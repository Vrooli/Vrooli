package agentharness

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
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

func TestReconcileDocumentAppliesOnlyPastTheGateAndOnDrift(t *testing.T) {
	for _, tc := range []struct {
		name    string
		caller  cliutil.CallerKind
		drift   bool
		args    []string
		applied bool
		refused bool
	}{
		{name: "human with drift", caller: cliutil.CallerKindHuman, drift: true, applied: true},
		{name: "human with no drift", caller: cliutil.CallerKindHuman},
		{name: "agent is refused", caller: cliutil.CallerKindExternalAgent, drift: true, refused: true},
		{name: "authorized agent", caller: cliutil.CallerKindExternalAgent, drift: true, args: []string{"--i-was-explicitly-authorized"}, applied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			applied := false
			h := NewPermissionHandlers(PermissionCommandConfig{
				DetectCaller: func() cliutil.CallerKind { return tc.caller }, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
				PlanDocument: func(string, string) (PermissionPlanResult, func() error, error) {
					return PermissionPlanResult{Drift: tc.drift}, func() error { applied = true; return nil }, nil
				},
			})
			err := h.ReconcileDocument(append([]string{"--document", "doc.json"}, tc.args...))
			if (err != nil) != tc.refused || applied != tc.applied {
				t.Fatalf("err = %v, applied = %v", err, applied)
			}
		})
	}
}

func TestPlanDocumentNeverApplies(t *testing.T) {
	var stdout bytes.Buffer
	h := NewPermissionHandlers(PermissionCommandConfig{
		ScopeDefault: "user", ScopeHelp: "scope", PlanReportsScope: true, Stdout: &stdout, Stderr: &bytes.Buffer{},
		PlanDocument: func(path, scope string) (PermissionPlanResult, func() error, error) {
			if path != "doc.json" || scope != "admin" {
				t.Fatalf("path = %q, scope = %q", path, scope)
			}
			return PermissionPlanResult{Runner: "r", Scope: scope, Drift: true, Enforcement: EnforcementPosture{Permissions: "p"}}, func() error { return errors.New("plan must not apply") }, nil
		},
	})
	if err := h.PlanDocument([]string{"--scope", "admin", "--document", "doc.json"}); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "runner=r scope=admin drift=true changes=0 enforcement=p\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if err := h.PlanDocument(nil); err == nil || !strings.Contains(err.Error(), "--document is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestBashToolPatternsRoundTrip(t *testing.T) {
	portable := []string{"git *", "rm -rf /"}
	native := BashToolPatterns(portable)
	if want := []string{"Bash(git *)", "Bash(rm -rf /)"}; !reflect.DeepEqual(native, want) {
		t.Fatalf("native = %v", native)
	}
	if got := PortableBashPatterns(append(native, "Read(*)")); !reflect.DeepEqual(got, append(portable, "Read(*)")) {
		t.Fatalf("portable = %v", got)
	}
}
