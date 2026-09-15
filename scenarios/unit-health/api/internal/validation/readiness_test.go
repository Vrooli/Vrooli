package validation

import (
	"context"
	"testing"
	"time"

	"unit-health/internal/evidence"
	"unit-health/internal/readiness"
)

type fakeReadiness struct{ report readiness.Report }

func (f fakeReadiness) Check(context.Context, string, string, string) (readiness.Report, error) {
	return f.report, nil
}

func TestMissingDependencyReadinessBlocksExecutionWithoutInstalling(t *testing.T) {
	service := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, loadSpec(t))
	runner := &countingRunner{}
	service.Executor = runner
	service.ReadinessResolver = fakeReadiness{report: readiness.Report{
		Status: readiness.Missing, Source: "scenario-dependency-analyzer",
		Requirements: []readiness.Requirement{{ID: "go-toolchain", Kind: "tool", Status: readiness.Missing, Source: "scenario-dependency-analyzer", Remediation: "Provision Go through Scenario Dependency Analyzer"}},
	}}
	response, err := service.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls.Load() != 0 {
		t.Fatalf("runner started despite missing readiness: %d", runner.calls.Load())
	}
	if !hasFinding(response.Findings, codeTestDependencyMissing) {
		t.Fatalf("findings=%v", codes(response.Findings))
	}
}

func TestMissingDependencyReadinessDoesNotReuseSuccessfulCache(t *testing.T) {
	store, err := evidence.NewStore(t.TempDir(), 1<<20, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	runner := &countingRunner{}
	service := newService(fakeDiscoverer{inv: goSurfaceInventory(t)}, loadSpec(t))
	service.Executor = runner
	service.EvidenceStore = store
	service.ReadinessResolver = fakeReadiness{report: readiness.Report{Status: readiness.Ready, Source: "scenario-dependency-analyzer"}}
	request := Request{Scenario: "demo", IncludeExecution: true, UseCache: true}
	first, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.CacheHit || runner.calls.Load() != 1 {
		t.Fatalf("first response hit=%v runner calls=%d", first.CacheHit, runner.calls.Load())
	}
	service.ReadinessResolver = fakeReadiness{report: readiness.Report{
		Status: readiness.Missing, Source: "scenario-dependency-analyzer",
		Requirements: []readiness.Requirement{{ID: "go-toolchain", Status: readiness.Missing, Source: "scenario-dependency-analyzer"}},
	}}
	second, err := service.Validate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if second.CacheHit || runner.calls.Load() != 1 || !hasFinding(second.Findings, codeTestDependencyMissing) {
		t.Fatalf("missing readiness reused cache or executed runner: hit=%v calls=%d findings=%v", second.CacheHit, runner.calls.Load(), codes(second.Findings))
	}
}
