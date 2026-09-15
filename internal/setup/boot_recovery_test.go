package setup

import (
	"errors"
	"strings"
	"testing"
)

// [REQ:BOOT-RECOVERY-001] `setup status` prints the autoheal verdict with
// every precondition when the API answers, and an explicit unknown when it
// does not; it never computes a verdict of its own.
func TestRenderBootRecoveryPrintsVerdictOrUnknown(t *testing.T) {
	var out strings.Builder
	status := BootRecoveryStatus{Status: "critical", Message: "Boot recovery would not work: unit-active failed", Remediation: "vrooli setup"}
	status.Preconditions = append(status.Preconditions, struct {
		Name   string `json:"name"`
		State  string `json:"state"`
		Reason string `json:"reason"`
	}{Name: "unit-active", State: "failed", Reason: "vrooli-runtime-supervisor.service is inactive"})
	renderBootRecovery(&out, status, nil)
	text := out.String()
	for _, want := range []string{"Boot recovery", "boot recovery: critical", "unit-active", "vrooli-runtime-supervisor.service is inactive", "remediation: vrooli setup"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}

	out.Reset()
	renderBootRecovery(&out, BootRecoveryStatus{}, errors.New("connection refused"))
	if got := out.String(); !strings.Contains(got, "boot recovery: unknown (autoheal API not reachable: connection refused)") {
		t.Fatalf("unreachable rendering = %q", got)
	}
}
