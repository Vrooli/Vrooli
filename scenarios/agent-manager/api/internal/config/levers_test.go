package config_test

import (
	"testing"
	"time"

	"agent-manager/internal/config"
)

// Tests for configuration lever validation

func TestDefaultLevers_Valid(t *testing.T) {
	levers := config.DefaultLevers()
	if err := levers.Validate(); err != nil {
		t.Errorf("default levers should be valid: %v", err)
	}
}

func TestExecutionLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.ExecutionLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(e *config.ExecutionLevers) {},
			wantErr: false,
		},
		{
			name:    "timeout too short",
			modify:  func(e *config.ExecutionLevers) { e.DefaultTimeout = 30 * time.Second },
			wantErr: true,
		},
		{
			name:    "timeout too long",
			modify:  func(e *config.ExecutionLevers) { e.DefaultTimeout = 5 * time.Hour },
			wantErr: true,
		},
		{
			name:    "max turns too low",
			modify:  func(e *config.ExecutionLevers) { e.DefaultMaxTurns = 0 },
			wantErr: true,
		},
		{
			name:    "max turns too high",
			modify:  func(e *config.ExecutionLevers) { e.DefaultMaxTurns = 1001 },
			wantErr: true,
		},
		{
			name:    "buffer size too small",
			modify:  func(e *config.ExecutionLevers) { e.EventBufferSize = 5 },
			wantErr: true,
		},
		{
			name:    "buffer size too large",
			modify:  func(e *config.ExecutionLevers) { e.EventBufferSize = 10001 },
			wantErr: true,
		},
		{
			name:    "flush interval too short",
			modify:  func(e *config.ExecutionLevers) { e.EventFlushInterval = 50 * time.Millisecond },
			wantErr: true,
		},
		{
			name:    "flush interval too long",
			modify:  func(e *config.ExecutionLevers) { e.EventFlushInterval = 60 * time.Second },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Execution)
			err := levers.Execution.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafetyLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.SafetyLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(s *config.SafetyLevers) {},
			wantErr: false,
		},
		{
			name:    "max files too low",
			modify:  func(s *config.SafetyLevers) { s.MaxFilesPerRun = 0 },
			wantErr: true,
		},
		{
			name:    "max files too high",
			modify:  func(s *config.SafetyLevers) { s.MaxFilesPerRun = 10001 },
			wantErr: true,
		},
		{
			name:    "max bytes too low",
			modify:  func(s *config.SafetyLevers) { s.MaxBytesPerRun = 512 },
			wantErr: true,
		},
		{
			name:    "max bytes too high",
			modify:  func(s *config.SafetyLevers) { s.MaxBytesPerRun = 2 * 1024 * 1024 * 1024 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Safety)
			err := levers.Safety.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConcurrencyLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.ConcurrencyLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(c *config.ConcurrencyLevers) {},
			wantErr: false,
		},
		{
			name:    "max runs too low",
			modify:  func(c *config.ConcurrencyLevers) { c.MaxConcurrentRuns = 0 },
			wantErr: true,
		},
		{
			name:    "max runs too high",
			modify:  func(c *config.ConcurrencyLevers) { c.MaxConcurrentRuns = 101 },
			wantErr: true,
		},
		{
			name:    "per scope too low",
			modify:  func(c *config.ConcurrencyLevers) { c.MaxConcurrentPerScope = 0 },
			wantErr: true,
		},
		{
			name:    "per scope too high",
			modify:  func(c *config.ConcurrencyLevers) { c.MaxConcurrentPerScope = 11 },
			wantErr: true,
		},
		{
			name:    "lock TTL too short",
			modify:  func(c *config.ConcurrencyLevers) { c.ScopeLockTTL = 4 * time.Minute },
			wantErr: true,
		},
		{
			name:    "lock TTL too long",
			modify:  func(c *config.ConcurrencyLevers) { c.ScopeLockTTL = 25 * time.Hour },
			wantErr: true,
		},
		{
			name:    "refresh too short",
			modify:  func(c *config.ConcurrencyLevers) { c.ScopeLockRefreshInterval = 20 * time.Second },
			wantErr: true,
		},
		{
			name:    "refresh too long",
			modify:  func(c *config.ConcurrencyLevers) { c.ScopeLockRefreshInterval = 11 * time.Minute },
			wantErr: true,
		},
		{
			name:    "queue timeout negative",
			modify:  func(c *config.ConcurrencyLevers) { c.QueueWaitTimeout = -1 * time.Second },
			wantErr: true,
		},
		{
			name:    "queue timeout too long",
			modify:  func(c *config.ConcurrencyLevers) { c.QueueWaitTimeout = 31 * time.Minute },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Concurrency)
			err := levers.Concurrency.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApprovalLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.ApprovalLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(a *config.ApprovalLevers) {},
			wantErr: false,
		},
		{
			name:    "timeout days too low",
			modify:  func(a *config.ApprovalLevers) { a.ReviewTimeoutDays = 0 },
			wantErr: true,
		},
		{
			name:    "timeout days too high",
			modify:  func(a *config.ApprovalLevers) { a.ReviewTimeoutDays = 91 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Approval)
			err := levers.Approval.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.RunnerLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(r *config.RunnerLevers) {},
			wantErr: false,
		},
		{
			name:    "health check too short",
			modify:  func(r *config.RunnerLevers) { r.HealthCheckInterval = 5 * time.Second },
			wantErr: true,
		},
		{
			name:    "health check too long",
			modify:  func(r *config.RunnerLevers) { r.HealthCheckInterval = 6 * time.Minute },
			wantErr: true,
		},
		{
			name:    "grace period negative",
			modify:  func(r *config.RunnerLevers) { r.StartupGracePeriod = -1 * time.Second },
			wantErr: true,
		},
		{
			name:    "grace period too long",
			modify:  func(r *config.RunnerLevers) { r.StartupGracePeriod = 6 * time.Minute },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Runners)
			err := levers.Runners.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.ServerLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(s *config.ServerLevers) {},
			wantErr: false,
		},
		{
			name:    "empty port",
			modify:  func(s *config.ServerLevers) { s.Port = "" },
			wantErr: true,
		},
		{
			name:    "read timeout too short",
			modify:  func(s *config.ServerLevers) { s.ReadTimeout = 4 * time.Second },
			wantErr: true,
		},
		{
			name:    "read timeout too long",
			modify:  func(s *config.ServerLevers) { s.ReadTimeout = 6 * time.Minute },
			wantErr: true,
		},
		{
			name:    "write timeout too short",
			modify:  func(s *config.ServerLevers) { s.WriteTimeout = 4 * time.Second },
			wantErr: true,
		},
		{
			name:    "write timeout too long",
			modify:  func(s *config.ServerLevers) { s.WriteTimeout = 11 * time.Minute },
			wantErr: true,
		},
		{
			name:    "idle timeout too short",
			modify:  func(s *config.ServerLevers) { s.IdleTimeout = 20 * time.Second },
			wantErr: true,
		},
		{
			name:    "idle timeout too long",
			modify:  func(s *config.ServerLevers) { s.IdleTimeout = 11 * time.Minute },
			wantErr: true,
		},
		{
			name:    "request body too small",
			modify:  func(s *config.ServerLevers) { s.MaxRequestBodyBytes = 512 },
			wantErr: true,
		},
		{
			name:    "request body too large",
			modify:  func(s *config.ServerLevers) { s.MaxRequestBodyBytes = 200 * 1024 * 1024 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Server)
			err := levers.Server.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStorageLevers_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.StorageLevers)
		wantErr bool
	}{
		{
			name:    "valid defaults",
			modify:  func(s *config.StorageLevers) {},
			wantErr: false,
		},
		{
			name:    "open conns too low",
			modify:  func(s *config.StorageLevers) { s.MaxOpenConns = 4 },
			wantErr: true,
		},
		{
			name:    "open conns too high",
			modify:  func(s *config.StorageLevers) { s.MaxOpenConns = 101 },
			wantErr: true,
		},
		{
			name:    "idle conns too low",
			modify:  func(s *config.StorageLevers) { s.MaxIdleConns = 0 },
			wantErr: true,
		},
		{
			name:    "idle conns too high",
			modify:  func(s *config.StorageLevers) { s.MaxIdleConns = 51 },
			wantErr: true,
		},
		{
			name:    "conn lifetime too short",
			modify:  func(s *config.StorageLevers) { s.ConnMaxLifetime = 30 * time.Second },
			wantErr: true,
		},
		{
			name:    "conn lifetime too long",
			modify:  func(s *config.StorageLevers) { s.ConnMaxLifetime = 2 * time.Hour },
			wantErr: true,
		},
		{
			name:    "event retention too low",
			modify:  func(s *config.StorageLevers) { s.EventRetentionDays = 0 },
			wantErr: true,
		},
		{
			name:    "event retention too high",
			modify:  func(s *config.StorageLevers) { s.EventRetentionDays = 366 },
			wantErr: true,
		},
		{
			name:    "artifact retention too low",
			modify:  func(s *config.StorageLevers) { s.ArtifactRetentionDays = 0 },
			wantErr: true,
		},
		{
			name:    "artifact retention too high",
			modify:  func(s *config.StorageLevers) { s.ArtifactRetentionDays = 366 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			levers := config.DefaultLevers()
			tt.modify(&levers.Storage)
			err := levers.Storage.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLevers_Validate_PropagatesErrors(t *testing.T) {
	// Test that main Validate() catches errors from each sub-lever
	levers := config.DefaultLevers()
	levers.Execution.DefaultMaxTurns = 0 // Invalid
	err := levers.Validate()
	if err == nil {
		t.Error("expected validation error")
	}
}

func TestDefaultLevers_SafetyDefaults(t *testing.T) {
	levers := config.DefaultLevers()

	// Verify safety-first defaults
	if !levers.Safety.RequireSandboxByDefault {
		t.Error("sandbox should be required by default")
	}
	if !levers.Approval.RequireApprovalByDefault {
		t.Error("approval should be required by default")
	}
	if levers.Concurrency.MaxConcurrentPerScope != 1 {
		t.Error("should default to exclusive scope locks")
	}

	// Verify deny patterns include dangerous paths
	denyPatterns := levers.Safety.DenyPathPatterns
	expectedPatterns := []string{".git/**", ".env*"}
	for _, expected := range expectedPatterns {
		found := false
		for _, pattern := range denyPatterns {
			if pattern == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected deny pattern %s not found", expected)
		}
	}
}
