package service_test

import (
	"bufio"
	"context"
	"fmt"
	"grpc/pb"
	"grpc/sample"
	"grpc/serializer"
	"grpc/service"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestClientCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStroe("img")
	ratingSore := service.NewInMemoryRatingStore()
	address := startTestLaptopServer(t, laptopStore, imageStore, ratingSore)
	client := newTestLaptopClient(t, address)

	laptopSample := sample.NewLaptop()
	expectedId := laptopSample.GetId()

	req := &pb.CreateLaptopRequest{
		Laptop: laptopSample,
	}

	res, err := client.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedId, res.Id)

	other, err := laptopStore.Find("dfd")
	st, ok := status.FromError(err)
	require.Nil(t, other)
	require.True(t, ok)
	require.Equal(t, st.Code(), codes.NotFound)

	other1, _ := laptopStore.Find(laptopSample.Id)
	requireSameLaptop(t, laptopSample, other1)
	// requireSameLaptop(t, req.Laptop, other1)

}

func requireSameLaptop(t *testing.T, laptop1 *pb.Laptop, laptop2 *pb.Laptop) {
	json1, err := serializer.ProtobufToJSON(laptop1)
	require.NoError(t, err)

	json2, err := serializer.ProtobufToJSON(laptop2)
	require.NoError(t, err)
	require.Equal(t, json1, json2)

}

// function that start grpc server and return the network address
func startTestLaptopServer(t *testing.T, store service.LaptopStore, imageStore service.ImageStore, ratingStore service.RatingStore) string {
	// laptopStore := service.NewInMemoryLaptopStore()
	laptopServer := service.NewLaptopServer(store, imageStore, ratingStore)
	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)
	listener, err := net.Listen("tcp", ":8080")
	require.NoError(t, err)
	go grpcServer.Serve(listener)
	return listener.Addr().String()
}

func newTestLaptopClient(t *testing.T, serverAddress string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn)
}

func TestClientSearchLaptop(t *testing.T) {
	t.Parallel()
	filter := &pb.Filter{
		MaxPriceUsd: 2000,
		MinCpuCores: 4,
		MinCpuGhz:   2.2,
		MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
	}

	store := service.NewInMemoryLaptopStore()

	expectedIDs := make(map[string]bool)

	for i := 0; i < 6; i++ {
		laptop := sample.NewLaptop()

		switch i {
		case 0:
			laptop.PriceUsd = 2500
		case 1:
			laptop.Cpu.NumberCores = 2
		case 2:
			laptop.Cpu.MinGhz = 2.0
		case 3:
			laptop.Ram = &pb.Memory{Value: 4096, Unit: pb.Memory_MEGABYTE}
		case 4:
			laptop.PriceUsd = 1999
			laptop.Cpu.NumberCores = 4
			laptop.Cpu.MinGhz = 2.5
			laptop.Cpu.MaxGhz = laptop.Cpu.MinGhz + 2.0
			laptop.Ram = &pb.Memory{Value: 16, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		case 5:
			laptop.PriceUsd = 2000
			laptop.Cpu.NumberCores = 6
			laptop.Cpu.MinGhz = 2.8
			laptop.Cpu.MaxGhz = laptop.Cpu.MinGhz + 2.0
			laptop.Ram = &pb.Memory{Value: 64, Unit: pb.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		}

		err := store.Save(laptop)
		require.NoError(t, err)
	}

	serverAddress := startTestLaptopServer(t, store, &service.DiskImageStore{}, &service.InMemoryRatingStore{})
	client := newTestLaptopClient(t, serverAddress)

	req := &pb.SearchLaptopRequest{
		Filter: filter,
	}

	stream, err := client.SearchLaptop(context.Background(), req)
	require.NoError(t, err)

	var found int = 0
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Contains(t, expectedIDs, res.GetLaptop().GetId())

		found += 1
	}
	require.Equal(t, len(expectedIDs), found)

}

func TestClientUploadImage(t *testing.T) {
	t.Parallel()

	testImageFolder := "../tmp"
	imageStore := service.NewDiskImageStroe(testImageFolder)
	laptopStore := service.NewInMemoryLaptopStore()

	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	serverAddress := startTestLaptopServer(t, laptopStore, imageStore, &service.InMemoryRatingStore{})
	laptopClient := newTestLaptopClient(t, serverAddress)

	imagePath := fmt.Sprintf("%s/laptop.jpg", testImageFolder)
	file, err := os.Open(imagePath)
	if err == nil {
		defer file.Close()
	}
	require.NoError(t, err)

	stream, err := laptopClient.UploadImage(context.Background())
	require.NoError(t, err)

	imageType := filepath.Ext(imagePath)
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_ImageInfo{
			ImageInfo: &pb.ImageInfo{
				LaptopId:  laptop.GetId(),
				ImageType: imageType,
			},
		},
	}
	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		size += n
		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_Chunk{
				Chunk: buffer[:n],
			},
		}
		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotZero(t, res.GetId())
	require.EqualValues(t, size, res.GetSize())

	savedImagePath := fmt.Sprintf("%s/%s%s", testImageFolder, res.GetId(), imageType)
	require.FileExists(t, savedImagePath)
	require.NoError(t, os.Remove(savedImagePath))

}

func TestClientRateLaptop(t *testing.T) {
	t.Parallel()

	laptopStore := service.NewInMemoryLaptopStore()
	ratingStore := service.NewInMemoryRatingStore()

	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	l, err := laptopStore.Find(laptop.GetId())
	require.NoError(t, err)
	require.NotNil(t, l)

	serverAddress := startTestLaptopServer(t, laptopStore, &service.DiskImageStore{}, ratingStore)
	laptopClient := newTestLaptopClient(t, serverAddress)

	stream, err := laptopClient.RateLaptop(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, stream)

	scores := []float64{8, 7.5, 10}

	n := len(scores)
	for i := 0; i < n; i++ {
		req := &pb.RateLaptopRequest{
			LaptopId: laptop.Id,
			Score:    scores[i],
		}
		err := stream.Send(req)
		require.NoError(t, err)
	}
	err = stream.CloseSend()
	require.NoError(t, err)

	averages := []float64{8, 7.75, 8.5}

	for idx := 0; ; idx++ {
		res, err := stream.Recv()
		if err == io.EOF {
			require.Equal(t, n, idx)
			return
		}
		require.NoError(t, err)
		require.Equal(t, laptop.GetId(), res.GetLaptopId())
		require.Equal(t, uint32(idx+1), res.GetRatedCount())
		s := res.GetAverageSocre()
		require.Equal(t, averages[idx], s)
	}

}
