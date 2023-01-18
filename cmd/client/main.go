package main

import (
	"bufio"
	"context"
	"flag"
	"grpc/pb"
	"grpc/sample"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var waitGroup = sync.WaitGroup{}

func main() {

	serverAdderss := flag.String("address", "", "the server address  : ")
	flag.Parse()
	log.Printf("dial server %s", *serverAdderss)

	conn, err := grpc.Dial(*serverAdderss, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}
	laptopClient := pb.NewLaptopServiceClient(conn)
	laptopClient1 := pb.NewLaptopServiceClient(conn)
	// laptopClient2 := pb.NewLaptopServiceClient(conn)
	// laptopClient3 := pb.NewLaptopServiceClient(conn)
	// laptopClient4 := pb.NewLaptopServiceClient(conn)
	// testCreateLaptop(laptopClient)
	// testSearchLaptop(laptopClient)
	waitGroup.Add(2)
	// go testUploadImage(laptopClient, time.Microsecond*1000, "user1")
	// go testUploadImage(laptopClient1, time.Microsecond*10, "user2")


	go testSearchLaptop(laptopClient, "user1")
	go testSearchLaptop(laptopClient1, "user2")


	waitGroup.Wait()

	log.Print("we end the recive data from user ")

}

func uploadImage(client pb.LaptopServiceClient, laptopId string, imagePath string, time1 time.Duration, userName string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open Image file", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	stream, err := client.UploadImage(ctx)
	if err != nil {
		log.Fatal("cannot create stream with the server")
	}

	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_ImageInfo{
			ImageInfo: &pb.ImageInfo{
				LaptopId:  laptopId,
				ImageType: filepath.Ext(imagePath),
				UserName:  userName,
			},
		},
	}
	err = stream.Send(req)
	if err != nil {
		log.Fatal("cannot send image info to server")
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {

		if err != nil {
			log.Fatal("cannot send more data")
		}

		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer ", err)
		}
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_Chunk{
				Chunk: buffer[:n],
			},
		}

		time.Sleep(time1)
		err = contextError(stream.Context())
		if err != nil {
			log.Fatal("some thing err ", err)
		}

		err = stream.Send(req)
		if err != nil {
			log.Fatal("cannot send chunk to server: ", err, stream.RecvMsg(nil))
		}
	}
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive the response", err)
	}

	log.Printf("image uploaded with id %s, and size %v", res.GetId(), res.GetSize())

}

func createLaptop(client pb.LaptopServiceClient, laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res, err := client.CreateLaptop(ctx, req)

	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Print("laptop already exists")
		} else {
			log.Fatal("cannot create laptop: ", err)
		}
		return
	}
	log.Printf("created laptop with id %s", res.GetId())

}

func searchLaptop(client pb.LaptopServiceClient, filter *pb.Filter, userName string) {
	log.Print("we search on laptop with filter ", filter)

	req := &pb.SearchLaptopRequest{
		Filter:   filter,
		UserName: userName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()
	stream, err := client.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatal("cannot search for laptop", err)
	}

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			return
		}
		if err != nil {
			waitGroup.Done()
			log.Fatal("cannot recive response : ", err)
		}
		laptop := res.GetLaptop()
		log.Print("- found: ", laptop.GetId())
		log.Print("  + brand: ", laptop.GetBrand())
		log.Print("  + name: ", laptop.GetName())
		log.Print("  + cpu cores: ", laptop.GetCpu().GetNumberCores())
		log.Print("  + cpu min ghz: ", laptop.GetCpu().GetMinGhz())
		log.Print("  + ram: ", laptop.GetRam())
		log.Print("  + price: ", laptop.GetPriceUsd())
	}

}

func testCreateLaptop(laptopClient pb.LaptopServiceClient) {
	createLaptop(laptopClient, sample.NewLaptop())
}

func testSearchLaptop(laptopClient pb.LaptopServiceClient, userName string) {
	for i := 0; i < 4; i++ {
		createLaptop(laptopClient, sample.NewLaptop())
	}
	filter := &pb.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 4,
		MinCpuGhz:   2.5,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}
	searchLaptop(laptopClient, filter, userName)
	waitGroup.Done()
}

func testUploadImage(laptopClient pb.LaptopServiceClient, times time.Duration, userName string) {
	laptop := sample.NewLaptop()
	createLaptop(laptopClient, laptop)
	uploadImage(laptopClient, laptop.GetId(), "tmp/laptop.jpg", times, userName)
	// uploadImage(laptopClient, laptop.GetId(), "tmp/laptop.jpg", time.Microsecond*500)
	waitGroup.Done()

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
