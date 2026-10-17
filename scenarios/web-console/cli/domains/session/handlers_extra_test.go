package session

import (
	"testing"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
)

func TestFormattingHelpers(t *testing.T) {
	if n, _ := atoiFlag(""); n != 0 {
		t.Fatal(n)
	}
	if n, err := atoiFlag("7"); err != nil || n != 7 {
		t.Fatalf("atoi = %d, %v", n, err)
	}
	if _, err := atoiFlag("bad"); err == nil {
		t.Fatal("invalid atoi succeeded")
	}
	if got := sessionRows(nil); len(got) != 1 {
		t.Fatal(got)
	}
	if got := recoverableRows(nil); len(got) != 1 {
		t.Fatal(got)
	}
	if got := sessionRows([]*sessionsv1.Session{{Id: "session-123", Shell: "bash", Backend: "standard", Cols: 80, Rows: 24, Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI, Owner: "owner", Target: &sharedv1.Target{Id: "local", Label: "Local"}}}); len(got) != 1 {
		t.Fatalf("session rows = %#v", got)
	}
	if got := recoverableRows([]*sessionsv1.RecoverableSession{{Id: "orphan", AgentType: "codex", AgentSessionId: "agent", Recoverable: false, NotRecoverableReason: "expired", OrphanedAt: "2026-01-01T00:00:00Z"}}); len(got) != 1 {
		t.Fatalf("recoverable rows = %#v", got)
	}
	if policyString(nil) != "never" || policyString(&sessionsv1.ExpirationPolicy{Mode: "days"}) != "days" || policyString(&sessionsv1.ExpirationPolicy{Mode: "days", Duration: "7d"}) != "days/7d" {
		t.Fatal("policy formatting mismatch")
	}
	if defaultIfEmpty("", "fallback") != "fallback" || defaultIfEmpty("value", "fallback") != "value" {
		t.Fatal("default formatting mismatch")
	}
}
