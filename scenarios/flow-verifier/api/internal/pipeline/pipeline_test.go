package pipeline

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/flows/kinds/temporal/model"
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

func TestRunAttachesFlowIdentityToPreflightFailure(t *testing.T) {
	root := t.TempDir()
	first := model.Flow{
		FlowID: "first.flow",
		Layout: layout.Layout{
			BaseDir:        "first/flow",
			TransitionPath: "first/flow/transition.go",
			TestPath:       "first/flow/flow_test.go",
		},
	}
	second := model.Flow{
		FlowID: "second.flow",
		Layout: layout.Layout{
			BaseDir:        "second/flow",
			TransitionPath: "second/flow/transition.go",
			TestPath:       "second/flow/flow_test.go",
		},
	}
	for _, rel := range []string{first.Layout.TransitionPath, first.Layout.TestPath} {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("// fixture\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	err := Run(context.Background(), Options{Root: root, Flows: []model.Flow{first, second}, Mode: ModeCheck})
	if err == nil {
		t.Fatal("expected second flow preflight to fail")
	}
	var flowErr *flowExecutionError
	if !errors.As(err, &flowErr) {
		t.Fatalf("expected flowExecutionError, got %T: %v", err, err)
	}
	if flowErr.flowID != second.FlowID {
		t.Fatalf("flow error identity = %q, want %q", flowErr.flowID, second.FlowID)
	}
}

func TestAssertFreshReportsMissingArtifact(t *testing.T) {
	m := &memFS{files: map[string][]byte{}}
	err := AssertFresh(m, "/gen/model.qnt", []byte("(* model *)"), "demo.flow")
	if err == nil {
		t.Fatal("expected error when artifact is missing")
	}
	fe, ok := AsFreshnessError(err)
	if !ok {
		t.Fatalf("expected *FreshnessError, got %T: %v", err, err)
	}
	if fe.Kind != FreshnessMissing {
		t.Fatalf("expected FreshnessMissing, got %q", fe.Kind)
	}
	if len(fe.Missing) != 1 || fe.Missing[0] != "model.qnt" {
		t.Fatalf("expected Missing=[model.qnt], got %v", fe.Missing)
	}
	if fe.FlowID != "demo.flow" {
		t.Fatalf("expected FlowID=demo.flow, got %q", fe.FlowID)
	}
	// Error() still mentions the flow id so CLI output stays informative.
	if !strings.Contains(err.Error(), "demo.flow") {
		t.Fatalf("error message should include flow id, got: %v", err)
	}
}

func TestAssertFreshReportsStaleArtifact(t *testing.T) {
	m := &memFS{files: map[string][]byte{"/gen/model.qnt": []byte("old")}}
	err := AssertFresh(m, "/gen/model.qnt", []byte("new"), "demo.flow")
	if err == nil {
		t.Fatal("expected error when artifact is stale")
	}
	fe, ok := AsFreshnessError(err)
	if !ok {
		t.Fatalf("expected *FreshnessError, got %T: %v", err, err)
	}
	if fe.Kind != FreshnessStale {
		t.Fatalf("expected FreshnessStale, got %q", fe.Kind)
	}
	if len(fe.Stale) != 1 || fe.Stale[0] != "model.qnt" {
		t.Fatalf("expected Stale=[model.qnt], got %v", fe.Stale)
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
