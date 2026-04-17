package teams

import (
	"flag"
	"io"
	"prompt-manager/teamconfig"
	"testing"
)

func parseTeamFlagsForTest(t *testing.T, includeDefaults bool, args ...string) teamConfigFlagSet {
	t.Helper()

	fs := flag.NewFlagSet("teams-test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	flags := registerTeamConfigFlags(fs, includeDefaults)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse(%v) error = %v", args, err)
	}

	return flags
}

func TestResolveCreateTeamConfigDefaultsToIndependentMultiProcess(t *testing.T) {
	flags := parseTeamFlagsForTest(t, true)

	runtime, coordination, execution, err := resolveCreateTeamConfig(flags)
	if err != nil {
		t.Fatalf("resolveCreateTeamConfig() error = %v", err)
	}

	if runtime.Mode != teamconfig.RuntimeModeMultiProcess {
		t.Fatalf("runtime.mode = %q, want %q", runtime.Mode, teamconfig.RuntimeModeMultiProcess)
	}
	if coordination.Pattern != teamconfig.CoordinationPatternIndependent {
		t.Fatalf("coordination.pattern = %q, want %q", coordination.Pattern, teamconfig.CoordinationPatternIndependent)
	}
	if coordination.Capabilities != teamconfig.DefaultIndependentCapabilities() {
		t.Fatalf(
			"coordination.capabilities = %+v, want %+v",
			coordination.Capabilities,
			teamconfig.DefaultIndependentCapabilities(),
		)
	}
	if execution.QueuePolicy != teamconfig.QueuePolicyBoundedParallel || execution.MaxConcurrentRuns != 2 {
		t.Fatalf("execution = %+v, want bounded-parallel/2", execution)
	}
}

func TestResolveCreateTeamConfigPromotesSingleProcessToLeaderLed(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		true,
		"--runtime-mode=single-process",
		"--lead-agent-id=director",
	)

	runtime, coordination, execution, err := resolveCreateTeamConfig(flags)
	if err != nil {
		t.Fatalf("resolveCreateTeamConfig() error = %v", err)
	}

	if runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("runtime.mode = %q, want %q", runtime.Mode, teamconfig.RuntimeModeSingleProcess)
	}
	if coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("coordination.pattern = %q, want %q", coordination.Pattern, teamconfig.CoordinationPatternLeaderLed)
	}
	if coordination.LeadAgentID != "director" {
		t.Fatalf("coordination.leadAgentId = %q, want %q", coordination.LeadAgentID, "director")
	}
	if coordination.MessagingMode != teamconfig.MessagingModeInSession {
		t.Fatalf("coordination.messagingMode = %q, want %q", coordination.MessagingMode, teamconfig.MessagingModeInSession)
	}
	if execution.QueuePolicy != teamconfig.QueuePolicySerialized || execution.MaxConcurrentRuns != 1 {
		t.Fatalf("execution = %+v, want serialized/1", execution)
	}
}

func TestResolveUpdatedTeamConfigPromotesRuntimeTransitions(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		false,
		"--runtime-mode=single-process",
		"--lead-agent-id=director",
	)

	current := TeamDetails{
		Team: Team{
			Runtime: Runtime{Mode: teamconfig.RuntimeModeMultiProcess},
			Coordination: Coordination{
				Pattern:       teamconfig.CoordinationPatternIndependent,
				ReportingMode: teamconfig.ReportingModeNone,
				MessagingMode: teamconfig.MessagingModeDisabled,
				Capabilities:  teamconfig.DefaultIndependentCapabilities(),
			},
			Execution: Execution{
				QueuePolicy:       teamconfig.QueuePolicyBoundedParallel,
				MaxConcurrentRuns: 3,
			},
		},
	}

	runtime, coordination, execution, err := resolveUpdatedTeamConfig(current, flags)
	if err != nil {
		t.Fatalf("resolveUpdatedTeamConfig() error = %v", err)
	}

	if runtime == nil || runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("runtime = %+v, want single-process", runtime)
	}
	if coordination == nil || coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("coordination = %+v, want leader-led", coordination)
	}
	if coordination.LeadAgentID != "director" {
		t.Fatalf("coordination.leadAgentId = %q, want %q", coordination.LeadAgentID, "director")
	}
	if execution == nil || execution.QueuePolicy != teamconfig.QueuePolicySerialized || execution.MaxConcurrentRuns != 1 {
		t.Fatalf("execution = %+v, want serialized/1", execution)
	}
}

func TestResolveUpdatedTeamConfigAppliesCapabilityOverrides(t *testing.T) {
	flags := parseTeamFlagsForTest(
		t,
		false,
		"--show-org-context=false",
		"--require-handoff=false",
	)

	current := TeamDetails{
		Team: Team{
			Runtime: Runtime{Mode: teamconfig.RuntimeModeMultiProcess},
			Coordination: Coordination{
				Pattern:       teamconfig.CoordinationPatternPeer,
				ReportingMode: teamconfig.ReportingModeOrgChart,
				MessagingMode: teamconfig.MessagingModeAsyncInbox,
				Capabilities:  teamconfig.DefaultPeerCapabilities(),
			},
			Execution: Execution{
				QueuePolicy:       teamconfig.QueuePolicyBoundedParallel,
				MaxConcurrentRuns: 2,
			},
		},
	}

	_, coordination, _, err := resolveUpdatedTeamConfig(current, flags)
	if err != nil {
		t.Fatalf("resolveUpdatedTeamConfig() error = %v", err)
	}

	if coordination == nil {
		t.Fatal("expected coordination update")
	}
	if coordination.Capabilities.ShowOrgContext {
		t.Fatal("expected showOrgContext override to be false")
	}
	if coordination.Capabilities.RequireHandoff {
		t.Fatal("expected requireHandoff override to be false")
	}
}
