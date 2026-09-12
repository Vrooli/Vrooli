package ai

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

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

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("resolve current file")
	}
	sidecarRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "sidecar", "py"))
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
