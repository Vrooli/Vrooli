package modelpolicydrift

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/upstreamcheck"
)

type recordingReporter struct{ reports []Report }

func (r *recordingReporter) Report(_ context.Context, report Report) (string, error) {
	r.reports = append(r.reports, report)
	return "knw-fixture", nil
}

func TestNewUsesDailyScheduleAndCapsLongIntervals(t *testing.T) {
	scheduler := New(t.TempDir(), "", 30*24*time.Hour, nil)
	if scheduler.interval != 24*time.Hour || scheduler.Snapshot().IntervalHours != 24 {
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
	if first.Status != "critical" || len(reporter.reports) != 4 || first.Measured != 1 {
		t.Fatalf("first snapshot=%+v reports=%d", first, len(reporter.reports))
	}
	second := scheduler.RunOnce(context.Background())
	if len(reporter.reports) != 4 || second.Status != "critical" {
		t.Fatalf("dedup failed snapshot=%+v reports=%d", second, len(reporter.reports))
	}
	loaded := New(root, filepath.Join(root, "state.json"), time.Hour, nil).Snapshot()
	if loaded.LastRun.IsZero() || len(loaded.Reported) != 4 {
		t.Fatalf("state was not durable: %+v", loaded)
	}
}

func TestRunOnceReportsNotMeasuredWithoutFalseDrift(t *testing.T) {
	scheduler := New(t.TempDir(), filepath.Join(t.TempDir(), "state.json"), 0, nil)
	scheduler.check = func(context.Context, string, string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
		return nil, agentcatalog.LiveModelCatalog{}, os.ErrNotExist
	}
	snapshot := scheduler.RunOnce(context.Background())
	if snapshot.Status != "not_measured" || len(snapshot.Findings) != len(runners) || snapshot.Measured != 0 {
		t.Fatalf("not-measured state=%+v", snapshot)
	}
}

type failingReporter struct{}

func (failingReporter) Report(context.Context, Report) (string, error) {
	return "", errors.New("receiver paused")
}
func TestRunOnceRetainsDeliveryFailureAndBoundsNewModelReports(t *testing.T) {
	root := t.TempDir()
	scheduler := New(root, filepath.Join(root, "state.json"), time.Hour, failingReporter{})
	scheduler.check = func(_ context.Context, runner, _ string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
		if runner != "codex" {
			return nil, agentcatalog.LiveModelCatalog{}, nil
		}
		found := make([]agentcatalog.PolicyValidationFinding, 100)
		for i := range found {
			found[i] = agentcatalog.PolicyValidationFinding{Type: "unnamed_live_model", Severity: "warning", Model: fmt.Sprintf("new-%d", i)}
		}
		return found, agentcatalog.LiveModelCatalog{}, nil
	}
	snapshot := scheduler.RunOnce(context.Background())
	if len(snapshot.LastErrors) != 1 || len(snapshot.Reported) != 0 {
		t.Fatalf("delivery failure disappeared: %+v", snapshot)
	}
	receiver := &recordingReporter{}
	scheduler.reporter = receiver
	scheduler.RunOnce(context.Background())
	scheduler.RunOnce(context.Background())
	if len(receiver.reports) != 1 || receiver.reports[0].Finding.Type != "unreviewed_live_catalog" {
		t.Fatalf("catalog review flooded or disappeared: %+v", receiver.reports)
	}
	scheduler.check = func(context.Context, string, string) ([]agentcatalog.PolicyValidationFinding, agentcatalog.LiveModelCatalog, error) {
		return nil, agentcatalog.LiveModelCatalog{}, nil
	}
	if snapshot = scheduler.RunOnce(context.Background()); len(snapshot.Reported) != 0 {
		t.Fatal("resolved findings retained permanent suppression")
	}
}

func TestUpstreamMaintenanceRetainsBehindAndUnknown(t *testing.T) {
	for _, tc := range []struct{ json, want string }{{`{"name":"codex","installed":"0.161.0","latest":"0.164.0","pinned":"0.161.0","status":"behind"}`, "upstream_cli_behind"}, {`{"name":"codex","status":"unknown","note":"network unavailable"}`, "upstream_cli_unknown"}, {`{"name":"codex","status":"up-to-date"}`, ""}} {
		run := upstreamcheck.RunnerFunc(func(_ context.Context, args []string) ([]byte, error) {
			if len(args) != 3 || args[0] != "resource-codex" {
				t.Fatalf("private version resolver introduced: %v", args)
			}
			return []byte(tc.json), nil
		})
		found := checkUpstream(context.Background(), "codex", run)
		if tc.want == "" {
			if len(found) != 0 {
				t.Fatal(found)
			}
		} else if len(found) != 1 || found[0].Type != tc.want {
			t.Fatalf("maintenance result lost: %+v", found)
		}
	}
}
