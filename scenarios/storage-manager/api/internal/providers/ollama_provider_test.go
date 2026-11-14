package providers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

type fakeOllamaInventory struct {
	models  []OllamaModel
	running []OllamaModel
}

type deleteCountingOllamaInventory struct {
	fakeOllamaInventory
	deleteCalls int
}

func (f *deleteCountingOllamaInventory) DeleteModel(context.Context, string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeOllamaInventory) ListModels(context.Context) ([]OllamaModel, error) {
	return append([]OllamaModel(nil), f.models...), nil
}

func (f *fakeOllamaInventory) ListRunningModels(context.Context) ([]OllamaModel, error) {
	return append([]OllamaModel(nil), f.running...), nil
}

type memoryOllamaLedger struct {
	entries   map[string]ollamaLedgerEntry
	inventory ollamaInventoryTotals
}

func (l *memoryOllamaLedger) RecordInventory(_ context.Context, now time.Time, models []OllamaModel) (ollamaInventoryTotals, error) {
	var current int64
	for _, model := range models {
		current = saturatingAddNonnegative(current, model.Size)
	}
	if l.inventory.FirstObservedAt.IsZero() {
		l.inventory.FirstObservedAt = now
		l.inventory.BaselineLogicalBytes = current
	}
	l.inventory.LastObservedAt = now
	l.inventory.CurrentLogicalBytes = current
	if current > l.inventory.PeakLogicalBytes {
		l.inventory.PeakLogicalBytes = current
	}
	return l.inventory, nil
}

func (l *memoryOllamaLedger) Record(_ context.Context, now time.Time, models []OllamaModel) error {
	if l.entries == nil {
		l.entries = map[string]ollamaLedgerEntry{}
	}
	for _, model := range models {
		entry := l.entries[model.Name]
		if entry.FirstObserved.IsZero() {
			entry.FirstObserved = now
		}
		entry.LastUsed = now
		l.entries[model.Name] = entry
	}
	return nil
}

func (l *memoryOllamaLedger) Eligible(model string, now time.Time, window time.Duration) bool {
	entry, ok := l.entries[model]
	return ok && now.Sub(entry.FirstObserved) >= window && now.Sub(entry.LastUsed) >= window
}

func TestOllamaRetentionPreviewHoldsAndReportsSaturatedLogicalInventory(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"roles":{"chat":{"model":"referenced:latest","fallbacks":["digestless:latest"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	ledger := &memoryOllamaLedger{entries: map[string]ollamaLedgerEntry{
		"referenced:latest": {FirstObserved: now.Add(-48 * time.Hour), LastUsed: now.Add(-48 * time.Hour)},
		"fallback:latest":   {FirstObserved: now.Add(-48 * time.Hour), LastUsed: now.Add(-48 * time.Hour)},
		"digestless:latest": {FirstObserved: now.Add(-48 * time.Hour), LastUsed: now.Add(-48 * time.Hour)},
	}}
	client := &fakeOllamaInventory{
		models: []OllamaModel{
			{Name: "referenced:latest", Size: 10, Digest: "digest-referenced"},
			{Name: "fallback:latest", Size: int64(^uint64(0) >> 1), Digest: "digest-fallback"},
			{Name: "orphan:latest", Size: 30, Digest: "digest-orphan"},
			{Name: "digestless:latest", Size: 25},
		},
		running: []OllamaModel{{Name: "referenced:latest"}},
	}
	provider := NewOllamaModelRetentionProvider(client, ledger, policyPath, nil)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: now},
		Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("preview items = %#v, want all models held until exact retention authority exists", preview.Items)
	}
	if preview.MinAge != 24*time.Hour || !strings.Contains(strings.Join(preview.Warnings, " "), "no supported atomic retention fence") || !strings.Contains(strings.Join(preview.Warnings, " "), "digest; retention identity is unknown") || !strings.Contains(strings.Join(preview.Warnings, " "), "9223372036854775807 bytes of reported logical model size") {
		t.Fatalf("preview policy/provenance = min_age %s, warnings %#v", preview.MinAge, preview.Warnings)
	}
	estimate, err := provider.Estimate(context.Background(), cleanup.EstimateRequest{
		Scope: cleanup.ObservationScope{Now: now}, Policy: cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
	})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.EstimatedBytes != 0 || estimate.ItemCount != 0 || !strings.Contains(strings.Join(estimate.Notices, " "), "reclaimed are unknown until an owner reports an authoritative delta") {
		t.Fatalf("estimate confused logical inventory with authorized reclaimable bytes: %+v", estimate)
	}
}

func TestSaturatingEstimateByteAdditionNeverWraps(t *testing.T) {
	max := int64(^uint64(0) >> 1)
	if got := saturatingAddNonnegative(max-4, 10); got != max {
		t.Fatalf("saturating sum = %d, want %d", got, max)
	}
	if got := saturatingAddNonnegative(7, -3); got != 7 {
		t.Fatalf("negative model size changed aggregate: %d", got)
	}
}

func TestProductionOllamaRetentionConstructorHasNoDeleteCapability(t *testing.T) {
	client := NewHTTPOllamaModelInventory("http://127.0.0.1:11434")
	provider := NewOllamaModelRetentionProvider(client, &memoryOllamaLedger{}, "model-policy.json", nil)
	if _, ok := any(provider.client).(interface {
		DeleteModel(context.Context, string) error
	}); ok {
		t.Fatal("production Ollama owner adapter unexpectedly exposes name-only deletion")
	}
}

func TestFileOllamaUsageLedgerSurvivesReloadAndProtectsWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ollama-usage-ledger.json")
	first, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	observed := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	if err := first.Record(context.Background(), observed, []OllamaModel{{Name: "model:latest"}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Eligible("model:latest", observed.Add(59*time.Minute), time.Hour) {
		t.Fatal("model became eligible before the full window after ledger reload")
	}
	if !reloaded.Eligible("model:latest", observed.Add(time.Hour), time.Hour) {
		t.Fatal("model did not become eligible after the full window")
	}
}

func TestFileOllamaInventoryAccountingTracksBaselineCurrentPeakAndRetainedPreview(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ollama-usage-ledger.json")
	first, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	if _, err := first.RecordInventory(context.Background(), t0, []OllamaModel{{Name: "chat:latest", Size: 100}, {Name: "embed:latest", Size: 50}}); err != nil {
		t.Fatal(err)
	}
	if _, err := first.RecordInventory(context.Background(), t0.Add(time.Hour), []OllamaModel{{Name: "chat:latest", Size: 125}}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	want := ollamaInventoryTotals{BaselineLogicalBytes: 150, CurrentLogicalBytes: 125, PeakLogicalBytes: 150, FirstObservedAt: t0, LastObservedAt: t0.Add(time.Hour)}
	if got := reloaded.Inventory; got != want {
		t.Fatalf("persisted inventory totals = %+v, want %+v", got, want)
	}

	policyPath := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"roles":{"chat":{"model":"chat:latest","fallbacks":["digestless:latest"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := NewOllamaModelRetentionProvider(&fakeOllamaInventory{
		models:  []OllamaModel{{Name: "chat:latest", Size: 125, Digest: "sha256:chat"}, {Name: "digestless:latest", Size: 25}},
		running: []OllamaModel{{Name: "chat:latest", Size: 125}},
	}, reloaded, policyPath, nil)
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope: cleanup.ObservationScope{Now: t0.Add(2 * time.Hour)}, Policy: cleanup.ProviderPolicy{Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	warnings := strings.Join(preview.Warnings, " ")
	for _, fact := range []string{"baseline=150 B", "current=150 B", "peak=150 B", "retained=150 B", "Explicit policy/running protected bytes in this preview=150 B at 2026-11-11T20:00:00Z", "digestless:latest: protected by the active Ollama model policy", "digestless:latest: Ollama did not report a digest", "no atomic owner deletion fence"} {
		if !strings.Contains(warnings, fact) {
			t.Fatalf("preview warnings missing %q: %s", fact, warnings)
		}
	}
	if len(preview.Items) != 0 {
		t.Fatalf("accounting preview emitted deletion items: %+v", preview.Items)
	}
}

func TestFileOllamaInventoryAccountingSerializesConcurrentHandles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ollama-usage-ledger.json")
	first, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)
	if _, err := first.RecordInventory(context.Background(), t0, []OllamaModel{{Name: "baseline:latest", Size: 100}}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, 40)
	for i := 1; i <= 40; i++ {
		ledger := first
		if i%2 == 0 {
			ledger = second
		}
		size := int64(i * 100)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := ledger.RecordInventory(context.Background(), t0.Add(time.Duration(size)*time.Second), []OllamaModel{{Name: "concurrent:latest", Size: size}})
			errCh <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	reloaded, err := NewFileOllamaUsageLedger(path)
	if err != nil {
		t.Fatal(err)
	}
	totals := reloaded.Inventory
	if totals.BaselineLogicalBytes != 100 || totals.PeakLogicalBytes != 4000 {
		t.Fatalf("concurrent handles lost baseline/high-water accounting: %+v", totals)
	}
	stale, err := reloaded.RecordInventory(context.Background(), t0.Add(time.Second), []OllamaModel{{Name: "late-arriving-old-snapshot:latest", Size: 5000}})
	if err != nil {
		t.Fatal(err)
	}
	if stale.CurrentLogicalBytes != 4000 || stale.PeakLogicalBytes != 5000 || !stale.LastObservedAt.Equal(t0.Add(4_000*time.Second)) {
		t.Fatalf("out-of-order snapshot rewrote current instead of only raising peak: %+v", stale)
	}
}

func TestOllamaRetentionApplyAndVerifyFailClosedWithoutOwnerFence(t *testing.T) {
	client := &deleteCountingOllamaInventory{}
	provider := NewOllamaModelRetentionProvider(client, &memoryOllamaLedger{}, "model-policy.json", nil)
	preview := cleanup.Preview{ProviderID: provider.Metadata().ID, ProviderVersion: "v3", Items: []cleanup.PreviewItem{{ID: "ollama-model:model:latest", Bytes: 42}}}
	for _, mode := range []cleanup.ApprovalMode{cleanup.ApprovalModeNone, cleanup.ApprovalModeOperator} {
		result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
			ProviderVersion: "v3", IdempotencyKey: string(mode), ApprovalMode: mode, Preview: preview,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Applied || result.ReclaimedBytes != 0 || len(result.AppliedItems) != 0 || len(result.SkippedItems) != 1 {
			t.Fatalf("mode %s did not fail closed: %+v", mode, result)
		}
		if mode == cleanup.ApprovalModeOperator && !strings.Contains(strings.Join(result.Warnings, " "), "reclaimed bytes remain unknown and are reported as zero") {
			t.Fatalf("zero reclaimed bytes were not distinguished from an authoritative zero delta: %+v", result)
		}
	}
	if client.deleteCalls != 0 {
		t.Fatalf("retention Apply invoked name-only model deletion %d times", client.deleteCalls)
	}

	partial := cleanup.ApplyResult{Applied: true, AppliedItems: []string{"ollama-model:removed:latest"}, SkippedItems: []string{"ollama-model:held:latest"}}
	verified, err := provider.Verify(context.Background(), cleanup.VerifyRequest{ApplyResult: partial})
	if err != nil {
		t.Fatal(err)
	}
	if verified.Verified || !strings.Contains(verified.Message, "skipped one or more items") {
		t.Fatalf("partial apply was incorrectly reported verified: %+v", verified)
	}
}

func TestPublicOllamaInventoryDoesNotExposeDelete(t *testing.T) {
	client := NewHTTPOllamaModelInventory("http://127.0.0.1:11434")
	if _, ok := any(client).(interface {
		DeleteModel(context.Context, string) error
	}); ok {
		t.Fatal("public Ollama API adapter unexpectedly exposes name-only model deletion")
	}
	provider := NewOllamaModelRetentionProvider(client, &memoryOllamaLedger{}, "model-policy.json", nil)
	result, err := provider.Verify(context.Background(), cleanup.VerifyRequest{ApplyResult: cleanup.ApplyResult{Applied: true, AppliedItems: []string{"ollama-model:missing:latest"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verified || !strings.Contains(result.Message, "no supported atomic retention deletion contract") {
		t.Fatalf("production API was reported verified without an owner mutation contract: %+v", result)
	}
}
