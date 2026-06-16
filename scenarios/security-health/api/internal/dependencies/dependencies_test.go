package dependencies

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"security-health/internal/clock"
	"security-health/internal/testutil/db"
	"security-health/internal/validation"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	d := db.NewSQLite(t)
	if _, err := d.ExecContext(context.Background(), schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return NewStore(d)
}

func TestSplitPnpmKey(t *testing.T) {
	cases := map[string][2]string{
		"esbuild@0.21.5":             {"esbuild", "0.21.5"},
		"@scope/pkg@1.2.3":           {"@scope/pkg", "1.2.3"},
		"/vite@5.0.0":                {"vite", "5.0.0"},
		"vite@5.0.0(@types/node@20)": {"vite", "5.0.0"},
		"nameonly":                   {"", ""},
	}
	for key, want := range cases {
		n, v := splitPnpmKey(key)
		if n != want[0] || v != want[1] {
			t.Errorf("splitPnpmKey(%q) = (%q,%q), want (%q,%q)", key, n, v, want[0], want[1])
		}
	}
}

func TestDiscoverScenario_GoAndPnpm(t *testing.T) {
	dir := t.TempDir()
	writeF(t, filepath.Join(dir, "api", "go.mod"), "module x\ngo 1.24\nrequire (\n\tgolang.org/x/net v0.17.0\n\tgithub.com/foo/bar v1.2.3 // indirect\n)\n")
	writeF(t, filepath.Join(dir, "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\npackages:\n  esbuild@0.21.5: {}\n  '@scope/pkg@1.0.0': {}\n")
	recs, err := DiscoverScenario(dir, "demo")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Ecosystem{}
	for _, r := range recs {
		got[r.Name+"@"+r.Version] = r.Ecosystem
	}
	if got["golang.org/x/net@v0.17.0"] != EcosystemGo {
		t.Errorf("expected go dep golang.org/x/net, got %+v", got)
	}
	if got["esbuild@0.21.5"] != EcosystemNPM {
		t.Errorf("expected npm dep esbuild, got %+v", got)
	}
	if got["@scope/pkg@1.0.0"] != EcosystemNPM {
		t.Errorf("expected scoped npm dep, got %+v", got)
	}
}

func TestStore_ApplyDiffSearch(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	recs := []DependencyRecord{
		{Scenario: "a", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", SourceFile: "api/go.mod", VulnIDs: []string{"GO-1"}, MaxSeverity: "high"},
		{Scenario: "a", Ecosystem: EcosystemNPM, Name: "react", Version: "18.0.0", SourceFile: "ui/pnpm-lock.yaml"},
		{Scenario: "b", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", SourceFile: "api/go.mod", VulnIDs: []string{"GO-1"}, MaxSeverity: "high"},
	}
	// Initial diff: all new, no deletes.
	up, del, err := s.Diff(ctx, "", recs)
	if err != nil || up != 3 || del != 0 {
		t.Fatalf("Diff = (%d,%d,%v), want (3,0,nil)", up, del, err)
	}
	if err := s.Apply(ctx, "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Count(ctx); n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
	if n, _ := s.VulnerableCount(ctx); n != 2 {
		t.Errorf("vulnerable count = %d, want 2", n)
	}

	// Structured filter: vulnerable-only across the fleet.
	hits, err := s.Search(ctx, SearchRequest{VulnerableOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("vulnerable-only search = %d hits, want 2", len(hits))
	}

	// name glob + ecosystem filter.
	hits, _ = s.Search(ctx, SearchRequest{Ecosystem: EcosystemGo, NameGlob: "golang.org/x/*"})
	if len(hits) != 2 {
		t.Fatalf("glob search = %d hits, want 2", len(hits))
	}

	// free-text query.
	hits, _ = s.Search(ctx, SearchRequest{Query: "react"})
	if len(hits) != 1 || hits[0].Record.Name != "react" {
		t.Fatalf("text search wrong: %+v", hits)
	}

	// Re-apply with one record dropped → that row is deleted.
	up, del, _ = s.Diff(ctx, "", recs[:2])
	if del != 1 {
		t.Errorf("expected 1 delete after dropping a record, got %d (up=%d)", del, up)
	}
}

func TestStore_ListVulnerabilitiesAggregatesScenarioExposure(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	vuln := VulnerabilityRecord{
		VulnerabilityID:    "GHSA-1234",
		Aliases:            []string{"CVE-2026-0001"},
		Ecosystem:          EcosystemNPM,
		Name:               "vite",
		Version:            "5.0.0",
		AffectedRanges:     []AffectedVersionRange{{Range: "<5.1.0", Fixed: "5.1.0"}},
		FixedRanges:        []FixedVersionRange{{Range: ">= 5.1.0", Version: "5.1.0"}},
		NormalizedSeverity: "high",
		Source:             VulnerabilitySourceOSV,
		Reachability:       ReachabilityLockfileAffected,
		Confidence:         EvidenceConfidenceAdvisory,
		Production:         true,
	}
	recs := []DependencyRecord{
		{Scenario: "a", Ecosystem: EcosystemNPM, Name: "vite", Version: "5.0.0", SourceFile: "ui/pnpm-lock.yaml", VulnIDs: []string{"GHSA-1234"}, MaxSeverity: "high", Vulnerabilities: []VulnerabilityRecord{vuln}},
		{Scenario: "b", Ecosystem: EcosystemNPM, Name: "vite", Version: "5.0.0", SourceFile: "worker/pnpm-lock.yaml", VulnIDs: []string{"GHSA-1234"}, MaxSeverity: "high", Vulnerabilities: []VulnerabilityRecord{vuln}},
	}
	if err := s.Apply(ctx, "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListVulnerabilities(ctx, VulnerabilityQuery{PackageName: "vite", MinimumConfidence: EvidenceConfidenceAdvisory})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Vulnerabilities) != 1 {
		t.Fatalf("expected one aggregated vulnerability, got %+v", list)
	}
	got := list.Vulnerabilities[0]
	if got.VulnerabilityID != "GHSA-1234" || got.FixedRanges[0].Range != ">= 5.1.0" {
		t.Fatalf("vulnerability details not preserved: %+v", got)
	}
	if len(got.Scenarios) != 2 || got.Scenarios[0] != "a" || got.Scenarios[1] != "b" {
		t.Fatalf("scenario exposure not aggregated: %+v", got.Scenarios)
	}
}

func TestStore_ListVulnerabilitiesFallsBackToDependencyRows(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	recs := []DependencyRecord{
		{Scenario: "a", Ecosystem: EcosystemNPM, Name: "vite", Version: "5.0.0", SourceFile: "ui/pnpm-lock.yaml", VulnIDs: []string{"GHSA-1234"}, MaxSeverity: "high"},
		{Scenario: "b", Ecosystem: EcosystemNPM, Name: "vite", Version: "5.0.0", SourceFile: "worker/pnpm-lock.yaml", VulnIDs: []string{"GHSA-1234"}, MaxSeverity: "moderate"},
	}
	if err := s.Apply(ctx, "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListVulnerabilities(ctx, VulnerabilityQuery{
		Ecosystem:       EcosystemNPM,
		PackageName:     "vite",
		VulnerabilityID: "GHSA-1234",
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Vulnerabilities) != 1 {
		t.Fatalf("expected one fallback vulnerability, got %+v", list)
	}
	got := list.Vulnerabilities[0]
	if got.VulnerabilityID != "GHSA-1234" || got.Confidence != EvidenceConfidenceDegraded || got.Reachability != ReachabilityLockfileAffected {
		t.Fatalf("fallback evidence not marked degraded lockfile evidence: %+v", got)
	}
	if got.NormalizedSeverity != "high" {
		t.Fatalf("severity = %q, want high", got.NormalizedSeverity)
	}
	if len(got.Scenarios) != 2 || got.Scenarios[0] != "a" || got.Scenarios[1] != "b" {
		t.Fatalf("scenario exposure not aggregated: %+v", got.Scenarios)
	}

	advisoryOnly, err := s.ListVulnerabilities(ctx, VulnerabilityQuery{
		PackageName:       "vite",
		MinimumConfidence: EvidenceConfidenceAdvisory,
	})
	if err != nil {
		t.Fatal(err)
	}
	if advisoryOnly.Total != 0 || len(advisoryOnly.Vulnerabilities) != 0 {
		t.Fatalf("degraded fallback should not satisfy advisory confidence: %+v", advisoryOnly)
	}
}

func TestService_ExplainVulnerabilityFallsBackToDependencyRows(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	if err := store.Apply(ctx, "", []DependencyRecord{{
		Scenario: "demo", Ecosystem: EcosystemNPM, Name: "vite", Version: "5.0.0",
		SourceFile: "ui/pnpm-lock.yaml", VulnIDs: []string{"GHSA-1234"}, MaxSeverity: "high",
	}}, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	svc := NewService(Deps{Store: store})
	got, found, err := svc.ExplainVulnerability(ctx, VulnerabilityQuery{
		VulnerabilityID: "GHSA-1234",
		Ecosystem:       EcosystemNPM,
		PackageName:     "vite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found || got.VulnerabilityID != "GHSA-1234" || got.Confidence != EvidenceConfidenceDegraded {
		t.Fatalf("expected degraded fallback explanation, found=%v got=%+v", found, got)
	}
}

func TestBuildVulnIndex_PreservesOSVRangesAndFixedVersions(t *testing.T) {
	report := validation.OSVReport{Results: []validation.OSVResult{{
		Packages: []validation.OSVPackage{{
			Package: validation.OSVPackageInfo{Name: "vite", Version: "5.0.0", Ecosystem: "npm"},
			Vulnerabilities: []validation.OSVVuln{{
				ID:      "GHSA-1234",
				Aliases: []string{"CVE-2026-0001"},
				Summary: "test vuln",
				DatabaseSpecific: struct {
					Severity string `json:"severity"`
				}{Severity: "HIGH"},
				Affected: []struct {
					Ranges []struct {
						Events []struct {
							Introduced   string `json:"introduced"`
							Fixed        string `json:"fixed"`
							LastAffected string `json:"last_affected"`
						} `json:"events"`
					} `json:"ranges"`
				}{{
					Ranges: []struct {
						Events []struct {
							Introduced   string `json:"introduced"`
							Fixed        string `json:"fixed"`
							LastAffected string `json:"last_affected"`
						} `json:"events"`
					}{{
						Events: []struct {
							Introduced   string `json:"introduced"`
							Fixed        string `json:"fixed"`
							LastAffected string `json:"last_affected"`
						}{{Introduced: "0", Fixed: "5.1.0"}},
					}},
				}},
			}},
		}},
	}}}
	index := buildVulnIndex(report)
	entry := index[depMatchKey(EcosystemNPM, "vite", "5.0.0")]
	if len(entry.vulnerabilities) != 1 {
		t.Fatalf("expected one vulnerability, got %+v", entry)
	}
	got := entry.vulnerabilities[0]
	if got.AffectedRanges[0].Range != ">=0 <5.1.0" || got.FixedRanges[0].Range != ">= 5.1.0" {
		t.Fatalf("range evidence not preserved: %+v", got)
	}
	if got.Confidence != EvidenceConfidenceAdvisory || got.Reachability != ReachabilityLockfileAffected {
		t.Fatalf("OSV evidence classification wrong: %+v", got)
	}
}

func TestService_ReindexDryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	repoRoot := t.TempDir()
	scen := filepath.Join(repoRoot, "scenarios", "demo")
	writeF(t, filepath.Join(scen, "api", "go.mod"), "module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")

	svc := NewService(Deps{
		RepoRoot:  repoRoot,
		Store:     s,
		Annotator: NewAnnotator(repoRoot, noopCommander{}),
		Clock:     clock.System{},
	})
	res, err := svc.Reindex(ctx, "demo", true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.PlannedUpserts != 1 {
		t.Fatalf("dry-run plan wrong: %+v", res)
	}
	if n, _ := s.Count(ctx); n != 0 {
		t.Errorf("dry-run must not write; count=%d", n)
	}
}

func TestService_ReindexRealRunReturnsBeforeAnnotationCompletes(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	repoRoot := t.TempDir()
	scen := filepath.Join(repoRoot, "scenarios", "demo")
	writeF(t, filepath.Join(scen, "api", "go.mod"), "module x\ngo 1.24\nrequire golang.org/x/net v0.17.0\n")
	cmd := &blockingCommander{started: make(chan struct{}), release: make(chan struct{})}

	svc := NewService(Deps{
		RepoRoot:  repoRoot,
		Store:     store,
		Annotator: NewAnnotator(repoRoot, cmd),
		Clock:     clock.System{},
	})

	started := make(chan ReindexResult, 1)
	errs := make(chan error, 1)
	go func() {
		res, err := svc.Reindex(ctx, "demo", false)
		if err != nil {
			errs <- err
			return
		}
		started <- res
	}()

	select {
	case err := <-errs:
		t.Fatalf("Reindex returned error: %v", err)
	case res := <-started:
		if res.DryRun || res.JobID == "" {
			t.Fatalf("real reindex should return job id immediately, got %+v", res)
		}
		state, _, _, _, ok := svc.ReindexStatus(res.JobID)
		if !ok || state != "pending" && state != "running" {
			t.Fatalf("job status = %q ok=%v, want pending/running", state, ok)
		}
		cancelled, ok := svc.ReindexCancel(res.JobID)
		if !ok || !cancelled {
			t.Fatalf("cancel = (%v,%v), want true,true", cancelled, ok)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("real reindex did not return before annotation completed")
	}

	close(cmd.release)
}

func TestService_StatusReportsCounts(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	_ = s.Apply(ctx, "", []DependencyRecord{
		{Scenario: "a", Ecosystem: EcosystemGo, Name: "m", Version: "1", VulnIDs: []string{"X"}, MaxSeverity: "high"},
	}, "2026-06-01T00:00:00Z")
	svc := NewService(Deps{RepoRoot: t.TempDir(), Store: s, Annotator: NewAnnotator("", noopCommander{}), Clock: clock.System{}})
	st, err := svc.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Available || st.IndexedCount != 1 || st.VulnerableCount != 1 {
		t.Fatalf("status wrong: %+v", st)
	}
}

func TestService_ReindexCancelUnknownJob(t *testing.T) {
	svc := NewService(Deps{RepoRoot: t.TempDir(), Store: newTestStore(t), Annotator: NewAnnotator("", noopCommander{}), Clock: clock.System{}})
	if _, _, _, _, ok := svc.ReindexStatus("ghost"); ok {
		t.Error("unknown job must report ok=false")
	}
	if _, ok := svc.ReindexCancel("ghost"); ok {
		t.Error("cancel of unknown job must report ok=false")
	}
}

// noopCommander makes osv-scanner "absent" so annotation is a no-op in tests.
type noopCommander struct{}

func (noopCommander) LookPath(string) (string, error) { return "", os.ErrNotExist }
func (noopCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, int, error) {
	return nil, nil, 0, nil
}

type blockingCommander struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingCommander) LookPath(string) (string, error) { return "/bin/osv-scanner", nil }

func (b *blockingCommander) Run(ctx context.Context, _ string, _ string, _ ...string) ([]byte, []byte, int, error) {
	select {
	case <-b.started:
	default:
		close(b.started)
	}
	select {
	case <-ctx.Done():
		return nil, nil, 1, ctx.Err()
	case <-b.release:
		return []byte(`{"results":[]}`), nil, 0, nil
	}
}

func writeF(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var _ SQLExecutor = (*sql.DB)(nil)
