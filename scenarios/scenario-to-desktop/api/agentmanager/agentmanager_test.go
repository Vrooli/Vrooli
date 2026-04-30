package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// TEST HELPERS
// =============================================================================

// staticResolver returns a fixed URL (used to point the client at httptest.Server).
type staticResolver struct {
	url string
}

func (s *staticResolver) ResolveBaseURL(_ context.Context) (string, error) {
	return s.url, nil
}

// newTestClient creates a Client that targets the given httptest.Server.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	return NewClientWithResolver(5*time.Second, &staticResolver{url: serverURL})
}

// newTestService creates an AgentService backed by the given httptest.Server.
func newTestService(t *testing.T, serverURL string) *AgentService {
	t.Helper()
	client := newTestClient(t, serverURL)
	return NewAgentServiceWithClient(client, "test-profile", "test-key", true)
}

// =============================================================================
// buildRunTag (pure function)
// =============================================================================

func TestBuildRunTag(t *testing.T) {
	tests := []struct {
		name          string
		invID         string
		additionalTag string
		want          string
	}{
		{"basic tag", "inv-123", "", "scenario-to-desktop-inv-123"},
		{"with additional tag", "inv-456", "fix", "scenario-to-desktop-inv-456|fix"},
		{"trims whitespace-only additional", "inv-789", "   ", "scenario-to-desktop-inv-789"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRunTag(tc.invID, tc.additionalTag)
			assert.Equal(t, tc.want, got)
		})
	}
}

// =============================================================================
// DefaultProfileConfig
// =============================================================================

func TestDefaultProfileConfig(t *testing.T) {
	cfg := DefaultProfileConfig()

	assert.Equal(t, domainpb.RunnerType_RUNNER_TYPE_CODEX, cfg.RunnerType)
	assert.Equal(t, domainpb.ModelPreset_MODEL_PRESET_SMART, cfg.ModelPreset)
	assert.Equal(t, int32(75), cfg.MaxTurns)
	assert.Equal(t, int32(600), cfg.TimeoutSeconds)
	assert.True(t, len(cfg.AllowedTools) > 0, "should have allowed tools")
	assert.True(t, cfg.SkipPermissions)
	assert.Equal(t, domainpb.SandboxMode_SANDBOX_MODE_OFF, cfg.SandboxMode,
		"scenario-to-desktop investigations run in-place against the local build tooling")
}

// =============================================================================
// AgentService.IsEnabled / IsAvailable
// =============================================================================

func TestAgentService_IsEnabled(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		svc := &AgentService{enabled: true}
		assert.True(t, svc.IsEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		svc := &AgentService{enabled: false}
		assert.False(t, svc.IsEnabled())
	})
}

func TestAgentService_IsAvailable(t *testing.T) {
	t.Run("returns true when healthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		svc := newTestService(t, srv.URL)
		assert.True(t, svc.IsAvailable(context.Background()))
	})

	t.Run("returns false when unhealthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		svc := newTestService(t, srv.URL)
		assert.False(t, svc.IsAvailable(context.Background()))
	})

	t.Run("returns false when disabled", func(t *testing.T) {
		svc := &AgentService{enabled: false}
		assert.False(t, svc.IsAvailable(context.Background()))
	})
}

// =============================================================================
// Client.Health
// =============================================================================

func TestClient_Health(t *testing.T) {
	t.Run("returns true for 200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		ok, err := c.Health(context.Background())
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("returns false for non-200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		ok, err := c.Health(context.Background())
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

// =============================================================================
// Client.StopRun
// =============================================================================

func TestClient_StopRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/runs/run-123/stop", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		err := c.StopRun(context.Background(), "run-123")
		assert.NoError(t, err)
	})

	t.Run("error response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"run not found"}`))
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		err := c.StopRun(context.Background(), "run-missing")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "run not found")
	})
}

// =============================================================================
// Client.GetRun
// =============================================================================

func TestClient_GetRun(t *testing.T) {
	t.Run("returns run", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/runs/run-abc", r.URL.Path)
			resp := &apipb.GetRunResponse{
				Run: &domainpb.Run{
					Id:     "run-abc",
					Status: domainpb.RunStatus_RUN_STATUS_RUNNING,
				},
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		run, err := c.GetRun(context.Background(), "run-abc")
		require.NoError(t, err)
		require.NotNil(t, run)
		assert.Equal(t, "run-abc", run.Id)
		assert.Equal(t, domainpb.RunStatus_RUN_STATUS_RUNNING, run.Status)
	})

	t.Run("returns nil for 404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		run, err := c.GetRun(context.Background(), "missing")
		require.NoError(t, err)
		assert.Nil(t, run)
	})
}

// =============================================================================
// Client.WaitForRun
// =============================================================================

func TestClient_WaitForRun(t *testing.T) {
	t.Run("returns when run reaches terminal state", func(t *testing.T) {
		var callCount int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			n := atomic.AddInt32(&callCount, 1)
			status := domainpb.RunStatus_RUN_STATUS_RUNNING
			if n >= 2 {
				status = domainpb.RunStatus_RUN_STATUS_COMPLETE
			}
			resp := &apipb.GetRunResponse{
				Run: &domainpb.Run{
					Id:     "run-wait",
					Status: status,
				},
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL)
		run, err := c.WaitForRun(context.Background(), "run-wait", 10*time.Millisecond)
		require.NoError(t, err)
		assert.Equal(t, domainpb.RunStatus_RUN_STATUS_COMPLETE, run.Status)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := &apipb.GetRunResponse{
				Run: &domainpb.Run{Id: "run-ctx", Status: domainpb.RunStatus_RUN_STATUS_RUNNING},
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		}))
		defer srv.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		c := newTestClient(t, srv.URL)
		_, err := c.WaitForRun(ctx, "run-ctx", 10*time.Millisecond)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

// =============================================================================
// AgentService.Initialize
// =============================================================================

func TestAgentService_Initialize(t *testing.T) {
	t.Run("noop when disabled", func(t *testing.T) {
		svc := &AgentService{enabled: false}
		err := svc.Initialize(context.Background(), DefaultProfileConfig())
		assert.NoError(t, err)
	})

	t.Run("creates profile and stores ID", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/profiles/reconcile-scenario", r.URL.Path)
			resp := &apipb.ReconcileScenarioProfilesResponse{
				Scenario: "scenario-to-desktop",
				Results: []*apipb.ProfileReconcileResult{
					{
						ProfileKey: "test-key",
						ProfileId:  "profile-xyz",
						Status:     apipb.ProfileReconcileStatus_PROFILE_RECONCILE_STATUS_CREATED,
					},
				},
				Created: 1,
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		}))
		defer srv.Close()

		svc := newTestService(t, srv.URL)
		err := svc.Initialize(context.Background(), DefaultProfileConfig())
		require.NoError(t, err)
		assert.Equal(t, "profile-xyz", svc.GetProfileID())
	})
}

// =============================================================================
// AgentService.Execute (disabled guard)
// =============================================================================

func TestAgentService_Execute_Disabled(t *testing.T) {
	svc := &AgentService{enabled: false}
	_, err := svc.Execute(context.Background(), ExecuteRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestAgentService_ExecuteAsync_Disabled(t *testing.T) {
	svc := &AgentService{enabled: false}
	_, err := svc.ExecuteAsync(context.Background(), ExecuteRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

// =============================================================================
// AgentService.ExecuteAsync (with mock server)
// =============================================================================

func TestAgentService_ExecuteAsync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/tasks" && r.Method == "POST":
			resp := &apipb.CreateTaskResponse{
				Task: &domainpb.Task{Id: "task-1"},
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(data)

		case r.URL.Path == "/api/v1/runs" && r.Method == "POST":
			resp := &apipb.CreateRunResponse{
				Run: &domainpb.Run{Id: "run-async-1"},
			}
			opts := protojson.MarshalOptions{UseProtoNames: false}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(data)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	runID, err := svc.ExecuteAsync(context.Background(), ExecuteRequest{
		InvestigationID: "inv-test",
		Prompt:          "Investigate the failure",
		WorkingDir:      "/tmp/test",
	})
	require.NoError(t, err)
	assert.Equal(t, "run-async-1", runID)
}

// =============================================================================
// AgentService.Execute (full flow with mock server)
// =============================================================================

func TestAgentService_Execute(t *testing.T) {
	var runPollCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opts := protojson.MarshalOptions{UseProtoNames: false}
		switch {
		case r.URL.Path == "/api/v1/tasks" && r.Method == "POST":
			resp := &apipb.CreateTaskResponse{
				Task: &domainpb.Task{Id: "task-exec"},
			}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(data)

		case r.URL.Path == "/api/v1/runs" && r.Method == "POST":
			resp := &apipb.CreateRunResponse{
				Run: &domainpb.Run{Id: "run-exec"},
			}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(data)

		case r.URL.Path == "/api/v1/runs/run-exec" && r.Method == "GET":
			n := atomic.AddInt32(&runPollCount, 1)
			status := domainpb.RunStatus_RUN_STATUS_RUNNING
			var summary *domainpb.RunSummary
			if n >= 2 {
				status = domainpb.RunStatus_RUN_STATUS_COMPLETE
				summary = &domainpb.RunSummary{
					Description:  "Found the bug in build config",
					TokensUsed:   1500,
					CostEstimate: 0.05,
				}
			}
			resp := &apipb.GetRunResponse{
				Run: &domainpb.Run{
					Id:        "run-exec",
					Status:    status,
					Summary:   summary,
					StartedAt: timestamppb.Now(),
					EndedAt:   timestamppb.Now(),
				},
			}
			data, _ := opts.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := newTestService(t, srv.URL)
	result, err := svc.Execute(context.Background(), ExecuteRequest{
		InvestigationID: "inv-exec",
		Prompt:          "Find the bug",
		WorkingDir:      "/tmp/test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "run-exec", result.RunID)
	assert.Equal(t, "Found the bug in build config", result.Output)
	assert.Equal(t, int32(1500), result.TokensUsed)
	assert.InDelta(t, 0.05, result.CostEstimate, 0.001)
}

// =============================================================================
// AgentService.GetRunStatus / StopRun / GetRunEvents (disabled guards)
// =============================================================================

func TestAgentService_DisabledGuards(t *testing.T) {
	svc := &AgentService{enabled: false}

	t.Run("GetRunStatus", func(t *testing.T) {
		_, err := svc.GetRunStatus(context.Background(), "run-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("StopRun", func(t *testing.T) {
		err := svc.StopRun(context.Background(), "run-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("GetRunEvents", func(t *testing.T) {
		_, err := svc.GetRunEvents(context.Background(), "run-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})

	t.Run("ResolveURL", func(t *testing.T) {
		_, err := svc.ResolveURL(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not enabled")
	})
}

// =============================================================================
// Client.parseError
// =============================================================================

func TestClient_ParseError(t *testing.T) {
	c := &Client{}

	t.Run("parses error field", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad request"}`))
		}))
		defer srv.Close()

		cl := newTestClient(t, srv.URL)
		resp, _ := cl.doRequest(context.Background(), "GET", "/test", nil)
		err := c.parseError(resp)
		assert.Contains(t, err.Error(), "bad request")
	})

	t.Run("parses message field", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"something went wrong"}`))
		}))
		defer srv.Close()

		cl := newTestClient(t, srv.URL)
		resp, _ := cl.doRequest(context.Background(), "GET", "/test", nil)
		err := c.parseError(resp)
		assert.Contains(t, err.Error(), "something went wrong")
	})
}

// =============================================================================
// NewUUID
// =============================================================================

func TestNewUUID(t *testing.T) {
	id := NewUUID()
	assert.NotEmpty(t, id)
	assert.Contains(t, id, "-") // UUID format

	// Uniqueness
	id2 := NewUUID()
	assert.NotEqual(t, id, id2)
}

// =============================================================================
// buildProfile
// =============================================================================

func TestBuildProfile(t *testing.T) {
	svc := &AgentService{
		profileName: "test-profile",
		profileKey:  "test-key",
	}

	cfg := DefaultProfileConfig()
	profile := svc.buildProfile(cfg)

	assert.Equal(t, "test-profile", profile.Name)
	assert.Equal(t, "test-key", profile.ProfileKey)
	assert.Equal(t, int32(75), profile.MaxTurns)
	assert.Equal(t, "scenario-to-desktop", profile.CreatedBy)
	assert.NotNil(t, profile.Timeout)
	assert.Equal(t, cfg.AllowedTools, profile.AllowedTools)
	assert.True(t, profile.SkipPermissionPrompt)
}
