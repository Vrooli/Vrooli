package condition

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/coverage"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"
)

// readerProjections are the projections this scenario has a live typed reader
// for. Everything else reports an explicitly unconfigured source rather than
// an empty-but-healthy result.
var readerProjections = []string{"availability", "substrate"}

// NewConfiguredService wires the condition domain's read-only source adapters.
// Handler packages only translate transport messages; source setup and band
// configuration stay with the condition domain.
func NewConfiguredService(root string, db *database.RoutedDB, clk schedule.Clock) *Service {
	retentionFloor, retentionErr := coverage.RetentionFloor(root)
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	httpClient := &http.Client{Timeout: 10 * time.Second}
	// One qualifier cache is shared by every projection reader. The qualifier
	// facts are registry-wide, and one of them derives the core set through a
	// multi-second subprocess, so a read covering N projections paid that cost
	// N times before this was shared.
	qualifiers := sources.NewQualifierCache(5 * time.Second)
	readers := make(map[string]sources.Reader, len(readerProjections))
	for _, projection := range readerProjections {
		readers[projection] = sources.AutohealReader{Resolver: resolver, HTTP: httpClient, Projection: projection, Qualifiers: qualifiers}
	}
	service := &Service{
		Source:         fanoutSource{readers: readers},
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
		return func(Observation) Band {
			return Band{NotGradeableReason: fmt.Sprintf("the setpoint could not be read: %v", err)}
		}
	}
	byCell := make(map[string]coverage.Bar, len(setpoint.Bars))
	for _, bar := range setpoint.Bars {
		byCell[bar.CellRef] = bar
	}
	return func(reading Observation) Band {
		bar, ok := byCell[reading.CellRef]
		if !ok {
			return Band{NeedsBaseline: true}
		}
		if !bar.Gradeable {
			reason := bar.NotGradeableReason
			if reason == "" {
				reason = "the bar authors no threshold"
			}
			return Band{NotGradeableReason: reason, Unit: bar.Unit}
		}
		// A reading in a different unit from its bar is a unit mismatch, not a
		// value to compare. Grading it would silently answer a question about
		// percent with a threshold authored in severities.
		if bar.Unit != "" && reading.Unit != "" && bar.Unit != reading.Unit {
			return Band{NotGradeableReason: fmt.Sprintf("reading unit %q does not match bar unit %q", reading.Unit, bar.Unit), Unit: bar.Unit}
		}
		return Band{
			Min: bar.Min, Max: bar.Max, SustainSatisfied: true,
			Unit: bar.Unit, Provisional: bar.Provisional,
		}
	}
}

// PeerSourceAvailability reports, per projection, which typed peer readers are
// configured. It names the owner for every projection so an unconfigured join
// is visible as a gap rather than as an absent row.
func PeerSourceAvailability(projection string, checkedAt time.Time) []SourceAvailability {
	owner := map[string]string{
		"supervision": "vrooli-autoheal", "availability": "vrooli-autoheal", "recovery": "vrooli-autoheal",
		"substrate": "vrooli-autoheal", "headroom": "storage-manager", "durability": "data-backup-manager",
		"attribution": "system-monitor", "validation-cost": "test-genie", "agent-throughput": "agent-manager",
		"commissioning": "control-plane", "capacity": "control-plane",
	}
	sourceIDs := []string{"vrooli-autoheal", "storage-manager", "data-backup-manager", "system-monitor", "test-genie", "agent-manager", "control-plane"}
	served := map[string]bool{}
	for _, served_ := range readerProjections {
		served[served_] = true
	}
	result := make([]SourceAvailability, 0, len(sourceIDs))
	for _, source := range sourceIDs {
		availability := SourceAvailability{Source: source, Available: false, CheckedAt: checkedAt}
		switch {
		case owner[projection] == source && served[projection]:
			availability.Available = true
			availability.Reason = fmt.Sprintf("served by the autoheal %s reader", projection)
		default:
			availability.Reason = fmt.Sprintf("typed reader for %s is not configured for projection %s", source, projection)
		}
		result = append(result, availability)
	}
	return result
}

// fanoutSource routes a projection to its typed reader. An unrouted projection
// returns an explicitly unavailable source, never an empty success.
type fanoutSource struct{ readers map[string]sources.Reader }

func (s fanoutSource) Read(ctx context.Context, projection string) ([]Observation, SourceAvailability, error) {
	checkedAt := time.Now().UTC()
	reader, ok := s.readers[projection]
	if !ok {
		return nil, SourceAvailability{
			Source:    "vrooli-autoheal",
			Available: false,
			Reason:    fmt.Sprintf("no typed reader is configured for projection %s", projection),
			CheckedAt: checkedAt,
		}, nil
	}
	result := sources.Read(ctx, []sources.Endpoint{{ID: "vrooli-autoheal", Reader: reader}}, 10*time.Second)[0]
	availability := SourceAvailability{Source: result.ID, Available: result.Available, Reason: result.Reason, CheckedAt: result.CheckedAt}
	readings := make([]Observation, 0, len(result.Observations))
	for _, reading := range result.Observations {
		trust := TrustInput{
			Available: result.Available, Ghost: reading.TrustHints.Ghost,
			Saturated: reading.TrustHints.Saturated, Shelved: reading.TrustHints.Shelved,
			UnitMatches: reading.TrustHints.UnitMatches,
		}
		if reading.TrustHints.Untrusted {
			trust.VerdictToken = "UNTRUSTED"
		}
		readings = append(readings, Observation{
			ID: reading.ID, CellRef: reading.CellRef, Value: reading.Value, Unit: reading.Unit,
			Source: reading.Source, ObservedAt: reading.ObservedAt, Trust: EvaluateTrust(trust),
			Band: Band{NeedsBaseline: true}, UnavailableReason: reading.TrustHints.UntrustedReason,
			OutOfScope: reading.TrustHints.OutOfScope,
		})
	}
	return readings, availability, nil
}

// ReadAll reads every projection that has a configured reader and merges the
// results. The trust triple is only honest over a known denominator, so a
// caller that names no projection gets the whole readable surface rather than
// an empty distribution.
func (s *Service) ReadAll(ctx context.Context) Snapshot {
	projections := append([]string(nil), readerProjections...)
	sort.Strings(projections)
	merged := Snapshot{Trust: TrustTriple{Distribution: map[TrustVerdict]int{}}}
	for _, projection := range projections {
		snapshot := s.Read(ctx, projection)
		merged.Readings = append(merged.Readings, snapshot.Readings...)
		merged.Sources = append(merged.Sources, snapshot.Sources...)
		for verdict, count := range snapshot.Trust.Distribution {
			merged.Trust.Distribution[verdict] += count
		}
		merged.Trust.CheckedDenominator += snapshot.Trust.CheckedDenominator
		merged.Trust.Total += snapshot.Trust.Total
		merged.At = snapshot.At
	}
	merged.Trust.CheckedAt = merged.At
	return merged
}
