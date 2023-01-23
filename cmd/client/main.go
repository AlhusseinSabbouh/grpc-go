package main

import (
	"context"
	"flag"
	"fmt"
	"grpc/client"
	"grpc/pb"
	"grpc/sample"
	"log"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var waitGroup = sync.WaitGroup{}

const (
	username = "user1"
	password = "secret"
)

func main() {

	serverAdderss := flag.String("address", "", "the server address  : ")
	flag.Parse()
	log.Printf("dial server %s", *serverAdderss)

	conn1, err := grpc.Dial(*serverAdderss, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	// laptopClient1 := pb.NewLaptopServiceClient(conn)
	// laptopClient2 := pb.NewLaptopServiceClient(conn)
	// laptopClient3 := pb.NewLaptopServiceClient(conn)
	// laptopClient4 := pb.NewLaptopServiceClient(conn)
	// testCreateLaptop(laptopClient)
	// testSearchLaptop(laptopClient)

	refreshDuration := time.Second * 15

	authClient := client.NewAuthClient(conn1, username, password)
	authInterceptor, err := client.NewAuthInterceptor(authClient, authMethods(), refreshDuration)

	conn2, err := grpc.Dial(*serverAdderss, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(authInterceptor.Unary()),
		grpc.WithStreamInterceptor(authInterceptor.Stream()),
	)

	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}
	laptopClient := client.NewLaptopClient(conn2)
	//! test rate Laptop
	testCreateLaptop(laptopClient)

	// waitGroup.Add(2)
	//! test Upload Image
	// go testUploadImage(laptopClient, time.Microsecond*1000, "user1")
	// go testUploadImage(laptopClient1, time.Microsecond*10, "user2")

	//! test Search Laptop
	// go testSearchLaptop(laptopClient, "user1")
	// go testSearchLaptop(laptopClient1, "user2")

	waitGroup.Wait()

	log.Print("we end the recive data from user ")
	//
}

func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptopClient.CreateLaptop(sample.NewLaptop())
}

func testSearchLaptop(laptopClient *client.LaptopClient, userName string) {
	for i := 0; i < 4; i++ {
		laptopClient.CreateLaptop(sample.NewLaptop())
	}
	filter := &pb.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 4,
		MinCpuGhz:   2.5,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	laptopClient.SearchLaptop(filter, userName)
	waitGroup.Done()
}

func testUploadImage(laptopClient *client.LaptopClient, times time.Duration, userName string) {
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.GetId(), "tmp/laptop.jpg", times, userName)
	// uploadImage(laptopClient, laptop.GetId(), "tmp/laptop.jpg", time.Microsecond*500)
	waitGroup.Done()
}

func testRateLaptop(client *client.LaptopClient) {
	n := 3
	laptopIDs := make([]string, 3)
	for i := 0; i < n; i++ {
		laptop := sample.NewLaptop()
		laptopIDs[i] = laptop.GetId()
		client.CreateLaptop(laptop)
	}
	scores := make([]float64, n)

	for {
		fmt.Println("rate laptop (y/n)? ")
		var answer string
		fmt.Scan(&answer)
		if strings.ToLower(answer) != "y" {
			break
		}

		for i := 0; i < n; i++ {
			scores[i] = sample.RandomLaptopScore()
		}

		err := client.RateLaptop(laptopIDs, scores)
		if err != nil {
			log.Fatal(err)
		}

	}
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Error(codes.Canceled, "request cancelled")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "deadline exceede")
	default:
		return nil
	}
}

func authMethods() map[string]bool {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"

	return map[string]bool{
		laptopServicePath + "CreateLaptop": true,
		laptopServicePath + "UploadImage":  true,
		laptopServicePath + "RateLaptop":   true,
	}
}
