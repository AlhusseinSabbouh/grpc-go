package serializer_test

import (
	"grpc/sample"
	"grpc/serializer"
	"testing"

	"grpc/pb"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFileSerializer(t *testing.T) {
	t.Parallel()
	binaryFile := "../tmp/laptop.bin"
	jsonFile := "../tmp/laptop.json"
	laptop1 := sample.NewLaptop()

	err := serializer.WriteProtobufToBinaryFile(laptop1, binaryFile)
	require.NoError(t, err)

	err = serializer.WriteProtobifToJSONFile(laptop1, jsonFile)
	require.NoError(t, err)

	laptop2 := &pb.Laptop{}
	err = serializer.ReadProtobufFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err)

	//

	// require.Equal(t, jsonT, json)

	require.True(t, proto.Equal(laptop1, laptop2))
}
