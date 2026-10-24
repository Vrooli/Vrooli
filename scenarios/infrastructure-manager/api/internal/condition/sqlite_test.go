package condition

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestSchemaPersistsReadingsWithoutBandVerdict(t *testing.T) {
	schema := strings.ToLower(Schema())
	require.Contains(t, schema, "condition_readings")
	require.Contains(t, schema, "trust_verdict")
	require.NotContains(t, schema, "band_verdict")

	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	at := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Save(context.Background(), []Observation{{
		ID: "check-1", CellRef: "availability/A1", Value: 99.5, Unit: "percent",
		Source: "test", ObservedAt: at, Trust: TrustValid,
	}}))

	readings, err := repo.History(context.Background(), "availability/A1", 10)
	require.NoError(t, err)
	require.Len(t, readings, 1)
	require.Equal(t, TrustValid, readings[0].Trust)
	require.True(t, readings[0].Band.NeedsBaseline, "history must not infer a band verdict from persisted columns")
}

func TestServiceHistoryRequiresRetentionFloor(t *testing.T) {
	at := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	service := &Service{
		Now:            func() time.Time { return at },
		RetentionFloor: 24 * time.Hour,
		Repository: memoryReadingRepository{readings: []Observation{{
			ID: "check-1", CellRef: "availability/A1", Value: 99, Unit: "percent", ObservedAt: at.Add(-time.Hour), Trust: TrustValid,
		}}},
	}
	history := service.History(context.Background(), "availability/A1", 10)
	require.False(t, history.Measurable)
	require.Contains(t, history.UnmeasurableReason, "retention")
}

func TestServiceHistoryRebandsAgainstCurrentBar(t *testing.T) {
	reading := Observation{ID: "check-1", CellRef: "availability/A1", Value: 99, Unit: "percent", ObservedAt: time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC), Trust: TrustValid}
	service := &Service{Repository: memoryReadingRepository{readings: []Observation{reading}}, BandResolver: func(observation Observation) Band {
		min := 99.5
		return Band{Min: &min, SustainSatisfied: true}
	}}
	first := service.History(context.Background(), "availability/A1", 10)
	require.Equal(t, BandOutOfBand, EvaluateBand(first.Readings[0].Value, first.Readings[0].Trust, first.Readings[0].Band))
	service.BandResolver = func(observation Observation) Band {
		min := 98.0
		return Band{Min: &min, SustainSatisfied: true}
	}
	second := service.History(context.Background(), "availability/A1", 10)
	require.Equal(t, BandInBand, EvaluateBand(second.Readings[0].Value, second.Readings[0].Trust, second.Readings[0].Band))
}

type memoryReadingRepository struct{ readings []Observation }

func (r memoryReadingRepository) Save(_ context.Context, readings []Observation) error {
	r.readings = append(r.readings, readings...)
	return nil
}

func (r memoryReadingRepository) History(_ context.Context, _ string, _ int) ([]Observation, error) {
	return r.readings, nil
}
