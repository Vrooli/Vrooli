package backup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) run(_ context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, args)
	return []byte("ok"), nil
}

// findRegisterCall returns the single targets-register invocation, or fails.
func findRegisterCall(t *testing.T, calls [][]string) []string {
	t.Helper()
	var found []string
	count := 0
	for _, c := range calls {
		if len(c) >= 2 && c[0] == "targets" && c[1] == "register" {
			found = c
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 targets-register call, got %d (%v)", count, calls)
	}
	return found
}

func argValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// T-B1: two EnsureBackupTargets calls each issue exactly one register for the
// same (owner,name) — idempotency is delegated to the upstream CLI, so the
// helper is safe to run repeatedly.
func TestEnsureBackupTargets_Idempotent(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	rec := &recordingRunner{}

	for i := 0; i < 2; i++ {
		if err := EnsureBackupTargets(context.Background(), rec.run); err != nil {
			t.Fatalf("EnsureBackupTargets call %d error = %v", i, err)
		}
	}
	if len(rec.calls) != 2 {
		t.Fatalf("expected 2 register invocations across 2 calls, got %d", len(rec.calls))
	}
	for _, c := range rec.calls {
		if argValue(c, "--owner") != Owner || argValue(c, "--name") != DomainTargetName {
			t.Fatalf("register call targets wrong (owner,name): %v", c)
		}
	}
}

// T-B2: captures (cache class) are never registered as a backup target.
func TestEnsureBackupTargets_CapturesExcluded(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	rec := &recordingRunner{}

	if err := EnsureBackupTargets(context.Background(), rec.run); err != nil {
		t.Fatalf("EnsureBackupTargets error = %v", err)
	}
	for _, c := range rec.calls {
		joined := strings.Join(c, " ")
		if strings.Contains(joined, "captures") || strings.Contains(joined, "/cache/") {
			t.Fatalf("captures/cache must not be registered, got: %v", c)
		}
	}
}

// T-B3: a failing runner (e.g. data-backup-manager not running) surfaces an
// error the caller can treat as non-fatal — it must not panic.
func TestEnsureBackupTargets_RunnerFailureSurfaces(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	failing := func(context.Context, ...string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}
	if err := EnsureBackupTargets(context.Background(), failing); err == nil {
		t.Fatalf("expected error from failing runner")
	}
}

// T-B4: the registered locator is the kind=filesystem data base under the
// runtime-home data class (so data-backup-manager's regenerable==false discovery
// sees it).
func TestEnsureBackupTargets_LocatorUnderDataClass(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	rec := &recordingRunner{}

	if err := EnsureBackupTargets(context.Background(), rec.run); err != nil {
		t.Fatalf("EnsureBackupTargets error = %v", err)
	}
	call := findRegisterCall(t, rec.calls)
	if argValue(call, "--kind") != "filesystem" {
		t.Fatalf("expected kind=filesystem, got %q", argValue(call, "--kind"))
	}
	locator := argValue(call, "--locator")
	wantPrefix := filepath.Join(root, "data", "vrooli", "swarm-manager")
	if locator != wantPrefix {
		t.Fatalf("locator = %q, want %q", locator, wantPrefix)
	}
}
