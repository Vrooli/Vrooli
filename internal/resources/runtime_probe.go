package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

// RuntimeProbeReceipt describes import readiness only. It cannot establish
// model fit, GPU placement, inference quality or service readiness.
type RuntimeProbeReceipt struct {
	Resource       string                         `json:"resource"`
	ArtifactPath   string                         `json:"artifact_path"`
	ArtifactSHA256 string                         `json:"artifact_sha256"`
	Imports        []string                       `json:"imports"`
	ElapsedMS      int64                          `json:"elapsed_ms"`
	Status         string                         `json:"status"`
	Intent         platform.OperationIntent       `json:"intent"`
	Observation    *platform.OperationObservation `json:"observation,omitempty"`
	Error          string                         `json:"error,omitempty"`
}

var pythonImportName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)

// ProbeManagedRuntime uses the installed resource's launch environment, never
// a reconstructed interpreter-only environment. It acquires no dependencies
// or model data. The caller supplies a new receipt directory, outside the
// immutable artifact, so pre-dispatch intent survives interruption.
func (c *Controller) ProbeManagedRuntime(ctx context.Context, name string, imports []string, output string) (receipt RuntimeProbeReceipt, resultErr error) {
	started := time.Now()
	if ctx == nil {
		return receipt, errors.New("runtime probe requires context")
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return receipt, errors.New("runtime probe requires one resource name")
	}
	if err := validateRuntimeImports(imports); err != nil {
		return receipt, err
	}
	manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json"))
	if err != nil {
		return receipt, err
	}
	if manifest.Name != name {
		return receipt, errors.New("runtime probe resource identity differs from manifest")
	}
	if manifest.ManagedService == nil {
		return receipt, errors.New("runtime probe requires a managed service")
	}
	path, err := managedServiceArtifactPath(c, manifest)
	if err != nil {
		return receipt, err
	}
	if err := (managedServiceDriver{}).verifyArtifactAt(ctx, manifest, path); err != nil {
		return receipt, err
	}
	artifact, err := managedServiceArtifactForLaunch(ctx, manifest, path)
	if err != nil {
		return receipt, err
	}
	executable, err := artifact.LaunchPath(path)
	if err != nil {
		return receipt, err
	}
	if !strings.HasPrefix(filepath.Base(executable), "python") {
		return receipt, errors.New("import probe currently requires a Python runtime entrypoint")
	}
	env, err := managedServiceRuntimeEnvironment(ctx, c, manifest, path, artifact)
	if err != nil {
		return receipt, err
	}
	bubblewrap, err := exec.LookPath("bwrap")
	if err != nil {
		return receipt, fmt.Errorf("runtime probe requires installed bubblewrap: %w", err)
	}
	spec, err := runtimeProbeSpec(bubblewrap, executable, env, imports)
	if err != nil {
		return receipt, err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return receipt, err
	}
	artifactRoot := managedServiceEnvValues(env)["RESOURCE_ARTIFACT_DIR"]
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return receipt, err
	}
	output = filepath.Join(parent, filepath.Base(output))
	artifactRoot, err = filepath.EvalSymlinks(artifactRoot)
	if err != nil {
		return receipt, err
	}
	relative, err := filepath.Rel(artifactRoot, output)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return receipt, errors.New("runtime probe receipts must be outside the immutable artifact")
	}
	if err := os.Mkdir(output, 0o700); err != nil {
		return receipt, fmt.Errorf("runtime probe requires a new receipt directory: %w", err)
	}
	intent, err := platform.NewOperationIntent("resource-runtime-probe", fmt.Sprintf("probe-%d", time.Now().UnixNano()), spec)
	if err != nil {
		return receipt, err
	}
	receipt = RuntimeProbeReceipt{Resource: name, ArtifactPath: path, ArtifactSHA256: artifact.SHA256, Imports: append([]string(nil), imports...), Status: "prepared", Intent: intent}
	if err := writeRuntimeProbeJSON(filepath.Join(output, "intent.json"), receipt); err != nil {
		return receipt, err
	}
	// The existing native owner supplies wall/memory/task ceilings and process
	// tree cleanup. No task-local signal/child tracking wrapper is introduced.
	defer func() {
		receipt.ElapsedMS = time.Since(started).Milliseconds()
		if resultErr != nil {
			receipt.Status = "failed"
			receipt.Error = resultErr.Error()
		}
		resultErr = errors.Join(resultErr, writeRuntimeProbeJSON(filepath.Join(output, "result.json"), receipt))
	}()
	operation, err := platform.StartOwnedOperation(ctx, intent, spec)
	if err != nil {
		return receipt, err
	}
	// Install cleanup immediately after spawn, before any fallible receipt write.
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stopped, stopErr := operation.Stop(cleanup)
		if stopErr == nil {
			if !stopped.NativeQuiescent {
				stopErr = platform.ErrOperationNotQuiescent
			} else {
				stopErr = platform.ReleaseOwnedOperation(cleanup, operation.Ref)
			}
		}
		resultErr = errors.Join(resultErr, stopErr)
	}()
	if err := operation.Wait(ctx); err != nil {
		return receipt, err
	}
	observation, err := platform.InspectOwnedOperation(ctx, operation.Ref)
	receipt.Observation = &observation
	if err != nil {
		return receipt, err
	}
	if !observation.ExitKnown || observation.Populated || observation.ExecMainCode != 1 || observation.ExecMainStatus != 0 || observation.Result != "success" {
		return receipt, fmt.Errorf("runtime import probe did not exit successfully: result=%s code=%d status=%d", observation.Result, observation.ExecMainCode, observation.ExecMainStatus)
	}
	receipt.Status = "passed"
	return receipt, nil
}

func validateRuntimeImports(imports []string) error {
	if len(imports) == 0 || len(imports) > 16 {
		return errors.New("runtime probe requires between 1 and 16 Python module names")
	}
	for _, name := range imports {
		if len(name) > 128 || !pythonImportName.MatchString(name) {
			return fmt.Errorf("invalid Python import %q", name)
		}
	}
	return nil
}

func runtimeProbeSpec(bubblewrap, executable string, env, imports []string) (platform.OperationSpec, error) {
	if err := validateRuntimeImports(imports); err != nil {
		return platform.OperationSpec{}, err
	}
	modules, err := json.Marshal(imports)
	if err != nil {
		return platform.OperationSpec{}, err
	}
	// The artifact and host filesystem are read-only. The private PID/network
	// namespaces isolate IP networking and writable scratch. Filesystem sockets
	// are not a certified isolation boundary; this is a trusted-runtime probe.
	args := []string{"--unshare-user", "--unshare-pid", "--unshare-net", "--die-with-parent", "--ro-bind", "/", "/", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--clearenv", "--setenv", "HOME", "/tmp", "--setenv", "PYTHONDONTWRITEBYTECODE", "1", "--setenv", "PYTHONNOUSERSITE", "1", "--setenv", "CUDA_VISIBLE_DEVICES", "", "--setenv", "OMP_NUM_THREADS", "1", "--setenv", "HF_HUB_OFFLINE", "1"}
	values := managedServiceEnvValues(env)
	// Deliberately exclude credentials and application/model configuration.
	// These keys are the owner's resolved import/loader closure, not argv guesses.
	for _, key := range []string{"PATH", "PYTHONPATH", "PYTHONHOME", "LD_LIBRARY_PATH", "LD_PRELOAD", "RESOURCE_ARTIFACT_DIR"} {
		if value := values[key]; value != "" {
			args = append(args, "--setenv", key, value)
		}
	}
	args = append(args, "--chdir", "/tmp", "--", executable, "-B", "-s", "-c", "import importlib; [importlib.import_module(m) for m in "+string(modules)+"]")
	return platform.OperationSpec{Path: bubblewrap, Args: args, Env: []string{}, Dir: "/", Limits: platform.OperationLimits{MemoryMaxBytes: 2 << 30, TasksMax: 64, CPUWeight: 20, WallTime: 30 * time.Second}}, nil
}

func writeRuntimeProbeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(b, '\n'))
	syncErr := f.Sync()
	return errors.Join(writeErr, syncErr, f.Close())
}
