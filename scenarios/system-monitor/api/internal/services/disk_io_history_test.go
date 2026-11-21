package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

func TestDiskIOHistoryObservationRejectsScalarOnlyPlaceholders(t *testing.T) {
	data, err := infrastructure.NewStaticProvider().GetInfrastructureMonitorData(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := diskIOHistoryObservation(data, time.Now()); got != nil {
		t.Fatalf("legacy scalar placeholders became persisted history: %#v", got.Values)
	}
}

func TestDiskIOHistoryObservationPreservesTypedStates(t *testing.T) {
	at := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	data := &models.InfrastructureMonitorData{
		Timestamp: at,
		StorageIO: models.StorageIOInfo{
			ReadRate:  models.MetricState{Status: "measured", Value: 0, Units: "MB/s", ObservedAt: at, Provenance: "device counters"},
			WriteRate: models.MetricState{Status: "failed", Reason: "counter reset", Units: "MB/s", ObservedAt: at},
		},
	}
	got := diskIOHistoryObservation(data, at.Add(time.Second))
	if got == nil {
		t.Fatal("typed disk rates were omitted")
	}
	if got.Values["read_rate_status"] != "measured" || got.Values["read_rate"] != float64(0) {
		t.Fatalf("measured zero lost: %#v", got.Values)
	}
	if got.Values["write_rate_status"] != "failed" || got.Values["write_rate_reason"] != "counter reset" {
		t.Fatalf("failed interval lost: %#v", got.Values)
	}
}

func TestFailedDiskIOHistoryObservationMarksBothSignals(t *testing.T) {
	at := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	got := failedDiskIOHistoryObservation(errors.New("provider unavailable"), at)
	for _, key := range []string{"read_rate", "write_rate"} {
		if got.Values[key+"_status"] != "failed" || got.Values[key+"_reason"] != "provider unavailable" {
			t.Errorf("%s state = %#v", key, got.Values)
		}
	}
}
