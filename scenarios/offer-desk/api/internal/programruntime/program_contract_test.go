package programruntime_test

import (
	"os/exec"
	"testing"
)

// Exercise setpoint-read's owner-specific partial-status rule with typed fake
// bridge failures through the production Program Runtime kernel.
func TestSetpointReadTypedFailures(t *testing.T) {
	out, err := exec.Command("python3", "../../../.vrooli/program-runtime/tests/test_setpoint_read.py").CombinedOutput()
	if err != nil {
		t.Fatalf("setpoint-read program probes: %v\n%s", err, out)
	}
}
