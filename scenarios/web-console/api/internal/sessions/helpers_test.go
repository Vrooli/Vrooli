package sessions

import (
	"strings"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestHelperMappingsAndRecoveryCommands(t *testing.T) {
	for _, raw := range []string{"ui", "remote", "programmatic", "", "unknown"} {
		_ = NormalizeOrigin(raw)
	}
	for _, raw := range []string{"none", "codex", "claude", "opencode", "grok", "unknown"} {
		_ = NormalizeAgentType(raw)
	}

	rows := map[string]sessionstore.Metadata{
		"a": {ID: "a", RecoveredInto: "b"},
		"b": {ID: "b", RecoveredInto: "c"},
		"c": {ID: "c"},
	}
	if got := ResolveLineage(rows["a"], rows); got.ID != "c" {
		t.Fatalf("lineage = %q, want c", got.ID)
	}
	rows["c"] = sessionstore.Metadata{ID: "c", RecoveredInto: "a"}
	if got := ResolveLineage(rows["a"], rows); got.ID != "c" {
		t.Fatalf("cyclic lineage = %q, want c", got.ID)
	}

	cases := []struct {
		agent sessionstore.Agent
		id    string
		ok    bool
		want  string
	}{
		{sessionstore.AgentNone, "", false, "no agent identity"},
		{sessionstore.AgentCodex, "", true, "codex --yolo resume --last"},
		{sessionstore.AgentCodex, "s1", true, "codex --yolo resume s1"},
		{sessionstore.AgentClaude, "s2", true, "claude --resume s2"},
		{sessionstore.AgentOpenCode, "", false, "opencode session id missing"},
		{sessionstore.AgentGrok, "s3", true, "grok --resume s3"},
	}
	for _, tc := range cases {
		meta := sessionstore.Metadata{AgentType: tc.agent, AgentSessionID: tc.id}
		ok, _ := Recoverability(meta)
		if ok != tc.ok {
			t.Errorf("Recoverability(%q) = %v, want %v", tc.agent, ok, tc.ok)
		}
		if command := BuildResumeCommand(meta); !strings.Contains(command, tc.want) {
			t.Errorf("BuildResumeCommand(%q) = %q, want %q", tc.agent, command, tc.want)
		}
	}
}

func TestFromSessionProjectsCachedResponse(t *testing.T) {
	created := time.Date(2026, time.August, 25, 12, 34, 56, 0, time.UTC)
	sess := &session.Session{
		ID:        "sess-1",
		Shell:     "/bin/bash",
		CreatedAt: created,
		Cols:      120,
		Rows:      40,
		Backend:   backend.Persistent,
	}
	sess.SetPolicy(policy.Policy{Mode: policy.Preset, Duration: "8h"})

	got := FromSession(sess)
	if got.ID != sess.ID || got.Shell != sess.Shell || got.CreatedAt != "2026-08-25T12:34:56Z" {
		t.Fatalf("basic projection = %+v", got)
	}
	if got.Cols != 120 || got.Rows != 40 || got.Backend != backend.Persistent {
		t.Fatalf("size/backend projection = %+v", got)
	}
	if !got.SurvivesRestart || got.Policy != (policy.Policy{Mode: policy.Preset, Duration: "8h"}) || got.Recovered {
		t.Fatalf("lifecycle projection = %+v", got)
	}

	sess.Backend = backend.Standard
	if got := FromSession(sess); got.SurvivesRestart {
		t.Fatal("standard sessions must not be marked restart-surviving")
	}
}
