package identity

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPrincipalStatesAreExplicit(t *testing.T) {
	cases := []struct {
		name   string
		kind   ActorKind
		verify bool
		human  bool
	}{
		{name: "unknown", kind: ActorUnknown},
		{name: "human", kind: ActorHuman, verify: true, human: true},
		{name: "agent", kind: ActorAgent, verify: true},
		{name: "service", kind: ActorService, verify: true},
		{name: "conflict", kind: ActorConflict, verify: true},
		{name: "missing subject", kind: ActorHuman, verify: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			principal := Principal{Kind: tc.kind, Subject: "subject", Verified: tc.verify}
			if tc.name == "missing subject" {
				principal.Subject = ""
			}
			expectedVerified := tc.verify && tc.name != "missing subject" && tc.kind != ActorConflict
			if got := principal.IsVerified(); got != expectedVerified {
				t.Fatalf("IsVerified()=%t", got)
			}
			if got := principal.IsHuman(); got != tc.human {
				t.Fatalf("IsHuman()=%t", got)
			}
		})
	}
}

func TestFailureFromErrorHandlesWrappingWithoutCredentialDetails(t *testing.T) {
	failure := NewFailure(FailureInvalid, SourceCloudflareAccess)
	wrapped := errors.New("outer: " + failure.Error())
	if got, ok := FailureFromError(wrapped); ok || got.Class != "" {
		t.Fatalf("plain error unexpectedly classified: %#v, %t", got, ok)
	}
	wrappedFailure := fmtWrap{err: failure}
	got, ok := FailureFromError(wrappedFailure)
	if !ok || got.Class != FailureInvalid || got.Source != SourceCloudflareAccess {
		t.Fatalf("failure=%#v ok=%t", got, ok)
	}
	if strings.Contains(got.Error(), "token") {
		t.Fatal("failure classification contains credential terminology")
	}
}

type fmtWrap struct{ err error }

func (w fmtWrap) Error() string { return "wrapped failure" }
func (w fmtWrap) Unwrap() error { return w.err }

func TestOperationGrantRulesNeverCrossPolicies(t *testing.T) {
	g := &OperationGrant{
		ID: "g", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour),
		Operations: []string{"agent.manage", "effort.reconcile"},
		Targets:    []string{"agent-manager:*", "agent-manager:effort/bas"},
		Rules: []OperationRule{
			{Policy: "agent-management", Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:*"}},
			{Policy: "effort-metadata", Operations: []string{"effort.reconcile"}, Targets: []string{"agent-manager:effort/bas"}},
		},
	}
	if !g.Allows("agent.manage", "agent-manager:run/1", time.Now()) || !g.Allows("effort.reconcile", "agent-manager:effort/bas", time.Now()) {
		t.Fatal("a rule's own operation and target were refused")
	}
	if g.Allows("effort.reconcile", "agent-manager:effort/other", time.Now()) {
		t.Fatal("effort.reconcile borrowed the agent-management wildcard target")
	}
	legacy := &OperationGrant{ID: "g", Approver: "owner", ExpiresAt: time.Now().Add(time.Hour), Operations: []string{"agent.manage"}, Targets: []string{"agent-manager:run/1"}}
	if !legacy.Allows("agent.manage", "agent-manager:run/1", time.Now()) || legacy.Allows("agent.manage", "agent-manager:run/2", time.Now()) {
		t.Fatal("a single-policy grant changed meaning")
	}
}

func TestOperationGrantScopedPrefixWildcard(t *testing.T) {
	now := time.Now()
	g := &OperationGrant{
		ID: "g", Approver: "owner", ExpiresAt: now.Add(time.Hour),
		Operations: []string{"effort.reconcile"},
		Targets:    []string{"agent-manager:effort/*"},
	}
	// Covers any effort under the prefix.
	for _, tgt := range []string{
		"agent-manager:effort/notification-hub-redesign",
		"agent-manager:effort/bas",
		"agent-manager:effort/nested/path",
	} {
		if !g.Allows("effort.reconcile", tgt, now) {
			t.Fatalf("effort/* wildcard did not cover %q", tgt)
		}
	}
	// Anchored at the path boundary: must not leak to a sibling prefix or
	// across domains, and the operation still has to match.
	for _, tgt := range []string{
		"agent-manager:effortish/x",      // sibling prefix, not under effort/
		"agent-manager:run/1",            // different object kind
		"prompt-manager:effort/anything", // different domain
	} {
		if g.Allows("effort.reconcile", tgt, now) {
			t.Fatalf("effort/* wildcard leaked to %q", tgt)
		}
	}
	if g.Allows("agent.manage", "agent-manager:effort/bas", now) {
		t.Fatal("prefix wildcard granted an operation it was never scoped to")
	}
}
