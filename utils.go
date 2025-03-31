package walrus

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func findLastSegmentId(directory string, maxFileSize int64) (int, error) {
	var lastSegmentId int
	files, err := filepath.Glob(filepath.Join(directory, fmt.Sprintf("%s*"), segmentPrefix))
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		segments := strings.Split(file, segmentPrefix)
		currSegmentId, err := strconv.Atoi(segments[len(segments)-1])
		if err != nil {
			return 0, err
		}
		if currSegmentId > lastSegmentId {
			// now, if the current file size is greater than the maxFileSize
			// then we'll have to create a new file.
			stat, err := os.Stat(file)
			if err != nil {
				return 0, err
			}

			if stat.Size() < maxFileSize {
				lastSegmentId = currSegmentId
			} else {
				lastSegmentId = currSegmentId + 1
			}
		}
	}

	return lastSegmentId, nil
}
