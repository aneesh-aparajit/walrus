package kv

import (
	"fmt"
	"log/slog"
	"sync"
)

type KVStore struct {
	mu *sync.RWMutex
	db map[string][]byte
}

func (kv *KVStore) Put(key string, val []byte) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if _, ok := kv.db[key]; ok {
		slog.Info(fmt.Sprintf("key %s found in db. replacing value", key))
	}

	kv.db[key] = val
}

func (kv *KVStore) Get(key string) ([]byte, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	val, ok := kv.db[key]
	if ok {
		return val, nil
	}
	return nil, fmt.Errorf("key %s not found", key)
}

func (kv *KVStore) Delete(key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	_, ok := kv.db[key]
	if ok {
		delete(kv.db, key)
	}
	return fmt.Errorf("key %s not found", key)
}
