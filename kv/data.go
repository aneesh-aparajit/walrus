package kv

import "encoding/json"

type WalData struct {
	Operation string
	Key       string
	Value     []byte
}

func (d *WalData) Marshal() ([]byte, error) {
	return json.Marshal(d)
}

func (d *WalData) Unmarshal(data []byte) error {
	return json.Unmarshal(data, d)
}
