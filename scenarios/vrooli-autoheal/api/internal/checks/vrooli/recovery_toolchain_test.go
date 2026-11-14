package vrooli

import (
	"path/filepath"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// langrecover copies the toolchain table because it must stay stdlib-only.
// The copy must still find every tool the rendered unit's PATH would: each
// platform-go entry, including the contract-resolved runtime-home bin, has
// to appear in the recovery table. Windows is rendered with backslashes on
// any host, so only the slash-separated targets compare literally here.
func TestRecoveryToolchainCoversTheUnitPath(t *testing.T) {
	home := t.TempDir()
	for _, target := range []string{"linux", "darwin"} {
		recovery := recoveryToolchainEntries(target, home)
		have := make(map[string]bool, len(recovery))
		for _, entry := range recovery {
			have[filepath.Clean(entry)] = true
		}
		for _, entry := range platformgo.DefaultPathEntries(target, home, "") {
			if !have[filepath.Clean(entry)] {
				t.Errorf("%s: unit PATH entry %q is missing from the recovery toolchain %v", target, entry, recovery)
			}
		}
	}
}

func TestRecoveryToolchainSearchesTheContractRuntimeBin(t *testing.T) {
	home := t.TempDir()
	want, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	entries := recoveryToolchainEntries("linux", home)
	if entries[len(entries)-1] != want {
		t.Fatalf("recovery toolchain = %v; want it to end with the contract bin %q", entries, want)
	}
}
