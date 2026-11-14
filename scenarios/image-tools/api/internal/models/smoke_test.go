package models

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/vrooli/packages/capacity"
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

	kind, args, ok = probeArgsFor(Model{ID: "sam-2.1-tiny", Backend: "sam2-transformers"}, "/models/sam-2.1-tiny", true)
	if !ok || kind != "sam2" {
		t.Fatalf("SAM 2 probe: ok=%v kind=%q", ok, kind)
	}
	if joined := strings.Join(args, " "); !strings.Contains(joined, "--kind sam2") || !strings.Contains(joined, "--deep") {
		t.Fatalf("SAM 2 probe arguments missing kind/deep flag: %v", args)
	}
}

func TestProbeArgsForDiffusersInpaintWithoutFamily(t *testing.T) {
	m := Model{
		ID: "sdxl-inpainting-0.1", Backend: BackendDiffusers, Architecture: "sdxl",
		Operations: []string{"inpaint", "outpaint"}, Runtime: Runtime{Variant: "fp16"},
	}
	kind, args, ok := probeArgsFor(m, "/models/sdxl-inpainting-0.1", true)
	if !ok || kind != "diffusers-inpaint" {
		t.Fatalf("inpaint probe: ok=%v kind=%q", ok, kind)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--kind diffusers-inpaint", "--architecture sdxl", "--variant fp16", "--deep"} {
		if !strings.Contains(joined, want) {
			t.Errorf("inpaint probe args %q missing %q", joined, want)
		}
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

	// An explicit retry revalidates failed evidence after runtime repair.
	if err := smoke.WriteVerdict(dir, smoke.Verdict{Pass: false, Reason: "broken", ModelHash: "h1", LockHash: "l1"}); err != nil {
		t.Fatal(err)
	}
	probed = false
	if err := in.ensureSmoke(context.Background(), diffusersModel(), dir, "h1", noEmit); err != nil {
		t.Fatalf("retry repaired model: %v", err)
	}
	if !probed {
		t.Error("failed evidence must be rechecked by an explicit install retry")
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

func TestSmokeStatusProjection(t *testing.T) {
	dir := t.TempDir()
	interp := fakeInterp(t)
	in := &Installer{Smoke: &SmokeConfig{Python: interp, LockHash: "l1"}}

	st := in.smokeStatusFor(diffusersModel(), dir, "")
	if !st.Applicable || !st.EnvProvisioned || st.HasVerdict {
		t.Fatalf("expected applicable+provisioned+no-verdict: %+v", st)
	}

	// not applicable for a binary backend.
	if in.smokeStatusFor(Model{ID: "sd", Backend: "stable-diffusion.cpp"}, t.TempDir(), "").Applicable {
		t.Error("binary backend must be not-applicable")
	}

	// reflects a cached verdict.
	_ = smoke.WriteVerdict(dir, smoke.Verdict{Pass: true, ModelHash: "h1", LockHash: "l1"})
	if st := in.smokeStatusFor(diffusersModel(), dir, "h1"); !st.HasVerdict || !st.Verdict.Pass {
		t.Errorf("status should reflect fresh cached verdict: %+v", st)
	}
}

func TestSmokeStatusRejectsStaleCachedVerdicts(t *testing.T) {
	dir := t.TempDir()
	in := &Installer{Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "l1"}}
	if err := smoke.WriteVerdict(dir, smoke.Verdict{Pass: true, ModelHash: "h1", LockHash: "l1", Kind: "diffusers"}); err != nil {
		t.Fatal(err)
	}
	if st := in.smokeStatusFor(diffusersModel(), dir, "h2"); st.HasVerdict {
		t.Fatalf("model digest change must invalidate cached pass: %+v", st)
	}
	in.Smoke.LockHash = "l2"
	if st := in.smokeStatusFor(diffusersModel(), dir, "h1"); st.HasVerdict {
		t.Fatalf("lock change must invalidate cached pass: %+v", st)
	}
}

func TestRuntimeReadinessRejectsStaleInstalledDigest(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := registry.ByID("qwen-image-edit")
	if !ok || model.Backend != BackendDiffusers {
		t.Fatalf("expected seeded diffusers model, got %+v (ok=%t)", model, ok)
	}
	dir := t.TempDir()
	state := NewInstallStore(db)
	if err := state.Set(ctx, InstallRecord{ID: model.ID, Installed: true, Path: dir, Checksum: "model-v1"}); err != nil {
		t.Fatal(err)
	}
	installer := &Installer{
		Root:  t.TempDir(),
		Reg:   registry,
		State: state,
		Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "lock-v1"},
	}
	if err := smoke.WriteVerdict(dir, smoke.Verdict{Pass: true, ModelHash: "model-v1", LockHash: "lock-v1", Kind: "diffusers"}); err != nil {
		t.Fatal(err)
	}
	if ready, state := installer.RuntimeReadiness(ctx, model.ID); !ready || state != "ready" {
		t.Fatalf("fresh verdict readiness = %t/%q, want ready", ready, state)
	}
	if err := state.Set(ctx, InstallRecord{ID: model.ID, Installed: true, Path: dir, Checksum: "model-v2"}); err != nil {
		t.Fatal(err)
	}
	if ready, state := installer.RuntimeReadiness(ctx, model.ID); ready || state != "smoke_pending" {
		t.Fatalf("stale artifact verdict readiness = %t/%q, want smoke_pending", ready, state)
	}
	installer.Smoke.LockHash = "lock-v2"
	if ready, state := installer.RuntimeReadiness(ctx, model.ID); ready || state != "smoke_pending" {
		t.Fatalf("stale lock verdict readiness = %t/%q, want smoke_pending", ready, state)
	}
}

func TestRuntimeReadinessRejectsPythonSidecarWithoutModelProbe(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	registry, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := registry.ByID("restoreformer-plus-plus")
	if !ok || model.Backend != "python-sidecar" {
		t.Fatalf("expected seeded python-sidecar model, got %+v (ok=%t)", model, ok)
	}
	dir := t.TempDir()
	state := NewInstallStore(db)
	if err := state.Set(ctx, InstallRecord{ID: model.ID, Installed: true, Path: dir, Checksum: "checkpoint-v1"}); err != nil {
		t.Fatal(err)
	}
	installer := &Installer{Root: t.TempDir(), Reg: registry, State: state, Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "lock-v1"}}
	if ready, state := installer.RuntimeReadiness(ctx, model.ID); ready || state != "smoke_unsupported" {
		t.Fatalf("unsupported model-load probe readiness = %t/%q, want smoke_unsupported", ready, state)
	}
}

func noEmit(int, string) {}

type smokeLease struct{ released bool }

func (*smokeLease) Heartbeat(context.Context) error { return nil }
func (l *smokeLease) Release(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.released = true
	return nil
}

func TestSmokeAdmissionCoversLoadAndReleasesAfterCanceledWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lease := &smokeLease{}
	m := diffusersModel()
	m.Hardware.MinRAMGB = 4
	dir := t.TempDir()
	in := &Installer{Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "lock"}}
	in.Smoke.Acquire = func(ctx context.Context, request capacity.Request) (*capacity.Scope, error) {
		if request.RAMBytes != 4<<30 || request.CPUMilli <= 0 || request.CPUMilli > 4000 || !request.NoReclaim {
			t.Fatalf("request = %+v", request)
		}
		return capacity.Maintain(ctx, lease), nil
	}
	in.Smoke.Run = func(ctx context.Context, _ string, _, env []string) ([]byte, error) {
		if lease.released {
			t.Fatal("load began after claim release")
		}
		for _, want := range []string{"CUDA_VISIBLE_DEVICES=", "IMAGE_TOOLS_DEVICE=cpu", "IMAGE_TOOLS_CPU_THREADS="} {
			if !strings.Contains(strings.Join(env, "\n"), want) {
				t.Fatalf("load env missing %q: %v", want, env)
			}
		}
		cancel()
		return nil, ctx.Err()
	}
	if err := in.ensureSmoke(ctx, m, dir, "weights", noEmit); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	if !lease.released {
		t.Fatal("canceled load retained its resource claim")
	}
	if _, ok := smoke.ReadVerdict(dir); ok {
		t.Fatal("cancellation poisoned reusable smoke evidence")
	}
}

func TestSmokeCapacityWaitCanBeCanceledBeforeLoad(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := diffusersModel()
	m.Hardware.MinRAMGB = 4
	in := &Installer{Smoke: &SmokeConfig{Python: fakeInterp(t), LockHash: "lock"}}
	in.Smoke.Acquire = func(context.Context, capacity.Request) (*capacity.Scope, error) {
		cancel()
		return nil, &capacity.UnavailableError{Verdict: capacity.Verdict{Kind: "queue", Reason: "RAM busy"}}
	}
	in.Smoke.Run = func(context.Context, string, []string, []string) ([]byte, error) {
		t.Fatal("load ran without admission")
		return nil, nil
	}
	if err := in.ensureSmoke(ctx, m, t.TempDir(), "weights", noEmit); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestLocalONNXReferenceRequiresSmokeAndKeepsSourceDirectoryUntouched(t *testing.T) {
	f := newInstallFixture(t)
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "local.onnx")
	if err := os.WriteFile(source, []byte("test-weight"), 0o444); err != nil {
		t.Fatal(err)
	}
	m := Model{ID: "local-onnx", Backend: "onnxruntime", Source: Source{LocalPath: source}, CapabilityLabels: CapabilityLabels{License: "MIT", CommercialUse: CommercialUseYes}}
	if err := f.in.Custom.Add(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	called := false
	f.in.Smoke = &SmokeConfig{Python: fakeInterp(t), LockHash: "runtime", Run: func(_ context.Context, _ string, args, _ []string) ([]byte, error) {
		called = true
		if !strings.Contains(strings.Join(args, " "), "--model-dir "+source) {
			t.Fatalf("probe did not load referenced file: %v", args)
		}
		return []byte("loaded"), nil
	}}
	if _, err := f.in.Install(context.Background(), m.ID, noEmit); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("file-backed ONNX skipped load validation")
	}
	state, err := f.in.Lifecycle(context.Background(), m.ID)
	if err != nil || !state.Ready() {
		t.Fatalf("validated local file not ready: %+v %v", state, err)
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("installation changed referenced directory: %v %v", entries, err)
	}
	if verdict, ok := smoke.ReadVerdict(f.in.ModelDir(m.ID)); !ok || !verdict.Pass {
		t.Fatal("managed smoke evidence missing")
	}
}
