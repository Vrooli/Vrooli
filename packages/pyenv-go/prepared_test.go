package pyenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyPreparedNeverRepairs(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "lock")
	raw := []byte("pinned fixture")
	os.WriteFile(lock, raw, 0600)
	spec := Spec{VenvDir: filepath.Join(dir, "venv"), LockFile: lock, BasePython: "3.12"}
	if _, err := VerifyPrepared(spec, hashBytes(raw)); err == nil {
		t.Fatal("missing prepared environment accepted")
	}
	if _, err := os.Stat(spec.VenvDir); !os.IsNotExist(err) {
		t.Fatal("created environment")
	}
	os.MkdirAll(filepath.Dir(InterpreterPath(spec.VenvDir)), 0700)
	os.WriteFile(InterpreterPath(spec.VenvDir), []byte("fixture"), 0700)
	os.WriteFile(sentinelPath(spec.VenvDir), []byte(hashBytes(raw)), 0600)
	if _, err := VerifyPrepared(spec, hashBytes(raw)); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(lock, []byte("changed"), 0600)
	if _, err := VerifyPrepared(spec, hashBytes(raw)); err == nil {
		t.Fatal("changed lock accepted")
	}
	data, _ := os.ReadFile(sentinelPath(spec.VenvDir))
	if string(data) != hashBytes(raw) {
		t.Fatal("sentinel repaired")
	}
}

func TestPreparedMetadataBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata")
	if err := os.WriteFile(path, make([]byte, (1<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPreparedFile(path, 1<<20); err == nil {
		t.Fatal("oversize lock accepted")
	}
	if err := os.WriteFile(path, make([]byte, 129), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPreparedFile(path, 128); err == nil {
		t.Fatal("oversize sentinel accepted")
	}
}
