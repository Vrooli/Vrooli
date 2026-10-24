package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageReaderReturnsHeadroomObservationsFromTypedFeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/infra-health/storage" {
			_ = json.NewEncoder(w).Encode(map[string]any{"snapshot_count": 4, "declared_ceiling_coverage": 0.8, "declared_ceiling_bytes": 100, "enforced_ceiling_coverage": 0.6, "declared_ceiling_measured_coverage": 0.7, "recovery_efficacy": 0.75, "budget_truth": 0.9, "growth_slope_bytes_per_hour": 100, "confidence": "full"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"bytes_per_hour": 200, "hot": true}})
	}))
	defer server.Close()

	readings, err := (StorageReader{BaseURL: server.URL, HTTP: server.Client()}).Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(readings) != 6 {
		t.Fatalf("readings = %d, want H1-H6", len(readings))
	}
	if readings[0].CellRef != "headroom/H1" || readings[0].Value != 0 || readings[0].Unit != "load-bearing devices absent from the census" {
		t.Fatalf("H1 = %#v", readings[0])
	}
	if readings[5].CellRef != "headroom/H6" || readings[5].Value != 200 || readings[5].Unit != "bytes per hour" {
		t.Fatalf("H6 = %#v", readings[5])
	}
	if readings[3].CellRef != "headroom/H4" || readings[3].Value != 75 {
		t.Fatalf("H4 = %#v", readings[3])
	}
	if readings[4].CellRef != "headroom/H5" || readings[4].Value != 90 {
		t.Fatalf("H5 = %#v", readings[4])
	}
}

func TestStorageReaderKeepsAllCellsWhenOwnerIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "storage manager is down", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	readings, err := (StorageReader{BaseURL: server.URL, HTTP: server.Client()}).Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(readings) != 6 {
		t.Fatalf("readings = %d, want six unavailable cells", len(readings))
	}
	for _, reading := range readings {
		if !reading.TrustHints.Unavailable || reading.TrustHints.UntrustedReason == "" {
			t.Fatalf("reading = %#v, want typed unavailable reason", reading)
		}
	}
}

func TestStorageReaderMarksGrowthUntrustedWhenCensusIsMostlyUnattributed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/infra-health/storage" {
			_ = json.NewEncoder(w).Encode(map[string]any{"snapshot_count": 4, "declared_ceiling_coverage": 1, "declared_ceiling_bytes": 1000, "growth_slope_bytes_per_hour": 100, "confidence": "full", "measured_bytes": 100, "unattributed_bytes": 11})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer server.Close()

	readings, err := (StorageReader{BaseURL: server.URL, HTTP: server.Client()}).Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !readings[1].TrustHints.Untrusted || readings[1].TrustHints.UntrustedReason == "" {
		t.Fatalf("H2 = %#v, want unattributed-rate trust warning", readings[1])
	}
}
