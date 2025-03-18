package kv

type KV interface {
	Put(string, []byte) error
	Get(string) ([]byte, error)
	Delete(string) error
}
