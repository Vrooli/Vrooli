package validation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apimetrics "github.com/vrooli/api-core/metrics"
)

type incrementalTestScanner struct {
	name        string
	fingerprint *string
	planErr     error
	runs        atomic.Int32
}

type rawOSVCache struct {
	mu      sync.Mutex
	payload map[string][]byte
}

func (c *rawOSVCache) GetOSVScanCache(_ context.Context, scenario, key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	payload, ok := c.payload[scenario+"\x00"+key]
	return append([]byte(nil), payload...), ok
}

func (c *rawOSVCache) PutOSVScanCache(_ context.Context, scenario, key string, report []byte, _ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.payload == nil {
		c.payload = make(map[string][]byte)
	}
	c.payload[scenario+"\x00"+key] = append([]byte(nil), report...)
	return nil
}

type osvCountingCommander struct {
	runs atomic.Int32
	out  []byte
}

func (c *osvCountingCommander) LookPath(string) (string, error) {
	return "/logical/osv-scanner-v1", nil
}

func (c *osvCountingCommander) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, []byte, int, error) {
	if len(args) > 0 && args[0] == "scan" {
		c.runs.Add(1)
	}
	return c.out, nil, 0, nil
}

func TestOSVRawReportCacheIsSharedWithoutNormalizationDrift(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module demo\nrequire example.com/vuln v1.0.0\n")
	cmd := &osvCountingCommander{out: []byte(`{"results":[{"source":{"path":"api/go.mod"},"packages":[{"package":{"name":"example.com/vuln","version":"1.0.0","ecosystem":"Go"},"vulnerabilities":[{"id":"GO-TEST-1","summary":"test vuln"}]}]}]}`)}
	cache := &rawOSVCache{}
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	coldScanner := newOSVScannerWithCache(cmd, cache, func() time.Time { return now })
	cold, err := coldScanner.Scan(context.Background(), dir, Substrate{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	warmScanner := newOSVScannerWithCache(cmd, cache, func() time.Time { return now })
	warm, err := warmScanner.Scan(context.Background(), dir, Substrate{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.runs.Load() != 1 {
		t.Fatalf("osv scan subprocesses = %d, want 1", cmd.runs.Load())
	}
	if !reflect.DeepEqual(cold, warm) {
		t.Fatalf("cached normalization drifted: cold=%+v warm=%+v", cold, warm)
	}
}

func (s *incrementalTestScanner) Name() string           { return s.name }
func (s *incrementalTestScanner) Binary() string         { return s.name }
func (s *incrementalTestScanner) Applies(Substrate) bool { return true }
func (s *incrementalTestScanner) Scan(context.Context, string, Substrate) ([]Finding, error) {
	s.runs.Add(1)
	return []Finding{{RuleID: s.name + ".finding", Severity: SeverityInfo, Scanner: s.name}}, nil
}

func (s *incrementalTestScanner) EvidencePlan(context.Context, string, Substrate, time.Time) (ScannerEvidencePlan, error) {
	if s.planErr != nil {
		return ScannerEvidencePlan{}, s.planErr
	}
	return ScannerEvidencePlan{Fingerprint: *s.fingerprint, Weight: 1, FreshFor: time.Hour}, nil
}

// [REQ:REQ-P0-022] [REQ:REQ-P0-023]
func TestServiceIncrementalEvidencePreservesOutputAndRerunsOnlyChangedScanner(t *testing.T) {
	t.Log("[REQ:REQ-P0-022] [REQ:REQ-P0-023]")
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "scenarios", "demo", "README.md"), "demo")
	firstFingerprint, secondFingerprint := "first-v1", "second-v1"
	first := &incrementalTestScanner{name: "first", fingerprint: &firstFingerprint}
	second := &incrementalTestScanner{name: "second", fingerprint: &secondFingerprint}
	store := &memoryEvidenceStore{}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: store, Capacity: 2})
	service := New(Deps{
		RepoRoot:            repoRoot,
		Commander:           stubCommander{present: map[string]bool{"first": true, "second": true}},
		Scanners:            []Scanner{first, second},
		EvidenceCoordinator: coordinator,
	})

	cold, err := service.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	warm, err := service.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold.Findings, warm.Findings) {
		t.Fatalf("warm findings differ from cold:\ncold=%+v\nwarm=%+v", cold.Findings, warm.Findings)
	}
	if first.runs.Load() != 1 || second.runs.Load() != 1 {
		t.Fatalf("warm scanner runs first=%d second=%d, want 1/1", first.runs.Load(), second.runs.Load())
	}

	firstFingerprint = "first-v2"
	changed, err := service.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold.Findings, changed.Findings) {
		t.Fatalf("selective rerun changed findings: cold=%+v changed=%+v", cold.Findings, changed.Findings)
	}
	if first.runs.Load() != 2 || second.runs.Load() != 1 {
		t.Fatalf("selective scanner runs first=%d second=%d, want 2/1", first.runs.Load(), second.runs.Load())
	}
}

// [REQ:REQ-P0-024]
func TestServiceEmitsPerScannerEvidenceMetrics(t *testing.T) {
	t.Log("[REQ:REQ-P0-024]")
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "scenarios", "demo", "README.md"), "demo")
	fingerprint := "stable"
	scanner := &incrementalTestScanner{name: "measured", fingerprint: &fingerprint}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: &memoryEvidenceStore{}, Capacity: 1})
	service := New(Deps{
		RepoRoot: repoRoot, Commander: stubCommander{present: map[string]bool{"measured": true}},
		Scanners: []Scanner{scanner}, EvidenceCoordinator: coordinator,
	})
	if _, err := service.ValidateScenario(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	collector := apimetrics.Start()
	if _, err := service.ValidateScenario(WithMetrics(context.Background(), collector), "demo"); err != nil {
		t.Fatal(err)
	}
	result := collector.Stop()
	var measured bool
	for _, stage := range result.GetStages() {
		if stage.GetName() != "scan" {
			continue
		}
		for _, child := range stage.GetChildren() {
			if child.GetName() != "scanner:measured" {
				continue
			}
			measured = true
			gauges := child.GetGauges()
			if gauges["cache_hit"] != 1 || gauges["executed"] != 0 || gauges["findings"] != 1 {
				t.Fatalf("scanner gauges = %v", gauges)
			}
			for _, required := range []string{"available", "fingerprint_ms", "cache_miss", "coalesced", "uncached", "failed", "cache_error", "admission_wait_ms", "execution_ms"} {
				if _, ok := gauges[required]; !ok {
					t.Errorf("scanner gauge %q missing: %v", required, gauges)
				}
			}
			if child.GetResources() == nil {
				t.Fatal("scanner stage must carry resource measurements")
			}
		}
	}
	if !measured {
		t.Fatalf("scanner child stage missing: %+v", result.GetStages())
	}
}

func TestServiceFingerprintFailureExecutesUncached(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, "scenarios", "demo", "README.md"), "demo")
	fingerprint := "unused"
	scanner := &incrementalTestScanner{name: "broken-plan", fingerprint: &fingerprint, planErr: errors.New("unreadable input")}
	coordinator := NewEvidenceCoordinator(EvidenceCoordinatorDeps{Store: &memoryEvidenceStore{}, Capacity: 1})
	service := New(Deps{
		RepoRoot:  repoRoot,
		Commander: stubCommander{present: map[string]bool{"broken-plan": true}},
		Scanners:  []Scanner{scanner}, EvidenceCoordinator: coordinator,
	})
	for range 2 {
		if _, err := service.ValidateScenario(context.Background(), "demo"); err != nil {
			t.Fatal(err)
		}
	}
	if scanner.runs.Load() != 2 {
		t.Fatalf("scanner runs = %d, want 2 uncached runs", scanner.runs.Load())
	}
	metrics := coordinator.Metrics().Scanners[scanner.name]
	if metrics.UncachedExecutions != 2 {
		t.Fatalf("metrics = %+v, want 2 uncached executions", metrics)
	}
}

type pathCommander struct{ path string }

func (c pathCommander) LookPath(string) (string, error) { return c.path, nil }
func (pathCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, int, error) {
	return nil, nil, 0, nil
}

type gitInventoryCommander struct{ git Commander }

func (gitInventoryCommander) LookPath(string) (string, error) { return "/logical/gitleaks-v1", nil }
func (c gitInventoryCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, error) {
	if name == "git" {
		return c.git.Run(ctx, dir, name, args...)
	}
	return nil, nil, 0, nil
}

func TestGitleaksEvidenceMatchesCommitEligibleBoundary(t *testing.T) {
	exec := NewExecCommander()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		if _, stderr, code, err := exec.Run(context.Background(), dir, "git", args...); err != nil || code != 0 {
			t.Fatalf("git %v: code=%d err=%v stderr=%s", args, code, err, stderr)
		}
	}
	runGit("init", "-q")
	writeFile(t, filepath.Join(dir, ".gitignore"), "data/\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n")
	writeFile(t, filepath.Join(dir, "data", "runtime.db"), "cache-v1")
	runGit("add", ".gitignore", "main.go")
	cmd := gitInventoryCommander{git: exec}
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	base, err := scannerEvidencePlan(context.Background(), cmd, "gitleaks", "gitleaks", dir, Substrate{}, now)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "data", "runtime.db"), "cache-v2")
	ignoredChanged, err := scannerEvidencePlan(context.Background(), cmd, "gitleaks", "gitleaks", dir, Substrate{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if ignoredChanged.Fingerprint != base.Fingerprint {
		t.Fatal("ignored runtime state invalidated gitleaks evidence")
	}
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n// tracked change\n")
	trackedChanged, err := scannerEvidencePlan(context.Background(), cmd, "gitleaks", "gitleaks", dir, Substrate{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if trackedChanged.Fingerprint == base.Fingerprint {
		t.Fatal("tracked source change did not invalidate gitleaks evidence")
	}
}

func TestScannerEvidencePlanInvalidatesRelevantSourceToolAndAdvisoryEpoch(t *testing.T) {
	t.Log("[REQ:REQ-P0-022]")
	dir := t.TempDir()
	goFile := filepath.Join(dir, "api", "main.go")
	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module demo\n")
	writeFile(t, goFile, "package main\n")
	writeFile(t, filepath.Join(dir, "ui", "node_modules", "ignored.go"), "ignored")
	tool := filepath.Join(t.TempDir(), "gosec")
	if err := os.WriteFile(tool, []byte("tool-v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := Substrate{Go: true, GoModDirs: []string{"api"}}
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	cmd := pathCommander{path: tool}

	base, err := scannerEvidencePlan(context.Background(), cmd, "gosec", "gosec", dir, sub, now)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "ui", "node_modules", "ignored.go"), "changed but irrelevant")
	irrelevant, err := scannerEvidencePlan(context.Background(), cmd, "gosec", "gosec", dir, sub, now)
	if err != nil {
		t.Fatal(err)
	}
	if irrelevant.Fingerprint != base.Fingerprint {
		t.Fatal("generated dependency content invalidated gosec evidence")
	}
	writeFile(t, goFile, "package main\n// relevant change\n")
	sourceChanged, err := scannerEvidencePlan(context.Background(), cmd, "gosec", "gosec", dir, sub, now)
	if err != nil {
		t.Fatal(err)
	}
	if sourceChanged.Fingerprint == base.Fingerprint {
		t.Fatal("relevant source change did not invalidate evidence")
	}
	if err := os.WriteFile(tool, []byte("tool-v2-expanded"), 0o755); err != nil {
		t.Fatal(err)
	}
	toolChanged, err := scannerEvidencePlan(context.Background(), cmd, "gosec", "gosec", dir, sub, now)
	if err != nil {
		t.Fatal(err)
	}
	if toolChanged.Fingerprint == sourceChanged.Fingerprint {
		t.Fatal("tool change did not invalidate evidence")
	}

	dayOne, err := scannerEvidencePlan(context.Background(), cmd, "govulncheck", "gosec", dir, sub, now)
	if err != nil {
		t.Fatal(err)
	}
	dayTwo, err := scannerEvidencePlan(context.Background(), cmd, "govulncheck", "gosec", dir, sub, now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if dayOne.Fingerprint == dayTwo.Fingerprint {
		t.Fatal("advisory epoch change did not invalidate evidence")
	}
}

func TestGoScannerEvidenceStopsAtNestedModuleBoundaries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module root\n")
	writeFile(t, filepath.Join(root, "internal", "owned.go"), "package internal\n")
	writeFile(t, filepath.Join(root, "scenarios", "runtime-only", "data", "runtime.db"), "volatile")
	writeFile(t, filepath.Join(root, "scenarios", "nested", "go.mod"), "module nested\n")
	writeFile(t, filepath.Join(root, "scenarios", "nested", "runtime.db"), "volatile")

	files, err := walkModuleEvidenceFiles(root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) == "runtime.db" || strings.Contains(filepath.ToSlash(file), "scenarios/nested/") {
			t.Fatalf("nested module input leaked into root-module evidence: %s", file)
		}
	}
	if !slices.Contains(files, filepath.Join(root, "internal", "owned.go")) {
		t.Fatalf("root-module source missing from evidence: %v", files)
	}
}
