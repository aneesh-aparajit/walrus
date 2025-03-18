package wal

import "encoding/json"

type Entry struct {
	LogSequenceNumber uint64 `json:"logSequenceNumber"`
	Data              []byte `json:"data"`
	Crc32             uint32 `json:"crc32"`
}

// TODO: update this to deal directly with protobuf for better storage and performance
func MarshalEntry(e Entry) ([]byte, error) {
	return json.Marshal(e)
}

func Unmarshal(b []byte) (*Entry, error) {
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
