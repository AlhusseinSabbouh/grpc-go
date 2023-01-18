package main

import (
	"flag"
	"fmt"
	"grpc/pb"
	"grpc/service"
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {

	port := flag.Int("port", 0, "the server port")
	flag.Parse()
	log.Printf("start sever on port %v", *port)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStroe("img")
	laptopServer := service.NewLaptopServer(laptopStore, imageStore)

	grpcServer := grpc.NewServer()

	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)

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
