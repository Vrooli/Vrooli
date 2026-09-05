package sandbox

import (
	"strings"
	"testing"
)

func TestPathsOverlapRespectsPathSegments(t *testing.T) {
	for _, tc := range []struct {
		left, right string
		want        bool
	}{
		{"", "src", false},
		{"src", "src", true},
		{"src", "src/api", true},
		{"src/api", "src", true},
		{"src", "scripts", false},
		{"src", "src-old", false},
		{"scenarios/agent-manager", "scenarios/agent-manager/api", true},
	} {
		if got := pathsOverlap(tc.left, tc.right); got != tc.want {
			t.Fatalf("pathsOverlap(%q, %q)=%t, want %t", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestFormatConflictErrorIncludesSafeIdentifiersAndGuidance(t *testing.T) {
	if got := FormatConflictError(nil); got != "" {
		t.Fatalf("empty conflicts=%q", got)
	}
	message := FormatConflictError([]ConflictingSandbox{{SandboxID: "1234567890abcdef", Scope: "scenarios/agent-manager"}, {SandboxID: "short", Scope: "scenarios/workspace-sandbox"}})
	for _, want := range []string{"2 existing sandbox", "Sandbox 12345678 manages scope: scenarios/agent-manager", "Sandbox short manages scope", "vrooli sandbox list", "vrooli sandbox delete"} {
		if !strings.Contains(message, want) {
			t.Fatalf("conflict message missing %q: %s", want, message)
		}
	}
}

func TestSandboxAPIErrorGetConflictsDecodesOnlyObjectEntries(t *testing.T) {
	if got := (&SandboxAPIError{}).GetConflicts(); got != nil {
		t.Fatalf("missing details=%+v", got)
	}
	err := &SandboxAPIError{Details: map[string]interface{}{
		"conflicts": []interface{}{
			map[string]interface{}{"sandboxId": "sandbox-1", "scope": "src", "conflictType": "scope_overlap"},
			"malformed",
			map[string]interface{}{"sandboxId": 3},
		},
	}}
	conflicts := err.GetConflicts()
	if len(conflicts) != 2 || conflicts[0].SandboxID != "sandbox-1" || conflicts[0].Scope != "src" || conflicts[1].SandboxID != "" {
		t.Fatalf("conflicts=%+v", conflicts)
	}
	if got := (&SandboxAPIError{Details: map[string]interface{}{"conflicts": "not-a-list"}}).GetConflicts(); got != nil {
		t.Fatalf("invalid conflicts payload=%+v", got)
	}
}
