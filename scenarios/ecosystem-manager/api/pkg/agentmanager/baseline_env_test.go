package agentmanager

import (
	"strings"
	"testing"
)

// TestCollectShadowEnv exercises the pure forwarding logic with an injected
// getenv so the propagation contract is verified without mutating the process
// environment.
func TestCollectShadowEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want map[string]string
	}{
		{
			name: "var set is forwarded verbatim",
			env:  map[string]string{"VROOLI_SHADOW_SCENARIOS": "swarm-manager,agent-manager"},
			want: map[string]string{"VROOLI_SHADOW_SCENARIOS": "swarm-manager,agent-manager"},
		},
		{
			name: "unset yields nil (no empty map materialized)",
			env:  map[string]string{},
			want: nil,
		},
		{
			name: "whitespace-only value is treated as unset",
			env:  map[string]string{"VROOLI_SHADOW_SCENARIOS": "   "},
			want: nil,
		},
		{
			name: "surrounding whitespace is trimmed",
			env:  map[string]string{"VROOLI_SHADOW_SCENARIOS": "  swarm-manager  "},
			want: map[string]string{"VROOLI_SHADOW_SCENARIOS": "swarm-manager"},
		},
		{
			name: "unrelated vars are not forwarded",
			env:  map[string]string{"PATH": "/usr/bin", "HOME": "/home/x"},
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := collectShadowEnv(func(k string) string { return tc.env[k] })
			if !equalEnv(got, tc.want) {
				t.Fatalf("collectShadowEnv() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAmbientShadowEnv_ProcessEnv proves the exported helper reads the real
// process environment (the path the queue call sites use).
func TestAmbientShadowEnv_ProcessEnv(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		t.Setenv("VROOLI_SHADOW_SCENARIOS", "ecosystem-manager")
		got := AmbientShadowEnv()
		if got["VROOLI_SHADOW_SCENARIOS"] != "ecosystem-manager" {
			t.Fatalf("AmbientShadowEnv() = %v, want VROOLI_SHADOW_SCENARIOS=ecosystem-manager", got)
		}
	})
	t.Run("unset returns nil", func(t *testing.T) {
		t.Setenv("VROOLI_SHADOW_SCENARIOS", "")
		if got := AmbientShadowEnv(); got != nil {
			t.Fatalf("AmbientShadowEnv() = %v, want nil when unset", got)
		}
	})
}

// TestShadowRoutingKeysSatisfyAgentManagerContract guards the forwarded keys
// against agent-manager's validateCustomEnvironment allowlist (VROOLI_ prefix,
// ≤20 entries) so a future addition to shadowRoutingEnvKeys can never produce a
// run-rejecting environment.
func TestShadowRoutingKeysSatisfyAgentManagerContract(t *testing.T) {
	if len(shadowRoutingEnvKeys) > 20 {
		t.Fatalf("shadowRoutingEnvKeys has %d entries; agent-manager rejects >20", len(shadowRoutingEnvKeys))
	}
	for _, k := range shadowRoutingEnvKeys {
		if !strings.HasPrefix(k, "VROOLI_") {
			t.Errorf("routing key %q lacks the VROOLI_ prefix agent-manager requires", k)
		}
	}
}

// TestExecuteRequestEnvironmentRoundTrips documents that Environment is part of
// the request contract the queue layer fills and the agent service forwards onto
// CreateRunRequest.environment. The MockAgentService captures the full request,
// so this asserts the field survives the call boundary.
func TestExecuteRequestEnvironmentRoundTrips(t *testing.T) {
	mock := NewMockAgentService()
	env := map[string]string{"VROOLI_SHADOW_SCENARIOS": "swarm-manager"}

	if _, err := mock.ExecuteTaskAsync(t.Context(), ExecuteRequest{Environment: env}); err != nil {
		t.Fatalf("ExecuteTaskAsync: %v", err)
	}
	if got := mock.LastExecuteTaskAsyncReq.Environment["VROOLI_SHADOW_SCENARIOS"]; got != "swarm-manager" {
		t.Fatalf("ExecuteTaskAsync did not carry Environment: got %v", mock.LastExecuteTaskAsyncReq.Environment)
	}

	if _, err := mock.ExecuteInsight(t.Context(), InsightRequest{Environment: env}); err != nil {
		t.Fatalf("ExecuteInsight: %v", err)
	}
	if got := mock.LastInsightReq.Environment["VROOLI_SHADOW_SCENARIOS"]; got != "swarm-manager" {
		t.Fatalf("ExecuteInsight did not carry Environment: got %v", mock.LastInsightReq.Environment)
	}
}

func equalEnv(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
