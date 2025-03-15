package wal

import "os"

type WAL struct {
	directory      string
	currentSegment *os.File
}
