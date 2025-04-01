package walrus

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fmt"
)

const (
	segmentPrefix string = "segment__"
)

type WAL struct {
	directory              string
	currSegment            *os.File
	buffer                 *bufio.Writer
	lock                   *sync.Mutex
	maxFileSize            int64
	maxSegmentCount        int
	lastSequenceNumber     uint64
	lastSegmentId          int
	enableFsync            bool
	checkpointingFrequency uint64
	syncInterval           time.Duration
	timer                  *time.Timer
	ctx                    context.Context
	cancel                 context.CancelFunc
}

func NewWal(
	directory string,
	maxFileSize int64,
	maxSegmentCount int,
	enableFsync bool,
	checkpointingFrequency uint64,
	syncInterval time.Duration) (*WAL, error) {
	// create directory to store the logs
	if err := os.MkdirAll(directory, os.ModeAppend); err != nil {
		return nil, err
	}

	lastSegmentId, err := findLastSegmentId(directory, maxFileSize)
	if err != nil {
		return nil, err
	}

	fileName := filepath.Join(directory, segmentPrefix+fmt.Sprint(lastSegmentId))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// seek to the end of the file
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	wal := &WAL{
		directory:              directory,
		currSegment:            file,
		buffer:                 bufio.NewWriter(file),
		lock:                   &sync.Mutex{},
		maxFileSize:            maxFileSize,
		lastSegmentId:          lastSegmentId,
		maxSegmentCount:        maxSegmentCount,
		checkpointingFrequency: checkpointingFrequency,
		timer:                  time.NewTimer(syncInterval),
		enableFsync:            enableFsync,
		syncInterval:           syncInterval,
		ctx:                    ctx,
		cancel:                 cancel,
	}

	wal.lastSequenceNumber, err = wal.findLastSequenceNumber()
	if err != nil {
		return nil, err
	}

	go wal.syncLoop()

	return wal, nil
}

func (wal *WAL) WriteEntry(data []byte) error {
	var isCkpt bool = false
	if wal.lastSequenceNumber%wal.checkpointingFrequency == 0 {
		isCkpt = true
	}

	return wal.writeEntry(data, isCkpt)
}

// this function will run forever in the background and will
// sync the buffer to the
func (wal *WAL) syncLoop() {
	for {
		select {
		case <-wal.ctx.Done():
			return
		case <-wal.timer.C:
			wal.lock.Lock()
			err := wal.Sync()
			wal.lock.Unlock()

			if err != nil {
				panic(err)
			}
		}
	}
}

// this function will sync the in memory buffer to the file.
func (wal *WAL) Sync() error {
	if err := wal.buffer.Flush(); err != nil {
		return err
	}

	if wal.enableFsync {
		if err := wal.currSegment.Sync(); err != nil {
			return err
		}
	}

	wal.timer.Reset(wal.syncInterval)

	return nil
}

// Replay all logs from the beginning.
func (wal *WAL) Replay() ([]*WalEntry, error) {
	return wal.readAllLogs()
}
