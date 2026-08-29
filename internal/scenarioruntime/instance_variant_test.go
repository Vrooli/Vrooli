//nolint:goconst // test data deliberately reuses stable instance fixtures.
package scenarioruntime

import (
	"context"
	"testing"
	"time"
)

// TestCreateInstancePerVariantGeneration proves the schema-5 invariant: two
// named instances of one scenario coexist, each with its own generation
// counter. Before schema 5 the UNIQUE(scenario, generation) constraint would
// have rejected the shadow's generation 1 because the live instance already
// owned it.
func TestCreateInstancePerVariantGeneration(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	live1 := mustCreate(t, store, Instance{Scenario: "alpha"})
	if live1.Variant != DefaultVariant {
		t.Fatalf("empty variant normalized to %q, want %q", live1.Variant, DefaultVariant)
	}
	if live1.Generation != 1 {
		t.Fatalf("live generation = %d, want 1", live1.Generation)
	}

	live2 := mustCreate(t, store, Instance{Scenario: "alpha"})
	if live2.Generation != 2 {
		t.Fatalf("second live generation = %d, want 2 (per-variant counter)", live2.Generation)
	}

	shadow1 := mustCreate(t, store, Instance{Scenario: "alpha", Variant: "shadow"})
	if shadow1.Variant != "shadow" {
		t.Fatalf("shadow variant = %q, want shadow", shadow1.Variant)
	}
	if shadow1.Generation != 1 {
		t.Fatalf("shadow generation = %d, want 1 (independent of live)", shadow1.Generation)
	}

	shadow2 := mustCreate(t, store, Instance{Scenario: "alpha", Variant: "shadow"})
	if shadow2.Generation != 2 {
		t.Fatalf("second shadow generation = %d, want 2", shadow2.Generation)
	}
}

// TestCreateInstanceNormalizesVariant confirms the store routes the variant
// through the InstanceKey SSOT so casing/whitespace can never fragment the
// uniqueness key or generation counter.
func TestCreateInstanceNormalizesVariant(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	a := mustCreate(t, store, Instance{Scenario: "beta", Variant: "  Shadow "})
	if a.Variant != "shadow" {
		t.Fatalf("variant = %q, want normalized shadow", a.Variant)
	}
	// A differently-cased spelling must share the same counter, not start over.
	b := mustCreate(t, store, Instance{Scenario: "beta", Variant: "SHADOW"})
	if b.Generation != 2 {
		t.Fatalf("generation = %d, want 2 (casing must not fork the counter)", b.Generation)
	}
}

// TestListInstancesVariantFilter checks that a variant-scoped query resolves
// only that variant's authoritative instance.
func TestListInstancesVariantFilter(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	mustCreate(t, store, Instance{Scenario: "gamma"})
	mustCreate(t, store, Instance{Scenario: "gamma", Variant: "shadow"})

	all, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma"})
	if err != nil {
		t.Fatalf("ListInstances(all) error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("unfiltered list = %d instances, want 2", len(all))
	}

	shadow, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma", Variant: "shadow"})
	if err != nil {
		t.Fatalf("ListInstances(shadow) error = %v", err)
	}
	if len(shadow) != 1 || shadow[0].Variant != "shadow" {
		t.Fatalf("variant filter = %+v, want exactly the shadow instance", shadow)
	}

	live, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma", Variant: DefaultVariant})
	if err != nil {
		t.Fatalf("ListInstances(live) error = %v", err)
	}
	if len(live) != 1 || live[0].Variant != DefaultVariant {
		t.Fatalf("live filter = %+v, want exactly the live instance", live)
	}
}

// TestAcquirePortClaimDenormalizesVariant verifies the claim carries (and
// filters by) its instance's variant.
func TestAcquirePortClaimDenormalizesVariant(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	live := mustCreate(t, store, Instance{Scenario: "delta"})
	shadow := mustCreate(t, store, Instance{Scenario: "delta", Variant: "shadow"})

	if _, err := store.AcquirePortClaim(ctx, PortClaim{InstanceID: live.InstanceID, Scenario: "delta", Port: 8100}); err != nil {
		t.Fatalf("acquire live claim error = %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, PortClaim{InstanceID: shadow.InstanceID, Scenario: "delta", Variant: "shadow", Port: 9100}); err != nil {
		t.Fatalf("acquire shadow claim error = %v", err)
	}

	shadowClaims, err := store.ListPortClaims(ctx, PortClaimFilter{Scenario: "delta", Variant: "shadow"})
	if err != nil {
		t.Fatalf("ListPortClaims(shadow) error = %v", err)
	}
	if len(shadowClaims) != 1 || shadowClaims[0].Port != 9100 || shadowClaims[0].Variant != "shadow" {
		t.Fatalf("shadow claims = %+v, want one shadow claim on 9100", shadowClaims)
	}

	liveClaims, err := store.ListPortClaims(ctx, PortClaimFilter{Scenario: "delta", Variant: DefaultVariant})
	if err != nil {
		t.Fatalf("ListPortClaims(live) error = %v", err)
	}
	if len(liveClaims) != 1 || liveClaims[0].Port != 8100 || liveClaims[0].Variant != DefaultVariant {
		t.Fatalf("live claims = %+v, want one live claim on 8100", liveClaims)
	}
}

func mustCreate(t *testing.T, store *SQLiteStore, in Instance) Instance {
	t.Helper()
	out, err := store.CreateInstance(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateInstance(%+v) error = %v", in, err)
	}
	return out
}
