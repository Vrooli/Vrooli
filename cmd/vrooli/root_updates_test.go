package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/buildinfo"
)

func TestRootUpdateInspectionAndHelpDoNotInitializeRuntimeState(t *testing.T) {
	home, root := t.TempDir(), t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_SOURCE_ROOT", root)
	t.Setenv("VROOLI_ROOT", root)
	old := buildinfo.RuntimeHomeSnapshotBase64
	buildinfo.RuntimeHomeSnapshotBase64 = ""
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	var out, stderr bytes.Buffer
	if handled, code := runRootUpdateReadCommand([]string{"runtime", "update", "status", "--json"}, &out, &stderr); !handled || code != 0 {
		t.Fatalf("inspection failed: %d %s", code, stderr.String())
	}
	var status map[string]any
	if err := json.Unmarshal(out.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status["operation"] != nil || status["enrollment"] != nil {
		t.Fatal("fresh status invented admission/enrollment")
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatal("read-only update status created runtime files")
	}
	out.Reset()
	if handled, code := runRootUpdateReadCommand([]string{"runtime", "update", "--help"}, &out, &stderr); !handled || code != 0 || out.Len() == 0 {
		t.Fatal("update help unavailable")
	}
}
