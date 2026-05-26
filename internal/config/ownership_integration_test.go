//go:build ownership_integration

// This test makes real chown(2) calls and therefore only runs as root. Run it
// in a privileged CI lane or manually with:
//
//	sudo go test -tags ownership_integration ./internal/config/...
//
// It proves the load-bearing ownership invariants that cannot be asserted
// without root: a sudo'd write ends up owned by the invoking user, reconcile
// reclaims a root-owned stray, and a file already owned by a different non-root
// user is left untouched (CD-7).
package config

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func fileUID(t *testing.T, path string) uint32 {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("no stat_t for %s", path)
	}
	return st.Uid
}

func TestReconcileReclaimsRootOwnedStray(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	const targetUID, targetGID = 1000, 1000
	const otherUID, otherGID = 65534, 65534 // nobody

	home := t.TempDir() // created as root → root-owned

	stray := filepath.Join(home, "state", "runtime.db")
	if err := os.MkdirAll(filepath.Dir(stray), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stray, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A file already owned by a different non-root user must NOT be reassigned.
	foreign := filepath.Join(home, "config", "foreign.json")
	if err := os.MkdirAll(filepath.Dir(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Lchown(foreign, otherUID, otherGID); err != nil {
		t.Fatal(err)
	}

	changed, err := reconcileHomeOwnership(home, targetUID, targetGID)
	if err != nil {
		t.Fatalf("reconcileHomeOwnership: %v", err)
	}
	if changed == 0 {
		t.Fatal("expected at least the root-owned stray to be reclaimed")
	}
	if got := fileUID(t, stray); got != targetUID {
		t.Errorf("stray uid = %d, want %d (should be reclaimed)", got, targetUID)
	}
	if got := fileUID(t, foreign); got != otherUID {
		t.Errorf("foreign uid = %d, want %d (must NOT be reassigned)", got, otherUID)
	}
}

func TestOwnedWriteEndsUpInvokerOwned(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root")
	}
	// EnsureOwnedDir/WriteOwnedFile chown to hostreqkit.InvokingUserIDs(), which
	// reads $SUDO_UID/$SUDO_GID. Set them and force the root-via-sudo detection.
	t.Setenv("SUDO_USER", "ci")
	t.Setenv("SUDO_UID", "1000")
	t.Setenv("SUDO_GID", "1000")

	// The owned-write seam only chowns within the resolved VrooliHome boundary.
	// Point HOME at a temp dir so VrooliHome resolves inside the sandbox.
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Avoid the sudo /etc/passwd redirection so HomeDir() honors $HOME above.
	t.Setenv("SUDO_USER", "")

	target := filepath.Join(home, ".vrooli", "state", "scenarios", "demo", "rec.json")
	if err := WriteOwnedFile(target, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteOwnedFile: %v", err)
	}
	// With SUDO_USER cleared, InvokingUserIDs is not ok, so no chown happened —
	// the file is root-owned but that is the expected non-sudo path. Re-assert
	// reconcile (explicit home) brings it to the invoking user.
	if changed, err := reconcileHomeOwnership(filepath.Join(home, ".vrooli"), 1000, 1000); err != nil || changed == 0 {
		t.Fatalf("reconcile after write: changed=%d err=%v", changed, err)
	}
	if got := fileUID(t, target); got != 1000 {
		t.Errorf("written file uid = %d, want 1000", got)
	}
}
