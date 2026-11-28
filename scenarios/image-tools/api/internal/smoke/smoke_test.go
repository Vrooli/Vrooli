package smoke

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProbe_AssemblesArgvAndPythonPath(t *testing.T) {
	var gotPython string
	var gotArgs, gotEnv []string
	in := Invoker{
		Python:     "/venv/bin/python",
		PythonPath: "/data/sidecar",
		Module:     "image_tools_sidecar.smoke",
		Run: func(_ context.Context, python string, args, env []string) ([]byte, error) {
			gotPython, gotArgs, gotEnv = python, args, env
			return []byte("onnx smoke OK\n"), nil
		},
	}
	out, err := in.Probe(context.Background(), []string{"--kind", "onnx", "--model-dir", "/m"})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if out != "onnx smoke OK" {
		t.Errorf("output not trimmed/returned: %q", out)
	}
	if gotPython != "/venv/bin/python" {
		t.Errorf("python = %q", gotPython)
	}
	want := []string{"-m", "image_tools_sidecar.smoke", "--kind", "onnx", "--model-dir", "/m"}
	if strings.Join(gotArgs, " ") != strings.Join(want, " ") {
		t.Errorf("args = %v, want %v", gotArgs, want)
	}
	if len(gotEnv) != 1 || gotEnv[0] != "PYTHONPATH=/data/sidecar" {
		t.Errorf("env = %v, want PYTHONPATH=/data/sidecar", gotEnv)
	}
}

func TestProbe_NoInterpreterFailsFast(t *testing.T) {
	in := Invoker{Module: "x"}
	called := false
	in.Run = func(context.Context, string, []string, []string) ([]byte, error) {
		called = true
		return nil, nil
	}
	_, err := in.Probe(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "not provisioned") {
		t.Fatalf("err = %v, want env-not-provisioned", err)
	}
	if called {
		t.Error("must not invoke the runner without an interpreter")
	}
}

func TestProbe_NonZeroExitSurfacesStderr(t *testing.T) {
	in := Invoker{
		Python: "/venv/bin/python", Module: "x",
		Run: func(context.Context, string, []string, []string) ([]byte, error) {
			return []byte("image_tools_sidecar: family not proven"), errors.New("exit status 4")
		},
	}
	out, err := in.Probe(context.Background(), nil)
	if err == nil {
		t.Fatal("expected probe failure")
	}
	if !strings.Contains(err.Error(), "family not proven") {
		t.Errorf("error must carry probe stderr: %v", err)
	}
	if !strings.Contains(out, "family not proven") {
		t.Errorf("output must carry probe stderr: %q", out)
	}
}

func TestVerdict_FreshnessAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	v := Verdict{Pass: true, Reason: "ok", Kind: "diffusers", ModelHash: "m1", LockHash: "l1"}
	if err := WriteVerdict(dir, v); err != nil {
		t.Fatal(err)
	}
	got, ok := ReadVerdict(dir)
	if !ok {
		t.Fatal("verdict not read back")
	}
	if !got.Pass || got.Kind != "diffusers" {
		t.Errorf("round-trip lost data: %+v", got)
	}
	if !got.Fresh("m1", "l1") {
		t.Error("verdict should be fresh for matching hashes")
	}
	if got.Fresh("m2", "l1") {
		t.Error("changed model hash must invalidate verdict")
	}
	if got.Fresh("m1", "l2") {
		t.Error("changed lock hash must invalidate verdict")
	}
	if got.Fresh("", "") {
		t.Error("empty hashes must never be fresh")
	}
}

func TestProbe_DefaultRunnerExecutes(t *testing.T) {
	// Use /bin/echo as a stand-in interpreter: defaultRunner should exec it with
	// the assembled argv and return its output with no error.
	if _, err := os.Stat("/bin/echo"); err != nil {
		t.Skip("/bin/echo unavailable")
	}
	in := Invoker{Python: "/bin/echo", PythonPath: "/p", Module: "mod"}
	out, err := in.Probe(context.Background(), []string{"--kind", "onnx"})
	if err != nil {
		t.Fatalf("default runner: %v", err)
	}
	if !strings.Contains(out, "-m mod --kind onnx") {
		t.Errorf("default runner output = %q", out)
	}
}

func TestWriteVerdict_UnwritableDir(t *testing.T) {
	dir := t.TempDir()
	notDir := filepath.Join(dir, "afile")
	if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteVerdict(notDir, Verdict{Pass: true}); err == nil {
		t.Error("expected WriteVerdict to fail when model dir is a file")
	}
}

func TestReadVerdict_AbsentOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	if _, ok := ReadVerdict(dir); ok {
		t.Error("absent verdict must report not-ok")
	}
	if err := os.WriteFile(filepath.Join(dir, VerdictFile), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadVerdict(dir); ok {
		t.Error("corrupt verdict must report not-ok")
	}
}
