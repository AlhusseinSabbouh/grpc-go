package service

import (
	"context"
	"grpc/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	userStore  UserStore
	jwtManager *JWTManager
	pb.UnimplementedAuthServiceServer
}

func NewAuthServer(userStore UserStore, jwtManager *JWTManager) *AuthServer {
	return &AuthServer{
		userStore:  userStore,
		jwtManager: jwtManager,
	}
}

func (server *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := server.userStore.Find(req.GetUsername())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot find the username , %v", err)
	}
	if user == nil || user.IsCorrectPassword(req.GetPassword()) == false {
		return nil, status.Errorf(codes.NotFound, "incorrect username or password %v", err)
	}

	token, err := server.jwtManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate the access token")
	}

	res := &pb.LoginResponse{
		AccessToken: token,
	}
	return res, nil
}
