//go:build !windows

package codingagentshims

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/testenv"
)

// stageHome redirects HOME and returns (binDir, shimDir, launcher).
func stageHome(t *testing.T) (string, string, string) {
	t.Helper()
	home := testenv.RuntimeHome(t)
	binDir := filepath.Join(home, ".vrooli", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(binDir, launcherBinary)
	if err := os.WriteFile(launcher, []byte(shelltest.POSIXShebang()+"exit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return binDir, filepath.Join(home, ".vrooli", "shims"), launcher
}

// The shims must not live in the shared install root. That coupling is what
// made this safeguard's storage declaration describe a directory it did not
// own, and put a 64MiB ceiling on a root holding gigabytes.
func TestShimDirIsSeparateFromTheSharedInstallRoot(t *testing.T) {
	_, shimDir, _ := stageHome(t)
	legacy, err := LegacyShimDir()
	if err != nil {
		t.Fatal(err)
	}
	if shimDir == legacy {
		t.Fatalf("shim dir %q must differ from the install root", shimDir)
	}
	got, err := ShimDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != shimDir {
		t.Fatalf("ShimDir() = %q, want %q", got, shimDir)
	}
}

// The directory the code installs into must be the one the contract declares,
// or the storage declaration describes a path nothing writes to.
func TestShimDirMatchesTheRepositoryContract(t *testing.T) {
	home := testenv.RuntimeHome(t)

	contract, err := repocontract.LoadDefault(repoRootForTest(t))
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	entry, err := contract.RuntimeHomeEntry(home, repocontract.HomeKeyShims)
	if err != nil {
		t.Fatalf("contract has no %q runtime-home entry: %v", repocontract.HomeKeyShims, err)
	}
	got, err := ShimDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != entry.AbsPath {
		t.Fatalf("ShimDir() = %q, contract declares %q", got, entry.AbsPath)
	}
	if entry.Protected {
		t.Fatal("the shim directory must not be contract-protected; it is regenerable and self-repairing")
	}
}

// The install root must be contract-protected, because a declared budget over
// it is otherwise a licence for the retention enforcer to prune it.
func TestInstallRootIsContractProtected(t *testing.T) {
	contract, err := repocontract.LoadDefault(repoRootForTest(t))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := contract.RuntimeHomeEntry(t.TempDir(), repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if !entry.Protected {
		t.Fatal("runtime-home entry \"bin\" must be protected")
	}
	if entry.Retention != nil || (entry.Cleanup != "" && entry.Cleanup != "never") {
		t.Fatalf("bin declares cleanup=%q retention=%v, want no bulk reclamation", entry.Cleanup, entry.Retention)
	}
}

// The declared storage path must be the directory the code uses. The previous
// declaration named ~/.vrooli/bin while claiming to describe five links.
func TestSafeguardManifestDeclaresTheShimDirectory(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), "internal", "safeguards", "coding-agent-shims", "safeguard.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Storage struct {
			Entries map[string]struct {
				Path map[string]string `json:"path"`
			} `json:"entries"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	entry, ok := manifest.Storage.Entries["shims"]
	if !ok {
		t.Fatal("manifest declares no \"shims\" storage entry")
	}
	for _, platform := range []string{"linux", "macos", "windows"} {
		want := "~/" + runtimeHomeDirName + "/" + shimsDirName
		if entry.Path[platform] != want {
			t.Fatalf("%s path = %q, want %q", platform, entry.Path[platform], want)
		}
	}
}

// EnsureInstalled is what makes the shims survive an install root that cannot
// be relied on, so it has to be able to rebuild the set from nothing.
func TestEnsureInstalledRecreatesEveryAliasAfterAWipe(t *testing.T) {
	_, shimDir, launcher := stageHome(t)

	installed, err := EnsureInstalled()
	if err != nil {
		t.Fatalf("EnsureInstalled: %v", err)
	}
	if len(installed) != len(cliutil.CodingAgentAliases()) {
		t.Fatalf("installed = %v, want every alias", installed)
	}

	if err := os.RemoveAll(shimDir); err != nil {
		t.Fatal(err)
	}
	installed, err = EnsureInstalled()
	if err != nil {
		t.Fatalf("EnsureInstalled after wipe: %v", err)
	}
	if len(installed) != len(cliutil.CodingAgentAliases()) {
		t.Fatalf("installed after wipe = %v, want every alias", installed)
	}
	for _, alias := range cliutil.CodingAgentAliases() {
		target, err := os.Readlink(filepath.Join(shimDir, alias))
		if err != nil {
			t.Fatalf("alias %q missing after repair: %v", alias, err)
		}
		if target != launcher {
			t.Fatalf("alias %q points at %q, want %q", alias, target, launcher)
		}
	}
}

// A healthy host must do no filesystem writes at all, because this runs on
// every control-plane invocation.
func TestEnsureInstalledIsANoOpWhenEverythingIsPresent(t *testing.T) {
	_, shimDir, _ := stageHome(t)
	if _, err := EnsureInstalled(); err != nil {
		t.Fatal(err)
	}
	before := statTimes(t, shimDir)

	installed, err := EnsureInstalled()
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 0 {
		t.Fatalf("second call installed %v, want nothing", installed)
	}
	if after := statTimes(t, shimDir); after != before {
		t.Fatalf("alias mtimes changed on a no-op call: %v -> %v", before, after)
	}
}

// Before the launcher exists there is nothing to link to, and that is the
// normal state on a fresh checkout -- not an error every CLI start reports.
func TestEnsureInstalledIsSilentWhenTheLauncherIsNotBuilt(t *testing.T) {
	testenv.RuntimeHome(t)
	installed, err := EnsureInstalled()
	if err != nil {
		t.Fatalf("EnsureInstalled: %v", err)
	}
	if len(installed) != 0 {
		t.Fatalf("installed = %v, want nothing", installed)
	}
}

// Replacement must be atomic. The previous implementation removed the target
// and then created it, so an interruption in between left the alias absent --
// the exact end state that made `codex` stop resolving.
func TestInstallAliasNeverLeavesTheAliasAbsent(t *testing.T) {
	_, shimDir, launcher := stageHome(t)
	if _, err := EnsureInstalled(); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(shimDir, "codex")

	// Point it somewhere else, then repair, asserting the path is occupied at
	// every observable moment.
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/bin/false", alias); err != nil {
		t.Fatal(err)
	}
	if err := InstallAlias(alias, launcher); err != nil {
		t.Fatalf("InstallAlias: %v", err)
	}
	target, err := os.Readlink(alias)
	if err != nil {
		t.Fatalf("alias absent after repair: %v", err)
	}
	if target != launcher {
		t.Fatalf("alias points at %q, want %q", target, launcher)
	}
	// No staging file may survive.
	entries, err := os.ReadDir(shimDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".new-") {
			t.Fatalf("staging file %q survived", entry.Name())
		}
	}
}

// A staging file left behind by a crashed earlier attempt must not wedge every
// later install.
func TestInstallAliasRecoversFromAbandonedStagingFile(t *testing.T) {
	_, shimDir, launcher := stageHome(t)
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(shimDir, "codex")
	if err := os.Symlink("/bin/false", alias+".new-"+processSuffix()); err != nil {
		t.Fatal(err)
	}
	if err := InstallAlias(alias, launcher); err != nil {
		t.Fatalf("InstallAlias: %v", err)
	}
	if target, err := os.Readlink(alias); err != nil || target != launcher {
		t.Fatalf("alias = %q, %v; want %q", target, err, launcher)
	}
}

// Two aliases for one agent, in two directories both on PATH, is a state no
// operator can reason about. The migration retires the old one and records it.
func TestRemoveLegacyAliasesRetiresOldLocationAndWritesReceipts(t *testing.T) {
	binDir, _, launcher := stageHome(t)
	legacy := filepath.Join(binDir, "codex")
	if err := os.Symlink(launcher, legacy); err != nil {
		t.Fatal(err)
	}
	// An operator's own binary sharing the name is not ours to remove.
	unrelated := filepath.Join(binDir, "grok")
	if err := os.Symlink("/bin/false", unrelated); err != nil {
		t.Fatal(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := artifactledger.New(home)
	if err != nil {
		t.Fatal(err)
	}

	removed, err := RemoveLegacyAliases(ledger)
	if err != nil {
		t.Fatalf("RemoveLegacyAliases: %v", err)
	}
	if len(removed) != 1 || removed[0] != "codex" {
		t.Fatalf("removed = %v, want [codex]", removed)
	}
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy alias survived: %v", err)
	}
	if _, err := os.Lstat(unrelated); err != nil {
		t.Fatalf("unrelated binary was removed: %v", err)
	}

	receipts, err := ledger.Read()
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, receipt := range receipts {
		if receipt.Path == legacy && receipt.Outcome == artifactledger.OutcomeRemoved {
			found = true
			if receipt.Predicate == "" {
				t.Fatal("receipt records no predicate")
			}
		}
	}
	if !found {
		t.Fatalf("no removal receipt for %s in %d receipts", legacy, len(receipts))
	}
}

func statTimes(t *testing.T, shimDir string) string {
	t.Helper()
	parts := make([]string, 0, len(cliutil.CodingAgentAliases()))
	for _, alias := range cliutil.CodingAgentAliases() {
		info, err := os.Lstat(filepath.Join(shimDir, alias))
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, alias+"="+info.ModTime().String())
	}
	return strings.Join(parts, ",")
}

// repoRootForTest walks up to the checkout carrying the repository contract.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no repository root above %s", dir)
		}
		dir = parent
	}
}
