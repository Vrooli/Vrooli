package stagedmessages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRegularFileBoundedRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	if err := os.WriteFile(path, []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readRegularFileBounded(path, 4); err == nil {
		t.Fatal("expected oversized file to be rejected")
	}
}

func TestReadRegularFileBoundedRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	path := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := readRegularFileBounded(path, 8); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
}

func TestReadRegularFileBoundedRejectsPathReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data")
	replacement := filepath.Join(dir, "replacement")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("replaced"), 0o600); err != nil {
		t.Fatal(err)
	}
	open := func(name string) (*os.File, error) {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		if err := os.Rename(replacement, name); err != nil {
			f.Close()
			return nil, err
		}
		return f, nil
	}
	if _, _, err := readRegularFileBoundedWithOpen(path, 32, open); err == nil {
		t.Fatal("expected replaced path to be rejected")
	}
}

func TestReadRegularFileBoundedRejectsSizeChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	if err := os.WriteFile(path, []byte("short"), 0o600); err != nil {
		t.Fatal(err)
	}
	open := func(name string) (*os.File, error) {
		f, err := os.OpenFile(name, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		if _, err := f.WriteAt([]byte(" appended"), 5); err != nil {
			f.Close()
			return nil, err
		}
		return f, nil
	}
	if _, _, err := readRegularFileBoundedWithOpen(path, 32, open); err == nil {
		t.Fatal("expected size change to be rejected")
	}
}
