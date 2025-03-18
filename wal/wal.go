package wal

import (
	"bufio"
	"context"
	"fmt"
	"hash/crc32"
	"log/slog"
	"os"
	"sync"
	"time"
)

type WAL struct {
	mu                *sync.RWMutex
	logSequenceNumber uint64
	currentSegment    int    // currentSegment keeps track of the segment numbers.
	maxFileSize       uint64 // maxFileSize will ensure the size of the file doesn't get too big.
	maxSegmentCount   int    // max number of segments in the wal directory
	filePrefix        string
	currentFile       *os.File
	buffer            *bufio.Writer
	currentFileName   string
	directory         string
	syncInterval      uint64
	syncTimer         *time.Timer
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewWAL(
	directory string,
	syncInterval uint64,
	maxFileSize uint64,
	maxSegmentSize int,
) *WAL {
	wal := &WAL{
		mu:              &sync.RWMutex{},
		currentSegment:  0,
		filePrefix:      "segment__",
		directory:       directory,
		syncInterval:    syncInterval,
		maxFileSize:     maxFileSize,
		maxSegmentCount: maxSegmentSize,
	}
	wal.currentFileName = wal.getCurrentSegmentFileName()

	// instantiate current file
	wal.createFile()

	// instantiate the ticker
	wal.syncTimer = time.NewTimer(time.Duration(wal.syncInterval))

	ctx, cancel := context.WithCancel(context.Background())
	wal.ctx, wal.cancel = ctx, cancel

	// start the sync timer.
	go wal.execSyncTicker()

	return wal
}

func (wal *WAL) createFile() {
	f, err := os.OpenFile(wal.currentFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	wal.currentFile = f
	if err != nil {
		slog.Error(fmt.Sprintf("unable to open file for %s", wal.currentFileName))
		panic(err)
	}
	// create in memory write buffer for the above created file.
	// this has to be changed every time a new file is created.
	wal.buffer = bufio.NewWriter(wal.currentFile)
}

func (wal *WAL) getCurrentSegmentFileName() string {
	return fmt.Sprintf("./%s/%s%d.log", wal.directory, wal.filePrefix, wal.currentSegment)
}

func (wal *WAL) execSyncTicker() {
	for {
		select {
		case <-wal.syncTimer.C:
			wal.mu.Lock()
			if err := wal.sync(); err != nil {
				slog.Error("error syncing files")
			}
			wal.mu.Unlock()
			wal.syncTimer.Reset(time.Duration(wal.syncInterval))
		case <-wal.ctx.Done():
			slog.Info("exiting timer")
			return
		}
	}
}

func (wal *WAL) Close() error {
	// call the cancel function to send an event to ctx.Done to
	// stop the timer.
	wal.cancel()

	// sync the file before closing
	if err := wal.sync(); err != nil {
		slog.Error("error syncing buffer and file")
		return err
	}

	if err := wal.currentFile.Close(); err != nil {
		slog.Error("error closing file")
		return err
	}

	return nil
}

// Write Logs to the current log file.
func (wal *WAL) WriteEntry(data []byte) error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	entry := &Entry{
		LogSequenceNumber: wal.logSequenceNumber,
		Data:              data,
		Crc32:             crc32.ChecksumIEEE(append(data, byte(wal.logSequenceNumber))),
	}
	v, err := MarshalEntry(*entry)
	if err != nil {
		slog.Error(fmt.Sprintf("unable marshal entry because: %v", err.Error()))
		return err
	}

	wal.writeAndRotateLogsIfRequired(v)

	wal.logSequenceNumber++
	return nil
}

func (wal *WAL) writeAndRotateLogsIfRequired(v []byte) error {
	// this function will first check the size of the current buffer size, if the
	// file size is greater than the maxFileSize, then we will create a new file.
	// while doing this, we will check if the total number of segments created is greater
	// than the max segment size, if yes, then delete the oldest file.

	// first check if a new record can be added to the current file.
	// if it can be added proceed to add the data to current file, else
	// create a new file and delete the oldest file.

	info, err := os.Stat(wal.currentFileName)
	if err != nil {
		slog.Error("error fetching file info stats")
		return err
	}

	if info.Size()+int64(wal.buffer.Buffered()) >= int64(wal.maxFileSize) {
		if err := wal.findAndDeleteOldestFileIfRequired(); err != nil {
			slog.Error("error when trying to delete oldest file")
			return err
		}

		// since the file size is maxed up, we can create a new file
		// and create a new buffer, before which should we fsync the file?
		if err := wal.sync(); err != nil {
			slog.Error("error when syncing file to disk and buffer")
			return err
		}

		wal.currentSegment++
		wal.currentFileName = wal.getCurrentSegmentFileName()
		wal.createFile()

		// once we create a new file check if we should delete any older files.
		if err := wal.findAndDeleteOldestFileIfRequired(); err != nil {
			slog.Error("error finding and deleting old files")
			return err
		}
	}

	// write entry to buffer
	_, err = wal.buffer.Write(v)
	if err != nil {
		slog.Error(fmt.Sprintf("enable to write string to buffer: %v", err.Error()))
		return err
	}

	return nil
}

func (wal *WAL) findAndDeleteOldestFileIfRequired() error {
	files, err := os.ReadDir(wal.directory)
	if err != nil {
		slog.Error("unable to fetch files from directory.")
		return err
	}

	var oldestFile os.DirEntry
	lastModified := time.Now().Unix()
	currentFiles := 0

	for _, file := range files {
		currentFiles++
		info, err := file.Info()
		if err != nil {
			slog.Error("unable to find file info")
			return err
		}
		modifiedTime := info.ModTime().Unix()
		if modifiedTime < lastModified {
			lastModified = modifiedTime
			oldestFile = file
		}
	}

	slog.Info(fmt.Sprintf("oldest file found: %s", oldestFile.Name()))

	if currentFiles > wal.maxSegmentCount {
		if err := os.Remove(oldestFile.Type().String()); err != nil {
			slog.Error("unable to delete file")
			return err
		}
	}

	return nil
}

func (wal *WAL) sync() error {
	// this function will write any data in the buffer to the file
	// and then will write it to the disk as well.
	if err := wal.buffer.Flush(); err != nil {
		slog.Error("error flusing buffer to file")
		return err
	}

	// now sync the file to the disk
	if err := wal.currentFile.Sync(); err != nil {
		slog.Error("error syncing file to disk")
		return err
	}

	return nil
}
