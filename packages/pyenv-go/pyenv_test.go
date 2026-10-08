package pyenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type recordingRunner struct {
	calls  [][]string
	failOn string
}

func (r *recordingRunner) run(_ context.Context, name string, args []string) ([]byte, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	if r.failOn != "" && strings.Contains(strings.Join(call, " "), r.failOn) {
		return []byte("boom"), errors.New("boom")
	}
	if len(args) > 1 && args[0] == "venv" {
		interpreter := InterpreterPath(args[1])
		if err := os.MkdirAll(filepath.Dir(interpreter), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(interpreter, []byte("python"), 0o755); err != nil {
			return nil, err
		}
	}
	return []byte("ok"), nil
}

func lockFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "requirements.lock")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEnsureBuildsAndPinsInterpreter(t *testing.T) {
	tmp := t.TempDir()
	lock := lockFile(t, tmp, "package==1.0\n")
	venv := filepath.Join(tmp, "venv")
	runner := &recordingRunner{}
	got, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock, BasePython: "3.12"}, runner.run)
	if err != nil {
		t.Fatal(err)
	}
	if got.Python != InterpreterPath(venv) || !filepath.IsAbs(got.Python) {
		t.Fatalf("interpreter = %q, want absolute %q", got.Python, InterpreterPath(venv))
	}
	if len(runner.calls) != 3 || runner.calls[0][0] != "uv" || runner.calls[0][1] != "python" || runner.calls[0][2] != "install" {
		t.Fatalf("unexpected uv calls: %v", runner.calls)
	}
	if runner.calls[0][3] != "3.12" || !containsPair(runner.calls[1], "--python", "3.12") || !containsPair(runner.calls[2], "--python", got.Python) {
		t.Fatalf("pinned interpreter missing from calls: %v", runner.calls)
	}
}

func TestEnsureReusesAndRepairsByLockHash(t *testing.T) {
	tmp := t.TempDir()
	lock := lockFile(t, tmp, "package==1.0\n")
	venv := filepath.Join(tmp, "venv")
	first := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, first.run); err != nil {
		t.Fatal(err)
	}
	second := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, second.run); err != nil {
		t.Fatal(err)
	}
	if len(second.calls) != 0 {
		t.Fatalf("up-to-date environment invoked uv: %v", second.calls)
	}
	if err := os.WriteFile(lock, []byte("package==2.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	third := &recordingRunner{}
	if _, err := Ensure(context.Background(), Spec{VenvDir: venv, LockFile: lock}, third.run); err != nil {
		t.Fatal(err)
	}
	if len(third.calls) != 1 || third.calls[0][1] != "pip" || third.calls[0][2] != "sync" {
		t.Fatalf("lock change should only sync: %v", third.calls)
	}
}

func TestEnsureNamesUVFailure(t *testing.T) {
	tmp := t.TempDir()
	lock := lockFile(t, tmp, "package==1.0\n")
	runner := &recordingRunner{failOn: "venv"}
	_, err := Ensure(context.Background(), Spec{VenvDir: filepath.Join(tmp, "venv"), LockFile: lock}, runner.run)
	if err == nil || !strings.Contains(err.Error(), "uv venv failed") {
		t.Fatalf("expected uv failure, got %v", err)
	}
}

func TestEnsureRejectsEmptyLock(t *testing.T) {
	tmp := t.TempDir()
	lock := lockFile(t, tmp, " \n")
	_, err := Ensure(context.Background(), Spec{VenvDir: filepath.Join(tmp, "venv"), LockFile: lock}, (&recordingRunner{}).run)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty lock error, got %v", err)
	}
}

func TestInterpreterPathLayout(t *testing.T) {
	path := InterpreterPath(filepath.Join("root", "venv"))
	if runtime.GOOS == "windows" && !strings.HasSuffix(path, filepath.Join("Scripts", "python.exe")) {
		t.Fatalf("windows path = %q", path)
	}
	if runtime.GOOS != "windows" && !strings.HasSuffix(path, filepath.Join("bin", "python")) {
		t.Fatalf("unix path = %q", path)
	}
}

func containsPair(argv []string, flag, value string) bool {
	for i := 0; i+1 < len(argv); i++ {
		if argv[i] == flag && argv[i+1] == value {
			return true
		}
	}
	return false
}
