package kv

import (
	"errors"
	"sync"

	"github.com/aneesh-aparajit/walrus"
)

type KV struct {
	sync.RWMutex
	wal *walrus.WAL
	db  map[string][]byte
}

func NewKV(config *KVConfig) (*KV, error) {
	wal, err := walrus.NewWal(
		config.directory,
		config.maxFileSize,
		config.maxSegmentCount,
		config.enableFsync,
		config.checkpointingFrequency,
		config.syncInterval,
	)
	if err != nil {
		return nil, err
	}
	return &KV{
		wal: wal,
		db:  make(map[string][]byte),
	}, nil
}

func (kv *KV) Write(key string, val []byte) error {
	kv.Lock()
	defer kv.Unlock()

	data := &WalData{
		Operation: "SET",
		Key:       key,
		Value:     val,
	}
	buf, err := data.Marshal()
	if err != nil {
		return err
	}
	if err := kv.wal.WriteEntry(buf); err != nil {
		return err
	}

	kv.db[key] = val

	return nil
}

func (kv *KV) Delete(key string, val []byte) error {
	kv.Lock()
	defer kv.Unlock()

	data := &WalData{
		Operation: "DELETE",
		Key:       key,
		Value:     val,
	}
	buf, err := data.Marshal()
	if err != nil {
		return err
	}
	if err := kv.wal.WriteEntry(buf); err != nil {
		return err
	}

	delete(kv.db, key)

	return nil
}

func (kv *KV) Get(key string) ([]byte, error) {
	kv.RLock()
	defer kv.RUnlock()

	val, ok := kv.db[key]
	if !ok {
		return nil, errors.New("not found")
	}

	return val, nil
}
