package serializer

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to binary file")
	}

	if err = os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("cannot write binary data to file")
	}
	return nil
}

func ReadProtobufFromBinaryFile(filename string, message proto.Message) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read binary data from file")
	}
	err = proto.Unmarshal(data, message)
	if err != nil {
		return fmt.Errorf("cannot unmarshal binary to proto message: %w", err)
	}
	return nil
}

func ProtobufToJSON(message proto.Message) (string, error) {
	json := protojson.MarshalOptions{
		Multiline:      true,
		UseEnumNumbers: false,
		UseProtoNames:  true,
		Indent:         "  ",
	}
	marshaller, err := json.Marshal(message)

	if err != nil {
		return "something is error", err
	}
	return string(marshaller), nil
}

func WriteProtobifToJSONFile(message proto.Message, filename string) error {
	data, err := ProtobufToJSON(message)
	if err != nil {
		return fmt.Errorf("cannot marshat proto message")
	}
	err = os.WriteFile(filename, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("cannot write JSON data to file")
	}
	return nil
}
