package aisearch

import (
	"context"
	"sync"
	"testing"
	"time"
)

type lifecycleCatalogFixture struct {
	mu       sync.Mutex
	records  map[string]GenerationRecord
	receipts map[string]GenerationCleanupReceipt
}

func newLifecycleCatalogFixture(records ...GenerationRecord) *lifecycleCatalogFixture {
	catalog := &lifecycleCatalogFixture{records: map[string]GenerationRecord{}, receipts: map[string]GenerationCleanupReceipt{}}
	for _, record := range records {
		catalog.records[record.CollectionName] = record
	}
	return catalog
}

func (c *lifecycleCatalogFixture) ListGenerationRecords(context.Context, string, string) ([]GenerationRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]GenerationRecord, 0, len(c.records))
	for _, record := range c.records {
		result = append(result, record)
	}
	return result, nil
}

func (c *lifecycleCatalogFixture) SaveGenerationRecord(_ context.Context, record GenerationRecord) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records[record.CollectionName] = record
	return nil
}

func (c *lifecycleCatalogFixture) LoadGenerationCleanupReceipt(_ context.Context, key string) (GenerationCleanupReceipt, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	receipt, ok := c.receipts[key]
	return receipt, ok, nil
}

func (c *lifecycleCatalogFixture) SaveGenerationCleanupReceipt(_ context.Context, receipt GenerationCleanupReceipt) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.receipts[receipt.IdempotencyKey] = receipt
	return nil
}

func TestGenerationLifecycleTransitionsAreExplicit(t *testing.T) {
	valid := [][2]GenerationLifecycleState{
		{GenerationStateBuilding, GenerationStateCandidate},
		{GenerationStateCandidate, GenerationStateActive},
		{GenerationStateActive, GenerationStateRetired},
		{GenerationStateRetired, GenerationStateExpired},
		{GenerationStateExpired, GenerationStateDeleted},
		{GenerationStateFailed, GenerationStateDeleted},
		{GenerationStateQuarantined, GenerationStateProtected},
	}
	for _, pair := range valid {
		if !ValidGenerationTransition(pair[0], pair[1]) {
			t.Errorf("transition %q -> %q should be valid", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]GenerationLifecycleState{
		{GenerationStateDeleted, GenerationStateActive},
		{GenerationStateQuarantined, GenerationStateDeleted},
		{GenerationStateFailed, GenerationStateActive},
	} {
		if ValidGenerationTransition(pair[0], pair[1]) {
			t.Errorf("transition %q -> %q should be rejected", pair[0], pair[1])
		}
	}
}

func TestQdrantOwnerCleanupProtectsActiveRollbackAndLeasedGenerations(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	now := time.Date(2026, 9, 9, 7, 0, 0, 0, time.UTC)
	old := now.Add(-96 * time.Hour)
	ids := []string{"gen-active", "gen-rollback", "gen-leased", "gen-expired", "gen-failed", "gen-unknown"}
	for _, id := range ids {
		fake.collections[generationCollectionName(alias, id)] = &fakeCollection{points: 1}
	}
	active := generationCollectionName(alias, "gen-active")
	fake.aliases[alias] = active
	records := []GenerationRecord{
		{Metadata: GenerationMetadata{ID: "gen-active", CreatedAt: old, Owner: "agent-manager", Namespace: alias, Alias: alias}, CollectionName: active, State: GenerationStateActive, UpdatedAt: old},
		{Metadata: GenerationMetadata{ID: "gen-rollback", CreatedAt: now.Add(-80 * time.Hour)}, CollectionName: generationCollectionName(alias, "gen-rollback"), State: GenerationStateRetired, UpdatedAt: now.Add(-80 * time.Hour)},
		{Metadata: GenerationMetadata{ID: "gen-leased", CreatedAt: old}, CollectionName: generationCollectionName(alias, "gen-leased"), State: GenerationStateRetired, Lease: GenerationLease{ID: "lease-1", ExpiresAt: now.Add(time.Hour)}, UpdatedAt: old},
		{Metadata: GenerationMetadata{ID: "gen-expired", CreatedAt: old}, CollectionName: generationCollectionName(alias, "gen-expired"), State: GenerationStateRetired, UpdatedAt: old},
		{Metadata: GenerationMetadata{ID: "gen-failed", CreatedAt: now.Add(-2 * time.Hour)}, CollectionName: generationCollectionName(alias, "gen-failed"), State: GenerationStateFailed, UpdatedAt: now.Add(-2 * time.Hour)},
	}
	catalog := newLifecycleCatalogFixture(records...)
	store, raw := newTestGenerationStore(t, fake, alias)
	raw.catalog = catalog
	raw.namespace = alias
	raw.now = func() time.Time { return now }
	policy := DefaultGenerationRetentionPolicy()
	policy.PolicyRevision = "test-v1"
	policy.KeepRollback = 1
	policy.MaxGenerationCount = 10
	inspection, err := raw.InspectGenerationLifecycle(context.Background(), policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Eligible) != 2 || !containsString(inspection.Eligible, generationCollectionName(alias, "gen-expired")) || !containsString(inspection.Eligible, generationCollectionName(alias, "gen-failed")) {
		t.Fatalf("eligible=%v, want expired and failed generations", inspection.Eligible)
	}
	for _, name := range []string{active, generationCollectionName(alias, "gen-rollback"), generationCollectionName(alias, "gen-leased")} {
		if !containsString(inspection.Protected, name) {
			t.Fatalf("%q should be protected: %v", name, inspection.Protected)
		}
	}
	if !containsString(inspection.Quarantined, generationCollectionName(alias, "gen-unknown")) {
		t.Fatalf("unknown collection must be quarantined: %v", inspection.Quarantined)
	}

	plan, err := raw.PreviewGenerationCleanup(context.Background(), policy, "plan-1")
	if err != nil {
		t.Fatal(err)
	}
	plan.Approved = true
	receipt, err := raw.ApplyGenerationCleanup(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.DeletedCollections) != 2 {
		t.Fatalf("deleted=%v, want two approved collections", receipt.DeletedCollections)
	}
	if len(fake.deletes) != 2 {
		t.Fatalf("qdrant deletes=%v, want two", fake.deletes)
	}
	for _, name := range []string{active, generationCollectionName(alias, "gen-rollback"), generationCollectionName(alias, "gen-leased"), generationCollectionName(alias, "gen-unknown")} {
		if _, ok := fake.collections[name]; !ok {
			t.Fatalf("protected/quarantined collection %q was deleted", name)
		}
	}
	second, err := raw.ApplyGenerationCleanup(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.deletes) != 2 || len(second.DeletedCollections) != 2 {
		t.Fatalf("repeated cleanup was not receipt-idempotent: deletes=%v second=%v", fake.deletes, second)
	}
	if second.IdempotencyKey != receipt.IdempotencyKey {
		t.Fatalf("receipt key changed: %q vs %q", second.IdempotencyKey, receipt.IdempotencyKey)
	}
	_ = store
}

func TestQdrantOwnerInspectionNormalizesStaleCatalogActiveState(t *testing.T) {
	fake := newFakeQdrant()
	const alias = "fixture_search"
	now := time.Date(2026, 9, 9, 7, 0, 0, 0, time.UTC)
	active := generationCollectionName(alias, "gen-new")
	stale := generationCollectionName(alias, "gen-old")
	fake.collections[active] = &fakeCollection{points: 1}
	fake.collections[stale] = &fakeCollection{points: 1}
	fake.aliases[alias] = active
	catalog := newLifecycleCatalogFixture(
		GenerationRecord{
			Metadata:       GenerationMetadata{ID: "gen-new", CreatedAt: now.Add(-time.Hour), Owner: "agent-manager", Namespace: alias, Alias: alias},
			CollectionName: active, State: GenerationStateActive, UpdatedAt: now.Add(-time.Hour),
		},
		GenerationRecord{
			Metadata:       GenerationMetadata{ID: "gen-old", CreatedAt: now.Add(-96 * time.Hour), Owner: "agent-manager", Namespace: alias, Alias: alias},
			CollectionName: stale, State: GenerationStateActive, UpdatedAt: now.Add(-96 * time.Hour),
		},
	)
	_, store := newTestGenerationStore(t, fake, alias)
	store.catalog = catalog
	store.namespace = alias
	store.now = func() time.Time { return now }
	policy := DefaultGenerationRetentionPolicy()
	policy.PolicyRevision = "test-v1"
	policy.KeepRollback = 0
	policy.MaxGenerationCount = 10

	inspection, err := store.InspectGenerationLifecycle(context.Background(), policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.Generations) != 2 {
		t.Fatalf("generations=%d, want 2", len(inspection.Generations))
	}
	for _, record := range inspection.Generations {
		if record.CollectionName == stale && record.State != GenerationStateExpired {
			t.Fatalf("stale catalog state=%q, want expired after retirement normalization", record.State)
		}
	}
	if len(inspection.Eligible) != 1 || inspection.Eligible[0] != stale {
		t.Fatalf("eligible=%v, want stale collection %q", inspection.Eligible, stale)
	}
}
