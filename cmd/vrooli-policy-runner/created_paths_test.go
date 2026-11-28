package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	agentharness "github.com/vrooli/agentharness"
)

func TestCreatedPathsSwitchAndAudit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_AGENT_POLICY_HOME", dir)
	run := func(args ...string) string {
		var out bytes.Buffer
		if err := runCreatedPaths(args, &out); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return out.String()
	}
	if got := run("status"); !strings.Contains(got, "session created itself: on") {
		t.Fatalf("the rule is on by default: %s", got)
	}
	if got := run("off", "--by", "operator"); !strings.Contains(got, ": off") || !strings.Contains(got, "by operator") {
		t.Fatalf("off: %s", got)
	}
	if agentharness.LoadCreationSettings(dir).Enabled {
		t.Fatal("the switch did not persist")
	}
	agentharness.AppendCreationAudit(dir, agentharness.CreationAudit{At: time.Now().UTC(), Path: "/x/y", Source: "birth-time", Certainty: "medium"})
	if got := run("audit"); !strings.Contains(got, "/x/y (birth-time, medium)") {
		t.Fatalf("audit: %s", got)
	}
	if got := run("on"); !strings.Contains(got, ": on") || !strings.Contains(got, "Allowed automatically in the last 24h0m0s: 1") {
		t.Fatalf("on: %s", got)
	}
	var out bytes.Buffer
	if err := runCreatedPaths([]string{"frobnicate"}, &out); err == nil {
		t.Fatal("an unknown command was accepted")
	}
}
