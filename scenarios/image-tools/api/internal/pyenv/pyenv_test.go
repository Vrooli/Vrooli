package pyenv

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// recordingRunner captures every invocation and simulates `uv venv` by creating
// the interpreter file, so the post-sync validation and idempotency checks see a
// realistic venv without a real uv on the host.
type recordingRunner struct {
	calls   [][]string
	failOn  string // substring of args that should return an error
	failErr error
}

func (r *recordingRunner) run(_ context.Context, name string, args []string) ([]byte, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	joined := strings.Join(call, " ")
	if r.failOn != "" && strings.Contains(joined, r.failOn) {
		return []byte("boom"), r.failErr
	}
	if len(args) > 0 && args[0] == "venv" {
		// Simulate venv creation: materialize the interpreter the spec resolves to.
		venvDir := args[1]
		interp := InterpreterPath(venvDir)
		if err := os.MkdirAll(filepath.Dir(interp), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(interp, []byte("#!/bin/sh\n"), 0o755); err != nil {
			return nil, err
		}
	}
	return []byte("ok"), nil
}

func writeLock(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "requirements.lock")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnsure_BuildsWhenAbsent(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")
	rr := &recordingRunner{}

	got, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock, BasePython: "python3.12"}, rr.run)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	wantInterp := InterpreterPath(venv)
	if got.Python != wantInterp {
		t.Fatalf("interpreter = %q, want %q", got.Python, wantInterp)
	}
	if !filepath.IsAbs(got.Python) {
		t.Fatalf("interpreter must be absolute: %q", got.Python)
	}
	if got.LockHash == "" {
		t.Fatalf("LockHash must be populated")
	}
	if len(rr.calls) != 2 {
		t.Fatalf("expected 2 uv calls (venv, pip sync), got %d: %v", len(rr.calls), rr.calls)
	}
	// uv venv <dir> --python python3.12
	venvCall := rr.calls[0]
	if venvCall[0] != "uv" || venvCall[1] != "venv" || venvCall[2] != venv {
		t.Fatalf("venv argv wrong: %v", venvCall)
	}
	if !containsPair(venvCall, "--python", "python3.12") {
		t.Fatalf("venv argv missing --python python3.12: %v", venvCall)
	}
	// uv pip sync --python <interp> <lock>
	syncCall := rr.calls[1]
	if syncCall[1] != "pip" || syncCall[2] != "sync" {
		t.Fatalf("sync argv wrong: %v", syncCall)
	}
	if !containsPair(syncCall, "--python", wantInterp) {
		t.Fatalf("sync argv missing --python <interp>: %v", syncCall)
	}
	if syncCall[len(syncCall)-1] != lock {
		t.Fatalf("sync argv must end with lockfile: %v", syncCall)
	}
	// sentinel recorded the lock hash
	sentinel, err := os.ReadFile(sentinelPath(venv))
	if err != nil {
		t.Fatalf("sentinel not written: %v", err)
	}
	if strings.TrimSpace(string(sentinel)) != got.LockHash {
		t.Fatalf("sentinel %q != lock hash %q", sentinel, got.LockHash)
	}
}

func TestEnsure_IdempotentWhenUpToDate(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")

	first := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, first.run); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}

	// Second Ensure with the SAME lock must be a no-op (zero uv calls).
	second := &recordingRunner{}
	got, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, second.run)
	if err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if len(second.calls) != 0 {
		t.Fatalf("up-to-date venv must invoke uv 0 times, got %v", second.calls)
	}
	if got.Python != InterpreterPath(venv) {
		t.Fatalf("interpreter mismatch on idempotent path: %q", got.Python)
	}
}

func TestEnsure_ResyncsWhenLockChanges(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")

	first := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, first.run); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}

	// Change the lock → must re-sync, but NOT recreate the venv (interpreter present).
	writeLock(t, tmp, "torch==2.6.0\n")
	second := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, second.run); err != nil {
		t.Fatalf("resync Ensure: %v", err)
	}
	if len(second.calls) != 1 {
		t.Fatalf("lock change must trigger exactly 1 uv call (pip sync), got %v", second.calls)
	}
	if second.calls[0][1] != "pip" || second.calls[0][2] != "sync" {
		t.Fatalf("resync must be a pip sync, got %v", second.calls[0])
	}
}

func TestEnsure_VenvFailurePropagates(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")
	rr := &recordingRunner{failOn: "venv", failErr: errBoom}

	_, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, rr.run)
	if err == nil {
		t.Fatalf("expected venv failure to propagate")
	}
	if !strings.Contains(err.Error(), "uv venv failed") {
		t.Fatalf("error should name the failing step: %v", err)
	}
}

func TestEnsure_SyncFailurePropagates(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")
	rr := &recordingRunner{failOn: "pip sync", failErr: errBoom}

	_, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, rr.run)
	if err == nil {
		t.Fatalf("expected sync failure to propagate")
	}
	if !strings.Contains(err.Error(), "uv pip sync failed") {
		t.Fatalf("error should name the failing step: %v", err)
	}
}

func TestEnsure_ValidatesSpec(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	cases := []struct {
		name string
		spec Spec
		want string
	}{
		{"no venv dir", Spec{LockFile: lock}, "VenvDir is required"},
		{"no lockfile", Spec{VenvDir: filepath.Join(tmp, "pyenv")}, "LockFile is required"},
		{"relative venv", Spec{VenvDir: "rel/pyenv", LockFile: lock}, "must be absolute"},
		{"missing lockfile", Spec{VenvDir: filepath.Join(tmp, "pyenv"), LockFile: filepath.Join(tmp, "nope.lock")}, "read lockfile"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := &recordingRunner{}
			_, err := Ensure(context.Background(), tc.spec, rr.run)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want substring %q", err, tc.want)
			}
			if len(rr.calls) != 0 {
				t.Fatalf("spec validation must reject before any uv call, got %v", rr.calls)
			}
		})
	}
}

func TestEnsure_EmptyLockRejected(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "   \n")
	rr := &recordingRunner{}
	_, err := Ensure(context.Background(), Spec{VenvDir: filepath.Join(tmp, "pyenv"), LockFile: lock}, rr.run)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty lock must be rejected, got %v", err)
	}
}

func TestInterpreterPath_OSLayout(t *testing.T) {
	p := InterpreterPath(filepath.Join("x", "pyenv"))
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(p, filepath.Join("Scripts", "python.exe")) {
			t.Fatalf("windows interpreter layout wrong: %q", p)
		}
		return
	}
	if !strings.HasSuffix(p, filepath.Join("bin", "python")) {
		t.Fatalf("unix interpreter layout wrong: %q", p)
	}
}

func TestEnsure_DefaultUVName(t *testing.T) {
	tmp := t.TempDir()
	lock := writeLock(t, tmp, "torch==2.5.1\n")
	venv := filepath.Join(tmp, "pyenv")
	rr := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, rr.run); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if rr.calls[0][0] != "uv" {
		t.Fatalf("default uv command must be \"uv\", got %q", rr.calls[0][0])
	}
}

func containsPair(argv []string, flag, val string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag && argv[i+1] == val {
			return true
		}
	}
	return false
}

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }
