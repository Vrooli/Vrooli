package providerreadiness

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/internal/orchestrator/phasepolicy"
)

// stageProvider materializes a provider api dir with one source file and a
// freshness manifest stamped from it, mirroring what a real build produces.
func stageProvider(t *testing.T, provider string) (repoRoot, digest string) {
	t.Helper()
	repoRoot = t.TempDir()
	apiDir := filepath.Join(repoRoot, "scenarios", provider, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(apiDir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(apiDir, provider+"-api")
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(repoRoot, src)
	if err != nil {
		t.Fatal(err)
	}
	spec := cliutil.FreshnessSpec{
		SourceRoot:  apiDir,
		ContextRoot: repoRoot,
		Inputs:      []string{filepath.ToSlash(rel)},
		SkipFiles:   []string{filepath.Base(binary)},
	}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if err := cliutil.WriteFreshnessManifest(cliutil.FreshnessManifestPath(binary), manifest); err != nil {
		t.Fatal(err)
	}
	return repoRoot, manifest.Digest
}

func TestStalenessFailsOpenWithoutRepoRoot(t *testing.T) {
	if v := EvaluateProviderStaleness("", "security-health", "abc"); v.Stale {
		t.Error("an unset repo root produced a stale verdict; the check must fail open")
	}
}

func TestStalenessFailsOpenWithoutManifest(t *testing.T) {
	repo := t.TempDir()
	if v := EvaluateProviderStaleness(repo, "security-health", "abc"); v.Stale {
		t.Error("a provider with no freshness manifest was reported stale")
	}
}

func TestStalenessFailsOpenOnUnknownReportedDigest(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	// A provider that could not stamp a digest reports "". That is unknown, and
	// unknown must never be grounds for a restart.
	if v := EvaluateProviderStaleness(repo, "security-health", ""); v.Stale {
		t.Errorf("empty reported digest treated as stale: %+v", v)
	}
}

func TestStalenessAcceptsMatchingDigest(t *testing.T) {
	repo, digest := stageProvider(t, "security-health")
	if v := EvaluateProviderStaleness(repo, "security-health", digest); v.Stale {
		t.Errorf("a provider running the on-disk build was reported stale: %+v", v)
	}
}

func TestStalenessDetectsRebuildWithoutRestart(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	// The running process reports the digest it started with; the manifest on
	// disk has since been rewritten by a rebuild.
	v := EvaluateProviderStaleness(repo, "security-health", "0000000000000000")
	if !v.Stale {
		t.Fatal("a rebuilt-but-not-restarted provider was not detected")
	}
	if v.Reason == "" {
		t.Error("verdict carried no reason")
	}
}

func TestStalenessDetectsSourceChange(t *testing.T) {
	repo, digest := stageProvider(t, "security-health")
	src := filepath.Join(repo, "scenarios", "security-health", "api", "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){println(1)}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := EvaluateProviderStaleness(repo, "security-health", digest)
	if !v.Stale {
		t.Fatal("a source change after the build was not detected")
	}
}

// countingLifecycle records restarts so the rails can be asserted precisely.
type countingLifecycle struct {
	restarted []string
	err       error
}

func (c *countingLifecycle) Start(context.Context, string, io.Writer) error { return nil }
func (c *countingLifecycle) Restart(_ context.Context, scenario string, _ io.Writer) error {
	c.restarted = append(c.restarted, scenario)
	return c.err
}

func staleManager(t *testing.T, repo string, lifecycle Lifecycle, max int) *Manager {
	t.Helper()
	return &Manager{
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{
				Reachable: true, ContractValid: true, IdentityMatch: true,
				FreshnessDigest: "0000000000000000", // never matches the staged digest
			}, nil
		},
		Lifecycle:        lifecycle,
		RepoRoot:         repo,
		MaxStaleRestarts: max,
	}
}

func staleInput() Input {
	in := input()
	in.Policy.ProviderLifecycle = phasepolicy.ProviderLifecycleStartIfNeeded
	return in
}

func TestStaleProviderIsRestartedOnce(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	lifecycle := &countingLifecycle{}
	m := staleManager(t, repo, lifecycle, 4)

	out := m.Check(context.Background(), staleInput(), io.Discard)
	if !out.Ready {
		t.Fatalf("a stale-but-restartable provider was not ready: %+v", out)
	}
	if len(lifecycle.restarted) != 1 || lifecycle.restarted[0] != "security-health" {
		t.Fatalf("restarted = %v, want exactly [security-health] — restarts must never cascade", lifecycle.restarted)
	}
}

func TestStaleRestartBudgetIsEnforced(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	lifecycle := &countingLifecycle{}
	m := staleManager(t, repo, lifecycle, 1)

	for i := 0; i < 3; i++ {
		if out := m.Check(context.Background(), staleInput(), io.Discard); !out.Ready {
			t.Fatalf("iteration %d: provider not ready: %+v", i, out)
		}
	}
	if len(lifecycle.restarted) != 1 {
		t.Errorf("restarted %d times, want 1 — the per-run budget did not hold", len(lifecycle.restarted))
	}
	report := m.StaleReport()
	if report.Restarted != 1 || len(report.Skipped) != 2 {
		t.Errorf("report = %+v, want 1 restarted and 2 skipped", report)
	}
}

func TestNegativeBudgetReportsWithoutRestarting(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	lifecycle := &countingLifecycle{}
	m := staleManager(t, repo, lifecycle, -1)

	out := m.Check(context.Background(), staleInput(), io.Discard)
	if len(lifecycle.restarted) != 0 {
		t.Errorf("a negative budget still restarted: %v", lifecycle.restarted)
	}
	if !out.Ready {
		t.Errorf("reporting-only mode must not block the phase: %+v", out)
	}
	if out.Message == "" {
		t.Error("staleness was neither acted on nor reported")
	}
}

// A restart that fails must not fail the phase: the provider is still answering,
// just with older code. Losing the whole run over it would be a worse outcome.
func TestFailedStaleRestartDoesNotBlockThePhase(t *testing.T) {
	repo, _ := stageProvider(t, "security-health")
	lifecycle := &countingLifecycle{err: errors.New("port in use")}
	m := staleManager(t, repo, lifecycle, 4)

	out := m.Check(context.Background(), staleInput(), io.Discard)
	if !out.Ready {
		t.Fatalf("a failed restart blocked the phase: %+v", out)
	}
	if out.Message == "" {
		t.Error("the failed restart was not surfaced")
	}
}

// With no repo root the gate is entirely inert — the fail-open default.
func TestNoRepoRootNeverRestarts(t *testing.T) {
	lifecycle := &countingLifecycle{}
	m := staleManager(t, "", lifecycle, 4)

	if out := m.Check(context.Background(), staleInput(), io.Discard); !out.Ready {
		t.Fatalf("provider not ready: %+v", out)
	}
	if len(lifecycle.restarted) != 0 {
		t.Errorf("restarted with no repo root configured: %v", lifecycle.restarted)
	}
}

func TestClassifyOffenderCategories(t *testing.T) {
	cases := []struct {
		name, reason, file string
		want               StalenessClass
	}{
		{"own code", "size changed", "scenarios/security-health/api/internal/scan/rules.go", StalenessOwnCode},
		{"shared package", "size changed", "packages/api-core/database/connect.go", StalenessSharedPackage},
		{"another scenario compiled in", "size changed", "scenarios/other-thing/api/lib.go", StalenessSharedPackage},
		{"dependency manifest", "size changed", "packages/maturity-go/go.mod", StalenessDependency},
		{"root dependency manifest", "size changed", "go.sum", StalenessDependency},
		{"toolchain key input", "build input changed", "toolchain", StalenessToolchain},
		{"unclassifiable", "content changed", "somefile.txt", StalenessOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyOffender("security-health", tc.reason, tc.file); got != tc.want {
				t.Errorf("classifyOffender(%q) = %q, want %q", tc.file, got, tc.want)
			}
		})
	}
}

// The message must say what kind of change happened, not just that something did.
func TestDescribeNamesTheCategoryAndFile(t *testing.T) {
	v := StalenessVerdict{Stale: true, Class: StalenessSharedPackage, File: "packages/api-core/database/connect.go"}
	got := v.Describe()
	if !strings.Contains(got, "shared package") || !strings.Contains(got, "connect.go") {
		t.Errorf("Describe() = %q, want it to name both the category and the file", got)
	}
	if (StalenessVerdict{}).Describe() != "" {
		t.Error("a non-stale verdict produced a description")
	}
}

func TestCooldownSuppressesRepeatSharedPackageRestarts(t *testing.T) {
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	now := time.Now()
	ledger.record("security-health", StalenessSharedPackage, now)

	cooling, remaining := ledger.cooling("security-health", StalenessSharedPackage, 30*time.Minute, now.Add(5*time.Minute))
	if !cooling {
		t.Fatal("a provider restarted 5 minutes ago was not held back from another shared-package restart")
	}
	if remaining <= 0 {
		t.Errorf("remaining = %v, want a positive window", remaining)
	}
}

// The cooldown must never hold back a provider whose own code changed: that is
// the case where the stale result is actively misleading rather than just dated.
func TestCooldownNeverSuppressesOwnCodeChanges(t *testing.T) {
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	now := time.Now()
	ledger.record("security-health", StalenessSharedPackage, now)

	for _, class := range []StalenessClass{StalenessOwnCode, StalenessRebuilt} {
		if cooling, _ := ledger.cooling("security-health", class, 30*time.Minute, now.Add(time.Second)); cooling {
			t.Errorf("class %q was held back by the cooldown; it must always act", class)
		}
	}
}

func TestCooldownExpires(t *testing.T) {
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	now := time.Now()
	ledger.record("security-health", StalenessSharedPackage, now)

	if cooling, _ := ledger.cooling("security-health", StalenessSharedPackage, 30*time.Minute, now.Add(31*time.Minute)); cooling {
		t.Error("cooldown did not expire after its window")
	}
}

func TestNilLedgerAndUnknownProviderNeverCool(t *testing.T) {
	var nilLedger *restartLedger
	if cooling, _ := nilLedger.cooling("x", StalenessSharedPackage, time.Hour, time.Now()); cooling {
		t.Error("a nil ledger reported cooling; the cooldown must fail open")
	}
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	if cooling, _ := ledger.cooling("never-restarted", StalenessSharedPackage, time.Hour, time.Now()); cooling {
		t.Error("a provider with no restart record reported cooling")
	}
}

// The ledger has to survive across runs, since the churn window it guards
// against spans many runs.
func TestLedgerPersistsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restarts.json")
	now := time.Now()
	NewRestartLedgerAt(path).record("security-health", StalenessSharedPackage, now)

	reopened := NewRestartLedgerAt(path)
	if cooling, _ := reopened.cooling("security-health", StalenessSharedPackage, 30*time.Minute, now.Add(time.Minute)); !cooling {
		t.Error("a fresh ledger instance did not see the persisted restart; the cooldown would never fire across runs")
	}
}

// stageProviderWithSharedPackage stages a provider whose manifest also covers a
// shared package, so a change there produces a shared_package verdict — the
// class the cooldown actually governs.
func stageProviderWithSharedPackage(t *testing.T, provider string) (repoRoot, digest, sharedFile string) {
	t.Helper()
	repoRoot = t.TempDir()
	apiDir := filepath.Join(repoRoot, "scenarios", provider, "api")
	pkgDir := filepath.Join(repoRoot, "packages", "api-core", "database")
	for _, d := range []string{apiDir, pkgDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(apiDir, "main.go")
	shared := filepath.Join(pkgDir, "connect.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shared, []byte("package database\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(apiDir, provider+"-api")
	if err := os.WriteFile(binary, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := cliutil.FreshnessSpec{
		SourceRoot:  apiDir,
		ContextRoot: repoRoot,
		Inputs: []string{
			filepath.ToSlash(filepath.Join("scenarios", provider, "api", "main.go")),
			"packages/api-core/database/connect.go",
		},
		SkipFiles: []string{filepath.Base(binary)},
	}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	if err := cliutil.WriteFreshnessManifest(cliutil.FreshnessManifestPath(binary), manifest); err != nil {
		t.Fatal(err)
	}
	return repoRoot, manifest.Digest, shared
}

func TestCooldownIsHonoredEndToEnd(t *testing.T) {
	repo, digest, shared := stageProviderWithSharedPackage(t, "security-health")
	// Churn the shared package: real staleness, but not this provider's own code.
	if err := os.WriteFile(shared, []byte("package database\nvar X = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lifecycle := &countingLifecycle{}
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	newManager := func() *Manager {
		return &Manager{
			Probe: func(context.Context, Input) (ProbeResult, error) {
				// Digest matches on disk, so only the source-change path fires.
				return ProbeResult{
					Reachable: true, ContractValid: true, IdentityMatch: true,
					FreshnessDigest: digest,
				}, nil
			},
			Lifecycle: lifecycle,
			RepoRoot:  repo,
			Ledger:    ledger,
		}
	}

	if out := newManager().Check(context.Background(), staleInput(), io.Discard); !out.Ready {
		t.Fatalf("first check: %+v", out)
	}
	if len(lifecycle.restarted) != 1 {
		t.Fatalf("first run restarted %d times, want 1", len(lifecycle.restarted))
	}

	// A second run inside the cooldown window must not rebuild it again.
	out := newManager().Check(context.Background(), staleInput(), io.Discard)
	if len(lifecycle.restarted) != 1 {
		t.Errorf("restarted %d times across two runs, want 1 — the cooldown did not hold", len(lifecycle.restarted))
	}
	if !strings.Contains(out.Message, "holding off") {
		t.Errorf("the cooldown decision was not explained: %q", out.Message)
	}
	if !strings.Contains(out.Message, "shared package") {
		t.Errorf("the message did not name the kind of change: %q", out.Message)
	}
}

// Own-code changes must punch through the cooldown even moments after a restart.
func TestOwnCodeChangeIgnoresCooldownEndToEnd(t *testing.T) {
	repo, digest, shared := stageProviderWithSharedPackage(t, "security-health")
	lifecycle := &countingLifecycle{}
	ledger := NewRestartLedgerAt(filepath.Join(t.TempDir(), "restarts.json"))
	newManager := func() *Manager {
		return &Manager{
			Probe: func(context.Context, Input) (ProbeResult, error) {
				return ProbeResult{
					Reachable: true, ContractValid: true, IdentityMatch: true,
					FreshnessDigest: digest,
				}, nil
			},
			Lifecycle: lifecycle,
			RepoRoot:  repo,
			Ledger:    ledger,
		}
	}

	// Round one: shared-package churn puts the provider into cooldown.
	if err := os.WriteFile(shared, []byte("package database\nvar X = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	newManager().Check(context.Background(), staleInput(), io.Discard)
	if len(lifecycle.restarted) != 1 {
		t.Fatalf("setup restart did not happen: %v", lifecycle.restarted)
	}

	// Round two: the provider's OWN code changes. Cooldown must not apply.
	own := filepath.Join(repo, "scenarios", "security-health", "api", "main.go")
	if err := os.WriteFile(own, []byte("package main\nfunc main(){println(2)}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := newManager().Check(context.Background(), staleInput(), io.Discard)
	if len(lifecycle.restarted) != 2 {
		t.Errorf("restarted %d times, want 2 — an own-code change was suppressed by the cooldown", len(lifecycle.restarted))
	}
	if !strings.Contains(out.Message, "own code") {
		t.Errorf("the message did not identify an own-code change: %q", out.Message)
	}
}
