package services

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

func TestRetentionSchedulerStopContextAndIdempotentStart(t *testing.T) {
	settings := NewSettingsManager(WithSettingsConfigStore(NewMemoryConfigStore()))
	next := settings.GetSettings()
	next.RetentionRunOnStartup = false
	if err := settings.UpdateSettings(next); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	maint := NewMetricsMaintenanceService(memory.NewRepository())
	scheduler := NewRetentionScheduler(maint, settings, nil)
	scheduler.Start()
	scheduler.Start() // A duplicate start must not create a second loop.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := scheduler.StopContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("StopContext error = %v, want canceled context", err)
	}
	if err := scheduler.StopContext(context.Background()); err != nil {
		t.Fatalf("join scheduler after canceled wait: %v", err)
	}
	scheduler.Start()
	if err := scheduler.StopContext(context.Background()); err != nil {
		t.Fatalf("stop restarted scheduler: %v", err)
	}
}
