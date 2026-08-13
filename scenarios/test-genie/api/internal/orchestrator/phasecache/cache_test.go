package phasecache

import (
	"os"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	"test-genie/internal/orchestrator/phases"
)

func TestKeyChangesForEachIdentityPart(t *testing.T) {
	base := Identity{ScopedInputDigest: "input", ProviderBuildIdentity: "provider", DescriptorSnapshotHash: "descriptor", ExecutionConfiguration: "config"}
	key := Key(base)
	for name, changed := range map[string]Identity{
		"input":      {ScopedInputDigest: "changed", ProviderBuildIdentity: base.ProviderBuildIdentity, DescriptorSnapshotHash: base.DescriptorSnapshotHash, ExecutionConfiguration: base.ExecutionConfiguration},
		"provider":   {ScopedInputDigest: base.ScopedInputDigest, ProviderBuildIdentity: "changed", DescriptorSnapshotHash: base.DescriptorSnapshotHash, ExecutionConfiguration: base.ExecutionConfiguration},
		"descriptor": {ScopedInputDigest: base.ScopedInputDigest, ProviderBuildIdentity: base.ProviderBuildIdentity, DescriptorSnapshotHash: "changed", ExecutionConfiguration: base.ExecutionConfiguration},
		"config":     {ScopedInputDigest: base.ScopedInputDigest, ProviderBuildIdentity: base.ProviderBuildIdentity, DescriptorSnapshotHash: base.DescriptorSnapshotHash, ExecutionConfiguration: "changed"},
	} {
		if got := Key(changed); got == key {
			t.Fatalf("%s change did not change cache key", name)
		}
	}
}

func TestAuditSamplingAndDemotionAreFailClosed(t *testing.T) {
	t.Setenv("TEST_GENIE_PHASE_CACHE_AUDIT_PERCENT", "100")
	store := New(t.TempDir())
	if !store.ShouldAudit("pc:key", "run-1") {
		t.Fatal("100 percent audit must select the cache hit")
	}
	if err := store.Demote("pc:key", "mismatch"); err != nil {
		t.Fatal(err)
	}
	if store.ShouldAudit("pc:key", "run-1") != true {
		t.Fatal("demotion records policy separately from sampling")
	}
	if err := store.Save("pc:key", "run-1", phases.ExecutionResult{Name: "x", Status: "passed"}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load("pc:key"); err != nil || ok {
		t.Fatalf("demoted entry must not load: ok=%v err=%v", ok, err)
	}
	if _, err := os.Stat(store.root + "/demotions.json"); err != nil {
		t.Fatal(err)
	}
}

func TestEquivalentIgnoresRunLocalTimingAndCacheProvenance(t *testing.T) {
	a := phases.ExecutionResult{Name: "x", Status: "passed", DurationSeconds: 1, DurationMilliseconds: 100, LogPath: "old", CacheHit: true, CacheSourceRunID: "old-run"}
	b := a
	b.DurationSeconds = 9
	b.DurationMilliseconds = 900
	b.LogPath = "new"
	b.CacheHit = false
	b.CacheSourceRunID = ""
	if !Equivalent(a, b) {
		t.Fatal("run-local fields should not affect cache audit comparison")
	}
}

func TestEquivalentIgnoresSchedulerAdmissionObservations(t *testing.T) {
	a := phases.ExecutionResult{
		Name: "proto", Status: "passed",
		Observations: []phases.Observation{
			{Prefix: "WARNING", Text: "scheduler serial fallback: insufficient cpu"},
			{Prefix: "INFO", Text: "provider result"},
		},
	}
	b := a
	b.Observations = []phases.Observation{{Prefix: "INFO", Text: "provider result"}}
	if !Equivalent(a, b) {
		t.Fatal("scheduler admission diagnostics must not demote an otherwise equivalent cache result")
	}
}

func TestEquivalentNormalizesFindingOrder(t *testing.T) {
	a := phases.ExecutionResult{Name: "quality", Status: "passed", Findings: []*architecturev1.ArchitectureFinding{
		{Code: "rule-b", Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, Locations: []string{"b.go"}},
		{Code: "rule-a", Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, Locations: []string{"a.go"}},
	}}
	b := a
	b.Findings = []*architecturev1.ArchitectureFinding{a.Findings[1], a.Findings[0]}
	if !Equivalent(a, b) {
		t.Fatal("finding order must not change the normalized verdict")
	}
}

func TestEquivalentDetectsFindingVerdictChanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*phases.ExecutionResult)
	}{
		{name: "severity", mutate: func(result *phases.ExecutionResult) {
			result.Findings[0].Severity = architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER
		}},
		{name: "status", mutate: func(result *phases.ExecutionResult) { result.Status = "failed" }},
		{name: "finding set", mutate: func(result *phases.ExecutionResult) {
			result.Findings = append(result.Findings, &architecturev1.ArchitectureFinding{Code: "new-rule"})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := phases.ExecutionResult{Name: "quality", Status: "passed", Findings: []*architecturev1.ArchitectureFinding{{Code: "rule", Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING}}}
			b := phases.ExecutionResult{Name: "quality", Status: "passed", Findings: []*architecturev1.ArchitectureFinding{{Code: "rule", Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING}}}
			tc.mutate(&b)
			if Equivalent(a, b) {
				t.Fatal("changed validation verdict must not compare equal")
			}
		})
	}
}

func TestStoreOnlyServesPassedResultsAndPreservesFindings(t *testing.T) {
	store := New(t.TempDir())
	phase := phases.ExecutionResult{Name: "structure", Status: "passed", Findings: nil}
	if err := store.Save("pc:test", "run-1", phase); err != nil {
		t.Fatal(err)
	}
	entry, ok, err := store.Load("pc:test")
	if err != nil || !ok || entry.RunID != "run-1" || entry.Phase.Status != "passed" {
		t.Fatalf("load = %+v, %v, %v", entry, ok, err)
	}
	if err := store.Save("pc:failed", "run-2", phases.ExecutionResult{Name: "structure", Status: "failed"}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Load("pc:failed"); err != nil || ok {
		t.Fatalf("failed result should not be cacheable: ok=%v err=%v", ok, err)
	}
}
