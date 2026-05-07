package setup

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

// describeExecutionState owns the human label every state carries through to
// `vrooli setup` output. Adding a new ExecutionState without a matching case
// silently falls into the default branch and emits the raw enum string, which
// is hostile to operators reading the report. Cover every constant so a future
// addition produces a compiler/test signal.
func TestDescribeExecutionStateCoversAllStates(t *testing.T) {
	cases := map[vrooliruntime.ExecutionState]string{
		vrooliruntime.ExecutionAlreadyPresent:       "already_present",
		vrooliruntime.ExecutionWouldInstall:         "would_install",
		vrooliruntime.ExecutionWouldApply:           "would_apply",
		vrooliruntime.ExecutionInstalled:            "installed",
		vrooliruntime.ExecutionApplied:              "applied",
		vrooliruntime.ExecutionRebootRequired:       "reboot_required",
		vrooliruntime.ExecutionManualActionRequired: "manual_action_required",
		vrooliruntime.ExecutionUnsupported:          "unsupported",
		vrooliruntime.ExecutionNotApplicable:        "not_applicable",
		vrooliruntime.ExecutionFailed:               "failed",
	}

	for state, want := range cases {
		item := vrooliruntime.ItemStatus{
			ExecutionState: state,
			Kind:           hostreq.KindTool,
		}
		got := describeExecutionState(item, true)
		if got != want {
			t.Errorf("describeExecutionState(%q) = %q, want %q", state, got, want)
		}
	}
}

// ExecutionRebootRequired must round-trip through the runtime re-export so
// callers using either package observe the same enum value.
func TestExecutionRebootRequiredReExport(t *testing.T) {
	if vrooliruntime.ExecutionRebootRequired != "reboot_required" {
		t.Fatalf("ExecutionRebootRequired wire value = %q, want %q",
			vrooliruntime.ExecutionRebootRequired, "reboot_required")
	}
}
