package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// StorageReader consumes storage-manager's read-only typed JSON projections.
// It never requests a census or recovery operation; infrastructure-manager is
// an observer and must not become an accidental actuator.
type StorageReader struct {
	Resolver *discovery.Resolver
	HTTP     *http.Client
	BaseURL  string // test seam; production resolves storage-manager by scenario name
}

type storageHealth struct {
	SnapshotCount                   int      `json:"snapshot_count"`
	DeclaredCeilingCoverage         float64  `json:"declared_ceiling_coverage"`
	DeclaredCeilingBytes            int64    `json:"declared_ceiling_bytes"`
	EnforcedCeilingCoverage         float64  `json:"enforced_ceiling_coverage"`
	DeclaredCeilingMeasuredCoverage float64  `json:"declared_ceiling_measured_coverage"`
	Confidence                      string   `json:"confidence"`
	GrowthSlopeBytesPerHour         *float64 `json:"growth_slope_bytes_per_hour"`
	SnapshotAgeSeconds              *float64 `json:"snapshot_age_seconds"`
	MeasuredBytes                   int64    `json:"measured_bytes"`
	UnattributedBytes               int64    `json:"unattributed_bytes"`
	RecoveryEfficacy                *float64 `json:"recovery_efficacy"`
	BudgetTruth                     *float64 `json:"budget_truth"`
}

type storageWriter struct {
	BytesPerHour int64 `json:"bytes_per_hour"`
	Hot          bool  `json:"hot"`
}

func (r StorageReader) Read(ctx context.Context) ([]Observation, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	base := r.BaseURL
	if base == "" {
		var err error
		base, err = resolver.ResolveScenarioURLDefault(ctx, "storage-manager")
		if err != nil {
			return nil, err
		}
	}
	var health storageHealth
	if err := getJSON(ctx, httpClient, base+"/api/v1/infra-health/storage?root=/", &health); err != nil {
		return unavailableStorageReadings(fmt.Sprintf("storage_manager_unreachable: %v", err)), nil
	}
	writers := []storageWriter{}
	if err := getJSON(ctx, httpClient, base+"/api/v1/storage/writers?top=10", &writers); err != nil {
		return unavailableStorageReadings(fmt.Sprintf("storage_manager_partial: %v", err)), nil
	}
	now := time.Now().UTC()
	if health.Confidence != "full" {
		return unavailableStorageReadings("storage_manager_partial: feed confidence is not full"), nil
	}
	if health.DeclaredCeilingBytes <= 0 {
		return unavailableStorageReadings("storage_manager_partial: declared ceiling bytes are unavailable"), nil
	}
	growthPercent := 0.0
	if health.GrowthSlopeBytesPerHour != nil {
		growthPercent = *health.GrowthSlopeBytesPerHour / float64(health.DeclaredCeilingBytes) * 100
	}
	h2Trust := TrustHints{UnitMatches: true}
	if health.SnapshotAgeSeconds != nil && *health.SnapshotAgeSeconds > 30*60 {
		h2Trust.Untrusted = true
		h2Trust.UntrustedReason = "storage_manager_stale: latest census is older than two census intervals"
	} else if health.MeasuredBytes > 0 && float64(health.UnattributedBytes)/float64(health.MeasuredBytes) > 0.10 {
		h2Trust.Untrusted = true
		h2Trust.UntrustedReason = "storage_manager_partial: unattributed roots exceed 10% of measured bytes"
	}
	out := []Observation{
		{ID: "storage-manager-H1", CellRef: "headroom/H1", Value: float64(boolInt(health.SnapshotCount == 0)), Unit: "load-bearing devices absent from the census", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{UnitMatches: true}},
		{ID: "storage-manager-H2", CellRef: "headroom/H2", Value: growthPercent, Unit: "percent of declared ceiling", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: h2Trust},
		{ID: "storage-manager-H3", CellRef: "headroom/H3", Value: health.DeclaredCeilingCoverage * 100, Unit: "percent of owners", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{UnitMatches: true}},
	}
	if health.RecoveryEfficacy != nil {
		out = append(out, Observation{ID: "storage-manager-H4", CellRef: "headroom/H4", Value: *health.RecoveryEfficacy * 100, Unit: "percent of recovery runs reaching target", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{UnitMatches: true}})
	} else {
		out = append(out, Observation{ID: "storage-manager-H4", CellRef: "headroom/H4", Unit: "percent of recovery runs reaching target", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{Unavailable: true, Untrusted: true, UntrustedReason: "storage_manager_insufficient_history: no terminal recovery run observed"}})
	}
	if health.BudgetTruth != nil {
		out = append(out, Observation{ID: "storage-manager-H5", CellRef: "headroom/H5", Value: *health.BudgetTruth * 100, Unit: "percent of measured declared bytes with agreeing enforcement", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{UnitMatches: true}})
	} else {
		out = append(out, Observation{ID: "storage-manager-H5", CellRef: "headroom/H5", Unit: "percent of measured declared bytes with agreeing enforcement", Source: "storage-manager/infra-health", ObservedAt: now, TrustHints: TrustHints{Unavailable: true, Untrusted: true, UntrustedReason: "storage_manager_insufficient_measurement: no declared bytes observed"}})
	}
	var hottestRate int64
	for _, writer := range writers {
		if writer.Hot && writer.BytesPerHour > hottestRate {
			hottestRate = writer.BytesPerHour
		}
	}
	out = append(out, Observation{ID: "storage-manager-H6", CellRef: "headroom/H6", Value: float64(hottestRate), Unit: "bytes per hour", Source: "storage-manager/storage-writers", ObservedAt: now, TrustHints: TrustHints{UnitMatches: true}})
	return out, nil
}

func unavailableStorageReadings(reason string) []Observation {
	units := []string{"load-bearing devices absent from the census", "percent of declared ceiling", "percent of owners", "percent of recovery runs reaching target", "percent of measured declared bytes with agreeing enforcement", "hot governed roots"}
	now := time.Now().UTC()
	readings := make([]Observation, 0, len(units))
	for i, unit := range units {
		readings = append(readings, Observation{ID: fmt.Sprintf("storage-manager-H%d", i+1), CellRef: fmt.Sprintf("headroom/H%d", i+1), Unit: unit, Source: "storage-manager", ObservedAt: now, TrustHints: TrustHints{Unavailable: true, Untrusted: true, UntrustedReason: reason}})
	}
	return readings
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("GET %s returned %s", endpoint, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
