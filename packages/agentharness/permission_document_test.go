package agentharness

import (
	"strings"
	"testing"
)

func TestPermissionDocumentValidatesAndProjectsPortableBashRules(t *testing.T) {
	document := PermissionDocument{SchemaVersion: PermissionDocumentSchemaVersion, Scope: "user", Rules: []PermissionRule{
		{ID: "allow-git", Action: "allow", Matcher: PermissionMatcher{Kind: "bash", Pattern: "git *"}},
		{ID: "deny-root", Action: "deny", Matcher: PermissionMatcher{Kind: "bash", Pattern: "rm -rf /"}},
	}}
	if err := ValidatePermissionDocument(document); err != nil {
		t.Fatalf("ValidatePermissionDocument() error = %v", err)
	}
	allow, ask, deny := PermissionPatterns(document)
	if got, want := strings.Join(allow, ","), "git *"; got != want {
		t.Fatalf("allow = %q, want %q", got, want)
	}
	if len(ask) != 0 {
		t.Fatalf("ask = %#v, want empty", ask)
	}
	if got, want := strings.Join(deny, ","), "rm -rf /"; got != want {
		t.Fatalf("deny = %q, want %q", got, want)
	}
}

func TestPermissionDocumentRejectsUnknownAndUnsupportedFields(t *testing.T) {
	_, err := parsePermissionDocument([]byte(`{"schema_version":"v1","rules":[{"id":"nope","action":"deny","matcher":{"kind":"tool","pattern":"x"}}],"extra":true}`), "test")
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("LoadPermissionDocumentFromBytes() error = %v, want unknown-field failure", err)
	}
	_, err = parsePermissionDocument([]byte(`{"schema_version":"v1","rules":[{"id":"nope","action":"deny","matcher":{"kind":"tool","pattern":"x"}}]}`), "test")
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("LoadPermissionDocumentFromBytes() error = %v, want unsupported matcher", err)
	}
}
