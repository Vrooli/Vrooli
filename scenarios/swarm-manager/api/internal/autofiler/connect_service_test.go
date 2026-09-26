package autofiler

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/settings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestConnectServiceGetStatus(t *testing.T) {
	cfg := settings.DefaultSettings().AutoFiler
	cfg.Enabled = true
	cfg.MaxOpenAutoFiled = 7
	sw := NewSweeper(fakeAutoFilerSettings{cfg: cfg}, fakeBacklogReader{}, fakeTransitionCounter{}, nil, nil)
	sw.remember(SweepResult{
		Enabled:          true,
		Strategy:         StrategyFeaturePending,
		Mode:             ModeSuggest,
		Candidates:       2,
		Findings:         3,
		Created:          1,
		SkippedDismissed: 1,
		ReconciledClosed: 1,
		ReconciledNoted:  1,
		Brake:            BrakeState{WindowDays: 7, Minimum: 1, Observed: 0, Braked: true},
	})
	dismissals := NewDismissalStorePath(filepath.Join(t.TempDir(), "dismissed.json"))
	if err := dismissals.Remember("gct:alpha:docs", "fix/a", "done"); err != nil {
		t.Fatalf("Remember: %v", err)
	}

	svc := NewConnectService(fakeAutoFilerSettings{cfg: cfg}, sw, nil, nil, dismissals)
	resp, err := svc.GetStatus(context.Background(), connect.NewRequest(&apipb.AutoFilerStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if !resp.Msg.GetEnabled() || resp.Msg.GetCreated() != 1 || resp.Msg.GetDismissalCount() != 1 {
		t.Fatalf("status response = %+v", resp.Msg)
	}
	if !resp.Msg.GetBrake().GetBraked() {
		t.Fatalf("brake = %+v, want braked", resp.Msg.GetBrake())
	}
}

func TestConnectServiceRunNow(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	backlogSvc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	cfg := settings.DefaultSettings().AutoFiler
	cfg.Enabled = true
	cfg.MinVelocityTransitions = 0
	sw := NewSweeper(
		fakeAutoFilerSettings{cfg: cfg},
		store,
		fakeTransitionCounter{},
		NewFiler(backlogSvc, &fakeGoals{}),
		&staticFindings{findings: []Finding{{ID: "gct:alpha:tests", Scenario: "alpha", Dimension: "tests"}}},
	)
	sw.Feature = staticTargets{targets: []Target{{Scenario: "alpha"}}}

	svc := NewConnectService(fakeAutoFilerSettings{cfg: cfg}, sw, nil, nil, nil)
	resp, err := svc.RunNow(context.Background(), connect.NewRequest(&apipb.AutoFilerRunNowRequest{}))
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	if resp.Msg.GetCreated() != 1 || resp.Msg.GetFindings() != 1 || resp.Msg.GetLastCycleTime() == "" {
		t.Fatalf("run-now response = %+v, want created cycle status", resp.Msg)
	}
}

func TestConnectServiceDismissSuggestion(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	backlogSvc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	filed, err := NewFiler(backlogSvc, &fakeGoals{}).File(context.Background(), Finding{
		ID:        "gct:alpha:docs",
		Scenario:  "alpha",
		Dimension: "docs",
	}, FileOptions{Mode: ModeSuggest, Strategy: StrategyFeaturePending, GoalName: "automated-maintenance"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	dismissals := NewDismissalStorePath(filepath.Join(t.TempDir(), "dismissed.json"))
	svc := NewConnectService(nil, nil, store, backlogSvc, dismissals)

	req := &apipb.DismissAutoFilerSuggestionRequest{
		Kind:   string(filed.Item.Kind),
		Name:   filed.Item.Name,
		Reason: strPtr("operator dismissed"),
	}
	resp, err := svc.DismissSuggestion(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("DismissSuggestion: %v", err)
	}
	if !resp.Msg.GetDismissed() || resp.Msg.GetItem().GetArchivedAt() == "" {
		t.Fatalf("dismiss response = %+v", resp.Msg)
	}
	if dismissed, err := dismissals.IsDismissed("gct:alpha:docs"); err != nil || !dismissed {
		t.Fatalf("dismissal memory dismissed=%v err=%v", dismissed, err)
	}

	resp, err = svc.DismissSuggestion(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("DismissSuggestion second: %v", err)
	}
	if !resp.Msg.GetDismissed() {
		t.Fatalf("second dismiss response = %+v", resp.Msg)
	}
}

func strPtr(value string) *string {
	return &value
}
