package service

import (
	"bytes"
	"context"
	"errors"
	"grpc/pb"
	"io"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type laptopServer struct {
	store       LaptopStore
	imageStore  ImageStore
	ratingStore RatingStore
	pb.UnimplementedLaptopServiceServer
}

func NewLaptopServer(store LaptopStore, imageStore ImageStore, ratingStore RatingStore) *laptopServer {
	return &laptopServer{store: store, imageStore: imageStore, ratingStore: ratingStore}
}

func (server *laptopServer) RateLaptop(stream pb.LaptopService_RateLaptopServer) error {
	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		if err != nil {
			return status.Error(codes.Unknown, "something got error while receiving stream")
		}

		laptopId := req.GetLaptopId()
		score := req.GetScore()

		log.Printf("we go a laptop with id : %v , and the score is %v ", laptopId, score)
		found, err := server.store.Find(laptopId)
		if err != nil || found == nil {
			return status.Error(codes.NotFound, "we cannot find the laptop with this id")
		}

		rating, err := server.ratingStore.Add(laptopId, score)

		if err != nil {
			return status.Error(codes.Internal, "cannot add the rate to laptop")
		}
		res := &pb.RateLaptopResponse{
			LaptopId:     laptopId,
			AverageSocre: rating.Avg(),
			RatedCount:   rating.Count,
		}

		err = stream.Send(res)

		if err != nil {
			return status.Error(codes.Unknown, "connot send the response to the client")
		}

	}

	return nil
}

const maxImageSize = 1 << 20

func (server *laptopServer) UploadImage(stream pb.LaptopService_UploadImageServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot recive a metadata")
	}
	laptopId := req.GetImageInfo().GetLaptopId()
	imageType := req.GetImageInfo().GetImageType()

	log.Print("we receive an upload image request for laptop id is : ", laptopId)

	laptop, err := server.store.Find(laptopId)
	if laptop == nil || err != nil {
		return status.Error(codes.NotFound, "we cannot find the laptop")
	}
	imageSize := 0
	imageData := bytes.Buffer{}

	userName := req.GetImageInfo().GetUserName()

	for {

		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		log.Print("waiting to receive more data from client", userName)

		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("we got all the data from user ", userName)

			break
		}
		if err != nil {
			return status.Error(codes.Internal, "something got error while receiveing the chunks")
		}

		imageSize += len(req.GetChunk())
		if imageSize > maxImageSize {
			return logError(status.Error(codes.InvalidArgument, "we reach to the maximum size for one image"))
		}

		_, err = imageData.Write(req.GetChunk())
		if err != nil {
			return status.Error(codes.Internal, "we can't wrtie to buffer")
		}

	}
	imageId, err := server.imageStore.Save(laptopId, imageType, imageData)
	if err != nil {
		return status.Error(codes.Internal, "cannot save image to internal storage")
	}

	res := &pb.UploadImageResponse{
		Id:   imageId,
		Size: uint32(imageSize),
	}

	err = stream.SendAndClose(res)
	if err != nil {
		return status.Error(codes.Unknown, "cannot close the stream")
	}

	log.Print("saved image with id : ", imageId)
	return nil

}

func (server *laptopServer) SearchLaptop(req *pb.SearchLaptopRequest, stream pb.LaptopService_SearchLaptopServer) error {
	filter := req.GetFilter()
	userName := req.GetUserName()
	log.Printf("receive a serach laptop request with server : %v", filter)

	err := server.store.Search(
		stream.Context(),
		filter,
		func(laptop *pb.Laptop) error {
			res := &pb.SearchLaptopResponse{
				Laptop: laptop,
			}
			t := time.Duration(rand.Intn(10) * int(time.Second))
			log.Printf("user: %v you will wait for %v second and the time is %v", userName, t, time.Now().Second())

			time.Sleep(t)

			err := stream.Send(res)
			if err != nil {
				return err
			}
			log.Printf("send a laptop response : to user %v", userName)

			log.Printf("we send to user %v  whos wait for %v second , and the time is %v", userName, t, time.Now().Second())

			return nil
		},
	)

	if err != nil {
		return status.Error(codes.Internal, "unexpected error")
	}

	return nil
}

func (server *laptopServer) CreateLaptop(ctx context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()
	log.Print("we got a create laptop request with laptop request is : &v", laptop.GetId())

	if len(laptop.GetId()) > 0 {
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop Id is not a valid uuid , %v", err)
		}
	} else {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate uuid for the new laptop")
		}

		laptop.Id = id.String()
	}

	if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
		log.Printf("error in the connection")
		return nil, status.Error(codes.Canceled, "the connection cut of")
	}

	err := server.store.Save(laptop)
	if err != nil {
		code := codes.Internal
		if errors.Is(err, ErrLaptopAlradyExists) {
			code = codes.AlreadyExists
		}

		return nil, status.Error(code, "cannot save the laptop")
	}

	res := &pb.CreateLaptopResponse{
		Id: laptop.Id,
	}
	return res, nil

}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
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
