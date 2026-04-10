package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func resetFSSeams() {
	mkdirAllFn = os.MkdirAll
	createTempFn = os.CreateTemp
	chmodFn = os.Chmod
	renameFn = os.Rename
	removeFn = os.Remove
	writeFn = func(f *os.File, b []byte) (int, error) { return f.Write(b) }
	syncFn = func(f *os.File) error { return f.Sync() }
	closeFn = func(f *os.File) error { return f.Close() }
}

func TestWriteFileAtomic_CreateTempError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	createTempFn = func(dir, pattern string) (*os.File, error) {
		return nil, errors.New("create temp failed")
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected create temp error")
	}
}

func TestWriteFileAtomic_WriteError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	writeFn = func(f *os.File, b []byte) (int, error) {
		return 0, errors.New("write failed")
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected write error")
	}
}

func TestWriteFileAtomic_SyncError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	syncFn = func(f *os.File) error {
		return errors.New("sync failed")
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected sync error")
	}
}

func TestWriteFileAtomic_CloseError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	closeCalls := 0
	closeFn = func(f *os.File) error {
		closeCalls++
		if closeCalls == 1 {
			return errors.New("close failed")
		}
		return f.Close()
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected close error")
	}
}

func TestWriteFileAtomic_ChmodError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	chmodFn = func(name string, mode os.FileMode) error {
		return errors.New("chmod failed")
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected chmod error")
	}
}

func TestWriteFileAtomic_RenameError(t *testing.T) {
	resetFSSeams()
	defer resetFSSeams()

	renameFn = func(oldpath, newpath string) error {
		return errors.New("rename failed")
	}

	err := WriteFileAtomic(filepath.Join(t.TempDir(), "file.txt"), []byte("x"), 0)
	if err == nil {
		t.Fatalf("expected rename error")
	}
}
