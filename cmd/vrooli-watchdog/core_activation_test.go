package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootWorkerModesCannotBeCombinedWithPressureReporting(t *testing.T) {
	for _, args := range [][]string{{"--core-update-capability", "--report-only"}, {"--reconcile-core-updates", "--report-only"}, {"--core-update-execute", "id", "store", "--report-only"}, {"--enroll-core-updates", "--report-only"}, {"--submit-core-update", "image", "root", "--report-only"}} {
		var out, stderr bytes.Buffer
		if handled, code := runCoreActivationCommand(args, &out, &stderr); !handled || code != 2 || out.Len() != 0 {
			t.Fatalf("mixed invocation reached root effects: %v %t %d", args, handled, code)
		}
	}
	var out, stderr bytes.Buffer
	if handled, _ := runCoreActivationCommand([]string{"--report-only"}, &out, &stderr); handled {
		t.Fatal("pressure reporting entered activation mode")
	}
}

func TestRefusalReasonStaysWithinTheInstallerReceiptBound(t *testing.T) {
	short := "core activation: another activation remains unresolved"
	if boundedRefusalReason(short) != short {
		t.Fatal("a short refusal reason was altered")
	}
	var long strings.Builder
	long.WriteString("candidate changes the persistent-state contract:")
	for long.Len() < 20<<10 {
		long.WriteString("\n  - owner resources: stored type internal/resources.ManagedServiceState field Example added (json \"example\", type string)")
	}
	bounded := boundedRefusalReason(long.String())
	if len(bounded) > 7<<10 || !strings.HasSuffix(bounded, "compare them in full)") || strings.Contains(bounded[:len(bounded)-200], "(json \"example\", type stri\n") {
		t.Fatalf("bounded reason is %d bytes or cut mid-entry", len(bounded))
	}
}

func TestOperatorModesRefuseAgentSessionsBeforeAnyEffect(t *testing.T) {
	old := operatorCallerEvidence
	t.Cleanup(func() { operatorCallerEvidence = old })
	operatorCallerEvidence = func() string { return "its process runs in the agent session scope vrooli-agent-codex-1.scope" }
	for _, args := range [][]string{{"--enroll-core-updates"}, {"--disable-core-updates"}, {"--retry-core-update", "/candidate", "/repo"}} {
		var out, stderr bytes.Buffer
		handled, code := runCoreActivationCommand(args, &out, &stderr)
		if !handled || code != 1 || out.Len() != 0 || !strings.Contains(stderr.String(), "operator decision") || !strings.Contains(stderr.String(), "vrooli-agent-codex-1") {
			t.Fatalf("%v from an agent session was not refused first: %t %d %q", args, handled, code, stderr.String())
		}
	}
}
