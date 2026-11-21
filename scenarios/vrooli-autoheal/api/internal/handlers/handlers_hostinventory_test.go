package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestHostInventoryFreshnessUsesLastObservedTime(t *testing.T) {
	now := time.Now().UTC()
	store := &mockStore{hostInventory: &hostinventory.SnapshotRecord{
		ID:          "hostinv-stable",
		CollectedAt: now.Add(-2 * time.Hour),
		LastSeenAt:  now.Add(-2 * time.Minute),
		Inventory: hostinventory.HostInventory{
			ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
		},
	}}
	handlers := setupTestHandlers(store)
	response := httptest.NewRecorder()
	handlers.HostInventory(response, httptest.NewRequest("GET", "/api/v1/host/inventory", nil))
	if response.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body hostinventory.InventoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host inventory response: %v", err)
	}
	if !body.Fresh || body.AgeSeconds > 300 {
		t.Fatalf("fresh=%t ageSeconds=%d, want a fresh unchanged inventory based on lastSeenAt", body.Fresh, body.AgeSeconds)
	}
	if body.Snapshot == nil || !body.Snapshot.CollectedAt.Equal(store.hostInventory.CollectedAt) {
		t.Fatalf("snapshot = %#v, want original collectedAt retained", body.Snapshot)
	}
}

func TestHostInventoryBecomesStaleWhenLastObservedTimeExpires(t *testing.T) {
	now := time.Now().UTC()
	store := &mockStore{hostInventory: &hostinventory.SnapshotRecord{
		ID:          "hostinv-stable",
		CollectedAt: now.Add(-2 * time.Hour),
		LastSeenAt:  now.Add(-10 * time.Minute),
		Inventory: hostinventory.HostInventory{
			ProbeStatus: map[string]hostinventory.IntegrityProbeState{"devices": hostinventory.IntegrityProbeOK},
		},
	}}
	handlers := setupTestHandlers(store)
	response := httptest.NewRecorder()
	handlers.HostInventory(response, httptest.NewRequest("GET", "/api/v1/host/inventory", nil))
	if response.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body hostinventory.InventoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode host inventory response: %v", err)
	}
	if body.Fresh || body.AgeSeconds < 300 {
		t.Fatalf("fresh=%t ageSeconds=%d, want stale inventory based on lastSeenAt", body.Fresh, body.AgeSeconds)
	}
}
