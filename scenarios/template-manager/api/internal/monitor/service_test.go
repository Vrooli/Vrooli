package monitor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"
)

func TestRunDuePersistsSchedulerAttributedDeepValidation(t *testing.T) {
	repo := newFakeRepo()
	validator := &fakeValidator{}
	service := NewService(repo, validator, Config{
		Enabled:     true,
		Interval:    time.Minute,
		TemplateIDs: []string{"react-vite"},
	}, nil)
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	status, err := service.RunDue(context.Background())
	if err != nil {
		t.Fatalf("RunDue: %v", err)
	}

	if validator.request.Mode != catalog.ModeDeep {
		t.Fatalf("mode = %q, want deep", validator.request.Mode)
	}
	if validator.request.Trigger != schedulerSource {
		t.Fatalf("trigger = %q, want %q", validator.request.Trigger, schedulerSource)
	}
	if status.InFlight {
		t.Fatal("status should not remain in-flight after RunDue")
	}
	if status.LastRunID != "validation-test" {
		t.Fatalf("last run = %q, want validation-test", status.LastRunID)
	}
	if status.GreenStreak != 1 {
		t.Fatalf("green streak = %d, want 1", status.GreenStreak)
	}
	if !status.NextRunAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("next run = %s, want %s", status.NextRunAt, now.Add(time.Minute))
	}
}

func TestRunDueSkipsWhenAlreadyRunning(t *testing.T) {
	repo := newFakeRepo()
	validator := &fakeValidator{block: make(chan struct{})}
	service := NewService(repo, validator, Config{
		Enabled:     true,
		Interval:    time.Minute,
		TemplateIDs: []string{"react-vite"},
	}, nil)

	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		validator.started = started
		_, err := service.RunDue(context.Background())
		done <- err
	}()
	<-started

	status, err := service.RunDue(context.Background())
	if err != nil {
		t.Fatalf("second RunDue: %v", err)
	}
	if status.LastStatus != "skipped_busy" {
		t.Fatalf("second status = %q, want skipped_busy", status.LastStatus)
	}

	close(validator.block)
	if err := <-done; err != nil {
		t.Fatalf("first RunDue: %v", err)
	}
}

type fakeRepo struct {
	mu     sync.Mutex
	status catalog.MonitorStatus
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		status: catalog.MonitorStatus{
			ID:              "default",
			Enabled:         true,
			IntervalSeconds: 60,
			LastStatus:      "scheduled",
			NextRunAt:       time.Date(2026, 7, 9, 12, 1, 0, 0, time.UTC),
		},
	}
}

func (r *fakeRepo) GetMonitorStatus(context.Context) (catalog.MonitorStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, nil
}

func (r *fakeRepo) SaveMonitorStatus(_ context.Context, status catalog.MonitorStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
	return nil
}

func (r *fakeRepo) ListTemplates(context.Context, catalog.TemplateKind) ([]catalog.TemplateRecord, error) {
	return []catalog.TemplateRecord{{ID: "react-vite", Kind: catalog.KindScenario, Status: "active"}}, nil
}

func (r *fakeRepo) DeepValidateGreenStreak(context.Context, string) (int64, error) {
	return 1, nil
}

type fakeValidator struct {
	request validationrunner.ValidateRequest
	started chan struct{}
	block   chan struct{}
}

func (v *fakeValidator) RunValidation(_ context.Context, req validationrunner.ValidateRequest) (catalog.ValidationRun, error) {
	v.request = req
	if v.started != nil {
		close(v.started)
	}
	if v.block != nil {
		<-v.block
	}
	return catalog.ValidationRun{
		ID:         "validation-test",
		TemplateID: req.TemplateID,
		Mode:       req.Mode,
		Status:     "passed",
		Trigger:    req.Trigger,
	}, nil
}
