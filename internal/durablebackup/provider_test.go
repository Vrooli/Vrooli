package durablebackup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscoverReadyRequiresCoverageAndChecksumBackedDrill(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1,
			Planned:    1,
			BackedUp:   1,
			Verified:   1,
			Drill:      DrillEvidence{ID: "drill-1", SnapshotID: "snapshot-1", RestoreID: "restore-1", Status: "verified", Checksum: strings.Repeat("a", 64), ObservedAt: time.Now()},
		}, nil
	})
	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "ready" {
		t.Fatalf("state = %q, want ready (%s)", status.State, status.Remediation)
	}
	if len(status.Evidence) != 2 || !status.Evidence[1].Verified || status.Evidence[1].Checksum == "" {
		t.Fatalf("evidence = %#v, want verified coverage and drill checksum", status.Evidence)
	}
	if strings.Contains(strings.ToLower(status.Evidence[1].ArtifactIdentity), "secret") {
		t.Fatalf("evidence identity contains secret wording: %#v", status.Evidence[1])
	}
}

func TestDiscoverDegradesWithoutVerifiedDrill(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{Registered: 1, Planned: 1, BackedUp: 1, Verified: 1, Drill: DrillEvidence{Status: "running"}}, nil
	})
	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "recovery drill") {
		t.Fatalf("status = %#v, want degraded recovery-drill remediation", status)
	}
}

func TestDiscoverDegradesWhenDBMIsUnavailable(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{}, context.DeadlineExceeded
	})
	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "unavailable") {
		t.Fatalf("status = %#v, want degraded unavailable evidence", status)
	}
}

func TestFetchProductionReadsPublicCoverageDrillAndRestoreContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/vrooli.data_backup_manager.v1.coverage.CoverageService/GetCoverageReport":
			_, _ = w.Write([]byte(`{"report":{"summary":{"registeredCount":1,"plannedCount":1,"backedUpCount":1,"verifiedCount":1}}}`))
		case "/vrooli.data_backup_manager.v1.drills.RecoveryDrillsService/ListDrills":
			_, _ = w.Write([]byte(`{"drills":[{"id":"drill-1","snapshotId":"snapshot-1","restoreId":"restore-1","status":"DRILL_STATUS_VERIFIED","finishedAt":"2026-08-19T16:00:00Z"}]}`))
		case "/vrooli.data_backup_manager.v1.restores.RestoresService/GetRestore":
			_, _ = w.Write([]byte(`{"restore":{"checksum":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","finishedAt":"2026-08-19T16:01:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	evidence, err := fetchProduction(context.Background(), func() string { return port })
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Registered != 1 || evidence.Drill.Status != "verified" || evidence.Drill.Checksum == "" || evidence.Drill.ObservedAt.IsZero() {
		t.Fatalf("evidence = %#v, want coverage and verified restore checksum", evidence)
	}
	if got := evidence.Drill.Checksum; got != strings.Repeat("a", 64) {
		t.Fatalf("checksum = %q", got)
	}
	if evidence.Drill.ID != "drill-1" || fmt.Sprint(evidence.Drill.SnapshotID) != "snapshot-1" {
		t.Fatalf("drill identity = %#v", evidence.Drill)
	}
}

func TestFetchProductionHonorsCallerDeadlineWhenDBMDoesNotRespond(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(200 * time.Millisecond):
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	port := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	_, err := fetchProduction(ctx, func() string { return port })
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "deadline") {
		t.Fatalf("error = %v, want caller deadline to bound evidence fetch", err)
	}
}
