package models

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/smoke"
)

// ErrSmokeFailed is returned when a model's install-time load-smoke probe ran and
// failed: the weights are on disk but the provisioned runtime could not construct
// the model. The install is reported failed (no silent "installed but broken");
// the weights are kept and the failure is cached so doctor/ready_state surface it.
var ErrSmokeFailed = errors.New("models: install-time load-smoke failed")

// SmokeConfig configures the install-time load-smoke gate. When the Installer's
// Smoke is nil the gate is disabled (e.g. in unit tests that don't exercise it).
type SmokeConfig struct {
	// Python is the absolute venv interpreter. Empty (or a not-yet-built venv)
	// means the env is not provisioned: install does NOT fail on it (the weights
	// fetch fine), but no pass verdict is written, so doctor/ready_state report
	// env-not-provisioned until the venv is ready.
	Python string
	// PythonPath is the sidecar PYTHONPATH (image_tools_sidecar resolution).
	PythonPath string
	// LockHash is the current venv lock hash; it is the smoke cache invalidator
	// (a venv resync re-smokes) and half of the cache key.
	LockHash string
	// Deep requests a full-weight load smoke (opt-in; expensive for large models).
	Deep bool
	// Run is the injected probe executor (nil → os/exec).
	Run smoke.Runner
	// Acquire admits the CPU-only model load. Production supplies the shared
	// broker; tests may inject isolated leases without host state.
	Acquire func(context.Context, capacity.Request) (*capacity.Scope, error)
}

// SmokeModule is the Python probe module the gate invokes in the venv.
const SmokeModule = "image_tools_sidecar.smoke"

// probeArgsFor decides the smoke probe for a model given its installed dir. It
// returns ok=false when no Python load-smoke applies (a binary-backend model, or
// no recognizable install shape) — those models are validated by other gates
// (host-tool availability), not by a Python probe.
func probeArgsFor(m Model, dir string, deep bool) (kind string, args []string, ok bool) {
	if m.Backend == "diffusers" {
		kind = "diffusers"
		if strings.TrimSpace(m.Runtime.Family) != "" {
			args = []string{"--kind", "diffusers", "--model-dir", dir, "--family", m.Runtime.Family}
		} else if m.Architecture != "" && (m.ServesOperation("inpaint") || m.ServesOperation("outpaint")) {
			kind = "diffusers-inpaint"
			args = []string{"--kind", "diffusers-inpaint", "--model-dir", dir, "--architecture", string(m.Architecture)}
			if variant := strings.TrimSpace(m.Runtime.Variant); variant != "" {
				args = append(args, "--variant", variant)
			}
		} else {
			return "", nil, false
		}
		if deep {
			args = append(args, "--deep")
		}
		return kind, args, true
	}
	if m.Backend == "sam2-transformers" {
		args = []string{"--kind", "sam2", "--model-dir", dir}
		if deep {
			args = append(args, "--deep")
		}
		return "sam2", args, true
	}
	if hasONNXWeight(dir) {
		args = []string{"--kind", "onnx", "--model-dir", dir}
		return "onnx", args, true
	}
	return "", nil, false
}

// HasLoadSmokeContract reports whether the catalog entry describes an install
// shape supported by the smoke gate. Host-fit provisioning uses this static
// check before downloading so a selected model cannot end in the known
// smoke_unsupported lifecycle state after a large transfer.
func HasLoadSmokeContract(m Model) bool {
	switch m.Backend {
	case BackendDiffusers:
		return strings.TrimSpace(m.Runtime.Family) != "" || m.Architecture != "" && (m.ServesOperation("inpaint") || m.ServesOperation("outpaint"))
	case "sam2-transformers":
		return true
	}
	for _, asset := range m.Source.Assets {
		if asset.Kind == ArtifactONNX || strings.EqualFold(filepath.Ext(asset.Filename), ".onnx") {
			return true
		}
	}
	return m.BuildProfile != nil && strings.EqualFold(filepath.Ext(m.BuildProfile.OutputFilename), ".onnx")
}

// NeedsLoadSmokeContract identifies model backends whose lifecycle requires a
// model-specific load-smoke before they can be Ready.
func NeedsLoadSmokeContract(m Model) bool {
	switch m.Backend {
	case BackendDiffusers, "python-sidecar", "onnxruntime", "sam2-transformers", "triposr":
		return true
	default:
		return false
	}
}

func hasONNXWeight(dir string) bool {
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		return strings.EqualFold(filepath.Ext(dir), ".onnx")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".onnx") {
			return true
		}
	}
	return false
}

// ensureSmoke runs (or reuses a cached) load-smoke for a freshly-installed model.
// Contract:
//   - gate disabled / no applicable probe → no-op (nil).
//   - env not provisioned (no interpreter on disk) → no-op (nil); readiness
//     surfaces env-not-provisioned separately, so install is not blocked on a
//     venv that may still be building.
//   - a fresh cached PASS for (modelHash, lockHash) → no-op (nil).
//   - probe runs and passes → cache PASS, nil.
//   - probe runs and fails → cache FAIL, return ErrSmokeFailed (install reported
//     failed; weights are kept for retry once the cause is fixed).
func (in *Installer) ensureSmoke(ctx context.Context, m Model, dir, modelHash string, emit func(int, string)) (resultErr error) {
	cfg := in.Smoke
	if cfg == nil {
		return nil
	}
	kind, args, ok := probeArgsFor(m, dir, cfg.Deep)
	if !ok {
		return nil
	}
	// Env not provisioned yet → don't block install; readiness reports it.
	if strings.TrimSpace(cfg.Python) == "" || !fileExists(cfg.Python) {
		return nil
	}
	evidenceDir := in.smokeEvidenceDir(m, dir)
	if v, ok := smoke.ReadVerdict(evidenceDir); ok && v.Pass && v.Fresh(modelHash, cfg.LockHash) {
		return nil
	}
	threads := min(4, runtime.NumCPU())
	if cfg.Acquire != nil {
		if m.Hardware.MinRAMGB <= 0 || m.Hardware.MinRAMGB > 1024 {
			return fmt.Errorf("models: smoke %q needs a bounded model RAM budget", m.ID)
		}
		request := capacity.Request{OwnerID: "image-tools:model-smoke:" + m.ID, RAMBytes: int64(m.Hardware.MinRAMGB) << 30, CPUMilli: int64(threads) * 1000, Priority: "service", NoReclaim: true}
		var owned *capacity.Scope
		for {
			var err error
			owned, err = cfg.Acquire(ctx, request)
			if err == nil {
				break
			}
			var unavailable *capacity.UnavailableError
			if !errors.As(err, &unavailable) {
				return err
			}
			emit(95, "waiting for model smoke capacity: "+err.Error())
			timer := time.NewTimer(2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return context.Cause(ctx)
			case <-timer.C:
			}
		}
		if owned == nil {
			return errors.New("models: smoke admission returned no scope")
		}
		defer func() { resultErr = errors.Join(resultErr, owned.Release(ctx)) }()
		ctx = owned.Context()
	}
	emit(96, "smoke-testing model load")
	threadLimit := strconv.Itoa(threads)
	env := []string{"CUDA_VISIBLE_DEVICES=", "IMAGE_TOOLS_DEVICE=cpu", "IMAGE_TOOLS_CPU_THREADS=" + threadLimit}
	for _, name := range []string{"OMP_NUM_THREADS", "MKL_NUM_THREADS", "OPENBLAS_NUM_THREADS", "NUMEXPR_NUM_THREADS", "VECLIB_MAXIMUM_THREADS"} {
		env = append(env, name+"="+threadLimit)
	}
	inv := smoke.Invoker{Python: cfg.Python, PythonPath: cfg.PythonPath, Module: SmokeModule, Env: env, Run: cfg.Run}
	out, probeErr := inv.Probe(ctx, args)
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		return fmt.Errorf("models: create smoke evidence directory: %w", err)
	}
	verdict := smoke.Verdict{
		Kind: kind, ModelHash: modelHash, LockHash: cfg.LockHash,
		CheckedAt: in.now().UTC().Format(time.RFC3339),
	}
	if probeErr != nil {
		verdict.Pass = false
		verdict.Reason = firstNonEmpty(out, probeErr.Error())
		return errors.Join(fmt.Errorf("%w: %s", ErrSmokeFailed, verdict.Reason), smoke.WriteVerdict(evidenceDir, verdict))
	}
	verdict.Pass = true
	verdict.Reason = out
	if err := smoke.WriteVerdict(evidenceDir, verdict); err != nil {
		return fmt.Errorf("models: persist smoke evidence: %w", err)
	}
	emit(99, "smoke passed")
	return nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return "smoke failed"
}

// SmokeStatus is the read-side evidence used by Lifecycle to derive model
// readiness without launching a probe.
type SmokeStatus struct {
	// Applicable is false for binary-backend models (no Python smoke).
	Applicable bool
	// EnvProvisioned reports whether the venv interpreter is present.
	EnvProvisioned bool
	// Verdict is the cached smoke outcome, if any.
	Verdict    smoke.Verdict
	HasVerdict bool
}

func (in *Installer) smokeStatusFor(m Model, dir, modelHash string) SmokeStatus {
	st := SmokeStatus{}
	if _, _, ok := probeArgsFor(m, dir, false); !ok {
		return st // not applicable
	}
	st.Applicable = true
	if in.Smoke != nil {
		st.EnvProvisioned = strings.TrimSpace(in.Smoke.Python) != "" && fileExists(in.Smoke.Python)
	}
	if v, ok := smoke.ReadVerdict(in.smokeEvidenceDir(m, dir)); ok && in.Smoke != nil && v.Fresh(modelHash, in.Smoke.LockHash) {
		st.Verdict, st.HasVerdict = v, true
	}
	return st
}

// RuntimeReadiness reports whether an installed model satisfies the gates needed
// before inference. Weight presence alone is intentionally weaker: build-profile
// models need their verified export, and Python-backed models need a provisioned
// environment plus a passing install-time load-smoke.
func (in *Installer) RuntimeReadiness(ctx context.Context, id string) (bool, string) {
	state, err := in.Lifecycle(ctx, id)
	if err != nil {
		return false, "lifecycle_unavailable"
	}
	return state.Ready(), state.Reason
}

// modelDirFor exposes the per-model install dir for readiness callers.
func (in *Installer) ModelDir(id string) string { return filepath.Join(in.Root, "models", id) }

// DoctorRuntime returns the lifecycle projection for the catalog together with
// the runtime "enabled ⇒ runnable" findings. The caller can render both from
// the same observation instead of hashing/reconciling every model twice.
func (in *Installer) DoctorRuntimeReport(ctx context.Context, catalog []Model, overlay map[string]bool) (CatalogDoctorReport, error) {
	report := CatalogDoctorReport{OK: true}
	for _, m := range catalog {
		state, err := in.Lifecycle(ctx, m.ID)
		if err != nil {
			return report, fmt.Errorf("models: doctor lifecycle %q: %w", m.ID, err)
		}
		report.Models = append(report.Models, state)
		if !EffectiveEnabled(m, overlay) {
			continue
		}
		finding := runtimeFinding(m, state)
		if finding != nil {
			report.Findings = append(report.Findings, *finding)
			if finding.Severity == FindingError {
				report.OK = false
			}
		}
	}
	return report, nil
}

// runtimeFinding derives the actionable doctor warning from the same lifecycle
// value used by the picker and admission.
func runtimeFinding(m Model, state ModelLifecycle) *CatalogFinding {
	if state.Ready() {
		return nil
	}
	severity := FindingWarning
	if state.State == ModelFailed {
		severity = FindingError
	}
	return &CatalogFinding{
		Severity: severity, Code: "enabled_model_" + state.Reason, ModelID: m.ID,
		Message: fmt.Sprintf("model lifecycle %s (%s): %s; run `image-tools models install %s --wait` to prepare or revalidate it", state.State, state.Reason, state.Detail, m.ID),
	}
}

// Referenced local artifacts are read-only; smoke evidence belongs to this
// scenario's managed model directory, never beside or inside the source.
func (in *Installer) smokeEvidenceDir(m Model, dir string) string {
	if m.Source.LocalPath != "" {
		return in.ModelDir(m.ID)
	}
	return dir
}
