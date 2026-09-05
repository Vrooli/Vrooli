package agentcontext

import "testing"

func TestIsAgentControlled(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want bool
	}{
		{name: "empty env", want: false},
		{name: "regular env", env: []string{"PATH=/usr/bin", "HOME=/tmp"}, want: false},
		{name: "sandbox id", env: []string{"VROOLI_SANDBOX_ID=sbx-1"}, want: true},
		{name: "sandbox merged", env: []string{"VROOLI_SANDBOX_MERGED=/workspace"}, want: true},
		{name: "identity token", env: []string{"VROOLI_AGENT_IDENTITY_TOKEN=tok"}, want: true},
		{name: "empty managed key ignored", env: []string{"VROOLI_SANDBOX_ID="}, want: false},
		{name: "bare managed key", env: []string{"VROOLI_AGENT_MANAGER_API_BASE"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAgentControlled(tt.env); got != tt.want {
				t.Fatalf("IsAgentControlled(%v) = %v, want %v", tt.env, got, tt.want)
			}
		})
	}
}
