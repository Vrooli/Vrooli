package executions

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/retention"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

type fakeSweeper struct {
	lastOpts retention.Options
	report   *retention.Report
}

func (f *fakeSweeper) Sweep(_ context.Context, opts retention.Options) (*retention.Report, error) {
	f.lastOpts = opts
	if f.report != nil {
		f.report.DryRun = !opts.Apply
		return f.report, nil
	}
	return &retention.Report{DryRun: !opts.Apply, RemovedByStatus: map[string]int{}}, nil
}

func newRetentionService(sw RetentionSweeper) *service {
	return &service{deps: Deps{Logger: logrus.New(), Executor: &stubExecutor{}, Retention: sw}}
}

func TestPreviewRetention_DryRunPassthrough(t *testing.T) {
	completed := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	sw := &fakeSweeper{report: &retention.Report{
		Removed: []retention.Item{{
			ExecutionID:    uuid.New(),
			Status:         database.ExecutionStatusCompleted,
			WorkflowID:     uuid.New(),
			StartedAt:      completed,
			CompletedAt:    &completed,
			ArtifactDir:    "/recordings/x",
			EstimatedBytes: 1234,
		}},
		EstimatedBytes:  1234,
		RemovedCount:    1,
		RemovedByStatus: map[string]int{database.ExecutionStatusCompleted: 1},
	}}
	svc := newRetentionService(sw)

	resp, err := svc.PreviewExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{MaxAgeDays: 3, KeepLatest: 5}))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if sw.lastOpts.Apply {
		t.Fatalf("preview must call Sweep with Apply=false")
	}
	if sw.lastOpts.MaxAgeDays != 3 || sw.lastOpts.KeepLatest != 5 {
		t.Fatalf("options not threaded: %+v", sw.lastOpts)
	}
	if !resp.Msg.GetDryRun() {
		t.Fatalf("expected dry_run=true")
	}
	if resp.Msg.GetRemovedCount() != 1 || resp.Msg.GetEstimatedBytes() != 1234 {
		t.Fatalf("report not mapped: %+v", resp.Msg)
	}
	if len(resp.Msg.GetRemoved()) != 1 || resp.Msg.GetRemoved()[0].GetEstimatedBytes() != 1234 {
		t.Fatalf("removed item not mapped")
	}
}

func TestRunRetention_RequiresConfirm(t *testing.T) {
	sw := &fakeSweeper{}
	svc := newRetentionService(sw)

	_, err := svc.RunExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{MaxAgeDays: 3}))
	if err == nil {
		t.Fatalf("expected error when confirm is false")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", connect.CodeOf(err))
	}
	if sw.lastOpts.Apply {
		t.Fatalf("Sweep must not be invoked with Apply when confirm=false")
	}
}

func TestRunRetention_ConfirmAppliesSweep(t *testing.T) {
	sw := &fakeSweeper{}
	svc := newRetentionService(sw)

	resp, err := svc.RunExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{MaxAgeDays: 3, Confirm: true}))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !sw.lastOpts.Apply {
		t.Fatalf("confirmed run must set Apply=true")
	}
	if resp.Msg.GetDryRun() {
		t.Fatalf("expected dry_run=false for confirmed run")
	}
}

func TestRunRetention_NilSweeperFailsPrecondition(t *testing.T) {
	svc := newRetentionService(nil)
	_, err := svc.PreviewExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition when retention unavailable, got %v", connect.CodeOf(err))
	}
}

func TestRunRetention_InvalidWorkflowID(t *testing.T) {
	svc := newRetentionService(&fakeSweeper{})
	bad := "not-a-uuid"
	_, err := svc.PreviewExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{WorkflowId: &bad}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestRunRetention_NonTerminalStatusRejected(t *testing.T) {
	svc := newRetentionService(&fakeSweeper{})
	st := basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
	_, err := svc.PreviewExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{Status: &st}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument for non-terminal status, got %v", connect.CodeOf(err))
	}
}

func TestRunRetention_TerminalStatusThreaded(t *testing.T) {
	sw := &fakeSweeper{}
	svc := newRetentionService(sw)
	st := basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
	if _, err := svc.PreviewExecutionArtifactRetention(context.Background(),
		connect.NewRequest(&basapi.ExecutionArtifactRetentionRequest{Status: &st})); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if sw.lastOpts.Status != database.ExecutionStatusFailed {
		t.Fatalf("expected failed status threaded, got %q", sw.lastOpts.Status)
	}
}
