package containment

import (
	"errors"
	"testing"
)

func TestContainmentFailure_Error(t *testing.T) {
	f := ContainmentFailure{
		Message: "test error message",
	}
	if f.Error() != "test error message" {
		t.Errorf("Error() = %q, want %q", f.Error(), "test error message")
	}
}

func TestContainmentFailure_WithInternalDetails(t *testing.T) {
	original := ContainmentFailure{
		Message:         "test",
		InternalDetails: "original",
	}
	modified := original.WithInternalDetails("modified")

	if original.InternalDetails != "original" {
		t.Error("Original was mutated")
	}
	if modified.InternalDetails != "modified" {
		t.Errorf("Modified has wrong details: %s", modified.InternalDetails)
	}
}

func TestNewDockerBinaryNotFoundFailure(t *testing.T) {
	f := NewDockerBinaryNotFoundFailure()

	if f.Code != FailureCodeDockerBinaryNotFound {
		t.Errorf("Code = %v, want %v", f.Code, FailureCodeDockerBinaryNotFound)
	}
	if f.Provider != ContainmentTypeDocker {
		t.Errorf("Provider = %v, want %v", f.Provider, ContainmentTypeDocker)
	}
	if !f.IsDegradable {
		t.Error("Expected IsDegradable to be true")
	}
	if f.RecoveryHint == "" {
		t.Error("Expected non-empty RecoveryHint")
	}
}

func TestNewDockerDaemonNotRunningFailure(t *testing.T) {
	f := NewDockerDaemonNotRunningFailure()

	if f.Code != FailureCodeDockerDaemonNotRunning {
		t.Errorf("Code = %v, want %v", f.Code, FailureCodeDockerDaemonNotRunning)
	}
	if !f.IsDegradable {
		t.Error("Expected IsDegradable to be true")
	}
}

func TestNewDockerTimeoutFailure(t *testing.T) {
	f := NewDockerTimeoutFailure(5)

	if f.Code != FailureCodeDockerTimeout {
		t.Errorf("Code = %v, want %v", f.Code, FailureCodeDockerTimeout)
	}
	if f.InternalDetails == "" {
		t.Error("Expected non-empty InternalDetails")
	}
	// Should mention the timeout value
	if !containsString(f.InternalDetails, "5") {
		t.Error("InternalDetails should mention timeout value")
	}
}

func TestNewNoContainmentAvailableFailure(t *testing.T) {
	checked := []ProviderCheckResult{
		{Type: ContainmentTypeDocker, Available: false, Reason: "daemon not running"},
		{Type: ContainmentTypeBubblewrap, Available: false, Reason: "not installed"},
	}

	f := NewNoContainmentAvailableFailure(checked)

	if f.Code != FailureCodeNoContainmentAvailable {
		t.Errorf("Code = %v, want %v", f.Code, FailureCodeNoContainmentAvailable)
	}
	if f.InternalDetails == "" {
		t.Error("Expected non-empty InternalDetails with provider details")
	}
	// Should mention both checked providers
	if !containsString(f.InternalDetails, "docker") {
		t.Error("InternalDetails should mention docker")
	}
	if !containsString(f.InternalDetails, "bubblewrap") {
		t.Error("InternalDetails should mention bubblewrap")
	}
}

func TestNewPrepareCommandFailure(t *testing.T) {
	err := errors.New("container creation failed: image not found")
	f := NewPrepareCommandFailure(ContainmentTypeDocker, err)

	if f.Code != FailureCodePrepareCommandFailed {
		t.Errorf("Code = %v, want %v", f.Code, FailureCodePrepareCommandFailed)
	}
	if f.Provider != ContainmentTypeDocker {
		t.Errorf("Provider = %v, want %v", f.Provider, ContainmentTypeDocker)
	}
	if f.InternalDetails != "container creation failed: image not found" {
		t.Errorf("InternalDetails = %q, want original error message", f.InternalDetails)
	}
}

func TestDecideOnContainmentFailure(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		failure       ContainmentFailure
		wantContinue  bool
		wantHasWarning bool
	}{
		{
			name: "AllowFallback=true with degradable failure continues",
			config: Config{
				AllowFallback: true,
			},
			failure: ContainmentFailure{
				Code:         FailureCodeDockerBinaryNotFound,
				IsDegradable: true,
			},
			wantContinue:   true,
			wantHasWarning: true,
		},
		{
			name: "AllowFallback=false stops",
			config: Config{
				AllowFallback: false,
			},
			failure: ContainmentFailure{
				Code:         FailureCodeDockerBinaryNotFound,
				IsDegradable: true,
			},
			wantContinue:   false,
			wantHasWarning: false,
		},
		{
			name: "Non-degradable failure stops even with AllowFallback=true",
			config: Config{
				AllowFallback: true,
			},
			failure: ContainmentFailure{
				Code:         FailureCodePrepareCommandFailed,
				IsDegradable: false,
			},
			wantContinue:   false,
			wantHasWarning: false,
		},
		{
			name: "Docker daemon not running continues with warning",
			config: Config{
				AllowFallback: true,
			},
			failure:        NewDockerDaemonNotRunningFailure(),
			wantContinue:   true,
			wantHasWarning: true,
		},
		{
			name: "Docker timeout continues with warning",
			config: Config{
				AllowFallback: true,
			},
			failure:        NewDockerTimeoutFailure(5),
			wantContinue:   true,
			wantHasWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := DecideOnContainmentFailure(tt.config, tt.failure)

			if decision.ShouldContinue != tt.wantContinue {
				t.Errorf("ShouldContinue = %v, want %v", decision.ShouldContinue, tt.wantContinue)
			}

			hasWarning := decision.WarningForUser != ""
			if hasWarning != tt.wantHasWarning {
				t.Errorf("HasWarning = %v, want %v", hasWarning, tt.wantHasWarning)
			}

			if decision.Reason == "" {
				t.Error("Expected non-empty Reason")
			}

			if decision.OriginalFailure == nil {
				t.Error("Expected OriginalFailure to be set")
			}
		})
	}
}

func TestContainmentFailure_ToLogFields(t *testing.T) {
	f := ContainmentFailure{
		Code:            FailureCodeDockerBinaryNotFound,
		Provider:        ContainmentTypeDocker,
		Message:         "Docker not found",
		IsDegradable:    true,
		InternalDetails: "should not appear in log fields directly",
	}

	fields := f.ToLogFields()

	if fields["containment_failure_code"] != "docker_binary_not_found" {
		t.Errorf("code = %v, want docker_binary_not_found", fields["containment_failure_code"])
	}
	if fields["containment_provider"] != "docker" {
		t.Errorf("provider = %v, want docker", fields["containment_provider"])
	}
	if fields["containment_failure_degradable"] != true {
		t.Errorf("degradable = %v, want true", fields["containment_failure_degradable"])
	}
	// InternalDetails should not be in log fields (keep logs cleaner)
	if _, ok := fields["internalDetails"]; ok {
		t.Error("log fields should not include internalDetails directly")
	}
}

func TestContainmentFailure_ToAPIResponse(t *testing.T) {
	f := ContainmentFailure{
		Code:            FailureCodeDockerDaemonNotRunning,
		Provider:        ContainmentTypeDocker,
		Message:         "Docker daemon not running",
		RecoveryHint:    "Start Docker",
		InternalDetails: "secret internal error",
	}

	resp := f.ToAPIResponse()

	if resp["code"] != "docker_daemon_not_running" {
		t.Errorf("code = %v, want docker_daemon_not_running", resp["code"])
	}
	if resp["message"] != "Docker daemon not running" {
		t.Errorf("message = %v", resp["message"])
	}
	if resp["recoveryHint"] != "Start Docker" {
		t.Errorf("recoveryHint = %v", resp["recoveryHint"])
	}
	// Internal details should NEVER be in API response
	if _, ok := resp["internalDetails"]; ok {
		t.Error("API response should not include internalDetails")
	}
}

func TestDecideOnContainmentFailure_WarningMessages(t *testing.T) {
	config := Config{AllowFallback: true}

	t.Run("Docker binary not found mentions install", func(t *testing.T) {
		decision := DecideOnContainmentFailure(config, NewDockerBinaryNotFoundFailure())
		if !containsString(decision.WarningForUser, "Install") {
			t.Errorf("Warning should mention Install: %q", decision.WarningForUser)
		}
	})

	t.Run("Docker daemon not running mentions start", func(t *testing.T) {
		decision := DecideOnContainmentFailure(config, NewDockerDaemonNotRunningFailure())
		if !containsString(decision.WarningForUser, "Start") {
			t.Errorf("Warning should mention Start: %q", decision.WarningForUser)
		}
	})

	t.Run("Docker timeout mentions timeout", func(t *testing.T) {
		decision := DecideOnContainmentFailure(config, NewDockerTimeoutFailure(5))
		if !containsString(decision.WarningForUser, "timeout") {
			t.Errorf("Warning should mention timeout: %q", decision.WarningForUser)
		}
	})
}

// Uses containsString from provider_test.go
