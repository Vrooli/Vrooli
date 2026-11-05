package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/api-core/targetmodel"
	sessionsH "web-console/handlers/sessions"
)

func TestLaunchCapabilityClassification(t *testing.T) {
	for _, test := range []struct {
		command string
		want    string
		ok      bool
	}{
		{"codex --yolo", "codex", true},
		{"vrooli agent launch --runner=opencode", "opencode", true},
		{"vrooli-agent-launcher --agent claude", "claude", true},
		{"/usr/local/bin/agy", "agy", true},
		{"bash", "", false},
	} {
		t.Run(test.command, func(t *testing.T) {
			got, ok := launchCapability(test.command)
			if got != test.want || ok != test.ok {
				t.Fatalf("launchCapability(%q) = %q, %t; want %q, %t", test.command, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestEnsureLaunchCapabilityRejectsMissingAndUnknown(t *testing.T) {
	target := targetConnection{Target: targetmodel.Target{
		Label: "Build node",
		Readiness: []targetmodel.ReadinessCheck{
			targetmodel.CapabilityReadinessCheck("codex", "Codex", targetmodel.ReadinessMissing, "codex is not installed", "Install Codex"),
		},
	}}
	if err := ensureLaunchCapability(target, "codex --yolo"); !errors.Is(err, sessionsH.ErrTargetUnavailable) {
		t.Fatalf("missing capability error = %v", err)
	}
	if err := ensureLaunchCapability(target, "claude"); !errors.Is(err, sessionsH.ErrTargetUnavailable) {
		t.Fatalf("absent inventory error = %v", err)
	}
	if err := ensureLaunchCapability(target, "bash"); err != nil {
		t.Fatalf("custom shell command rejected: %v", err)
	}
}

// A refusal must carry what the machine observed. Web Console used to report
// the state plus a generic "Refresh the capability probe and check that the
// node is reporting" — an operation it does not offer, aimed at a node that was
// already reporting correctly — while the observation itself said the agent was
// installed and its runtime was unreachable.
func TestEnsureLaunchCapabilityReportsWhatTheMachineObserved(t *testing.T) {
	target := targetConnection{Target: targetmodel.Target{
		Label: "minimouse",
		Readiness: []targetmodel.ReadinessCheck{
			targetmodel.CapabilityReadinessCheck("codex", "Codex", targetmodel.ReadinessUnknown,
				"codex is installed, but its node runtime is not on this service's PATH (env: node: No such file or directory)",
				"fix the cause named above on that machine"),
		},
	}}
	err := ensureLaunchCapability(target, "codex --yolo")
	if err == nil {
		t.Fatal("an unrunnable agent was allowed to launch")
	}
	if !errors.Is(err, sessionsH.ErrTargetUnavailable) {
		t.Fatalf("error is not a target-unavailable refusal: %v", err)
	}
	for _, want := range []string{"Codex", "minimouse", "unknown", "node runtime", "PATH"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not carry %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "Refresh the capability probe") {
		t.Fatalf("error still names an operation the product does not offer: %v", err)
	}
}

func TestEnsureLaunchCapabilityAllowsReady(t *testing.T) {
	target := targetConnection{Target: targetmodel.Target{
		Label: "Build node",
		Readiness: []targetmodel.ReadinessCheck{
			targetmodel.CapabilityReadinessCheck("codex", "Codex", targetmodel.ReadinessReady, "codex 1", ""),
		},
	}}
	if err := ensureLaunchCapability(target, "codex --yolo"); err != nil {
		t.Fatalf("ready capability rejected: %v", err)
	}
}
