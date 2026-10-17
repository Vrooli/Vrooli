package recovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/baselinefloor"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenario"
)

// declaredScenario seeds a scenario manifest with an api component, runtime
// data, build outputs and .vrooli configuration, plus a shared package.
func declaredScenario(t *testing.T, svc Service, scenarioDir string) {
	t.Helper()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(scenarioDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".vrooli/service.json", `{"service":{"name":"demo"},"components":{"api":{"role":"api","build":{"kind":"go_module","dir":"api","output":"api/demo-api"},"run":{"argv":["{{bin.api}}"],"cwd":"api","data_dirs":["var/uploads"]}}}}`)
	write(".vrooli/testing.json", `{}`)
	write(".vrooli/secrets.json", `{"token":"x"}`)
	write("api/demo-api", "ELF")
	write("var/uploads/a.png", "png")
	write("data/demo.db", "live data")
	write("server.log", "log line")
	if err := os.WriteFile(filepath.Join(svc.Root, ".gitignore"), []byte("data/\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(svc.Root, "packages", "shared")
	if err := os.MkdirAll(filepath.Join(shared, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "dist", "index.js"), []byte("export const v = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixedRoots(roots ...lifecycle.SharedSourceRoot) func(context.Context, string, scenario.Scenario) ([]lifecycle.SharedSourceRoot, error) {
	return func(context.Context, string, scenario.Scenario) ([]lifecycle.SharedSourceRoot, error) {
		return roots, nil
	}
}

// TestCaptureKeepsSourceAndConfigAndDropsRuntimeData is the O16 restore-point
// scope regression: the restore point holds source and the scenario's
// .vrooli configuration (a BAS UI file imported service.json), and leaves out
// databases, logs, build outputs, declared data directories and secrets.
func TestCaptureKeepsSourceAndConfigAndDropsRuntimeData(t *testing.T) {
	svc, scenarioDir := newTestService(t, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	declaredScenario(t, svc, scenarioDir)
	svc.SharedSourceRoots = fixedRoots(lifecycle.SharedSourceRoot{Path: "packages/shared", Reason: "file:@demo/shared"})

	out, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	for _, kept := range []string{"api/main.go", ".vrooli/service.json", ".vrooli/testing.json"} {
		if _, err := os.Stat(filepath.Join(out.RestorePointPath, kept)); err != nil {
			t.Errorf("restore point lacks %s: %v", kept, err)
		}
	}
	for _, dropped := range []string{"data", "server.log", "api/demo-api", "var/uploads", ".vrooli/secrets.json", "node_modules"} {
		if _, err := os.Stat(filepath.Join(out.RestorePointPath, dropped)); !os.IsNotExist(err) {
			t.Errorf("restore point holds %s (err=%v)", dropped, err)
		}
	}
	if !out.SharedSources.Frozen() || len(out.SharedSources.Roots) != 1 || out.CaptureID == "" {
		t.Fatalf("capture output = %+v", out)
	}
	frozen := filepath.Join(svc.Store.SharedSourcesPath("demo", "abc"), "packages", "shared", "dist", "index.js")
	if got, err := os.ReadFile(frozen); err != nil || string(got) != "export const v = 1;\n" {
		t.Fatalf("frozen generated output = %q, %v", got, err)
	}
	record, ok, err := svc.Store.ReadCaptureRecord("demo", "abc")
	if err != nil || !ok || record.CaptureID != out.CaptureID || !record.SharedSources.Frozen() {
		t.Fatalf("capture record = %+v ok=%t err=%v", record, ok, err)
	}
}

// A recapture (baseline start --replace) gets a new capture ID, so the next
// serving start builds a tree from it instead of reusing the old copy.
func TestRecaptureGetsNewCaptureID(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	svc, scenarioDir := newTestService(t, now)
	declaredScenario(t, svc, scenarioDir)
	svc.SharedSourceRoots = fixedRoots()
	first, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow"}); err != nil {
		t.Fatal(err)
	}
	svc.Clock = fixedClock(now.Add(time.Hour))
	second, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true, Replace: true})
	if err != nil {
		t.Fatalf("replace recapture: %v", err)
	}
	if second.CaptureID == first.CaptureID {
		t.Fatalf("recapture kept capture ID %q", first.CaptureID)
	}
}

// O15: a capture into an open engagement would overwrite the restore point
// the engaged live instance serves; it is refused without --replace.
func TestCaptureRefusesToOverwriteOpenEngagement(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	svc, scenarioDir := newTestService(t, now)
	svc.SharedSourceRoots = fixedRoots()
	if _, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true})
	if !errors.Is(err, ErrEngagementLive) || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("capture over open engagement: %v", err)
	}
	frozen, _ := os.ReadFile(filepath.Join(svc.Store.RestorePointPath("demo", "abc"), "api", "main.go"))
	if string(frozen) != "package main\n" {
		t.Fatalf("refused capture still changed the restore point: %q", frozen)
	}
	// An expired engagement is no longer open.
	svc.Clock = fixedClock(now.Add(2 * time.Hour))
	if _, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true}); err != nil {
		t.Fatalf("capture over expired engagement: %v", err)
	}
}

// Freeze failure policy (P-11): a capture fails closed when the shared
// sources cannot be resolved, before the restore point changes; with
// --allow-unfrozen it records an unfrozen capture.
func TestCaptureFailsClosedWhenSharedSourcesUnresolved(t *testing.T) {
	svc, scenarioDir := newTestService(t, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	declaredScenario(t, svc, scenarioDir)
	svc.SharedSourceRoots = func(context.Context, string, scenario.Scenario) ([]lifecycle.SharedSourceRoot, error) {
		return nil, errors.New("go list -deps: exit status 1")
	}
	_, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true})
	if !errors.Is(err, ErrSharedSourcesUnresolved) || !strings.Contains(err.Error(), "--allow-unfrozen") {
		t.Fatalf("unresolved freeze: %v", err)
	}
	if _, err := os.Stat(svc.Store.RestorePointPath("demo", "abc")); !os.IsNotExist(err) {
		t.Fatalf("failed freeze still wrote a restore point (err=%v)", err)
	}

	out, err := svc.Capture(CaptureRequest{Scenario: "demo", Slug: "abc", NoReflink: true, AllowUnfrozen: true})
	if err != nil {
		t.Fatalf("allow-unfrozen capture: %v", err)
	}
	if out.SharedSources.Status != baselinefloor.SharedSourcesUnfrozen || !strings.Contains(out.SharedSources.Reason, "go list") {
		t.Fatalf("unfrozen capture = %+v", out.SharedSources)
	}
	view, err := svc.WriteEngagement(WriteRequest{Scenario: "demo", Slug: "abc", Mode: "shadow"})
	if err != nil || view.Capture == nil || view.Capture.SharedSources.Frozen() {
		t.Fatalf("engagement view must report the unfrozen capture: %+v %v", view.Capture, err)
	}
}

// Restore uses the destination's ignore rules: a restore point captured
// before runtime data was excluded never overlays its stale database onto
// the working tree.
func TestRestoreKeepsWorkingTreeRuntimeData(t *testing.T) {
	svc, scenarioDir := newTestService(t, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	declaredScenario(t, svc, scenarioDir)
	legacy := svc.Store.RestorePointPath("demo", "abc")
	for rel, content := range map[string]string{"api/main.go": "package main\n", "data/demo.db": "last week"} {
		path := filepath.Join(legacy, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Restore(RestoreRequest{Scenario: "demo", Slug: "abc", NoReflink: true}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(scenarioDir, "api", "main.go")); string(got) != "package main\n" {
		t.Errorf("source not restored: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(scenarioDir, "data", "demo.db")); string(got) != "live data" {
		t.Errorf("restore overlaid stale runtime data: %q", got)
	}
}
