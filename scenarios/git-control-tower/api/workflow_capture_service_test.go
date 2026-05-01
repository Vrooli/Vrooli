package main

import (
	"testing"

	"git-control-tower/internal/testutil/fixtures"
)

func TestParseWorkflowMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		wantName string
		wantMode string
		wantErr  bool
	}{
		{
			name:     "full metadata",
			json:     `{"metadata":{"description":"Login test","execution_mode":"mutating"},"nodes":[],"edges":[]}`,
			wantName: "Login test",
			wantMode: "mutating",
		},
		{
			name:     "observer mode",
			json:     `{"metadata":{"description":"Dashboard check","execution_mode":"observer"},"nodes":[],"edges":[]}`,
			wantName: "Dashboard check",
			wantMode: "observer",
		},
		{
			name:     "missing execution_mode defaults to observer",
			json:     `{"metadata":{"description":"No mode set"},"nodes":[],"edges":[]}`,
			wantName: "No mode set",
			wantMode: "observer",
		},
		{
			name:     "missing description defaults to unnamed",
			json:     `{"metadata":{"execution_mode":"destructive"},"nodes":[],"edges":[]}`,
			wantName: "unnamed",
			wantMode: "destructive",
		},
		{
			name:    "invalid JSON",
			json:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, mode, err := parseWorkflowMetadata([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", mode, tt.wantMode)
			}
		})
	}
}

func TestFilterByExecutionMode(t *testing.T) {
	t.Parallel()

	workflows := []discoveredWorkflow{
		{Name: "observer-wf", ExecutionMode: "observer"},
		{Name: "mutating-wf", ExecutionMode: "mutating"},
		{Name: "destructive-wf", ExecutionMode: "destructive"},
		{Name: "another-observer", ExecutionMode: "observer"},
	}

	tests := []struct {
		name    string
		allowed []string
		want    int
	}{
		{"observer only", []string{"observer"}, 2},
		{"observer and mutating", []string{"observer", "mutating"}, 3},
		{"all modes", []string{"observer", "mutating", "destructive"}, 4},
		{"destructive only", []string{"destructive"}, 1},
		{"empty filter", []string{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByExecutionMode(workflows, tt.allowed)
			if len(got) != tt.want {
				t.Errorf("filterByExecutionMode returned %d workflows, want %d", len(got), tt.want)
			}
		})
	}
}

func TestDiscoverWorkflows_EmptyDir(t *testing.T) {
	t.Parallel()

	repoDir := writeWorkflowCaptureRepoFixture(t, "test-scenario")
	fs := NewFakeFileIO()
	workflows, err := discoverWorkflows(fs, repoDir, "test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(workflows))
	}
}

func writeWorkflowCaptureRepoFixture(t *testing.T, scenarios ...string) string {
	t.Helper()
	root := t.TempDir()
	fixtures.WriteRepoContract(t, root)
	for _, scenario := range scenarios {
		fixtures.WriteScenarioServiceJSON(t, root, scenario, `{"service":{"name":"`+scenario+`"}}`)
	}
	return root
}
