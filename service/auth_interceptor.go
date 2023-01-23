package service

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	jwtManager      *JWTManager
	accessibleRoles map[string][]string
}

func NewAuthInterceptor(jwtManager *JWTManager, accessableRoles map[string][]string) *AuthInterceptor {
	return &AuthInterceptor{
		jwtManager:      jwtManager,
		accessibleRoles: accessableRoles,
	}
}
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		log.Println("---> unary interceptor: ", info.FullMethod)

		err = interceptor.authorize(ctx, info.FullMethod)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		log.Println("---> unary interceptor: ", info.FullMethod)

		err := interceptor.authorize(stream.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		return handler(srv, stream)
	}

}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {

	accessibleRoles, ok := interceptor.accessibleRoles[method]
	if !ok {
		// every one can access to this rpc
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token not found")
	}

	accessToken := values[0]
	claims, err := interceptor.jwtManager.Verify(accessToken)

	if err != nil {
		return status.Errorf(codes.Unauthenticated, "access token is invalid")
	}

	for _, role := range accessibleRoles {
		if role == claims.Role {
			return nil
		}
	}

	return status.Error(codes.PermissionDenied, "no permission to access to this rpc")

}
