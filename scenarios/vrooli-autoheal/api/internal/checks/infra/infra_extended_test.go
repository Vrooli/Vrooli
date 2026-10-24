package infra

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/testutil"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func TestNetworkCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		dialResponse   testutil.MockDialResponse
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name: "successful connection",
			dialResponse: testutil.MockDialResponse{
				Conn:  &testutil.MockConn{},
				Error: nil,
			},
			expectedStatus: checks.StatusOK,
			expectedMsg:    "Network connectivity OK",
		},
		{
			name: "connection refused",
			dialResponse: testutil.MockDialResponse{
				Conn:  nil,
				Error: testutil.ErrConnectionRefused,
			},
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Network connectivity failed",
		},
		{
			name: "timeout",
			dialResponse: testutil.MockDialResponse{
				Conn:  nil,
				Error: testutil.ErrTimeout,
			},
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Network connectivity failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDialer := testutil.NewMockDialer()
			mockDialer.DefaultResponse = tt.dialResponse

			check := NewNetworkCheck(testTarget, WithDialer(mockDialer))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}

			// Verify the mock was called
			if len(mockDialer.Calls) != 1 {
				t.Errorf("Expected 1 dial call, got %d", len(mockDialer.Calls))
			}
			if len(mockDialer.Calls) > 0 && mockDialer.Calls[0] != testTarget {
				t.Errorf("Dial target = %q, want %q", mockDialer.Calls[0], testTarget)
			}
		})
	}
}

// TestNetworkCheckResponseTime tests that response time is recorded
// [REQ:INFRA-NET-001]
func TestNetworkCheckResponseTime(t *testing.T) {
	mockDialer := testutil.NewMockDialer()
	mockDialer.DefaultResponse = testutil.MockDialResponse{
		Conn:  &testutil.MockConn{},
		Error: nil,
	}

	check := NewNetworkCheck(testTarget, WithDialer(mockDialer))
	result := check.Run(context.Background())

	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if _, ok := result.Details["responseTimeMs"]; !ok {
		t.Error("Details should contain responseTimeMs")
	}
}

// TestDockerCheckRunWithMock tests DockerCheck.Run() using mock executor
// [REQ:INFRA-DOCKER-001] [REQ:TEST-SEAM-001]
func TestDockerCheckRunWithMock(t *testing.T) {
	tests := []struct {
		name           string
		dockerInfo     string
		execError      error
		expectedStatus checks.Status
		expectedMsg    string
	}{
		{
			name:           "docker healthy",
			dockerInfo:     `{"ServerVersion":"24.0.7","Containers":5,"ContainersRunning":3}`,
			execError:      nil,
			expectedStatus: checks.StatusOK,
			expectedMsg:    "Docker daemon is healthy",
		},
		{
			name:           "docker not responsive",
			dockerInfo:     "",
			execError:      testutil.ErrConnectionRefused,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Docker daemon not responsive",
		},
		{
			name:           "docker command not found",
			dockerInfo:     "",
			execError:      testutil.ErrCommandNotFound,
			expectedStatus: checks.StatusCritical,
			expectedMsg:    "Docker daemon not responsive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			mockExecutor.Responses["docker info --format {{json .}}"] = testutil.MockResponse{
				Output: []byte(tt.dockerInfo),
				Error:  tt.execError,
			}

			check := NewDockerCheckWithOptions(testCaps(), WithDockerExecutor(mockExecutor))
			result := check.Run(context.Background())

			if result.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.expectedStatus)
			}
			if result.Message != tt.expectedMsg {
				t.Errorf("Message = %q, want %q", result.Message, tt.expectedMsg)
			}

			// Verify the mock was called
			if len(mockExecutor.Calls) != 1 {
				t.Errorf("Expected 1 executor call, got %d", len(mockExecutor.Calls))
			}
		})
	}
}

// TestDockerCheckParsesInfo tests that Docker info is correctly parsed
// [REQ:INFRA-DOCKER-001]
func TestDockerCheckParsesInfo(t *testing.T) {
	mockExecutor := testutil.NewMockExecutor()
	mockExecutor.Responses["docker info --format {{json .}}"] = testutil.MockResponse{
		Output: []byte(`{"ServerVersion":"24.0.7","Containers":10,"ContainersRunning":5}`),
		Error:  nil,
	}

	check := NewDockerCheckWithOptions(testCaps(), WithDockerExecutor(mockExecutor))
	result := check.Run(context.Background())

	if result.Details == nil {
		t.Fatal("Details should not be nil")
	}
	if version, ok := result.Details["version"].(string); !ok || version != "24.0.7" {
		t.Errorf("version = %v, want %q", result.Details["version"], "24.0.7")
	}
	if containers, ok := result.Details["containers"].(int); !ok || containers != 10 {
		t.Errorf("containers = %v, want %d", result.Details["containers"], 10)
	}
	if running, ok := result.Details["running"].(int); !ok || running != 5 {
		t.Errorf("running = %v, want %d", result.Details["running"], 5)
	}
}

// TestDockerCheckExecuteActionWithMock tests DockerCheck.ExecuteAction() using mock
// [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
func TestDockerCheckExecuteActionWithMock(t *testing.T) {
	tests := []struct {
		name          string
		actionID      string
		cmdKey        string
		cmdOutput     string
		cmdError      error
		expectSuccess bool
	}{
		{
			name:          "prune delegates to storage-manager",
			actionID:      "prune",
			cmdKey:        "",
			cmdOutput:     "",
			expectSuccess: false,
		},
		{
			name:          "info success",
			actionID:      "info",
			cmdKey:        "docker info",
			cmdOutput:     "Docker version 24.0.7",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "logs success",
			actionID:      "logs",
			cmdKey:        "journalctl --no-pager -o short-iso -u docker -n 100",
			cmdOutput:     "Docker daemon logs...",
			cmdError:      nil,
			expectSuccess: true,
		},
		{
			name:          "unknown action",
			actionID:      "invalid-action",
			cmdKey:        "",
			cmdOutput:     "",
			cmdError:      nil,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutil.NewMockExecutor()
			if tt.cmdKey != "" {
				mockExecutor.Responses[tt.cmdKey] = testutil.MockResponse{
					Output: []byte(tt.cmdOutput),
					Error:  tt.cmdError,
				}
			}

			check := NewDockerCheckWithOptions(testCaps(), WithDockerExecutor(mockExecutor))
			result := check.ExecuteAction(context.Background(), tt.actionID)

			if result.Success != tt.expectSuccess {
				t.Errorf("Success = %v, want %v", result.Success, tt.expectSuccess)
			}
			if result.ActionID != tt.actionID {
				t.Errorf("ActionID = %q, want %q", result.ActionID, tt.actionID)
			}
			if result.CheckID != check.ID() {
				t.Errorf("CheckID = %q, want %q", result.CheckID, check.ID())
			}
			if tt.actionID == "prune" {
				if len(mockExecutor.Calls) != 0 {
					t.Fatalf("prune must not execute Docker cleanup directly, got calls: %+v", mockExecutor.Calls)
				}
				if !strings.Contains(result.Message, "storage-manager") {
					t.Fatalf("prune message = %q, want storage-manager handoff", result.Message)
				}
			}
		})
	}
}
