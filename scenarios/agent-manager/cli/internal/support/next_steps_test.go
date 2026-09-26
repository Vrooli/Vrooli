package support

import (
	"strings"
	"testing"
)

func TestFormatNextSteps(t *testing.T) {
	got := FormatNextSteps("agent-manager run report run-1", "", "agent-manager run tools run-1 --failed")
	if !strings.HasPrefix(got, "Next: ") || strings.Contains(got, "; ;") || !strings.Contains(got, "tools run-1 --failed") {
		t.Fatalf("unexpected next-step footer: %q", got)
	}
}
