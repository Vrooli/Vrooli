package stateformat

import (
	"strings"
	"testing"
)

func digestOf(t *testing.T, label string) string {
	t.Helper()
	digest, err := DigestSchema(label)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func manifestOf(t *testing.T, stores ...Store) Manifest {
	t.Helper()
	if err := SortStores(stores); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Format: ManifestFormat, Stores: stores}
	if err := manifest.Seal(); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func storeOf(t *testing.T, name string, write Version, reads []Version, additive Version, versions ...Version) Store {
	t.Helper()
	schemas := make(map[Version]string, len(versions))
	for _, v := range versions {
		schemas[v] = digestOf(t, name+"-"+strings.Repeat("v", int(v)))
	}
	return Store{Name: name, Write: write, Reads: reads, AdditiveFrom: additive, Schemas: schemas}
}

func TestCheckIdenticalIsCompatible(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 17, []Version{16, 17}, 0, 16, 17))
	candidate := manifestOf(t, storeOf(t, "registry", 17, []Version{16, 17}, 0, 16, 17))
	got := Check(prior, candidate)
	if !got.AllowsAutomaticRollback() {
		t.Fatalf("identical manifests must be compatible: %+v", got)
	}
}

func TestCheckAdditiveBumpIsCompatible(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{15, 16}, 0, 15, 16))
	candidate := manifestOf(t, storeOf(t, "registry", 17, []Version{15, 16, 17}, 16, 15, 16, 17))
	got := Check(prior, candidate)
	if !got.AllowsAutomaticRollback() {
		t.Fatalf("an additive bump must be compatible: %+v", got)
	}
}

func TestCheckAdditiveChainIsCompatible(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{14, 15, 16}, 0, 14, 15, 16))
	candidate := manifestOf(t, storeOf(t, "registry", 18, []Version{14, 15, 16, 17, 18}, 16, 14, 15, 16, 17, 18))
	got := Check(prior, candidate)
	if !got.AllowsAutomaticRollback() {
		t.Fatalf("an additive chain from the serving version must be compatible: %+v", got)
	}
}

func TestCheckBreakingBumpIsBreaking(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{15, 16}, 0, 15, 16))
	candidate := manifestOf(t, storeOf(t, "registry", 17, []Version{15, 16, 17}, 0, 15, 16, 17))
	got := Check(prior, candidate)
	if got.AllowsAutomaticRollback() || got.Verdict != Breaking {
		t.Fatalf("a bump without an additive claim must break: %+v", got)
	}
	if !hasKind(got, ChangePriorUnreadable) {
		t.Fatalf("expected prior-cannot-read change: %+v", got)
	}
}

func TestCheckCandidateMustReadServing(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{15, 16}, 0, 15, 16))
	candidate := manifestOf(t, storeOf(t, "registry", 17, []Version{17}, 16, 17))
	got := Check(prior, candidate)
	if got.Verdict != Breaking || !hasKind(got, ChangeCandidateUnreadable) {
		t.Fatalf("a candidate that cannot read the serving data must break: %+v", got)
	}
}

func TestCheckMigrationReachIsEnough(t *testing.T) {
	// A candidate that upgrades v16 in place does not need to decode v16.
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{15, 16}, 0, 15, 16))
	candidate := storeOf(t, "registry", 17, []Version{17}, 16, 17)
	candidate.MigratesFrom = 16
	got := Check(prior, manifestOf(t, candidate))
	if !got.AllowsAutomaticRollback() {
		t.Fatalf("a candidate that migrates the serving version must be compatible: %+v", got)
	}
}

func TestCheckSharedVersionMustKeepItsSchema(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{16}, 0, 16))
	candidate := storeOf(t, "registry", 16, []Version{16}, 0, 16)
	candidate.Schemas[16] = digestOf(t, "rewritten-v16")
	got := Check(prior, manifestOf(t, candidate))
	if got.Verdict != Breaking || !hasKind(got, ChangeSchema) {
		t.Fatalf("a rewritten version must break: %+v", got)
	}
}

func TestCheckNewStoreIsCompatible(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{16}, 0, 16))
	candidate := manifestOf(t,
		storeOf(t, "registry", 16, []Version{16}, 0, 16),
		storeOf(t, "cache", 1, []Version{1}, 0, 1),
	)
	got := Check(prior, candidate)
	if !got.AllowsAutomaticRollback() {
		t.Fatalf("a store the serving image never reads is inert: %+v", got)
	}
	if !hasKind(got, ChangeAdded) {
		t.Fatalf("expected an informational added-store change: %+v", got)
	}
}

func TestCheckRemovedStoreIsBreaking(t *testing.T) {
	prior := manifestOf(t,
		storeOf(t, "registry", 16, []Version{16}, 0, 16),
		storeOf(t, "cache", 1, []Version{1}, 0, 1),
	)
	candidate := manifestOf(t, storeOf(t, "registry", 16, []Version{16}, 0, 16))
	got := Check(prior, candidate)
	if got.Verdict != Breaking || !hasKind(got, ChangeRemoved) {
		t.Fatalf("removing a store a rollback may read must break: %+v", got)
	}
}

func TestCheckUnknownManifestIsUnknown(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{16}, 0, 16))
	candidate := prior
	candidate.Format = 99
	got := Check(prior, candidate)
	if got.Verdict != Unknown || got.AllowsAutomaticRollback() {
		t.Fatalf("an unknown manifest format must be unknown, never compatible: %+v", got)
	}
	if !hasKind(got, ChangeUnqualified) {
		t.Fatalf("expected an unqualified change: %+v", got)
	}
}

func TestCheckTamperedDigestIsUnknown(t *testing.T) {
	prior := manifestOf(t, storeOf(t, "registry", 16, []Version{16}, 0, 16))
	candidate := prior
	candidate.Digest = digestOf(t, "not-the-manifest")
	got := Check(prior, candidate)
	if got.Verdict != Unknown {
		t.Fatalf("a manifest that does not match its digest must be unknown: %+v", got)
	}
}

func TestOpaqueStoresPreserveIdentitySemantics(t *testing.T) {
	priorStore, err := Opaque("legacy", "same-digest")
	if err != nil {
		t.Fatal(err)
	}
	changedStore, err := Opaque("legacy", "different-digest")
	if err != nil {
		t.Fatal(err)
	}
	if got := Check(manifestOf(t, priorStore), manifestOf(t, priorStore)); !got.AllowsAutomaticRollback() {
		t.Fatalf("unchanged opaque store must be compatible: %+v", got)
	}
	if got := Check(manifestOf(t, priorStore), manifestOf(t, changedStore)); got.Verdict != Breaking {
		t.Fatalf("changed opaque store must break: %+v", got)
	}
}

func TestDigestSchemaIsStableAcrossMapOrder(t *testing.T) {
	first, err := DigestSchema(map[string]any{"a": 1, "b": []int{2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := DigestSchema(map[string]any{"b": []int{2, 3}, "a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("canonical digest depends on map order: %s != %s", first, second)
	}
}

func TestSortStoresRejectsDuplicatesAndMisorderedReads(t *testing.T) {
	if err := SortStores([]Store{
		{Name: "a", Write: 1, Reads: []Version{1}, Schemas: map[Version]string{1: digestOf(t, "a")}},
		{Name: "a", Write: 1, Reads: []Version{1}, Schemas: map[Version]string{1: digestOf(t, "a")}},
	}); err == nil {
		t.Fatal("duplicate store names must be rejected")
	}
	if err := SortStores([]Store{
		{Name: "a", Write: 2, Reads: []Version{2, 1}, Schemas: map[Version]string{1: digestOf(t, "1"), 2: digestOf(t, "2")}},
	}); err == nil {
		t.Fatal("unsorted reads must be rejected")
	}
}

func hasKind(assessment Assessment, kind string) bool {
	for _, change := range assessment.Changes {
		if change.Kind == kind {
			return true
		}
	}
	return false
}
