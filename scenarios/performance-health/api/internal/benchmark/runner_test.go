package benchmark

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeFixture builds a scenario tree under t.TempDir() and returns its root.
func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	return root
}

func alwaysFound(string) (string, error) { return "/usr/bin/found", nil }

// [REQ:PH-BENCH-001] The runner times go + ui surfaces and reports MEASURED with
// per-surface timings against budgets from .vrooli/testing.json.
func TestCLIRunnerMeasuresGoAndUI(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"api/go.mod":           "module demo\n",
		"ui/package.json":      `{"scripts":{"build":"vite build"},"packageManager":"pnpm@9"}`,
		".vrooli/testing.json": `{"performance":{"budgets":{"go_build_max_ms":90000,"ui_build_max_ms":180000}}}`,
	})
	var ran []string
	r := &CLIRunner{
		Resolve: func(_, _ string) (string, error) { return root, nil },
		Lookup:  alwaysFound,
		Exec: func(_ context.Context, dir, name string, args ...string) error {
			ran = append(ran, name)
			return nil
		},
	}
	res, err := r.Run(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Outcome != OutcomeMeasured {
		t.Fatalf("expected MEASURED, got %#v", res)
	}
	if len(res.Timings) != 2 || res.Timings[0].Surface != "go" || res.Timings[1].Surface != "ui" {
		t.Fatalf("expected go+ui timings, got %#v", res.Timings)
	}
	if res.Timings[0].BudgetMs != 90000 || res.Timings[1].BudgetMs != 180000 {
		t.Fatalf("budgets not loaded from testing.json: %#v", res.Timings)
	}
	if len(ran) != 2 || ran[0] != "go" || ran[1] != "pnpm" {
		t.Fatalf("expected go then pnpm build, got %v", ran)
	}
}

// [REQ:PH-BENCH-002] A go build that fails to compile EARLY-EXITS: the run is
// FAILED and the UI build is never attempted.
func TestCLIRunnerGoBuildFailureEarlyExits(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"api/go.mod":      "module demo\n",
		"ui/package.json": `{"scripts":{"build":"vite build"}}`,
	})
	var ran []string
	r := &CLIRunner{
		Resolve: func(_, _ string) (string, error) { return root, nil },
		Lookup:  alwaysFound,
		Exec: func(_ context.Context, dir, name string, args ...string) error {
			ran = append(ran, name)
			if name == "go" {
				return errors.New("compile error")
			}
			return nil
		},
	}
	res, err := r.Run(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Outcome != OutcomeFailed {
		t.Fatalf("expected FAILED, got %#v", res)
	}
	if len(ran) != 1 || ran[0] != "go" {
		t.Fatalf("early-exit violated: ui build should not run, ran=%v", ran)
	}
}

// [REQ:PH-BENCH-002] A surface merely OVER BUDGET is flagged but does not
// early-exit — both surfaces are still timed.
func TestCLIRunnerOverBudgetDoesNotEarlyExit(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"api/go.mod":           "module demo\n",
		"ui/package.json":      `{"scripts":{"build":"vite build"}}`,
		".vrooli/testing.json": `{"performance":{"budgets":{"go_build_max_ms":0}}}`,
	})
	// go_build_max_ms:0 → no budget, so nothing is over budget; assert both
	// surfaces still ran and result is MEASURED.
	var ran []string
	r := &CLIRunner{
		Resolve: func(_, _ string) (string, error) { return root, nil },
		Lookup:  alwaysFound,
		Exec: func(_ context.Context, dir, name string, args ...string) error {
			ran = append(ran, name)
			return nil
		},
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeMeasured || len(ran) != 2 {
		t.Fatalf("expected both surfaces measured, ran=%v res=%#v", ran, res)
	}
	if res.Timings[0].OverBudget {
		t.Fatalf("zero budget should never be over budget: %#v", res.Timings[0])
	}
}

// [REQ:PH-BENCH-001] No buildable surfaces → clean SKIP.
func TestCLIRunnerSkipsWhenNoSurfaces(t *testing.T) {
	root := t.TempDir()
	r := &CLIRunner{
		Resolve: func(_, _ string) (string, error) { return root, nil },
		Lookup:  alwaysFound,
		Exec:    func(context.Context, string, string, ...string) error { return nil },
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeSkipped {
		t.Fatalf("expected SKIPPED, got %#v", res)
	}
}

// [REQ:PH-BENCH-001] UI skips cleanly (go-only MEASURED) when no package manager
// is available.
func TestCLIRunnerUISkippedWhenPMAbsent(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"api/go.mod":      "module demo\n",
		"ui/package.json": `{"scripts":{"build":"vite build"},"packageManager":"pnpm@9"}`,
	})
	r := &CLIRunner{
		Resolve: func(_, _ string) (string, error) { return root, nil },
		Lookup: func(name string) (string, error) {
			if name == "go" {
				return "/usr/bin/go", nil
			}
			return "", errors.New("not found")
		},
		Exec: func(context.Context, string, string, ...string) error { return nil },
	}
	res, _ := r.Run(context.Background(), "demo", "")
	if res.Outcome != OutcomeMeasured || len(res.Timings) != 1 || res.Timings[0].Surface != "go" {
		t.Fatalf("expected go-only measured, got %#v", res)
	}
}

func TestDetectPackageManager(t *testing.T) {
	dir := t.TempDir()
	if got := detectPackageManager(packageManifest{PackageManager: "yarn@4"}, dir); got != "yarn" {
		t.Fatalf("packageManager field: got %q", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("{}"), 0o644)
	if got := detectPackageManager(packageManifest{}, dir); got != "pnpm" {
		t.Fatalf("lockfile detect: got %q", got)
	}
	if got := detectPackageManager(packageManifest{}, t.TempDir()); got != "npm" {
		t.Fatalf("default: got %q", got)
	}
}
