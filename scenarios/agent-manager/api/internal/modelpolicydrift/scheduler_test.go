package modelpolicydrift

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
)

type recordingReporter struct{ reports []Report }

func (r *recordingReporter) Report(_ context.Context, report Report) error {
	r.reports = append(r.reports, report)
	return nil
}

func TestNewUsesWeeklyScheduleAndCapsLongIntervals(t *testing.T) {
	scheduler := New(t.TempDir(), "", 30*24*time.Hour, nil)
	if scheduler.interval != 7*24*time.Hour || scheduler.Snapshot().IntervalHours != 168 {
		t.Fatalf("schedule=%s snapshot=%+v", scheduler.interval, scheduler.Snapshot())
	}
}

func TestRunOncePersistsStatusAndDeduplicatesFindings(t *testing.T) {
	root := t.TempDir()
	reporter := &recordingReporter{}
	scheduler := New(root, filepath.Join(root, "state.json"), time.Hour, reporter)
	scheduler.check = func(_ context.Context, runner, _ string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
		if runner == "codex" {
			return []agentcatalog.PolicyValidationFinding{{Type: "missing_primary_model", Severity: "error", Role: "code.default", Model: "gone", Message: "missing"}}, agentcatalog.LiveModelCatalog{}, nil
		}
		return nil, agentcatalog.LiveModelCatalog{}, os.ErrNotExist
	}
	first := scheduler.RunOnce(context.Background())
	if first.Status != "critical" || len(reporter.reports) != 1 || first.Measured != 1 {
		t.Fatalf("first snapshot=%+v reports=%d", first, len(reporter.reports))
	}
	second := scheduler.RunOnce(context.Background())
	if len(reporter.reports) != 1 || second.Status != "critical" {
		t.Fatalf("dedup failed snapshot=%+v reports=%d", second, len(reporter.reports))
	}
	loaded := New(root, filepath.Join(root, "state.json"), time.Hour, nil).Snapshot()
	if loaded.LastRun.IsZero() || len(loaded.Reported) != 1 {
		t.Fatalf("state was not durable: %+v", loaded)
	}
}

func TestRunOnceReportsNotMeasuredWithoutFalseDrift(t *testing.T) {
	scheduler := New(t.TempDir(), filepath.Join(t.TempDir(), "state.json"), 0, nil)
	scheduler.check = func(context.Context, string, string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
		return nil, agentcatalog.LiveModelCatalog{}, os.ErrNotExist
	}
	snapshot := scheduler.RunOnce(context.Background())
	if snapshot.Status != "not_measured" || len(snapshot.Findings) != 0 || snapshot.Measured != 0 {
		t.Fatalf("not-measured state=%+v", snapshot)
	}
}
