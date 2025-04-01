package kv

import "time"

type KVConfig struct {
	directory              string
	maxFileSize            int64
	maxSegmentCount        int
	enableFsync            bool
	checkpointingFrequency uint64
	syncInterval           time.Duration
}

func NewConfig(
	directory string,
	maxFileSize int64,
	maxSegmentCount int,
	enableFsync bool,
	checkpointingFrequency uint64,
	syncInterval int64,
) *KVConfig {
	return &KVConfig{
		directory:              directory,
		maxFileSize:            maxFileSize,
		maxSegmentCount:        maxSegmentCount,
		enableFsync:            enableFsync,
		checkpointingFrequency: checkpointingFrequency,
		syncInterval:           time.Duration(syncInterval),
	}
}
