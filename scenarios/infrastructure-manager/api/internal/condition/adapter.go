package condition

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"
	"infrastructure-manager/internal/coverage"
	"infrastructure-manager/internal/sources"
)

// NewConfiguredService wires the condition domain's read-only source adapter.
// Handler packages only translate transport messages; source setup and band
// configuration stay with the condition domain.
func NewConfiguredService(root string, db *database.RoutedDB, clk schedule.Clock) *Service {
	retentionFloor, retentionErr := coverage.RetentionFloor(root)
	service := &Service{
		Source:         fanoutSource{reader: sources.AutohealReader{Resolver: discovery.NewResolver(discovery.ResolverConfig{}), HTTP: &http.Client{Timeout: 10 * time.Second}, Projection: "availability"}},
		RetentionFloor: retentionFloor,
	}
	service.BandResolver = currentBandResolver(root)
	if retentionErr != nil {
		service.RetentionFloorReason = fmt.Sprintf("condition retention floor is unavailable: %v", retentionErr)
	}
	if db != nil {
		service.Repository = NewSQLiteRepository(db)
	}
	if clk != nil {
		service.Now = clk.Now
	}
	return service
}

func currentBandResolver(root string) func(Observation) Band {
	setpoint, err := coverage.LoadSetpoint(root)
	if err != nil {
		return func(Observation) Band { return Band{NeedsBaseline: true} }
	}
	byCell := make(map[string]coverage.Bar, len(setpoint.Bars))
	for _, bar := range setpoint.Bars {
		byCell[bar.CellRef] = bar
	}
	return func(reading Observation) Band {
		bar, ok := byCell[reading.CellRef]
		if !ok || (bar.Min == nil && bar.Max == nil) {
			return Band{NeedsBaseline: true}
		}
		return Band{Min: bar.Min, Max: bar.Max, SustainSatisfied: true}
	}
}

func PeerSourceAvailability(projection string, checkedAt time.Time) []SourceAvailability {
	owner := map[string]string{
		"supervision": "vrooli-autoheal", "availability": "vrooli-autoheal", "recovery": "vrooli-autoheal",
		"headroom": "storage-manager", "durability": "data-backup-manager", "attribution": "system-monitor",
		"validation-cost": "test-genie", "agent-throughput": "agent-manager", "commissioning": "control-plane",
	}
	sourceIDs := []string{"vrooli-autoheal", "storage-manager", "data-backup-manager", "system-monitor", "test-genie", "agent-manager", "control-plane"}
	result := make([]SourceAvailability, 0, len(sourceIDs))
	for _, source := range sourceIDs {
		availability := SourceAvailability{Source: source, Available: false, CheckedAt: checkedAt}
		if owner[projection] == source && source == "vrooli-autoheal" && projection == "availability" {
			availability.Available = true
			availability.Reason = "served by the autoheal availability reader"
		} else {
			availability.Reason = fmt.Sprintf("typed reader for %s is not configured for projection %s", source, projection)
		}
		result = append(result, availability)
	}
	return result
}

type fanoutSource struct{ reader sources.Reader }

func (s fanoutSource) Read(ctx context.Context, projection string) ([]Observation, SourceAvailability, error) {
	checkedAt := time.Now().UTC()
	if projection != "availability" {
		return nil, SourceAvailability{Source: "vrooli-autoheal", Available: false, Reason: "autoheal currently supplies availability legs only", CheckedAt: checkedAt}, nil
	}
	result := sources.Read(ctx, []sources.Endpoint{{ID: "vrooli-autoheal", Reader: s.reader}}, 10*time.Second)[0]
	availability := SourceAvailability{Source: result.ID, Available: result.Available, Reason: result.Reason, CheckedAt: result.CheckedAt}
	readings := make([]Observation, 0, len(result.Observations))
	for _, reading := range result.Observations {
		trust := TrustInput{Available: result.Available, Ghost: reading.TrustHints.Ghost, Saturated: reading.TrustHints.Saturated, Shelved: reading.TrustHints.Shelved, UnitMatches: reading.TrustHints.UnitMatches}
		if reading.TrustHints.Untrusted {
			trust.VerdictToken = "UNTRUSTED"
		}
		readings = append(readings, Observation{ID: reading.ID, CellRef: reading.CellRef, Value: reading.Value, Unit: reading.Unit, Source: reading.Source, ObservedAt: reading.ObservedAt, Trust: EvaluateTrust(trust), Band: Band{NeedsBaseline: true}})
	}
	return readings, availability, nil
}
