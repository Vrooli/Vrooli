package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRediscoveryGate(t *testing.T) {
	var report output
	if err := fromFixture("../../internal/hostpressure/testdata/host-2026-08-22", &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 4 {
		t.Fatalf("expected four evidence-backed findings, got %d: %v", len(report.Findings), report.Findings)
	}
	for _, key := range []string{"fork-rate", "stranded-memory", "abandoned-workload", "idle-vrooli-service"} {
		if len(report.Evidence[key]) == 0 {
			t.Fatalf("finding %q has no evidence", key)
		}
	}
}

func TestSustainedFailureUsesDurableHysteresis(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	base := time.Date(2026, 8, 22, 20, 0, 0, 0, time.UTC)
	previous := watchdogNow
	defer func() { watchdogNow = previous }()
	watchdogNow = func() time.Time { return base }
	if sustainedFailure("test", true, time.Minute) {
		t.Fatal("first observation must be held")
	}
	watchdogNow = func() time.Time { return base.Add(time.Minute - time.Second) }
	if sustainedFailure("test", true, time.Minute) {
		t.Fatal("failure before sustain window must be held")
	}
	watchdogNow = func() time.Time { return base.Add(time.Minute) }
	if !sustainedFailure("test", true, time.Minute) {
		t.Fatal("failure at sustain window must be reported")
	}
	if sustainedFailure("test", false, time.Minute) {
		t.Fatal("recovered failure must clear")
	}
}

func TestDisposalProposalIsStructuredAndPreviewOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDisposalProposal(disposalProposal{Workload: "airbyte-abctl-control-plane", Class: "abandoned", Posture: "vrooli_only", Evidence: []string{"agent-experiments/airbyte"}, Reason: "historical evidence", ProposedAction: "preview undeclared-workload disposal"})
	f, err := os.Open(filepath.Join(home, ".vrooli", "state", "watchdog-disposal-proposals.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var line map[string]any
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("proposal file is empty")
	}
	if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
		t.Fatal(err)
	}
	if line["workload"] != "airbyte-abctl-control-plane" || line["proposed_action"] != "preview undeclared-workload disposal" {
		t.Fatalf("unexpected proposal: %#v", line)
	}
}
