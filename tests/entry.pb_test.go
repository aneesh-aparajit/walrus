package tests

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"testing"

	"github.com/aneesh-aparajit/walrus"
	"google.golang.org/protobuf/proto"
)

func writeProtobufMessage(filename string, messages []*walrus.WalEntry) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for _, message := range messages {
		data, err := proto.Marshal(message)
		if err != nil {
			panic(err)
		}
		if err := binary.Write(file, binary.LittleEndian, int32(len(data))); err != nil {
			panic(err)
		}
		if _, err := file.Write(data); err != nil {
			panic(err)
		}
	}
}

func readProtobufFromFile(filename string) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}

	for {
		var size int32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(file, data); err != nil {
			panic(err)
		}
		e := &walrus.WalEntry{}
		if err := proto.Unmarshal(data, e); err != nil {
			panic(err)
		}
	}

}

func TestProtoBufReadWrite(t *testing.T) {
	entries := make([]*walrus.WalEntry, 0)
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("Hello World #%d", i))
		entries = append(entries, &walrus.WalEntry{
			LogSequenceNo: uint64(i),
			Data:          data,
			Crc:           crc32.ChecksumIEEE(data),
			IsCkpt:        nil,
		})
	}

	writeProtobufMessage("./file.dat", entries)
	readProtobufFromFile("./file.dat")
}
