package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func findAndSortLogFiles(dir, file string) ([]string, error) {
	if dir == "" {
		dir = "."
	}

	fileExtension := filepath.Ext(file)
	filename := strings.TrimSuffix(file, fileExtension)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var logFiles []string
	for _, f := range files {
		if strings.HasPrefix(f.Name(), filename) && strings.HasSuffix(f.Name(), fileExtension) {
			logFiles = append(logFiles, f.Name())
		}
	}
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i] < logFiles[j]
	})
	return logFiles, nil
}

// readFileStream reads supply data from the specified file.
// It supports reading log rotated files.
func readFileStream(path, skipUntilFile string, errCh chan error) (<-chan interface{}, error) {
	dir, originalFile := filepath.Split(path)

	files, err := findAndSortLogFiles(dir, originalFile)
	if err != nil {
		return nil, fmt.Errorf("failed to list and sort log files: %v", err)
	}

	// Skip until the designated file
	skipping := true

	// If skipUntilFile is empty, don't skip any files
	if skipUntilFile == "" {
		skipping = false
	}

	linesCh := make(chan interface{}, 1024)

	go func() {
		defer close(linesCh)

		for _, fileName := range files {
			if skipping {
				if fileName == skipUntilFile {
					skipping = false
				}

				continue
			}

			waitForMore := fileName == originalFile
			processLogFile(fileName, waitForMore, linesCh, errCh)
		}
	}()

	return linesCh, nil
}

func processLogFile(fileName string, waitForMore bool, linesCh chan interface{}, errCh chan error) {
	file, err := os.Open(fileName)
	if err != nil {
		errCh <- fmt.Errorf("failed to open file %s: %v", fileName, err)
		return
	}
	defer file.Close()

	var pos int64
	scanner := bufio.NewScanner(file)
	for {
		for scanner.Scan() {
			var supply supplyInfo
			bytes := scanner.Bytes()
			if len(bytes) == 0 {
				continue
			}
			if err := json.Unmarshal(bytes, &supply); err != nil {
				errCh <- fmt.Errorf("error unmarshalling line: %v", err)
				return
			}
			linesCh <- supply

			// Update current position
			pos, _ = file.Seek(0, io.SeekCurrent)
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("error reading file: %v", err)
			return
		}

		if waitForMore {
			// EOF is reached; wait for new lines to be appended
			time.Sleep(1 * time.Second)

			// Seek to the last known position before continuing the loop
			_, err = file.Seek(pos, io.SeekStart)
			if err != nil {
				errCh <- fmt.Errorf("failed to seek in file: %v", err)
				return
			}

			// Reset scanner with the current file position
			scanner = bufio.NewScanner(file)

		} else {
			// Save state when we finish reading a file
			// skip the last file, where we "waitForMore"
			linesCh <- SaveLastParsedFile(fileName)

			return
		}
	}
}
