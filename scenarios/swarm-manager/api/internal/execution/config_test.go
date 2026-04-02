package execution

import (
	"testing"
	"time"
)

func TestDefaultFinalizationConfig(t *testing.T) {
	cfg := DefaultFinalizationConfig()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"HealthPollInterval", cfg.HealthPollInterval, 5 * time.Second},
		{"HealthPollTimeout", cfg.HealthPollTimeout, 2 * time.Minute},
		{"ReviewPollInterval", cfg.ReviewPollInterval, 5 * time.Second},
		{"ReviewPollTimeout", cfg.ReviewPollTimeout, 10 * time.Minute},
		{"MaxRestartAttempts", cfg.MaxRestartAttempts, 2},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}

func TestNewServiceAppliesDefaultFinalizationConfig(t *testing.T) {
	svc := NewService(ServiceConfig{
		RootDir:   t.TempDir(),
		StorePath: t.TempDir() + "/exec.json",
	})

	want := DefaultFinalizationConfig()
	got := svc.finalizationCfg

	if got != want {
		t.Errorf("NewService with zero FinalizationConfig should apply defaults\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestNewServiceRespectsCustomFinalizationConfig(t *testing.T) {
	custom := FinalizationConfig{
		HealthPollInterval: 1 * time.Second,
		HealthPollTimeout:  30 * time.Second,
		ReviewPollInterval: 2 * time.Second,
		ReviewPollTimeout:  1 * time.Minute,
		MaxRestartAttempts: 5,
	}

	svc := NewService(ServiceConfig{
		RootDir:      t.TempDir(),
		StorePath:    t.TempDir() + "/exec.json",
		Finalization: custom,
	})

	if svc.finalizationCfg != custom {
		t.Errorf("NewService should use custom FinalizationConfig\ngot:  %+v\nwant: %+v", svc.finalizationCfg, custom)
	}
}
