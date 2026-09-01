package main

import (
	"errors"
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
