package durablebackup

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/operatorcapability"
)

func TestDiscoverReadyRequiresCoverageAndChecksumBackedDrill(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1,
			Planned:    1,
			BackedUp:   1,
			Verified:   1,
			Sources:    []SourceCoverage{{ID: "source-1", Planned: true, BackedUp: true, Verified: true, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()}},
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
	if !containsCoverage(status.Evidence[0].Coverage, "source:source-1") || !containsCoverage(status.Evidence[0].Coverage, "source:source-1:verified_at:") {
		t.Fatalf("coverage = %#v, want exact source identity and freshness", status.Evidence[0].Coverage)
	}
	if strings.Contains(strings.ToLower(status.Evidence[1].ArtifactIdentity), "secret") {
		t.Fatalf("evidence identity contains secret wording: %#v", status.Evidence[1])
	}
}

func TestRegistryVerifyBindsFreshOwnerEvidence(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	provider := NewProviderWithFetcherAndFreshness(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1, Planned: 1, BackedUp: 1, Verified: 1,
			Sources: []SourceCoverage{{ID: "source-1", Planned: true, BackedUp: true, Verified: true, LastSuccessAt: now.Add(-time.Hour), LastVerifiedAt: now.Add(-time.Hour)}},
			Drill:   DrillEvidence{ID: "drill-1", SnapshotID: "snapshot-1", RestoreID: "restore-1", Status: providerVerified, Checksum: strings.Repeat("b", 64), ObservedAt: now.Add(-time.Hour)},
		}, nil
	}, 24*time.Hour)
	provider.now = func() time.Time { return now }
	registry, err := operatorcapability.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	credential := &operatorcapability.CredentialEvidenceRef{LogicalID: "vrooli/test", Field: "api-key", Version: "opaque-version-1"}
	receipts, err := registry.Verify(context.Background(), operatorcapability.VerificationRequest{
		CapabilityID: CapabilityID, CredentialRef: credential, TargetID: "local", Environment: "development", AccountIdentity: "operator", Operation: "recovery-readiness",
		Effect: operatorcapability.EffectBudget{Class: operatorcapability.EffectReadOnly},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 2 {
		t.Fatalf("receipts = %#v, want coverage and drill evidence", receipts)
	}
	for _, receipt := range receipts {
		if receipt.CapabilityID != CapabilityID || receipt.TargetID != "local" || receipt.Operation != "recovery-readiness" || receipt.CredentialRef == nil || receipt.CredentialRef.Version != credential.Version || receipt.Status != "exercised" || !receipt.FreshAt(now) {
			t.Fatalf("receipt was not bound to the request: %#v", receipt)
		}
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

func TestDiscoverDegradesWhenVerifiedDrillIsStale(t *testing.T) {
	provider := NewProviderWithFetcherAndFreshness(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1, Planned: 1, BackedUp: 1, Verified: 1,
			Drill: DrillEvidence{ID: "drill-1", SnapshotID: "snapshot-1", Status: "verified", Checksum: "checksum", ObservedAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)},
		}, nil
	}, 24*time.Hour)
	provider.now = func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }

	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "older than 24h") {
		t.Fatalf("status = %#v, want stale-drill degradation", status)
	}
	if len(status.Evidence) != 2 || status.Evidence[1].Verified {
		t.Fatalf("evidence = %#v, want stale drill unverified", status.Evidence)
	}
}

func TestDiscoverRejectsMatchingCountsWithWrongSourceCoverage(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 2, Planned: 2, BackedUp: 2, Verified: 2,
			Sources: []SourceCoverage{
				{ID: "source-1", Planned: true, BackedUp: true, Verified: true, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()},
				{ID: "source-2", Planned: true, BackedUp: true, Verified: false, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()},
			},
			Drill: DrillEvidence{ID: "drill-1", Status: "verified", Checksum: "checksum", ObservedAt: time.Now()},
		}, nil
	})
	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "every planned source") {
		t.Fatalf("status = %#v, want source-specific coverage degradation", status)
	}
}

func TestDiscoverDegradesWhenSourceCoverageIsStale(t *testing.T) {
	provider := NewProviderWithFetcherAndFreshness(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1, Planned: 1, BackedUp: 1, Verified: 1,
			Sources: []SourceCoverage{{
				ID: "source-1", Planned: true, BackedUp: true, Verified: true,
				LastSuccessAt:  time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
				LastVerifiedAt: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
			}},
			Drill: DrillEvidence{ID: "drill-1", Status: "verified", Checksum: "checksum", ObservedAt: time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)},
		}, nil
	}, 24*time.Hour)
	provider.now = func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) }

	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "source evidence") {
		t.Fatalf("status = %#v, want stale source degradation", status)
	}
}

func TestDiscoverRejectsAggregateCountsWithoutSourceRows(t *testing.T) {
	provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
		return Evidence{
			Registered: 1, Planned: 1, BackedUp: 1, Verified: 1,
			Drill: DrillEvidence{ID: "drill-1", Status: "verified", Checksum: "checksum", ObservedAt: time.Now()},
		}, nil
	})

	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "degraded" || !strings.Contains(status.Remediation, "source evidence") {
		t.Fatalf("status = %#v, want aggregate-only evidence rejection", status)
	}
}

func TestDiscoverRejectsDuplicateOrUnidentifiedSourceRows(t *testing.T) {
	for _, sources := range [][]SourceCoverage{
		{{Planned: true, BackedUp: true, Verified: true, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()}},
		{
			{ID: "source-1", Planned: true, BackedUp: true, Verified: true, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()},
			{ID: "source-1", Planned: true, BackedUp: true, Verified: true, LastSuccessAt: time.Now(), LastVerifiedAt: time.Now()},
		},
	} {
		provider := NewProviderWithFetcher(func(context.Context) (Evidence, error) {
			return Evidence{Registered: len(sources), Planned: len(sources), BackedUp: len(sources), Verified: len(sources), Sources: sources}, nil
		})
		status, err := provider.Discover(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if status.State != "degraded" || !strings.Contains(status.Remediation, "every planned source") {
			t.Fatalf("status = %#v, want source identity degradation", status)
		}
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
			_, _ = w.Write([]byte(`{"report":{"summary":{"registeredCount":1,"plannedCount":1,"backedUpCount":1,"verifiedCount":1},"registeredTargets":[{"id":"target-1","owner":"vrooli","name":"config","planned":true,"lastSuccessAt":"2026-08-19T15:00:00Z","lastVerifiedAt":"2026-08-19T15:30:00Z"}]}}`))
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
	if len(evidence.Sources) != 1 || !evidence.Sources[0].BackedUp || !evidence.Sources[0].Verified {
		t.Fatalf("source coverage = %#v, want owner-identified covered source", evidence.Sources)
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

func containsCoverage(coverage []string, want string) bool {
	for _, item := range coverage {
		if item == want || strings.HasPrefix(item, want) {
			return true
		}
	}
	return false
}
