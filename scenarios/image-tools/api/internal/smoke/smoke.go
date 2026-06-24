// Package smoke runs install-time "load-smoke" probes: it invokes a Python probe
// module inside the scenario's private venv to prove a freshly-installed model can
// be CONSTRUCTED/LOADED by the provisioned runtime before a user op depends on it,
// and caches the pass/fail verdict keyed by (model hash, lock hash).
//
// Like internal/pyenv, it is deliberately scenario-agnostic: it knows an
// interpreter path, a PYTHONPATH, a module name and argv — nothing about
// image-tools models. The model→probe-args mapping lives with the caller
// (internal/models). That keeps this a clean lift to a platform package when a
// second model-managing scenario appears (see docs/internal/SEAMS.md).
package smoke

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// VerdictFile is the per-model-dir filename the cached verdict is stored under.
const VerdictFile = ".vrooli-smoke-verdict.json"

// Runner executes the probe interpreter and returns combined output. Injected so
// tests assert argv/PYTHONPATH assembly without a real python. nil → os/exec.
type Runner func(ctx context.Context, python string, args, extraEnv []string) ([]byte, error)

// Invoker runs a probe module in a specific interpreter.
type Invoker struct {
	// Python is the absolute venv interpreter. Empty ⇒ Probe fails fast (the venv
	// is not provisioned), which the caller surfaces as env-not-provisioned.
	Python string
	// PythonPath is prepended to PYTHONPATH so the embedded sidecar package
	// (image_tools_sidecar) resolves regardless of CWD.
	PythonPath string
	// Module is the probe module run as `python -m <Module> <args...>`.
	Module string
	// Run is the injected executor; nil uses os/exec.
	Run Runner
}

// Probe runs `python -m <Module> <args...>` and returns its trimmed output. A
// non-nil error means the probe failed (non-zero exit) or could not run; the
// returned string carries the probe's stderr/stdout for the caller's message.
func (in Invoker) Probe(ctx context.Context, args []string) (string, error) {
	if strings.TrimSpace(in.Python) == "" {
		return "", fmt.Errorf("smoke: no venv interpreter (Python environment not provisioned)")
	}
	full := append([]string{"-m", in.Module}, args...)
	var env []string
	if strings.TrimSpace(in.PythonPath) != "" {
		env = append(env, "PYTHONPATH="+in.PythonPath)
	}
	run := in.Run
	if run == nil {
		run = defaultRunner
	}
	out, err := run(ctx, in.Python, full, env)
	text := strings.TrimSpace(string(out))
	if err != nil {
		return text, fmt.Errorf("smoke probe failed: %w: %s", err, text)
	}
	return text, nil
}

func defaultRunner(ctx context.Context, python string, args, extraEnv []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, python, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	return cmd.CombinedOutput()
}

// Verdict is the cached outcome of a smoke probe for one installed model.
type Verdict struct {
	Pass      bool   `json:"pass"`
	Reason    string `json:"reason"`
	Kind      string `json:"kind"`
	ModelHash string `json:"model_hash"`
	LockHash  string `json:"lock_hash"`
	CheckedAt string `json:"checked_at,omitempty"`
}

// Fresh reports whether this cached verdict still applies for the given model and
// lock hashes (a changed model revision or a changed venv lock invalidates it).
func (v Verdict) Fresh(modelHash, lockHash string) bool {
	return v.ModelHash == modelHash && v.LockHash == lockHash && modelHash != "" && lockHash != ""
}

// ReadVerdict loads the cached verdict from a model dir, if present and parseable.
func ReadVerdict(modelDir string) (Verdict, bool) {
	data, err := os.ReadFile(filepath.Join(modelDir, VerdictFile))
	if err != nil {
		return Verdict{}, false
	}
	var v Verdict
	if err := json.Unmarshal(data, &v); err != nil {
		return Verdict{}, false
	}
	return v, true
}

// WriteVerdict persists a verdict into a model dir (best-effort cache; the caller
// treats a write error as non-fatal).
func WriteVerdict(modelDir string, v Verdict) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(modelDir, VerdictFile), append(data, '\n'), 0o644)
}
