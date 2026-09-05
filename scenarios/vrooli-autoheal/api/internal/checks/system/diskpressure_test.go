package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/cleanupmanager"
)

func TestSystemMonitorDiskPressureReaderReadsTypedObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/disk-pressure" {
			t.Fatalf("path = %q, want /api/v1/disk-pressure", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"observed":true,"band":"critical","mount_path":"/data","used_percent":96.5,"available_bytes":42,"fill_rate_bytes_per_hour":200,"hot_writers":[{"root":"/tmp/hot","current_bytes":300,"bytes_per_hour":400,"window_seconds":60}]}`))
	}))
	defer server.Close()

	reader := NewSystemMonitorDiskPressureReader(server.Client(), server.URL)
	got, err := reader.ReadDiskPressure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Band != "critical" || got.MountPath != "/data" || got.UsedPercent != 96.5 || got.FillRateBytesPerHour != 200 || len(got.HotWriters) != 1 || got.HotWriters[0].CurrentBytes != 300 {
		t.Fatalf("observation = %+v", got)
	}
}

type fixedDiskPressureReader struct{ pressure DiskPressure }

func (r fixedDiskPressureReader) ReadDiskPressure(context.Context) (DiskPressure, error) {
	return r.pressure, nil
}

func TestDiskCheckUsesSystemMonitorBandInsteadOfLocalThresholds(t *testing.T) {
	reporter := &fakeCleanupReporter{}
	check := NewDiskCheck(
		WithDiskThresholds(99, 100),
		WithDiskPressureReader(fixedDiskPressureReader{pressure: DiskPressure{
			Observed: true, Band: "high", MountPath: "/data", UsedPercent: 86,
		}}),
		WithCleanupReporter(reporter),
	)

	result := check.Run(context.Background())
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning from authoritative high band", result.Status)
	}
	if result.Details["evidence_source"] != "system-monitor" {
		t.Fatalf("details = %+v", result.Details)
	}

	action := check.ExecuteAction(context.Background(), requestCleanupActionID)
	if !action.Success || len(reporter.reports) != 1 {
		t.Fatalf("action = %+v, reports = %+v", action, reporter.reports)
	}
	if reporter.reports[0].Band != cleanupmanager.BandHigh || reporter.reports[0].Partition != "/data" {
		t.Fatalf("report = %+v", reporter.reports[0])
	}
}
