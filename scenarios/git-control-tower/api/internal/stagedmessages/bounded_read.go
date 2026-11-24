package stagedmessages

import (
	"errors"
	"fmt"
	"io"
	"os"
)

var errBoundedReadLimit = errors.New("file exceeds bounded read limit")

// readRegularFileBounded reads at most maxBytes+1 bytes from a regular file.
// It verifies that the path still names the opened file and that neither the
// file's size nor its observable metadata changed during the read.
func readRegularFileBounded(path string, maxBytes int64) ([]byte, os.FileInfo, error) {
	return readRegularFileBoundedWithOpen(path, maxBytes, os.Open)
}

func readRegularFileBoundedWithOpen(path string, maxBytes int64, open func(string) (*os.File, error)) ([]byte, os.FileInfo, error) {
	if maxBytes < 0 || maxBytes == int64(^uint64(0)>>1) {
		return nil, nil, fmt.Errorf("invalid bounded read limit")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("file is not a regular file")
	}
	if before.Size() > maxBytes {
		return nil, nil, errBoundedReadLimit
	}
	f, err := open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || !sameReadMetadata(before, opened) {
		return nil, nil, fmt.Errorf("file changed before bounded read")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, nil, errBoundedReadLimit
	}
	afterOpen, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	afterPath, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !afterPath.Mode().IsRegular() || !os.SameFile(opened, afterOpen) || !os.SameFile(opened, afterPath) || !sameReadMetadata(opened, afterOpen) || !sameReadMetadata(opened, afterPath) || int64(len(data)) != opened.Size() {
		return nil, nil, fmt.Errorf("file changed during bounded read")
	}
	return data, opened, nil
}

func sameReadMetadata(a, b os.FileInfo) bool {
	return a.Size() == b.Size() && a.Mode() == b.Mode() && a.ModTime().Equal(b.ModTime())
}
