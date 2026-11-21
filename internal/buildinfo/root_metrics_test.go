package buildinfo

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestDirectRootInstallObservationOnlyForCanonicalTarget(t *testing.T) {
	home := useTestRebuildHome(t)
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	BeginDirectRootInstall(filepath.Join(t.TempDir(), "vrooli")).Event("terminal", "succeeded")
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(state, "root-rebaseline", "attempts")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("noncanonical target created telemetry: %v", err)
	}
	a := BeginDirectRootInstall(filepath.Join(bin, "vrooli"))
	if a == nil {
		t.Fatal("canonical target not observed")
	}
	a.Event("terminal", "succeeded")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("canonical observation: %v, %v", entries, err)
	}
}
