// Package infra tests for DNS health check with mock resolver
// [REQ:INFRA-DNS-001] [REQ:TEST-SEAM-001] [REQ:HEAL-ACTION-001]
package infra

import (
	"context"
	"testing"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// =============================================================================
// DNSCheck Unit Tests with Mock DNS Resolver
// =============================================================================

// TestDNSCheckRunWithMock_Successful tests successful DNS resolution
// [REQ:INFRA-DNS-001] [REQ:TEST-SEAM-001]
func TestDNSCheckRunWithMock_Successful(t *testing.T) {
	mockResolver := checks.NewMockDNSResolver()
	mockResolver.Responses["google.com"] = []string{"142.250.80.46", "142.250.80.78"}

	check := NewDNSCheck("google.com", testCaps(), WithDNSResolver(mockResolver))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Message != "DNS resolution OK" {
		t.Errorf("Message = %q, want %q", result.Message, "DNS resolution OK")
	}
	if result.Details["resolved"] != 2 {
		t.Errorf("Details[resolved] = %v, want 2", result.Details["resolved"])
	}

	// Verify mock was called
	if len(mockResolver.Calls) != 1 {
		t.Errorf("Expected 1 DNS call, got %d", len(mockResolver.Calls))
	}
	if len(mockResolver.Calls) > 0 && mockResolver.Calls[0] != "google.com" {
		t.Errorf("DNS query = %q, want %q", mockResolver.Calls[0], "google.com")
	}
}

// TestDNSCheckRunWithMock_Failed tests failed DNS resolution
func TestDNSCheckRunWithMock_Failed(t *testing.T) {
	mockResolver := checks.NewMockDNSResolver()
	mockResolver.Errors["nonexistent.invalid"] = checks.ErrDNSLookupFailed

	check := NewDNSCheck("nonexistent.invalid", testCaps(), WithDNSResolver(mockResolver))
	result := check.Run(context.Background())

	if result.Status != checks.StatusCritical {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusCritical)
	}
	if result.Message != "DNS resolution failed" {
		t.Errorf("Message = %q, want %q", result.Message, "DNS resolution failed")
	}
	if result.Details["error"] == nil {
		t.Error("Details should contain error")
	}
}

// TestDNSCheckRunWithMock_MultipleAddresses tests resolution returning multiple IPs
func TestDNSCheckRunWithMock_MultipleAddresses(t *testing.T) {
	mockResolver := checks.NewMockDNSResolver()
	mockResolver.Responses["cdn.example.com"] = []string{
		"192.168.1.1", "192.168.1.2", "192.168.1.3",
		"192.168.1.4", "192.168.1.5",
	}

	check := NewDNSCheck("cdn.example.com", testCaps(), WithDNSResolver(mockResolver))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, checks.StatusOK)
	}
	if result.Details["resolved"] != 5 {
		t.Errorf("Details[resolved] = %v, want 5", result.Details["resolved"])
	}
}

// TestDNSCheckRunWithMock_DifferentDomains tests various domain resolutions
func TestDNSCheckRunWithMock_DifferentDomains(t *testing.T) {
	tests := []struct {
		name           string
		domain         string
		addresses      []string
		err            error
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name:           "google.com resolves",
			domain:         "google.com",
			addresses:      []string{"142.250.80.46"},
			err:            nil,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "DNS resolution OK",
		},
		{
			name:           "localhost resolves",
			domain:         "localhost",
			addresses:      []string{"127.0.0.1"},
			err:            nil,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "DNS resolution OK",
		},
		{
			name:           "nonexistent domain fails",
			domain:         "does.not.exist.example",
			addresses:      nil,
			err:            checks.ErrDNSLookupFailed,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "DNS resolution failed",
		},
		{
			name:           "timeout error",
			domain:         "slow.example.com",
			addresses:      nil,
			err:            checks.ErrTimeout,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "DNS resolution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := checks.NewMockDNSResolver()
			if tt.err != nil {
				mockResolver.Errors[tt.domain] = tt.err
			} else {
				mockResolver.Responses[tt.domain] = tt.addresses
			}

			check := NewDNSCheck(tt.domain, testCaps(), WithDNSResolver(mockResolver))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}
			if result.Details["domain"] != tt.domain {
				t.Errorf("Details[domain] = %v, want %q", result.Details["domain"], tt.domain)
			}
		})
	}
}

// TestDNSCheckRunWithMock_DefaultResolver tests that default resolver is used
func TestDNSCheckRunWithMock_DefaultResolver(t *testing.T) {
	mockResolver := checks.NewMockDNSResolver()
	mockResolver.DefaultAddresses = []string{"1.2.3.4"}

	check := NewDNSCheck("any.domain.com", testCaps(), WithDNSResolver(mockResolver))
	result := check.Run(context.Background())

	if result.Status != checks.StatusOK {
		t.Errorf("Status = %v, want %v (using default response)", result.Status, checks.StatusOK)
	}
}

// =============================================================================
// DNSCheck Recovery Action Tests with Mocks
// =============================================================================

// TestDNSCheckRecoveryActions tests recovery action availability
// [REQ:HEAL-ACTION-001]
func TestDNSCheckRecoveryActions(t *testing.T) {
	tests := []struct {
		name            string
		caps            *platform.Capabilities
		expectAvailable map[string]bool
	}{
		{
			name: "linux with systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: true,
			},
			expectAvailable: map[string]bool{
				"restart-resolved": true,
				"flush-cache":      true,
				"test-external":    true,
			},
		},
		{
			name: "linux without systemd",
			caps: &platform.Capabilities{
				Platform:        platform.Linux,
				SupportsSystemd: false,
			},
			expectAvailable: map[string]bool{
				"restart-resolved": false,
				"flush-cache":      false,
				"test-external":    true,
			},
		},
		{
			name: "macos",
			caps: &platform.Capabilities{
				Platform:        platform.MacOS,
				SupportsSystemd: false,
			},
			expectAvailable: map[string]bool{
				"restart-resolved": false,
				"flush-cache":      false,
				"test-external":    true,
			},
		},
		{
			name: "nil caps",
			caps: nil,
			expectAvailable: map[string]bool{
				"restart-resolved": false,
				"flush-cache":      false,
				"test-external":    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewDNSCheck("google.com", tt.caps)
			actions := check.RecoveryActions(nil)

			actionMap := make(map[string]checks.RecoveryAction)
			for _, a := range actions {
				actionMap[a.ID] = a
			}

			for id, expectAvail := range tt.expectAvailable {
				action, exists := actionMap[id]
				if !exists {
					t.Errorf("Action %q not found", id)
					continue
				}
				if action.Available != expectAvail {
					t.Errorf("Action %q.Available = %v, want %v", id, action.Available, expectAvail)
				}
			}
		})
	}
}

// TestDNSCheckExecuteAction_RestartResolved tests restart-resolved action
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDNSCheckExecuteAction_RestartResolved(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo systemctl restart systemd-resolved"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	mockResolver := checks.NewMockDNSResolver()
	mockResolver.DefaultAddresses = []string{"142.250.80.46"} // Success after restart

	check := NewDNSCheck("google.com", testCaps(),
		WithDNSExecutor(mockExec),
		WithDNSResolver(mockResolver),
	)

	result := check.ExecuteAction(context.Background(), "restart-resolved")

	if !result.Success {
		t.Errorf("Success = %v, want true (Error: %s)", result.Success, result.Error)
	}
	if result.ActionID != "restart-resolved" {
		t.Errorf("ActionID = %q, want %q", result.ActionID, "restart-resolved")
	}

	// Verify executor was called
	if len(mockExec.Calls) < 1 {
		t.Error("Expected executor to be called for restart")
	}
}

// TestDNSCheckExecuteAction_RestartResolved_Failed tests restart failure
func TestDNSCheckExecuteAction_RestartResolved_Failed(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo systemctl restart systemd-resolved"] = checks.MockResponse{
		Output: []byte("Failed to restart"),
		Error:  checks.ErrPermissionDenied,
	}

	check := NewDNSCheck("google.com", testCaps(), WithDNSExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "restart-resolved")

	if result.Success {
		t.Error("Success should be false for failed restart")
	}
	if result.Message != "Failed to restart systemd-resolved" {
		t.Errorf("Message = %q, want %q", result.Message, "Failed to restart systemd-resolved")
	}
}

// TestDNSCheckExecuteAction_FlushCache tests flush-cache action
func TestDNSCheckExecuteAction_FlushCache(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo resolvectl flush-caches"] = checks.MockResponse{
		Output: []byte(""),
		Error:  nil,
	}

	check := NewDNSCheck("google.com", testCaps(), WithDNSExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "flush-cache")

	if !result.Success {
		t.Errorf("Success = %v, want true", result.Success)
	}
	if result.Message != "Flushed DNS cache" {
		t.Errorf("Message = %q, want %q", result.Message, "Flushed DNS cache")
	}
}

// TestDNSCheckExecuteAction_FlushCache_Failed tests flush failure
func TestDNSCheckExecuteAction_FlushCache_Failed(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	mockExec.Responses["sudo resolvectl flush-caches"] = checks.MockResponse{
		Output: []byte("Command not found"),
		Error:  checks.ErrCommandNotFound,
	}

	check := NewDNSCheck("google.com", testCaps(), WithDNSExecutor(mockExec))
	result := check.ExecuteAction(context.Background(), "flush-cache")

	if result.Success {
		t.Error("Success should be false for failed flush")
	}
	if result.Message != "Failed to flush DNS cache" {
		t.Errorf("Message = %q, want %q", result.Message, "Failed to flush DNS cache")
	}
}

// TestDNSCheckExecuteAction_TestExternal tests test-external action
func TestDNSCheckExecuteAction_TestExternal(t *testing.T) {
	tests := []struct {
		name          string
		cmdOutput     string
		cmdError      error
		expectSuccess bool
		expectMsg     string
	}{
		{
			name:          "external DNS works",
			cmdOutput:     "Server: 8.8.8.8\nAddress: 142.250.80.46",
			cmdError:      nil,
			expectSuccess: true,
			expectMsg:     "External DNS resolution works - issue is likely with local resolver",
		},
		{
			name:          "external DNS fails",
			cmdOutput:     "Connection timed out",
			cmdError:      checks.ErrTimeout,
			expectSuccess: false,
			expectMsg:     "External DNS test failed - possible network issue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := checks.NewMockExecutor()
			mockExec.Responses["nslookup google.com 8.8.8.8"] = checks.MockResponse{
				Output: []byte(tt.cmdOutput),
				Error:  tt.cmdError,
			}

			check := NewDNSCheck("google.com", testCaps(), WithDNSExecutor(mockExec))
			result := check.ExecuteAction(context.Background(), "test-external")

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectSuccess)
			}
			if result.Message != tt.expectMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectMsg)
			}
		})
	}
}

// TestDNSCheckExecuteAction_Unknown tests unknown action handling
func TestDNSCheckExecuteAction_Unknown(t *testing.T) {
	check := NewDNSCheck("google.com", testCaps())
	result := check.ExecuteAction(context.Background(), "unknown-action")

	if result.Success {
		t.Error("Success should be false for unknown action")
	}
	if result.Error != "unknown action: unknown-action" {
		t.Errorf("Error = %q, want %q", result.Error, "unknown action: unknown-action")
	}
}

// TestDNSCheckHealable verifies DNSCheck implements HealableCheck
func TestDNSCheckHealable(t *testing.T) {
	var _ checks.HealableCheck = (*DNSCheck)(nil)

	check := NewDNSCheck("google.com", testCaps())
	actions := check.RecoveryActions(nil)

	if len(actions) != 3 {
		t.Errorf("Expected 3 recovery actions, got %d", len(actions))
	}

	// Verify action IDs
	actionIDs := make(map[string]bool)
	for _, a := range actions {
		actionIDs[a.ID] = true
	}
	expectedActions := []string{"restart-resolved", "flush-cache", "test-external"}
	for _, expected := range expectedActions {
		if !actionIDs[expected] {
			t.Errorf("DNSCheck should have %s action", expected)
		}
	}
}

// TestDNSCheckResolverInjection verifies resolver is properly injected
func TestDNSCheckResolverInjection(t *testing.T) {
	mockResolver := checks.NewMockDNSResolver()
	check := NewDNSCheck("google.com", testCaps(), WithDNSResolver(mockResolver))

	if check.resolver != mockResolver {
		t.Error("Resolver was not properly injected")
	}
}

// TestDNSCheckExecutorInjection verifies executor is properly injected
func TestDNSCheckExecutorInjection(t *testing.T) {
	mockExec := checks.NewMockExecutor()
	check := NewDNSCheck("google.com", testCaps(), WithDNSExecutor(mockExec))

	if check.executor != mockExec {
		t.Error("Executor was not properly injected")
	}
}

// TestDNSCheckDefaultsUsed verifies defaults are used when no options provided
func TestDNSCheckDefaultsUsed(t *testing.T) {
	check := NewDNSCheck("google.com", testCaps())

	if check.executor != checks.DefaultExecutor {
		t.Error("Default executor should be used")
	}
	// resolver should be realDNSResolver (we can't compare directly, but it should not be nil)
	if check.resolver == nil {
		t.Error("Default resolver should be set")
	}
}
