package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeFile creates a file (and parents) with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// goFixture lays out a minimal Go scenario with a SQLite engine and two
// domains under a temp scenarios/ root. Returns repoRoot.
func goFixture(t *testing.T, scenario string) string {
	t.Helper()
	repoRoot := t.TempDir()
	dir := filepath.Join(repoRoot, "scenarios", scenario)
	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module "+scenario+"\n\nrequire modernc.org/sqlite v1.50.1\n")
	writeFile(t, filepath.Join(dir, "api", "internal", "database", "system.sql"), "-- system\n")
	writeFile(t, filepath.Join(dir, "api", "internal", "orders", "repo.go"), "package orders\n")
	writeFile(t, filepath.Join(dir, "api", "handlers", "billing", "handler.go"), "package billing\n")
	writeFile(t, filepath.Join(dir, ".vrooli", "service.json"), `{"maturity":"production","dependencies":{"resources":["postgres","redis"]}}`)
	return repoRoot
}

func TestFilesystemDetector_Go(t *testing.T) {
	repoRoot := goFixture(t, "demo-go")
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo-go")
	got := FilesystemDetector{}.Detect(context.Background(), "demo-go", scenarioDir)
	if got.Language != "go" {
		t.Fatalf("language = %q, want go", got.Language)
	}
	wantDomains := map[string]bool{"orders": true, "billing": true}
	if len(got.Domains) != len(wantDomains) {
		t.Fatalf("domains = %v, want orders+billing only (infra excluded)", got.Domains)
	}
	for _, d := range got.Domains {
		if !wantDomains[d] {
			t.Fatalf("unexpected domain %q in %v", d, got.Domains)
		}
	}
}

func TestFilesystemDetector_NonGo(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo-ts")
	writeFile(t, filepath.Join(scenarioDir, "api", "package.json"), `{"name":"demo-ts"}`)
	got := FilesystemDetector{}.Detect(context.Background(), "demo-ts", scenarioDir)
	if got.Language != "typescript" {
		t.Fatalf("language = %q, want typescript", got.Language)
	}
}

type blockingResolver struct{}

func (blockingResolver) ResolveScenarioURLDefault(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func TestCodeFactsDetector_TimeoutFallsBackToFilesystem(t *testing.T) {
	repoRoot := goFixture(t, "demo-go")
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo-go")

	start := time.Now()
	got := CodeFactsDetector{
		Resolver: blockingResolver{},
		Timeout:  5 * time.Millisecond,
	}.Detect(context.Background(), "demo-go", scenarioDir)

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Detect took %s, want short fallback", elapsed)
	}
	if got.Language != "go" {
		t.Fatalf("language = %q, want filesystem fallback go", got.Language)
	}
	if len(got.Domains) == 0 {
		t.Fatalf("domains = %v, want filesystem fallback domains", got.Domains)
	}
}

func TestDetectEnginesAndStage(t *testing.T) {
	repoRoot := goFixture(t, "demo-go")
	scenarioDir := filepath.Join(repoRoot, "scenarios", "demo-go")

	engines := detectEngines(scenarioDir)
	want := map[Engine]bool{EnginePostgres: true, EngineRedis: true, EngineSQLite: true}
	if len(engines) != len(want) {
		t.Fatalf("engines = %v, want postgres+redis+sqlite", engines)
	}
	for _, e := range engines {
		if !want[e] {
			t.Fatalf("unexpected engine %q", e)
		}
	}

	stage, hasMigrations := deriveStorageStage(scenarioDir)
	if stage != "production" {
		t.Fatalf("stage = %q, want production", stage)
	}
	if hasMigrations {
		t.Fatalf("hasMigrations = true, want false (no migrations dir)")
	}
}

func TestDeriveStorageStage_DefaultsGreenfield(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "bare")
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{}`)
	stage, _ := deriveStorageStage(scenarioDir)
	if stage != "greenfield" {
		t.Fatalf("stage = %q, want greenfield (absent maturity)", stage)
	}
}

func TestValidateScenario_MissingTargetIsGracefulSkip(t *testing.T) {
	svc := New(Deps{RepoRoot: t.TempDir(), Detector: FilesystemDetector{}})
	rep, err := svc.ValidateScenario(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("ValidateScenario error = %v, want nil (graceful skip)", err)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Code != "STORAGE_TARGET_UNRESOLVABLE" {
		t.Fatalf("findings = %+v, want single STORAGE_TARGET_UNRESOLVABLE", rep.Findings)
	}
	if rep.Findings[0].Severity != SeverityInfo {
		t.Fatalf("unresolvable severity = %v, want INFO", rep.Findings[0].Severity)
	}
}

func TestValidateScenario_CleanWithZeroAnalyzers(t *testing.T) {
	repoRoot := goFixture(t, "demo-go")
	// Inject an empty analyzer set so this exercises the detect→aggregate path
	// independent of which analyzer tiers are registered.
	svc := New(Deps{RepoRoot: repoRoot, Detector: FilesystemDetector{}, Analyzers: []Analyzer{}})
	rep, err := svc.ValidateScenario(context.Background(), "demo-go")
	if err != nil {
		t.Fatalf("ValidateScenario error = %v", err)
	}
	// The accountability rung marker is not an analyzer tier: it is the
	// structural guard that stops an owner declaring nothing from inheriting
	// L3 "governed end to end" by producing no findings. It must survive an
	// empty analyzer set, so the fixture (which declares no storage entries)
	// reports exactly the undeclared rung and nothing else.
	if len(rep.Findings) != 1 || rep.Findings[0].Code != "STORAGE_ACCOUNTABILITY_UNDECLARED" {
		t.Fatalf("findings = %+v, want only STORAGE_ACCOUNTABILITY_UNDECLARED", rep.Findings)
	}
	if rep.Findings[0].Severity != SeverityInfo {
		t.Fatalf("rung severity = %v, want INFO so adoption never fails the phase", rep.Findings[0].Severity)
	}
	if rep.Status != "passed" {
		t.Fatalf("status = %q, want passed", rep.Status)
	}
	if !rep.IsGo() {
		t.Fatalf("language = %q, want go", rep.Language)
	}
	if !rep.HasEngine(EngineSQLite) {
		t.Fatalf("engines = %v, want sqlite present", rep.Engines)
	}
}
