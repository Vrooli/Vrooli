package models

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
}

// SmokeModule is the Python probe module the gate invokes in the venv.
const SmokeModule = "image_tools_sidecar.smoke"

// probeArgsFor decides the smoke probe for a model given its installed dir. It
// returns ok=false when no Python load-smoke applies (a binary-backend model, or
// no recognizable install shape) — those models are validated by other gates
// (host-tool availability), not by a Python probe.
func probeArgsFor(m Model, dir string, deep bool) (kind string, args []string, ok bool) {
	if m.Backend == "diffusers" {
		args = []string{"--kind", "diffusers", "--model-dir", dir, "--family", m.Runtime.Family}
		if deep {
			args = append(args, "--deep")
		}
		return "diffusers", args, true
	}
	if hasONNXWeight(dir) {
		args = []string{"--kind", "onnx", "--model-dir", dir}
		return "onnx", args, true
	}
	return "", nil, false
}

func hasONNXWeight(dir string) bool {
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
func (in *Installer) ensureSmoke(ctx context.Context, m Model, dir, modelHash string, emit func(int, string)) error {
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
	if v, ok := smoke.ReadVerdict(dir); ok && v.Fresh(modelHash, cfg.LockHash) {
		if v.Pass {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrSmokeFailed, v.Reason)
	}

	emit(96, "smoke-testing model load")
	inv := smoke.Invoker{Python: cfg.Python, PythonPath: cfg.PythonPath, Module: SmokeModule, Run: cfg.Run}
	out, probeErr := inv.Probe(ctx, args)
	verdict := smoke.Verdict{
		Kind: kind, ModelHash: modelHash, LockHash: cfg.LockHash,
		CheckedAt: in.now().UTC().Format(time.RFC3339),
	}
	if probeErr != nil {
		verdict.Pass = false
		verdict.Reason = firstNonEmpty(out, probeErr.Error())
		_ = smoke.WriteVerdict(dir, verdict)
		return fmt.Errorf("%w: %s", ErrSmokeFailed, verdict.Reason)
	}
	verdict.Pass = true
	verdict.Reason = out
	_ = smoke.WriteVerdict(dir, verdict)
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

// SmokeStatus is the read-side view of a model's env-provisioning + smoke state,
// consumed by doctor, the health summary, and ready_state. It is computed without
// running a probe (cache + interpreter presence only), so it is cheap to query.
type SmokeStatus struct {
	// Applicable is false for binary-backend models (no Python smoke).
	Applicable bool
	// EnvProvisioned reports whether the venv interpreter is present.
	EnvProvisioned bool
	// Verdict is the cached smoke outcome, if any.
	Verdict    smoke.Verdict
	HasVerdict bool
}

// SmokeStatusFor reports the cached smoke/env state for an installed model dir
// without probing. dir should be the model's installed path.
func (in *Installer) SmokeStatusFor(m Model, dir string) SmokeStatus {
	st := SmokeStatus{}
	if _, _, ok := probeArgsFor(m, dir, false); !ok {
		return st // not applicable
	}
	st.Applicable = true
	if in.Smoke != nil {
		st.EnvProvisioned = strings.TrimSpace(in.Smoke.Python) != "" && fileExists(in.Smoke.Python)
	}
	if v, ok := smoke.ReadVerdict(dir); ok {
		st.Verdict, st.HasVerdict = v, true
	}
	return st
}

// modelDirFor exposes the per-model install dir for readiness callers.
func (in *Installer) ModelDir(id string) string { return filepath.Join(in.Root, "models", id) }

// DoctorRuntime emits the runtime "enabled ⇒ runnable" findings that the static
// catalog doctor cannot see: for every effectively-enabled, installed,
// smoke-applicable model it reports whether the Python env is provisioned and
// whether the cached load-smoke passed. This is what surfaces "installed but
// broken" (and "enabled but env not ready") in doctor/health BEFORE a user op.
func (in *Installer) DoctorRuntime(ctx context.Context, catalog []Model, overlay map[string]bool) []CatalogFinding {
	var out []CatalogFinding
	for _, m := range catalog {
		if !EffectiveEnabled(m, overlay) || !in.Installed(ctx, m.ID) {
			continue
		}
		dir := in.ModelDir(m.ID)
		if in.State != nil {
			if rec, ok, _ := in.State.Get(ctx, m.ID); ok && rec.Path != "" {
				dir = rec.Path
			}
		}
		st := in.SmokeStatusFor(m, dir)
		if !st.Applicable {
			continue
		}
		switch {
		case !st.EnvProvisioned:
			out = append(out, CatalogFinding{
				Severity: FindingWarning,
				Code:     "enabled_model_env_not_provisioned",
				ModelID:  m.ID,
				Message:  "enabled model's Python venv is not provisioned yet (still building, or uv missing); ensure `vrooli host install uv`, then restart the scenario (`vrooli scenario start image-tools`) to build/repair the venv",
			})
		case st.HasVerdict && !st.Verdict.Pass:
			out = append(out, CatalogFinding{
				Severity: FindingError,
				Code:     "enabled_model_smoke_failed",
				ModelID:  m.ID,
				Message:  "enabled model failed its install-time load-smoke (installed but not runnable): " + st.Verdict.Reason + " — re-validate with `vrooli scenario image-tools models install " + m.ID + "` after fixing the runtime",
			})
		case !st.HasVerdict:
			out = append(out, CatalogFinding{
				Severity: FindingWarning,
				Code:     "enabled_model_smoke_pending",
				ModelID:  m.ID,
				Message:  "enabled model has not been load-smoke verified yet; re-run `vrooli scenario image-tools models install " + m.ID + "` (idempotent) to smoke it",
			})
		}
	}
	return out
}
