package sessions

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/nodereach"
	"web-console/internal/backend"
	"web-console/internal/policy"
	intsessions "web-console/internal/sessions"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestAdapterPureMappingsAndErrorClassification(t *testing.T) {
	for _, in := range []error{session.ErrSessionLimitReached, session.ErrBackendUnavailable, session.ErrBackendUnknown, session.ErrPTYSpawnFailed, errors.New("x")} {
		if got := mapCreateError(in); got == nil {
			t.Fatalf("mapCreateError(%v) returned nil", in)
		}
	}
	missingScope := mapCreateError(&nodereach.Error{Kind: nodereach.ErrMissingScope, Scope: "vrooli-bridge:write", Err: errors.New("missing")})
	if !strings.Contains(missingScope.Error(), "vrooli-bridge:write") || !strings.Contains(missingScope.Error(), "manage the machine permissions") {
		t.Fatalf("missing-scope mapping = %q", missingScope)
	}
	capability := mapCreateError(fmt.Errorf("%w: capability %q on minimouse is missing; Install this coding agent on the selected machine", ErrTargetUnavailable, "codex"))
	if !strings.Contains(capability.Error(), `capability "codex"`) || !strings.Contains(capability.Error(), "Install this coding agent") {
		t.Fatalf("capability mapping = %q", capability)
	}
	resp := intsessions.Response{ID: "s", Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: backend.Standard, SurvivesRestart: true, Policy: policy.Policy{Mode: policy.Preset, Duration: "1h"}, Recovered: true, Origin: "ui", Owner: "owner", DisplayLabel: "label"}
	converted := responseToHandlerSession(resp)
	if converted.ID != "s" || converted.Policy.Duration != "1h" || converted.Origin != "ui" {
		t.Fatalf("response mapping: %+v", converted)
	}
	if back := handlerSessionToResponse(converted); back.ID != "s" || back.Backend != backend.Standard {
		t.Fatalf("reverse mapping: %+v", back)
	}
	if createFingerprint(CreateInput{Shell: "sh"}) == createFingerprint(CreateInput{Shell: "bash"}) {
		t.Fatal("fingerprint collision")
	}
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	meta := sessionstore.Metadata{ID: "r", Backend: backend.Persistent, Shell: "/bin/sh", Cols: 80, Rows: 24, Created: now, LastActivityAt: now, OrphanedAt: now, AgentType: sessionstore.AgentCodex, AgentSessionID: "agent", LastRolloutPath: "/tmp/rollout"}
	recoverable := toHandlerRecoverable(meta)
	if recoverable.ID != "r" || recoverable.AgentSessionID != "agent" || recoverable.CreatedAt == "" {
		t.Fatalf("recoverable: %+v", recoverable)
	}
	if formatTimeOrEmpty(time.Time{}) != "" || formatTimeOrEmpty(now) == "" || sanitizeID("a\x00b") != "ab" || len(sanitizeID("1234567890123456789012345678901234567890123")) != 43 {
		t.Fatal("time/id helpers")
	}
	for _, m := range []sessionstore.Agent{sessionstore.AgentNone, sessionstore.AgentCodex} {
		_ = m
	}
	adapter := &Adapter{AgentHistoryPresent: func(sessionstore.Metadata) bool { return true }}
	if state, _ := adapter.restoreState(sessionstore.Metadata{AgentType: sessionstore.AgentNone}, 0); state != RestoreStateNothingToRestore {
		t.Fatal(state)
	}
	if state, _ := adapter.restoreState(meta, 1); state != RestoreStateReopenable {
		t.Fatal(state)
	}
	if view := policyViewFor(&session.Session{ID: "s", CreatedAt: now}, policy.Policy{Mode: policy.Preset, Duration: "1h"}); !view.HasExpiry || view.ExpiresAt == "" {
		t.Fatalf("policy view: %+v", view)
	}
	if view := policyViewFor(&session.Session{ID: "s", CreatedAt: now}, policy.Policy{Mode: policy.Never}); view.HasExpiry {
		t.Fatalf("never policy: %+v", view)
	}
}
