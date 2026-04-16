package tasks

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type fakeContext struct {
	postPath    string
	postPayload any
}

func (f *fakeContext) Get(path string, result interface{}) error { return nil }
func (f *fakeContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	return nil
}

func (f *fakeContext) Post(path string, payload interface{}, result interface{}) error {
	f.postPath = path
	f.postPayload = payload
	if out, ok := result.(*TaskCreateResponse); ok {
		out.Success = true
		out.Task = TaskResponse{ID: "task-1", Title: "created"}
	}
	return nil
}
func (f *fakeContext) Put(path string, payload interface{}, result interface{}) error { return nil }
func (f *fakeContext) Delete(path string) error                                       { return nil }
func (f *fakeContext) DeleteWithResult(path string, result interface{}) error         { return nil }

func TestApplyTaskContextFlags(t *testing.T) {
	tmp := t.TempDir()
	handoffDir, notesPath := writeHandoffPackage(t, tmp, "brief notes\n")

	req := &TaskCreateRequest{}
	err := applyTaskContextFlags(req, "", notesPath, taskOriginInput{
		source:      "swarm-manager",
		backlogItem: "idea/alpha",
		itemFolder:  "/tmp/ideas/alpha",
		handoffDir:  handoffDir,
	})
	if err != nil {
		t.Fatalf("applyTaskContextFlags() error = %v", err)
	}

	if req.Notes != "brief notes" {
		t.Fatalf("notes = %q", req.Notes)
	}
	if req.Origin == nil {
		t.Fatal("expected origin to be populated")
	}
	if req.Origin.HandoffManifestPath != filepath.Join(handoffDir, "manifest.json") {
		t.Fatalf("handoff manifest path = %q", req.Origin.HandoffManifestPath)
	}
}

func TestCmdAdd_PassesNotesAndOrigin(t *testing.T) {
	tmp := t.TempDir()
	handoffDir, notesPath := writeHandoffPackage(t, tmp, "brief notes\n")

	ctx := &fakeContext{}
	err := cmdAdd(ctx, []string{
		"--notes-file", notesPath,
		"--origin-source", "swarm-manager",
		"--origin-backlog-item", "idea/alpha",
		"--origin-item-folder", "/tmp/ideas/alpha",
		"--handoff-dir", handoffDir,
		"scenario", "alpha",
		"--json",
	})
	if err != nil {
		t.Fatalf("cmdAdd() error = %v", err)
	}

	if ctx.postPath != "/tasks" {
		t.Fatalf("post path = %q", ctx.postPath)
	}
	req, ok := ctx.postPayload.(TaskCreateRequest)
	if !ok {
		data, _ := json.Marshal(ctx.postPayload)
		t.Fatalf("unexpected payload type %T: %s", ctx.postPayload, string(data))
	}
	if req.Notes != "brief notes" {
		t.Fatalf("notes = %q", req.Notes)
	}
	if req.Origin == nil || req.Origin.BacklogItem != "idea/alpha" {
		t.Fatalf("origin = %#v", req.Origin)
	}
}

func TestApplyTaskContextFlags_AutoLoadsNotesFromHandoffDir(t *testing.T) {
	tmp := t.TempDir()
	handoffDir, _ := writeHandoffPackage(t, tmp, "brief from handoff\n")

	req := &TaskCreateRequest{}
	err := applyTaskContextFlags(req, "", "", taskOriginInput{
		source:      "swarm-manager",
		backlogItem: "idea/alpha",
		itemFolder:  "/tmp/ideas/alpha",
		handoffDir:  handoffDir,
	})
	if err != nil {
		t.Fatalf("applyTaskContextFlags() error = %v", err)
	}

	if req.Notes != "brief from handoff" {
		t.Fatalf("notes = %q", req.Notes)
	}
}

func writeHandoffPackage(t *testing.T, root, brief string) (string, string) {
	t.Helper()
	handoffDir := filepath.Join(root, "handoff")
	if err := os.MkdirAll(handoffDir, 0o755); err != nil {
		t.Fatalf("mkdir handoff: %v", err)
	}

	briefPath := filepath.Join(handoffDir, "brief.md")
	if err := os.WriteFile(briefPath, []byte(brief), 0o644); err != nil {
		t.Fatalf("write brief: %v", err)
	}
	if err := os.WriteFile(filepath.Join(handoffDir, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(handoffDir, "source-index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write source index: %v", err)
	}

	return handoffDir, briefPath
}
