package walrus

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"
)

// this function finds the finds the lastly used LSN.
func (wal *WAL) findLastSequenceNumber() (uint64, error) {
	entry, err := wal.findLastEntry()
	if err != nil {
		return 0, err
	}
	return entry.LogSequenceNo, nil
}

// find the last entry of the current log segment.
func (wal *WAL) findLastEntry() (*WalEntry, error) {
	// we cannot use the same file ptr as the struct, because that points to the end of file.
	fileName := filepath.Join(wal.directory, segmentPrefix+fmt.Sprint(wal.lastSegmentId))
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	entry := &WalEntry{}

	for {
		// protobufs don't have delimters, so we have to read a fixed size.
		var size int32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			if err == io.EOF {
				break
			}
			return entry, err
		}

		buf := make([]byte, size)
		if _, err := io.ReadFull(file, buf); err != nil {
			return entry, err
		}

		if err := proto.Unmarshal(buf, entry); err != nil {
			return entry, nil
		}
	}

	return entry, nil
}

// write entry to the current log segment file.
func (wal *WAL) writeEntry(data []byte, isCkpt bool) error {
	// we will assume the file ptr will be at the end of the file.
	entry := wal.createEntry(data, isCkpt)
	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	if wal.buffer.Size()+len(data) >= int(wal.maxFileSize) {
		if err := wal.rotateLogs(); err != nil {
			return err
		}
	}

	// size protobufs don't have a delimiter, we'll have to write the
	// the size. We'll write to the in memory buffer.
	if err := binary.Write(wal.buffer, binary.LittleEndian, int32(len(data))); err != nil {
		return err
	}

	if _, err := wal.buffer.Write(data); err != nil {
		return err
	}

	return nil
}

// create an entry for the given data.
func (wal *WAL) createEntry(data []byte, isCkpt bool) *WalEntry {
	wal.lock.Lock()
	defer wal.lock.Unlock()

	entry := &WalEntry{
		LogSequenceNo: wal.lastSequenceNumber,
		Data:          data,
		Crc:           crc32.ChecksumIEEE(append(data, byte(wal.lastSequenceNumber))),
		IsCkpt:        &isCkpt,
	}

	wal.lastSequenceNumber++

	return entry
}

// rotateLogs: this function will simply delete the oldest file and ensure
// we have limited number of files in the log segment files.
func (wal *WAL) rotateLogs() error {
	files, err := filepath.Glob(filepath.Join(wal.directory, segmentPrefix+"*"))
	if err != nil {
		return err
	}

	oldestTime, oldestFile := time.Now(), ""

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			return err
		}
		if stat.ModTime().Before(oldestTime) {
			oldestFile = file
			oldestTime = stat.ModTime()
		}
	}

	// now we have the oldest file, which we can delete.
	if err := os.Remove(oldestFile); err != nil {
		return err
	}

	if err := wal.repairLog(); err != nil {
		return err
	}

	// now create a new file
	fileName := filepath.Join(wal.directory, segmentPrefix+fmt.Sprint(wal.lastSegmentId))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	wal.currSegment = file
	// now that we have created a new file, we need to flush the old buffer
	if err := wal.Sync(); err != nil {
		return err
	}
	wal.buffer = bufio.NewWriter(wal.currSegment)

	return nil
}

// we will repair log right before we create a new segment.
func (wal *WAL) repairLog() error {
	// let's say there is an issue with the log file
	// in that case, we simply truncate the entire file.
	// we do this just for the latest file.
	filePath := filepath.Join(wal.directory, segmentPrefix+fmt.Sprint(wal.lastSegmentId))
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}

	for {
		var size uint32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			return err
		}

		data := make([]byte, size)
		currentOffset, err := file.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}

		if _, err := io.ReadFull(file, data); err != nil {
			if err == io.EOF {
				return nil
			}
			// in case there is some error, then we will simply truncate the remaining file.
			if err := file.Truncate(currentOffset); err != nil {
				return err
			}
		}
	}
}

func (wal *WAL) readAllLogs() ([]*WalEntry, error) {
	entries := make([]*WalEntry, 0)
	files, err := filepath.Glob(filepath.Join(segmentPrefix + "*"))
	if err != nil {
		return nil, err
	}

	// assuming all the files in sorted order
	for _, file := range files {
		f, err := os.OpenFile(file, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}

		wals, err := wal.readAllLogsFromFile(f)
		if err != nil {
			return nil, err
		}

		entries = append(entries, wals...)
	}

	return entries, nil
}

func (wal *WAL) readAllLogsFromFile(file *os.File) ([]*WalEntry, error) {
	entries := make([]*WalEntry, 0)
	for {
		var size int32
		if err := binary.Read(file, binary.LittleEndian, &size); err != nil {
			return nil, err
		}

		data := make([]byte, size)
		if _, err := io.ReadFull(file, data); err != nil {
			if err == io.EOF {
				return entries, nil
			}
			return nil, err
		}

		log := &WalEntry{}
		if err := proto.Unmarshal(data, log); err != nil {
			return nil, err
		}

		entries = append(entries, log)
	}
}
