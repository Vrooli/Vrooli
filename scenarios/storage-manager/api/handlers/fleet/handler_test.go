package fleet

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	internalfleet "storage-manager/internal/fleet"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/fleet"
)

type fakeClassifier struct {
	entries map[string]internalfleet.ScenarioEntry
}

func (f fakeClassifier) Classify(_ context.Context, s string) (internalfleet.ScenarioEntry, error) {
	return f.entries[s], nil
}

type memStore struct{ res internalfleet.Result }

func (m *memStore) Save(_ context.Context, r internalfleet.Result) error { m.res = r; return nil }
func (m *memStore) Load(_ context.Context) (internalfleet.Result, error) { return m.res, nil }

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Unix(42, 0).UTC() }

func newTestHandler() *Handler {
	cls := fakeClassifier{entries: map[string]internalfleet.ScenarioEntry{
		"alpha": {Scenario: "alpha", Engines: []string{"postgres"}, PrimaryEngine: "postgres", StorageStage: "production", IsolationReady: false, IsolationReason: "seams unwired", HasBackupTarget: false, FindingCount: 1, ErrorCount: 1},
		"beta":  {Scenario: "beta", Engines: []string{"sqlite"}, PrimaryEngine: "sqlite", StorageStage: "greenfield", IsolationReady: true, NamespaceAdopted: true, HasBackupTarget: true},
	}}
	svc := internalfleet.NewService(cls, nil, &memStore{}, fixedClock{})
	return NewHandler(svc, nil)
}

func TestScanFleetAndGetInventory(t *testing.T) {
	h := newTestHandler()

	resp, err := h.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{Scenarios: []string{"alpha", "beta"}}))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	msg := resp.Msg
	if msg.GetScenarioCount() != 2 {
		t.Fatalf("scenario_count: got %d want 2", msg.GetScenarioCount())
	}
	if msg.GetIsolationUnreadyCount() != 1 {
		t.Fatalf("isolation_unready: got %d want 1", msg.GetIsolationUnreadyCount())
	}
	if msg.GetNoBackupCount() != 1 {
		t.Fatalf("no_backup: got %d want 1", msg.GetNoBackupCount())
	}
	if msg.GetScannedAt() == "" {
		t.Fatal("expected scanned_at stamped")
	}
	if len(msg.GetEntries()) != 2 {
		t.Fatalf("entries: got %d want 2", len(msg.GetEntries()))
	}

	// GetInventory reads back the persisted snapshot.
	inv, err := h.GetInventory(context.Background(), connect.NewRequest(&fleetv1.GetInventoryRequest{}))
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if inv.Msg.GetScenarioCount() != 2 {
		t.Fatalf("inventory scenario_count: got %d want 2", inv.Msg.GetScenarioCount())
	}
}
