package models

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"image-tools/internal/smoke"
)

// fakeInterp creates an executable stand-in venv interpreter file so the
// env-provisioned check (fileExists) passes without a real python.
func fakeInterp(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "python")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func diffusersModel() Model {
	return Model{ID: "qwen", Backend: "diffusers", Runtime: Runtime{Family: "qwen-image-edit-plus"}}
}

func TestProbeArgsFor(t *testing.T) {
	dir := t.TempDir()

	kind, args, ok := probeArgsFor(diffusersModel(), dir, false)
	if !ok || kind != "diffusers" {
		t.Fatalf("diffusers: ok=%v kind=%q", ok, kind)
	}
	if args[len(args)-1] != "qwen-image-edit-plus" || args[1] != "diffusers" {
		t.Errorf("diffusers args wrong: %v", args)
	}
	if _, deepArgs, _ := probeArgsFor(diffusersModel(), dir, true); deepArgs[len(deepArgs)-1] != "--deep" {
		t.Errorf("deep flag not appended: %v", deepArgs)
	}

	// onnx dir → onnx probe.
	onnxDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(onnxDir, "m.onnx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if kind, _, ok := probeArgsFor(Model{ID: "seg", Backend: "onnxruntime"}, onnxDir, false); !ok || kind != "onnx" {
		t.Errorf("onnx: ok=%v kind=%q", ok, kind)
	}

	// binary backend with no onnx weight → not applicable.
	if _, _, ok := probeArgsFor(Model{ID: "sd", Backend: "stable-diffusion.cpp"}, t.TempDir(), false); ok {
		t.Error("binary backend must not be smoke-applicable")
	}
}

func TestEnsureSmoke_DisabledIsNoop(t *testing.T) {
	in := &Installer{} // Smoke nil
	if err := in.ensureSmoke(context.Background(), diffusersModel(), t.TempDir(), "h", noEmit); err != nil {
		t.Fatalf("disabled gate must be a no-op: %v", err)
	}
}

func TestEnsureSmoke_EnvNotProvisionedDoesNotBlock(t *testing.T) {
	called := false
	in := &Installer{Smoke: &SmokeConfig{Python: "", LockHash: "l1", Run: func(context.Context, string, []string, []string) ([]byte, error) {
		called = true
		return nil, nil
	}}}
	dir := t.TempDir()
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h", noEmit); err != nil {
		t.Fatalf("env-not-provisioned must not block install: %v", err)
	}
	if called {
		t.Error("must not probe without an interpreter")
	}
	if _, ok := smoke.ReadVerdict(dir); ok {
		t.Error("must not write a verdict when env is not provisioned")
	}
}

func TestEnsureSmoke_PassWritesVerdict(t *testing.T) {
	dir := t.TempDir()
	in := &Installer{Smoke: &SmokeConfig{
		Python: fakeInterp(t), LockHash: "l1",
		Run: func(_ context.Context, _ string, _, _ []string) ([]byte, error) {
			return []byte("diffusers smoke OK (shallow)"), nil
		},
	}}
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit); err != nil {
		t.Fatalf("pass: %v", err)
	}
	v, ok := smoke.ReadVerdict(dir)
	if !ok || !v.Pass || v.ModelHash != "h1" || v.LockHash != "l1" || v.Kind != "diffusers" {
		t.Fatalf("verdict not persisted correctly: %+v ok=%v", v, ok)
	}
}

func TestEnsureSmoke_FailReturnsErrAndCaches(t *testing.T) {
	dir := t.TempDir()
	in := &Installer{Smoke: &SmokeConfig{
		Python: fakeInterp(t), LockHash: "l1",
		Run: func(_ context.Context, _ string, _, _ []string) ([]byte, error) {
			return []byte("image_tools_sidecar: family not proven"), errors.New("exit status 4")
		},
	}}
	err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit)
	if !errors.Is(err, ErrSmokeFailed) {
		t.Fatalf("want ErrSmokeFailed, got %v", err)
	}
	v, ok := smoke.ReadVerdict(dir)
	if !ok || v.Pass {
		t.Fatalf("failure must be cached as not-pass: %+v ok=%v", v, ok)
	}
}

func TestEnsureSmoke_FreshCacheSkipsProbe(t *testing.T) {
	dir := t.TempDir()
	// Pre-seed a fresh PASS verdict.
	if err := smoke.WriteVerdict(dir, smoke.Verdict{Pass: true, ModelHash: "h1", LockHash: "l1", Kind: "diffusers"}); err != nil {
		t.Fatal(err)
	}
	probed := false
	in := &Installer{Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "l1", Run: func(context.Context, string, []string, []string) ([]byte, error) {
		probed = true
		return []byte("ok"), nil
	}}}
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit); err != nil {
		t.Fatalf("fresh pass: %v", err)
	}
	if probed {
		t.Error("a fresh-pass verdict must skip the probe")
	}

	// A fresh FAIL verdict must short-circuit to ErrSmokeFailed without probing.
	if err := smoke.WriteVerdict(dir, smoke.Verdict{Pass: false, Reason: "broken", ModelHash: "h1", LockHash: "l1"}); err != nil {
		t.Fatal(err)
	}
	probed = false
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit); !errors.Is(err, ErrSmokeFailed) {
		t.Fatalf("fresh fail must return ErrSmokeFailed, got %v", err)
	}
	if probed {
		t.Error("a fresh-fail verdict must skip the probe")
	}

	// A stale verdict (lock changed) must re-probe.
	probed = false
	in.Smoke.LockHash = "l2"
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit); err != nil {
		t.Fatalf("stale re-probe: %v", err)
	}
	if !probed {
		t.Error("a stale verdict (lock changed) must re-probe")
	}
}

func TestSmokeStatusFor(t *testing.T) {
	dir := t.TempDir()
	interp := fakeInterp(t)
	in := &Installer{Smoke: &SmokeConfig{Python: interp, LockHash: "l1"}}

	st := in.SmokeStatusFor(diffusersModel(), dir)
	if !st.Applicable || !st.EnvProvisioned || st.HasVerdict {
		t.Fatalf("expected applicable+provisioned+no-verdict: %+v", st)
	}

	// not applicable for a binary backend.
	if in.SmokeStatusFor(Model{ID: "sd", Backend: "stable-diffusion.cpp"}, t.TempDir()).Applicable {
		t.Error("binary backend must be not-applicable")
	}

	// reflects a cached verdict.
	_ = smoke.WriteVerdict(dir, smoke.Verdict{Pass: true, ModelHash: "h1", LockHash: "l1"})
	if st := in.SmokeStatusFor(diffusersModel(), dir); !st.HasVerdict || !st.Verdict.Pass {
		t.Errorf("status should reflect cached verdict: %+v", st)
	}
}

func noEmit(int, string) {}
