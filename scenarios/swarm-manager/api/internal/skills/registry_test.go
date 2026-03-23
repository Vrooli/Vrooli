package skills

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		kind     string
		expected string
	}{
		// Workshop mode
		{"workshop/idea", "workshop", "idea", "swarm-manager-workshop"},
		{"workshop/research", "workshop", "research", "swarm-manager-workshop"},
		{"workshop/fix", "workshop", "fix", "swarm-manager-workshop"},
		{"workshop/execute", "workshop", "execute", "swarm-manager-workshop"},
		{"workshop/chore", "workshop", "chore", "swarm-manager-workshop"},

		// Research mode
		{"research/idea", "research", "idea", "swarm-manager-research-idea"},
		{"research/fix", "research", "fix", "swarm-manager-research-fix"},
		{"research/execute", "research", "execute", "swarm-manager-research-general"},
		{"research/research", "research", "research", "swarm-manager-research-general"},
		{"research/chore", "research", "chore", "swarm-manager-research-general"},

		// Initialize mode
		{"initialize/idea", "initialize", "idea", "swarm-manager-initialize-backlog"},
		{"initialize/research", "initialize", "research", "swarm-manager-initialize-backlog"},
		{"initialize/fix", "initialize", "fix", "swarm-manager-initialize-backlog"},
		{"initialize/execute", "initialize", "execute", "swarm-manager-initialize-backlog"},
		{"initialize/chore", "initialize", "chore", "swarm-manager-initialize-backlog"},

		// Fallbacks
		{"unknown mode", "unknown", "idea", "swarm-manager-research-general"},
		{"unknown kind", "workshop", "unknown", "swarm-manager-research-general"},
		{"both unknown", "foo", "bar", "swarm-manager-research-general"},
		{"empty strings", "", "", "swarm-manager-research-general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.mode, tt.kind)
			if got != tt.expected {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.mode, tt.kind, got, tt.expected)
			}
		})
	}
}

func TestClassifyCaptureSkillID(t *testing.T) {
	got := ClassifyCaptureSkillID()
	if got != "swarm-manager-classify-capture" {
		t.Errorf("ClassifyCaptureSkillID() = %q, want %q", got, "swarm-manager-classify-capture")
	}
}
