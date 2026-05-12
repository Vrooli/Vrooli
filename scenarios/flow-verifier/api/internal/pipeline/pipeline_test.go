package pipeline

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

type memFS struct {
	files map[string][]byte
}

func (m *memFS) MkdirAll(string, os.FileMode) error { return nil }
func (m *memFS) WriteFile(p string, data []byte, _ os.FileMode) error {
	m.files[p] = data
	return nil
}

func (m *memFS) ReadFile(p string) ([]byte, error) {
	v, ok := m.files[p]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: p, Err: fs.ErrNotExist}
	}
	return v, nil
}

func TestModeIdentifiersAreDistinct(t *testing.T) {
	if ModeGenerate == ModeCheck {
		t.Fatal("ModeGenerate and ModeCheck must be distinct")
	}
}

func TestAssertFreshReportsMissingArtifact(t *testing.T) {
	m := &memFS{files: map[string][]byte{}}
	err := AssertFresh(m, "/gen/model.qnt", []byte("(* model *)"), "demo.flow")
	if err == nil {
		t.Fatal("expected error when artifact is missing")
	}
	if !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("error should call out missing artifact, got: %v", err)
	}
	if !strings.Contains(err.Error(), "demo.flow") {
		t.Fatalf("error should include the flow id, got: %v", err)
	}
}

func TestAssertFreshReportsStaleArtifact(t *testing.T) {
	m := &memFS{files: map[string][]byte{"/gen/model.qnt": []byte("old")}}
	err := AssertFresh(m, "/gen/model.qnt", []byte("new"), "demo.flow")
	if err == nil {
		t.Fatal("expected error when artifact is stale")
	}
	if !strings.Contains(err.Error(), "is stale") {
		t.Fatalf("error should call out stale artifact, got: %v", err)
	}
}

func TestAssertFreshHappyPath(t *testing.T) {
	want := []byte("identical")
	m := &memFS{files: map[string][]byte{"/gen/model.qnt": want}}
	if err := AssertFresh(m, "/gen/model.qnt", want, "demo.flow"); err != nil {
		t.Fatalf("AssertFresh should accept identical content, got: %v", err)
	}
	var pathErr *fs.PathError
	if errors.As(nil, &pathErr) { // silence unused import check on builds with errors not used elsewhere
		_ = pathErr
	}
}
