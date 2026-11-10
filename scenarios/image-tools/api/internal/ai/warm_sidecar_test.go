package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWarmResidencyRetainsClaimsAcrossJobsAndReleasesOnIdle(t *testing.T) {
	root, imports, outputs := writeSleepySidecarPackage(t)
	setPythonPath(t, root)
	runner := newWarmSidecarRunner()
	runner.idleTimeout = 20 * time.Millisecond
	t.Cleanup(func() { _ = runner.Close() })
	var acquired, released atomic.Int32
	closed := make(chan struct{}, 2)
	acquire := func(ctx context.Context) (*leaseGroup, error) {
		acquired.Add(1)
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
	for i := range 2 {
		meta := map[string]string{}
		admitted, err := runner.reserve(context.Background(), "same-model", fmt.Sprintf("job-%d", i), acquire, meta)
		if err != nil {
			t.Fatal(err)
		}
		if got := admitted.Meta["warm_reused"]; got != fmt.Sprint(i > 0) {
			t.Fatalf("reuse %d: %s", i, got)
		}
		if err := runner.Run(admitted.Context, "python3", []string{"-m", "benchmark_sidecar.sleepy", "--out", filepath.Join(outputs, fmt.Sprint(i))}); err != nil {
			t.Fatal(err)
		}
		if meta[workerResidencyIDMetadataKey] == "" || meta[workerResidencyModelMetadataKey] != "same-model" {
			t.Fatalf("residency identity metadata = %#v", meta)
		}
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
