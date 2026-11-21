package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestProcessSidecarFormatParticipatesInContract(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	const path = "internal/process/pid_sidecar.go"
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	writeTestFile(t, fixture, path, string(data))
	digest := func() string {
		t.Helper()
		d, err := wholeSourceDigest(fixture, statecontract.WholeSources["process-records"])
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	before := digest()
	changed := strings.Replace(string(data), `".pid"`, `".process"`, 1)
	if changed == string(data) {
		t.Fatal("PID sidecar name anchor missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("PID sidecar name change escaped process-records contract")
	}
	changed = strings.Replace(string(data), `"\n"`, `"\r\n"`, 1)
	if changed == string(data) {
		t.Fatal("PID sidecar encoding anchor missing")
	}
	writeTestFile(t, fixture, path, changed)
	if digest() == before {
		t.Fatal("PID sidecar encoding change escaped process-records contract")
	}
}
