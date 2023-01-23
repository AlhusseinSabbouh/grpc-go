package main

import (
	"context"
	"flag"
	"fmt"
	"grpc/pb"
	"grpc/service"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
)

func main() {

	port := flag.Int("port", 0, "the server port")
	flag.Parse()
	log.Printf("start sever on port %v", *port)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStroe("img")
	ratingStore := service.NewInMemoryRatingStore()
	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)

	userStore := service.NewInMemoryUserStore()

	err := seedUsers(userStore)
	if err != nil {
		log.Fatal("cannot seed users", err)
	}

	jwtManager := service.NewJwtManager(secretKey, tokenDuration)
	authServer := service.NewAuthServer(userStore, jwtManager)
	interceptor := service.NewAuthInterceptor(jwtManager, accessobleRoles())

	grpcServer := grpc.NewServer(
		// grpc.UnaryInterceptor(unaryInterceptor),
		// grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	)
	// grpcServer = grpc.NewServer(
	// 	grpc.UnaryInterceptor(unaryInterceptor),
	// 	grpc.StreamInterceptor(streamInterceptor),
	// )

	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	pb.RegisterAuthServiceServer(grpcServer, authServer)

	reflection.Register(grpcServer)

	address := fmt.Sprintf("localhost:%d", *port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatal("cannot start the server ", err)
	}

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start the server", err)
	}

}

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	log.Println("---> unary interceptor: ", info.FullMethod)
	// log.Println("---> type of request is ", reflect.TypeOf(req))
	// if k, ok := req.(*pb.CreateLaptopRequest); ok {
	// 	log.Println("---> laptop id request is ", k.Laptop.GetId())
	// }
	// log.Println("---> request is  : ", req)
	return handler(ctx, req)
}

func streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("---> unary interceptor: ", info.FullMethod)
	return handler(srv, stream)
}

func createUser(userStore service.UserStore, username, password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return err
	}
	return createUser(userStore, "user1", "secret", "user")
}

func accessobleRoles() map[string][]string {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"
	return map[string][]string{
		laptopServicePath + "CreateLaptop": {"admin"},
		laptopServicePath + "UploadImage":  {"admin"},
		laptopServicePath + "RateLaptop":   {"admin", "user"},
	}
}
