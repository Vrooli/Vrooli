package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

type fakeOllamaInventory struct {
	models  []OllamaModel
	running []OllamaModel
	deleted []string
}

func (f *fakeOllamaInventory) ListModels(context.Context) ([]OllamaModel, error) {
	return append([]OllamaModel(nil), f.models...), nil
}

func (f *fakeOllamaInventory) ListRunningModels(context.Context) ([]OllamaModel, error) {
	return append([]OllamaModel(nil), f.running...), nil
}

func (f *fakeOllamaInventory) DeleteModel(_ context.Context, model string) error {
	f.deleted = append(f.deleted, model)
	return nil
}

type memoryOllamaLedger struct {
	entries map[string]ollamaLedgerEntry
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

func TestOllamaRetentionPreviewIncludesUnreferencedAndLedgerAgedModels(t *testing.T) {
	policyPath := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"roles":{"chat":{"model":"referenced:latest","fallbacks":["fallback:latest"]}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	ledger := &memoryOllamaLedger{entries: map[string]ollamaLedgerEntry{
		"referenced:latest": {FirstObserved: now.Add(-48 * time.Hour), LastUsed: now.Add(-48 * time.Hour)},
		"fallback:latest":   {FirstObserved: now.Add(-48 * time.Hour), LastUsed: now.Add(-48 * time.Hour)},
	}}
	client := &fakeOllamaInventory{
		models: []OllamaModel{
			{Name: "referenced:latest", Size: 10},
			{Name: "fallback:latest", Size: 20},
			{Name: "orphan:latest", Size: 30},
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
	if len(preview.Items) != 2 {
		t.Fatalf("preview items = %#v, want aged fallback and unreferenced orphan", preview.Items)
	}
	if preview.Items[0].ID != "ollama-model:fallback:latest" || preview.Items[1].ID != "ollama-model:orphan:latest" {
		t.Fatalf("preview IDs = %#v", preview.Items)
	}
	if preview.Items[0].Bytes != 20 || preview.Items[1].Bytes != 30 {
		t.Fatalf("preview bytes = %#v", preview.Items)
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

func TestOllamaRetentionApplyRequiresOperatorAndUsesAPI(t *testing.T) {
	client := &fakeOllamaInventory{}
	provider := NewOllamaModelRetentionProvider(client, &memoryOllamaLedger{}, "", nil)
	preview := cleanup.Preview{ProviderID: provider.Metadata().ID, ProviderVersion: "v1", Items: []cleanup.PreviewItem{{ID: "ollama-model:orphan:latest", Bytes: 42}}}
	skipped, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "test", ApprovalMode: cleanup.ApprovalModeNone, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped.SkippedItems) != 1 || len(client.deleted) != 0 {
		t.Fatalf("non-operator apply = %#v, deleted=%v", skipped, client.deleted)
	}
	applied, err := provider.Apply(context.Background(), cleanup.ApplyRequest{ProviderVersion: "v1", IdempotencyKey: "test-approved", ApprovalMode: cleanup.ApprovalModeOperator, Preview: preview})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.ReclaimedBytes != 42 || len(client.deleted) != 1 || client.deleted[0] != "orphan:latest" {
		t.Fatalf("operator apply = %#v, deleted=%v", applied, client.deleted)
	}
}
