package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	internaljobs "image-tools/internal/jobs"
)

func TestWarmResidencyRetainsClaimsAcrossJobsAndReleasesOnIdle(t *testing.T) {
	root, imports, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	runner.idleTimeout = 20 * time.Millisecond
	t.Cleanup(func() { _ = runner.Close() })
	var acquired, released atomic.Int32
	var lastResidencyID string
	closed := make(chan struct{}, 2)
	for i := range 2 {
		meta := map[string]string{"disk_job_attempt_id": fmt.Sprintf("job-%d", i)}
		acquire := func(ctx context.Context) (*leaseGroup, error) {
			acquired.Add(1)
			meta["gpu_capacity_claim_id"] = "resident-claim"
			group := newLeaseGroup(ctx)
			group.renew("RAM", &memoryTestLease{onRelease: func(context.Context) {
				if runner.cmd != nil || runner.exited != nil {
					t.Error("resident RAM released before worker reaped")
				}
				released.Add(1)
				closed <- struct{}{}
			}})
			return group, nil
		}
		admitted, err := runner.reserve(context.Background(), "same-model", fmt.Sprintf("job-%d", i), acquire, meta)
		if err != nil {
			t.Fatal(err)
		}
		if got := admitted.Meta["warm_reused"]; got != fmt.Sprint(i > 0) {
			t.Fatalf("reuse %d: %s", i, got)
		}
		if got := meta["disk_job_attempt_id"]; got != fmt.Sprintf("job-%d", i) {
			t.Fatalf("job-scoped metadata leaked across warm jobs: %q", got)
		}
		if meta["gpu_capacity_claim_id"] != "resident-claim" {
			t.Fatalf("resident GPU claim metadata was not retained: %#v", meta)
		}
		if !runner.hasResidency(meta[workerResidencyIDMetadataKey]) {
			t.Fatal("successful request lost its resource-backed warm residency before idle timeout")
		}
		if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, fmt.Sprint(i))}); err != nil {
			t.Fatal(err)
		}
		if meta[workerResidencyIDMetadataKey] == "" || meta[workerResidencyModelMetadataKey] != "same-model" {
			t.Fatalf("residency identity metadata = %#v", meta)
		}
		lastResidencyID = meta[workerResidencyIDMetadataKey]
		if meta[workerResidencyPeakRSSStatusKey] != "measured" && meta[workerResidencyPeakRSSStatusKey] != "unavailable" {
			t.Fatalf("worker residency RSS was not explicitly classified: %#v", meta)
		}
		if runtime.GOOS == "linux" {
			if meta[workerResidencyPeakRSSStatusKey] != "measured" || meta[workerResidencyPeakRSSSourceKey] != workerResidencyPeakRSSLinuxSource || meta[workerResidencyPIDMetadataKey] == "" || meta[workerResidencyPeakRSSBytesKey] == "" {
				t.Fatalf("Linux worker high-water RSS missing: %#v", meta)
			}
		} else if meta[workerResidencyPeakRSSStatusKey] != "unavailable" || !strings.Contains(meta[workerResidencyPeakRSSReasonKey], "unsupported") {
			t.Fatalf("unsupported platform should remain unavailable: %#v", meta)
		}
		if err := admitted.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
		if acquired.Load() != 1 || released.Load() != 0 {
			t.Fatalf("claims not retained across jobs: acquired=%d released=%d", acquired.Load(), released.Load())
		}
	}
	if got := importCount(t, imports); got != 1 {
		t.Fatalf("warm jobs reloaded Python module %d times", got)
	}
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("idle worker retained capacity")
	}
	if released.Load() != 1 {
		t.Fatalf("resident release count=%d", released.Load())
	}
	if runner.hasResidency(lastResidencyID) {
		t.Fatal("idle worker still reports its RAM/GPU residency after releasing its claim")
	}
}

func TestWarmResidencyDoesNotReportCanceledLeaseGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := newWarmSidecarRunner()
	runner.resident = newLeaseGroup(ctx)
	runner.residencyID = "resident-claim"
	cancel()
	if runner.hasResidency("resident-claim") {
		t.Fatal("canceled residency still reports its GPU/RAM claim as held")
	}
}

func TestWarmResidencyCancellationAndModelSwitchReapBeforeRelease(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	writeWarmSleepModule(t, root)
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	var live atomic.Int32
	acquire := func(ctx context.Context) (*leaseGroup, error) {
		if live.Add(1) != 1 {
			t.Error("model switch retained old RAM")
		}
		group := newLeaseGroup(ctx)
		group.renew("RAM", &memoryTestLease{onRelease: func(cleanup context.Context) {
			if cleanup.Err() != nil || runner.cmd != nil {
				t.Error("canceled cleanup or unreaped worker")
			}
			live.Add(-1)
		}})
		return group, nil
	}
	first, err := runner.reserve(context.Background(), "model-a", "job-model-a", acquire, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(first.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, "one")}); err != nil {
		t.Fatal(err)
	}
	if err := first.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	second, err := runner.reserve(ctx, "model-b", "job-model-b", acquire, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(second.Context, "python3", []string{"-m", "benchmark_sidecar.long_sleep"}); err == nil {
		t.Fatal("cancel did not stop warm worker")
	}
	if err := second.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if live.Load() != 0 || runner.resident != nil || len(runner.gate) != 0 {
		t.Fatalf("canceled residency leaked claims or gate: live=%d", live.Load())
	}
}

func TestWarmSidecarRunnerMarksStaleResidencyIdentityUnavailable(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	meta := map[string]string{}
	admitted, err := runner.reserve(context.Background(), "model-a", "job-a", func(ctx context.Context) (*leaseGroup, error) {
		return newLeaseGroup(ctx), nil
	}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, "stale-id.txt")}); err != nil {
		t.Fatal(err)
	}
	meta[workerResidencyIDMetadataKey] = "previous-residency"
	runner.mu.Lock()
	runner.recordResidencyPeakRSSLocked(admitted.Context)
	runner.mu.Unlock()
	if meta[workerResidencyPeakRSSStatusKey] != "unavailable" || !strings.Contains(meta[workerResidencyPeakRSSReasonKey], "identity changed") || meta[workerResidencyPeakRSSBytesKey] != "" {
		t.Fatalf("stale residency identity was not rejected: %#v", meta)
	}
	if err := admitted.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWarmSidecarRunnerReturnsPerRequestInferenceMetrics(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	moduleDir := filepath.Join(root, "benchmark_sidecar")
	if err := os.WriteFile(filepath.Join(moduleDir, "metrics.py"), []byte("def main():\n    return {'device': 'cuda', 'forward_ms': 17, 'model_cache_hit': True}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	meta := map[string]string{}
	admitted, err := runner.reserve(context.Background(), "metric-model", "metric-job", func(ctx context.Context) (*leaseGroup, error) {
		return newLeaseGroup(ctx), nil
	}, meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.metrics", "--out", filepath.Join(outputs, "metrics")}); err != nil {
		t.Fatal(err)
	}
	if meta["inference_device"] != "cuda" || meta["inference_forward_ms"] != "17" || meta["inference_model_cache_hit"] != "True" {
		t.Fatalf("per-request worker metrics missing from admission metadata: %#v", meta)
	}
	if err := admitted.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWarmSidecarRunner_AmortizesPythonImport(t *testing.T) {
	root, imports, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)

	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })

	for i := 0; i < 2; i++ {
		out := filepath.Join(outputs, fmt.Sprintf("warm-%d.txt", i))
		if err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", out}); err != nil {
			t.Fatalf("warm run %d: %v", i, err)
		}
	}
	if got := importCount(t, imports); got != 1 {
		t.Fatalf("warm worker should import the module once, got %d imports", got)
	}

	if err := os.WriteFile(imports, nil, 0o600); err != nil {
		t.Fatalf("reset import marker: %v", err)
	}
	for i := 0; i < 2; i++ {
		out := filepath.Join(outputs, fmt.Sprintf("cold-%d.txt", i))
		cmd := exec.Command("python3", "-m", "benchmark_sidecar.sleepy", "--out", out)
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("one-shot run %d: %v\n%s", i, err, out)
		}
	}
	if got := importCount(t, imports); got != 2 {
		t.Fatalf("one-shot runner should import the module every call, got %d imports", got)
	}
}

func TestWarmSidecarRunner_ReapsOperationAdmittedWorkerBeforeReturning(t *testing.T) {
	root, imports, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	ctx := context.WithValue(context.Background(), operationMemoryContextKey{}, true)
	for i := 0; i < 2; i++ {
		out := filepath.Join(outputs, fmt.Sprintf("admitted-%d.txt", i))
		if err := runner.Run(ctx, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", out}); err != nil {
			t.Fatal(err)
		}
		if runner.cmd != nil || runner.exited != nil || runner.idleTimer != nil {
			t.Fatal("operation returned with a resident worker after its RAM ownership ended")
		}
		if _, err := os.Stat(out); err != nil {
			t.Fatalf("worker did not complete output before reaping: %v", err)
		}
	}
	if got := importCount(t, imports); got != 2 {
		t.Fatalf("each admitted operation should own and reap its worker, imports=%d", got)
	}
}

func TestWarmSidecarRunner_ReturnsSidecarFailuresAndRestarts(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	writeWarmFailureModule(t, root)
	setPythonPath(t, root)

	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })

	if err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.fail"}); err == nil || !strings.Contains(err.Error(), "sidecar exited with code 12") {
		t.Fatalf("expected SystemExit error, got %v", err)
	}
	if err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.explode"}); err == nil || !strings.Contains(err.Error(), "boom") || !strings.Contains(err.Error(), "Traceback") {
		t.Fatalf("expected traceback error, got %v", err)
	}
	out := filepath.Join(outputs, "after-fail.txt")
	if err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", out}); err != nil {
		t.Fatalf("worker should restart after request failure: %v", err)
	}
}

func TestWarmSidecarRunner_StartAndCancelFailures(t *testing.T) {
	runner := newWarmSidecarRunner()
	if err := runner.Run(context.Background(), "definitely-missing-python-binary", []string{"-m", "x"}); err == nil || !strings.Contains(err.Error(), "warm sidecar start") {
		t.Fatalf("expected start failure, got %v", err)
	}

	root, _, _ := writeSleepySidecarPackage(t)
	writeWarmSleepModule(t, root)
	setPythonPath(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := runner.Run(ctx, "python3", []string{"-m", "benchmark_sidecar.long_sleep"}); err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("expected context deadline, got %v", err)
	}
}

func TestWarmSidecarRunner_ReapsIdleProcessAndStartsFresh(t *testing.T) {
	root, imports, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	runner.idleTimeout = 20 * time.Millisecond
	t.Cleanup(func() { _ = runner.Close() })
	args := []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, "idle.txt")}
	if err := runner.Run(context.Background(), "python3", args); err != nil {
		t.Fatal(err)
	}
	runner.mu.Lock()
	exited := runner.exited
	runner.mu.Unlock()
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		t.Fatal("unused model worker remained resident")
	}
	if err := runner.Run(context.Background(), "python3", args); err != nil {
		t.Fatal(err)
	}
	if got := importCount(t, imports); got != 2 {
		t.Fatalf("expected fresh model process after eviction; imports=%d", got)
	}
}

func TestWarmSidecarRunner_QueuedCancellationDoesNotLaunch(t *testing.T) {
	runner := newWarmSidecarRunner()
	runner.gate <- struct{}{} // simulate an active inference
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runner.Run(ctx, "must-not-launch", nil); err != context.Canceled {
		t.Fatalf("queued cancellation = %v", err)
	}
	<-runner.gate
}

func BenchmarkWarmSidecarRunner_AmortizesModuleLoad(b *testing.B) {
	root, _, outputs := writeSleepySidecarPackage(b)
	setPythonPath(b, root)

	b.Run("one-shot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			out := filepath.Join(outputs, fmt.Sprintf("cold-bench-%d.txt", i))
			cmd := exec.Command("python3", "-m", "benchmark_sidecar.sleepy", "--out", out)
			cmd.Env = os.Environ()
			if cmdOut, err := cmd.CombinedOutput(); err != nil {
				b.Fatalf("one-shot run: %v\n%s", err, cmdOut)
			}
		}
	})

	b.Run("warm", func(b *testing.B) {
		runner := newWarmSidecarRunner()
		b.Cleanup(func() { _ = runner.Close() })
		for i := 0; i < b.N; i++ {
			out := filepath.Join(outputs, fmt.Sprintf("warm-bench-%d.txt", i))
			if err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", out}); err != nil {
				b.Fatalf("warm run: %v", err)
			}
		}
	})
}

func writeSleepySidecarPackage(tb testing.TB) (root string, imports string, outputs string) {
	tb.Helper()

	root = tb.TempDir()
	imports = filepath.Join(root, "imports.txt")
	outputs = filepath.Join(root, "outputs")
	pkg := filepath.Join(root, "benchmark_sidecar")
	if err := os.MkdirAll(pkg, 0o755); err != nil {
		tb.Fatalf("mkdir package: %v", err)
	}
	if err := os.MkdirAll(outputs, 0o755); err != nil {
		tb.Fatalf("mkdir outputs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "__init__.py"), []byte(""), 0o600); err != nil {
		tb.Fatalf("write package init: %v", err)
	}

	module := fmt.Sprintf(`
import argparse
import os
import time

time.sleep(0.025)
with open(%q, "a", encoding="utf-8") as marker:
    marker.write("import\n")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    with open(args.out, "w", encoding="utf-8") as out:
        out.write("ok\n")
`, imports)
	if err := os.WriteFile(filepath.Join(pkg, "sleepy.py"), []byte(module), 0o600); err != nil {
		tb.Fatalf("write sleepy module: %v", err)
	}
	if err := os.WriteFile(imports, nil, 0o600); err != nil {
		tb.Fatalf("write import marker: %v", err)
	}
	return root, imports, outputs
}

func writeWarmFailureModule(tb testing.TB, root string) {
	tb.Helper()
	pkg := filepath.Join(root, "benchmark_sidecar")
	if err := os.WriteFile(filepath.Join(pkg, "fail.py"), []byte(`
def main():
    raise SystemExit(12)
`), 0o600); err != nil {
		tb.Fatalf("write fail module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "explode.py"), []byte(`
def main():
    raise RuntimeError("boom")
`), 0o600); err != nil {
		tb.Fatalf("write explode module: %v", err)
	}
}

func writeWarmSleepModule(tb testing.TB, root string) {
	tb.Helper()
	pkg := filepath.Join(root, "benchmark_sidecar")
	if err := os.WriteFile(filepath.Join(pkg, "long_sleep.py"), []byte(`
import time

def main():
    time.sleep(1.0)
`), 0o600); err != nil {
		tb.Fatalf("write sleep module: %v", err)
	}
}

func setPythonPath(tb testing.TB, extra string) {
	tb.Helper()

	sidecarRoot := packageRelative(tb, "..", "sidecar", "py")
	parts := []string{extra, sidecarRoot}
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		parts = append(parts, existing)
	}
	tb.Setenv("PYTHONPATH", strings.Join(parts, string(os.PathListSeparator)))
}

func importCount(tb testing.TB, path string) int {
	tb.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read import marker: %v", err)
	}
	return strings.Count(string(data), "import\n")
}

// A graceful shutdown closes the backend registry. A resident warm worker must
// give back every claim it holds then, not leave them to the heartbeat expiry
// (which keeps the memory refused for other work after the restart).
func TestClosingTheWarmRunnerReleasesEveryResidentClaim(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })
	var ram, vram atomic.Int32
	acquire := func(ctx context.Context) (*leaseGroup, error) {
		group := newLeaseGroup(ctx)
		group.renew("RAM", &memoryTestLease{onRelease: func(context.Context) {
			if runner.cmd != nil {
				t.Error("resident RAM released before the worker was reaped")
			}
			ram.Add(1)
		}})
		group.renew("GPU residency", &memoryTestLease{onRelease: func(context.Context) { vram.Add(1) }})
		return group, nil
	}
	admitted, err := runner.reserve(context.Background(), "model", "job", acquire, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, "one")}); err != nil {
		t.Fatal(err)
	}
	if err := admitted.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ram.Load() != 0 || vram.Load() != 0 {
		t.Fatalf("the idle window must keep the claims: ram=%d vram=%d", ram.Load(), vram.Load())
	}
	if err := runner.Close(); err != nil {
		t.Fatal(err)
	}
	if ram.Load() != 1 || vram.Load() != 1 {
		t.Fatalf("Close left a resident claim held: ram=%d vram=%d", ram.Load(), vram.Load())
	}
	if runner.resident != nil || runner.cmd != nil {
		t.Fatal("Close left the worker or its residency in place")
	}
}

// A request the module rejects as given is the caller's fault, not the model's:
// the worker's kind reaches the job as a failure kind, and an ordinary crash stays a fault.
func TestWarmSidecarRunnerClassifiesAnInvalidRequestApartFromAFault(t *testing.T) {
	root, _, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	moduleDir := filepath.Join(root, "benchmark_sidecar")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(moduleDir, name+".py"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("rejects", "from image_tools_sidecar import _common\n\ndef main():\n    raise _common.InputError(\"this parser has no label for 'book'\")\n")
	write("crashes", "def main():\n    raise RuntimeError('CUDA error: an illegal memory access')\n")
	runner := newWarmSidecarRunner()
	t.Cleanup(func() { _ = runner.Close() })

	err := runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.rejects", "--out", filepath.Join(outputs, "r")})
	if err == nil || !strings.Contains(err.Error(), "no label for 'book'") || internaljobs.KindOf(err) != internaljobs.FailureInvalidInput {
		t.Fatalf("invalid request: err=%v kind=%q", err, internaljobs.KindOf(err))
	}
	err = runner.Run(context.Background(), "python3", []string{"-m", "benchmark_sidecar.crashes", "--out", filepath.Join(outputs, "c")})
	if err == nil || !strings.Contains(err.Error(), "illegal memory access") || internaljobs.KindOf(err) != internaljobs.FailureFault {
		t.Fatalf("crash: err=%v kind=%q", err, internaljobs.KindOf(err))
	}
}

func TestOneShotSidecarExitStatusSeparatesABadRequestFromACrash(t *testing.T) {
	_, badRequest := defaultRunOutput(context.Background(), "sh", []string{"-c", fmt.Sprintf("echo no label >&2; exit %d", sidecarExitInvalidInput)})
	if got := internaljobs.KindOf(classifySidecarFailure(badRequest)); got != internaljobs.FailureInvalidInput {
		t.Fatalf("exit %d kind = %q", sidecarExitInvalidInput, got)
	}
	_, crash := defaultRunOutput(context.Background(), "sh", []string{"-c", "exit 1"})
	if got := internaljobs.KindOf(classifySidecarFailure(crash)); got != internaljobs.FailureFault {
		t.Fatalf("exit 1 kind = %q", got)
	}
	if classifySidecarFailure(nil) != nil {
		t.Fatal("no failure must stay no failure")
	}
}

func TestPolicyRefusalsAreNotFaultsOfTheModel(t *testing.T) {
	for name, err := range map[string]error{
		"consent":  fmt.Errorf("%w: the edit leaves this machine", ErrConsentRequired),
		"spend":    fmt.Errorf("%w: $0.30 over the cap", ErrSpendCap),
		"cooldown": ErrCloudCooldown,
		"safety":   errors.Join(&HardRefusalError{Reason: "policy"}, errors.New("deleted outputs")),
	} {
		if got := internaljobs.KindOf(err); got != internaljobs.FailureRefused {
			t.Errorf("%s kind = %q, want refused", name, got)
		}
		if !errors.Is(err, ErrConsentRequired) && name == "consent" {
			t.Error("the consent sentinel must still match errors.Is")
		}
	}
}
