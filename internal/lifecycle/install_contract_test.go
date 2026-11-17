package lifecycle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func TestInstallDigestTracksOperationAndRejectsLegacyRecord(t *testing.T) {
	root := t.TempDir()
	input := []byte("module example\n")
	if err := os.WriteFile(filepath.Join(root, "go.mod"), input, 0o644); err != nil {
		t.Fatal(err)
	}
	component := scenario.Component{Build: scenario.ComponentBuild{Kind: "go_module"}}
	spec := BuilderSpec{Kind: "go_module", Install: []string{"go", "mod", "download"}, InstallInputs: []string{"go.mod"}}
	// This is the previous reader's digest calculation for this single-file fixture.
	h := sha256.New()
	fmt.Fprintf(h, "go.mod\x00%d\x00", len(input))
	h.Write(input)
	legacy := fmt.Sprintf("%x", h.Sum(nil))
	data, err := json.Marshal(installDigestRecord{Digest: legacy})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installDigestPath(root, component, spec), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if needed, _, err := installNeeded(root, root, component, spec); err != nil || !needed {
		t.Fatalf("legacy record: needed=%v err=%v", needed, err)
	}
	if err := recordInstallDigest(root, root, component, spec); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(installDigestPath(root, component, spec))
	if err != nil {
		t.Fatal(err)
	}
	var current installDigestRecord
	if err := json.Unmarshal(data, &current); err != nil {
		t.Fatal(err)
	}
	if current.Digest == legacy {
		t.Fatal("previous reader would accept the new record")
	}
	if needed, _, err := installNeeded(root, root, component, spec); err != nil || needed {
		t.Fatalf("unchanged record: needed=%v err=%v", needed, err)
	}
	changed := spec
	changed.Install = []string{"go", "mod", "download", "-x"}
	if needed, _, err := installNeeded(root, root, component, changed); err != nil || !needed {
		t.Fatalf("changed command: needed=%v err=%v", needed, err)
	}
	changed = spec
	changed.Environment = map[string]string{"GOPROXY": "off"}
	if needed, _, err := installNeeded(root, root, component, changed); err != nil || !needed {
		t.Fatalf("changed environment: needed=%v err=%v", needed, err)
	}
}
