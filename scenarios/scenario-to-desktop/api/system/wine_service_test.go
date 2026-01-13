package system

import (
	"log/slog"
	"testing"
)

func TestNewWineService(t *testing.T) {
	logger := slog.Default()
	s := NewWineService(logger)

	if s == nil {
		t.Fatal("expected service to be created")
	}
	if s.installs == nil {
		t.Error("expected installs map to be initialized")
	}
	if s.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestWineServiceCheckStatus(t *testing.T) {
	s := NewWineService(slog.Default())
	resp := s.CheckStatus()

	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Platform == "" {
		t.Error("expected platform to be set")
	}
	// RequiredFor should include "win"
	found := false
	for _, r := range resp.RequiredFor {
		if r == "win" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected RequiredFor to include 'win'")
	}
}

func TestWineServiceStartInstallation(t *testing.T) {
	s := NewWineService(slog.Default())

	tests := []struct {
		name        string
		method      string
		expectError bool
	}{
		{"flatpak", "flatpak", false},
		{"flatpak-auto", "flatpak-auto", false},
		{"appimage", "appimage", false},
		{"skip", "skip", false},
		{"invalid", "invalid", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			installID, err := s.StartInstallation(tc.method)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if installID != "" {
					t.Error("expected empty installID on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if installID == "" {
					t.Error("expected installID to be set")
				}

				// Verify we can get the status
				status, exists := s.GetInstallStatus(installID)
				if !exists {
					t.Error("expected status to exist")
				}
				if status == nil {
					t.Error("expected status to be non-nil")
				}
				if status.Method != tc.method {
					t.Errorf("method = %q, want %q", status.Method, tc.method)
				}
			}
		})
	}
}

func TestWineServiceGetInstallStatus(t *testing.T) {
	s := NewWineService(slog.Default())

	t.Run("non-existent", func(t *testing.T) {
		status, exists := s.GetInstallStatus("nonexistent")
		if exists {
			t.Error("expected exists to be false")
		}
		if status != nil {
			t.Error("expected status to be nil")
		}
	})

	t.Run("existing", func(t *testing.T) {
		installID, _ := s.StartInstallation("skip")
		status, exists := s.GetInstallStatus(installID)
		if !exists {
			t.Error("expected exists to be true")
		}
		if status == nil {
			t.Fatal("expected status to be non-nil")
		}
		if status.InstallID != installID {
			t.Errorf("InstallID = %q, want %q", status.InstallID, installID)
		}
	})
}

func TestWineServiceSkipMethod(t *testing.T) {
	s := NewWineService(slog.Default())

	installID, err := s.StartInstallation("skip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Skip should complete immediately
	status, _ := s.GetInstallStatus(installID)
	if status.Status != "completed" {
		t.Errorf("skip status = %q, want 'completed'", status.Status)
	}
	if status.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
	if len(status.Log) == 0 {
		t.Error("expected log to have entries")
	}
}

func TestWineServiceAddLog(t *testing.T) {
	s := NewWineService(slog.Default())

	installID, _ := s.StartInstallation("skip")
	s.addLog(installID, "test message")

	status, _ := s.GetInstallStatus(installID)
	found := false
	for _, log := range status.Log {
		if log == "test message" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log message to be added")
	}
}

func TestWineServiceAddError(t *testing.T) {
	s := NewWineService(slog.Default())

	installID, _ := s.StartInstallation("skip")
	s.addError(installID, "test error")

	status, _ := s.GetInstallStatus(installID)
	found := false
	for _, log := range status.ErrorLog {
		if log == "test error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error message to be added")
	}
}

func TestWineServiceSetStatus(t *testing.T) {
	s := NewWineService(slog.Default())

	installID, _ := s.StartInstallation("skip")

	// Reset status to test setStatus
	s.setStatus(installID, "installing")
	status, _ := s.GetInstallStatus(installID)
	if status.Status != "installing" {
		t.Errorf("status = %q, want 'installing'", status.Status)
	}

	// Set to completed - should set CompletedAt
	s.setStatus(installID, "completed")
	status, _ = s.GetInstallStatus(installID)
	if status.Status != "completed" {
		t.Errorf("status = %q, want 'completed'", status.Status)
	}
	if status.CompletedAt == nil {
		t.Error("expected CompletedAt to be set on completion")
	}

	// Set to failed - should also set CompletedAt
	installID2, _ := s.StartInstallation("skip")
	s.setStatus(installID2, "failed")
	status2, _ := s.GetInstallStatus(installID2)
	if status2.Status != "failed" {
		t.Errorf("status = %q, want 'failed'", status2.Status)
	}
	if status2.CompletedAt == nil {
		t.Error("expected CompletedAt to be set on failure")
	}
}

func TestWineServiceLogToNonExistent(t *testing.T) {
	s := NewWineService(slog.Default())

	// These should not panic when called with non-existent IDs
	s.addLog("nonexistent", "test")
	s.addError("nonexistent", "test")
	s.setStatus("nonexistent", "test")
}
