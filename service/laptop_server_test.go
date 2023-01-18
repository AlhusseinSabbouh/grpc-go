package service_test

import (
	"context"
	"grpc/pb"
	"grpc/sample"
	"grpc/service"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Case struct {
	name   string
	laptop *pb.Laptop
	store  service.LaptopStore
	code   codes.Code
}

type Cases []Case

func TestServerCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopNoId := sample.NewLaptop()
	laptopNoId.Id = ""

	laptopInvalidID := sample.NewLaptop()
	laptopInvalidID.Id = "invalid_id"

	laptopDuplicateID := sample.NewLaptop()
	storeDuplicateID := service.NewInMemoryLaptopStore()
	err := storeDuplicateID.Save(laptopDuplicateID)
	require.Nil(t, err)

	testCases := Cases{
		{
			name:   "succes_with_id",
			laptop: sample.NewLaptop(),
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "succes_without_id",
			laptop: laptopNoId,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "failure_invalid_id",
			laptop: laptopInvalidID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.InvalidArgument,
		},
		{
			name:   "failure_duplicate_id",
			laptop: laptopDuplicateID,
			store:  storeDuplicateID,
			code:   codes.AlreadyExists,
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := &pb.CreateLaptopRequest{
				Laptop: tc.laptop,
			}

			server := service.NewLaptopServer(tc.store, &service.DiskImageStore{}, &service.InMemoryRatingStore{})
			res, err := server.CreateLaptop(context.Background(), req)

			if tc.code == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.Id)
				if len(tc.laptop.Id) > 0 {
					require.Equal(t, tc.laptop.Id, res.Id)
				}
			} else {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.code, st.Code())
			}

		})
	}
}

// filter := &pb.Filter{
// 	MaxPriceUsd: 2000,
// 	MinCpuCores: 4,
// 	MinCpuGhz:   2.2,
// 	MinRam:      &pb.Memory{Value: 8, Unit: pb.Memory_GIGABYTE},
// }
