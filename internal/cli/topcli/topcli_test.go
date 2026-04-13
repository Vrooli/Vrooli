package topcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

func TestParseStatusRequestRejectsConflictingFilters(t *testing.T) {
	if _, err := ParseStatusRequest([]string{"--resources", "--scenarios"}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestRunInfoListJSONOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, repocontractmeta.ProjectConfigDir), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(repocontractmeta.InfoManifestPath(root), []byte(`{"files":["docs/context.md"]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "context.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write context: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunInfo(root, cliout.FormatJSON, InfoRequest{ListOnly: true}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunInfo: %v", err)
	}

	var payload struct {
		Success bool     `json:"success"`
		Root    string   `json:"root"`
		Files   []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if !payload.Success || payload.Root != root {
		t.Fatalf("root = %q", payload.Root)
	}
	if len(payload.Files) != 1 || payload.Files[0] != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("files = %v", payload.Files)
	}
}

func TestCollectInfoSourcesDetailedFallsBackOnInvalidManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, repocontractmeta.ProjectConfigDir), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(repocontractmeta.InfoManifestPath(root), []byte(`{"files":`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	files, warnings, err := CollectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("CollectInfoSourcesDetailed: %v", err)
	}
	if got, want := strings.Join(files, ","), strings.Join(DefaultInfoFiles, ","); got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Invalid info manifest") {
		t.Fatalf("warnings = %#v", warnings)
	}
}
