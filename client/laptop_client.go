package client

import (
	"bufio"
	"context"
	"fmt"
	"grpc/pb"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LaptopClient struct {
	service pb.LaptopServiceClient
}

func NewLaptopClient(conn *grpc.ClientConn) *LaptopClient {
	service := pb.NewLaptopServiceClient(conn)
	return &LaptopClient{service: service}
}

func (laptopClient *LaptopClient) UploadImage(laptopId string, imagePath string, time1 time.Duration, userName string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open Image file", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	stream, err := laptopClient.service.UploadImage(ctx)
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

func (laptopClient *LaptopClient) CreateLaptop(laptop *pb.Laptop) {
	req := &pb.CreateLaptopRequest{
		Laptop: laptop,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res, err := laptopClient.service.CreateLaptop(ctx, req)

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

func (laptopClient *LaptopClient) SearchLaptop(filter *pb.Filter, userName string) {
	log.Print("we search on laptop with filter ", filter)

	req := &pb.SearchLaptopRequest{
		Filter:   filter,
		UserName: userName,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()
	stream, err := laptopClient.service.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatal("cannot search for laptop", err)
	}

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			return
		}
		if err != nil {
			// waitGroup.Done()
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

func (laptopClient *LaptopClient) RateLaptop(laptopIds []string, laptopScores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	stream, err := laptopClient.service.RateLaptop(ctx)
	if err != nil {
		return fmt.Errorf("cannot rate the laptop %v", err)
	}

	var waitGroup = sync.WaitGroup{}

	waitResponse := make(chan error)

	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				log.Print("no more responses")
				waitResponse <- nil
				return
			}
			if err != nil {
				waitResponse <- fmt.Errorf("cannot receive stream response")
				return
			}
			log.Print("received response: ", res)
		}
	}()

	waitGroup.Add(1)
	go func() {
		for i, laptopId := range laptopIds {
			req := &pb.RateLaptopRequest{
				LaptopId: laptopId,
				Score:    laptopScores[i],
			}
			err := stream.Send(req)
			if err != nil {
				waitResponse <- fmt.Errorf("cannot send stream response")
				waitGroup.Done()
				return
			}
			log.Print("sent request: , ", req)
		}
		err = stream.CloseSend()
		if err != nil {
			waitResponse <- fmt.Errorf("cannot send stream response")
			waitGroup.Done()
			return
		}
		waitGroup.Done()
	}()

	// for i, laptopId := range laptopIds {
	// 	req := &pb.RateLaptopRequest{
	// 		LaptopId: laptopId,
	// 		Score:    laptopScores[i],
	// 	}
	// 	err := stream.Send(req)
	// 	if err != nil {
	// 		return fmt.Errorf("cannot sent stream request")
	// 	}
	// 	log.Print("sent request: , ", req)

	// }

	waitGroup.Wait()
	err = <-waitResponse
	return err

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
