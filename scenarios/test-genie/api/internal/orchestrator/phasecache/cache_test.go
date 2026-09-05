package phasecache

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"test-genie/internal/orchestrator/phases"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func writePruneEntry(t *testing.T, store *Store, name string, size int, modTime time.Time) string {
	t.Helper()
	if err := os.MkdirAll(store.root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.root, name)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPruneRemovesOldestEntriesFirst(t *testing.T) {
	store := New(t.TempDir())
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	oldest := writePruneEntry(t, store, "pc_oldest.json", 10, now.Add(-3*time.Hour))
	middle := writePruneEntry(t, store, "pc_middle.json", 10, now.Add(-2*time.Hour))
	newest := writePruneEntry(t, store, "pc_newest.json", 10, now.Add(-time.Hour))

	result, err := store.Prune(context.Background(), PrunePolicy{MaxBytes: 10, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if result.BoundBy != PruneBoundBytes || !reflect.DeepEqual(result.DeletedEntries, []string{"pc_oldest.json", "pc_middle.json"}) {
		t.Fatalf("prune result = %+v", result)
	}
	for _, deleted := range []string{oldest, middle} {
		if _, err := os.Stat(deleted); !os.IsNotExist(err) {
			t.Fatalf("old entry %s survived: %v", deleted, err)
		}
	}
	if _, err := os.Stat(newest); err != nil {
		t.Fatalf("newest entry was removed: %v", err)
	}
}

func TestPruneReportsByteBoundWhenAgeDoesNotBind(t *testing.T) {
	store := New(t.TempDir())
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	writePruneEntry(t, store, "pc_a.json", 8, now.Add(-2*time.Minute))
	writePruneEntry(t, store, "pc_b.json", 8, now.Add(-time.Minute))

	result, err := store.Prune(context.Background(), PrunePolicy{
		MaxAge: 24 * time.Hour, MaxBytes: 8, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.BoundBy != PruneBoundBytes || result.AfterBytes > 8 || result.AfterEntries != 1 {
		t.Fatalf("byte-bound result = %+v", result)
	}
}

func TestPruneAlwaysPreservesDemotionsMetadata(t *testing.T) {
	store := New(t.TempDir())
	if err := store.Demote("pc:stale", "audit mismatch"); err != nil {
		t.Fatal(err)
	}
	writePruneEntry(t, store, "pc_entry.json", 8, time.Now().Add(-time.Hour))

	result, err := store.Prune(context.Background(), PrunePolicy{MaxBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.DeletedEntries) != 1 {
		t.Fatalf("deleted entries = %v", result.DeletedEntries)
	}
	if !store.IsDemoted("pc:stale") {
		t.Fatal("demotions.json did not survive pruning")
	}
}

func TestPruneFractionGuardAlarmsWithoutDeleting(t *testing.T) {
	store := New(t.TempDir())
	first := writePruneEntry(t, store, "pc_first.json", 8, time.Now().Add(-2*time.Hour))
	second := writePruneEntry(t, store, "pc_second.json", 8, time.Now().Add(-time.Hour))

	result, err := store.Prune(context.Background(), PrunePolicy{MaxBytes: 1, MaxDeleteFraction: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Refused || result.RefusedReason == "" || len(result.DeletedEntries) != 0 {
		t.Fatalf("fraction guard result = %+v", result)
	}
	for _, path := range []string{first, second} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("guard deleted %s: %v", path, err)
		}
	}
}

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

func TestStoreServesBothDeterminedVerdicts(t *testing.T) {
	store := New(t.TempDir())

	// A pass under an unchanged identity is reusable.
	pass := phases.ExecutionResult{Name: "structure", Status: "passed", Findings: nil}
	if err := store.Save("pc:passed", "run-1", pass); err != nil {
		t.Fatal(err)
	}
	entry, ok, err := store.Load("pc:passed")
	if err != nil || !ok || entry.RunID != "run-1" || entry.Phase.Status != "passed" {
		t.Fatalf("load passed = %+v, %v, %v", entry, ok, err)
	}

	// So is a failure. The cache identity covers everything that could change
	// the verdict, so re-deriving a byte-identical failure buys nothing — and
	// that re-derivation was 36% of all phase time.
	fail := phases.ExecutionResult{
		Name:   "structure",
		Status: "failed",
		Findings: []*architecturev1.ArchitectureFinding{
			{Code: "rule", Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR},
		},
	}
	if err := store.Save("pc:failed", "run-2", fail); err != nil {
		t.Fatal(err)
	}
	entry, ok, err = store.Load("pc:failed")
	if err != nil || !ok {
		t.Fatalf("a failure under an unchanged identity must be reusable: ok=%v err=%v", ok, err)
	}
	if entry.Phase.Status != "failed" {
		t.Fatalf("cached verdict = %q, want failed", entry.Phase.Status)
	}
	if len(entry.Phase.Findings) != 1 || entry.Phase.Findings[0].GetCode() != "rule" {
		t.Fatalf("findings must survive the round trip, got %+v", entry.Phase.Findings)
	}
}

// TestStoreRejectsNonVerdictStatuses keeps the extension narrow. These statuses
// describe the state of the RUN rather than the verdict of the phase — they say
// the phase did not happen, which is not a result to reuse.
func TestStoreRejectsNonVerdictStatuses(t *testing.T) {
	store := New(t.TempDir())
	for _, status := range []string{"skipped", "missing", "not_executable", "not_run", "provider_unavailable", ""} {
		key := "pc:" + status
		if err := store.Save(key, "run-x", phases.ExecutionResult{Name: "structure", Status: status}); err != nil {
			t.Fatalf("save %q: %v", status, err)
		}
		if _, ok, err := store.Load(key); err != nil || ok {
			t.Fatalf("status %q must not be cacheable: ok=%v err=%v", status, ok, err)
		}
	}
}

func TestCacheableIsCaseAndSpaceInsensitive(t *testing.T) {
	for _, status := range []string{"passed", "PASSED", " failed ", "Failed"} {
		if !Cacheable(status) {
			t.Fatalf("%q should be cacheable", status)
		}
	}
	for _, status := range []string{"skipped", "pass", "fail", ""} {
		if Cacheable(status) {
			t.Fatalf("%q should not be cacheable", status)
		}
	}
}

// --- audit diagnosis ----------------------------------------------------
//
// A mismatch count tells an operator a number. Naming the difference tells
// them whether the cache was wrong or the world moved — the only question
// worth asking when an audit fails.

func TestDiffNamesAVerdictFlip(t *testing.T) {
	cached := phases.ExecutionResult{Name: "unit", Status: "passed"}
	fresh := phases.ExecutionResult{Name: "unit", Status: "failed"}

	diff := Diff(cached, fresh)
	if !strings.Contains(diff, "passed") || !strings.Contains(diff, "failed") {
		t.Fatalf("a verdict flip must be named, got %q", diff)
	}
}

func TestDiffNamesAppearingAndDisappearingFindings(t *testing.T) {
	cached := phases.ExecutionResult{
		Name:   "quality",
		Status: "failed",
		Findings: []*architecturev1.ArchitectureFinding{
			{Code: "kept", StableId: "kept"},
			{Code: "gone", StableId: "gone"},
		},
	}
	fresh := phases.ExecutionResult{
		Name:   "quality",
		Status: "failed",
		Findings: []*architecturev1.ArchitectureFinding{
			{Code: "kept", StableId: "kept"},
			{Code: "new", StableId: "new"},
		},
	}

	diff := Diff(cached, fresh)
	if !strings.Contains(diff, "appeared") || !strings.Contains(diff, "new") {
		t.Fatalf("appearing findings must be named, got %q", diff)
	}
	if !strings.Contains(diff, "disappeared") || !strings.Contains(diff, "gone") {
		t.Fatalf("disappearing findings must be named, got %q", diff)
	}
	if strings.Contains(diff, "kept") {
		t.Fatalf("an unchanged finding must not be reported as churn: %q", diff)
	}
}

// TestDiffIsBounded keeps the message readable for a phase that emits hundreds
// of findings: an unreadable diagnosis is no better than a count.
func TestDiffIsBounded(t *testing.T) {
	fresh := phases.ExecutionResult{Name: "quality", Status: "failed"}
	for i := 0; i < 50; i++ {
		id := "finding-" + strconv.Itoa(i)
		fresh.Findings = append(fresh.Findings, &architecturev1.ArchitectureFinding{Code: id, StableId: id})
	}

	diff := Diff(phases.ExecutionResult{Name: "quality", Status: "failed"}, fresh)
	if !strings.Contains(diff, "50 finding(s) appeared") {
		t.Fatalf("the full count must be reported, got %q", diff)
	}
	if !strings.Contains(diff, "more") {
		t.Fatalf("expected the example list to be truncated, got %q", diff)
	}
	if len(diff) > 300 {
		t.Fatalf("diff is %d chars; it must stay readable: %q", len(diff), diff)
	}
}

// TestDiffOnEquivalentResultsIsHonest covers the case where two results compare
// equal under Equivalent. Reaching Diff then means they differ only in a field
// normalization ignores, and saying so beats an empty explanation.
func TestDiffOnEquivalentResultsIsHonest(t *testing.T) {
	a := phases.ExecutionResult{Name: "unit", Status: "passed"}
	b := phases.ExecutionResult{Name: "unit", Status: "passed", DurationMilliseconds: 900}
	if !Equivalent(a, b) {
		t.Fatal("these should compare equivalent")
	}
	if diff := Diff(a, b); diff == "" {
		t.Fatal("Diff must never return an empty explanation")
	}
}
