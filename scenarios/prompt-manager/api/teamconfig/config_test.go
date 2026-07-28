package teamconfig

import "testing"

func TestBuildCoordinationPresetIndependent(t *testing.T) {
	got, err := BuildCoordinationPreset(CoordinationPatternIndependent, RuntimeModeMultiProcess, "")
	if err != nil {
		t.Fatalf("BuildCoordinationPreset() error = %v", err)
	}

	if got.Pattern != CoordinationPatternIndependent {
		t.Fatalf("pattern = %q, want %q", got.Pattern, CoordinationPatternIndependent)
	}
	if got.ReportingMode != ReportingModeNone {
		t.Fatalf("reportingMode = %q, want %q", got.ReportingMode, ReportingModeNone)
	}
	if got.MessagingMode != MessagingModeDisabled {
		t.Fatalf("messagingMode = %q, want %q", got.MessagingMode, MessagingModeDisabled)
	}
	if got.Capabilities != DefaultIndependentCapabilities() {
		t.Fatalf("capabilities = %+v, want %+v", got.Capabilities, DefaultIndependentCapabilities())
	}
}

func TestBuildCoordinationPresetLeaderLedSingleProcess(t *testing.T) {
	got, err := BuildCoordinationPreset(CoordinationPatternLeaderLed, RuntimeModeSingleProcess, "director")
	if err != nil {
		t.Fatalf("BuildCoordinationPreset() error = %v", err)
	}

	if got.LeadAgentID != "director" {
		t.Fatalf("leadAgentId = %q, want %q", got.LeadAgentID, "director")
	}
	if got.MessagingMode != MessagingModeInSession {
		t.Fatalf("messagingMode = %q, want %q", got.MessagingMode, MessagingModeInSession)
	}
	if got.Capabilities != DefaultLeaderLedCapabilities(RuntimeModeSingleProcess) {
		t.Fatalf(
			"capabilities = %+v, want %+v",
			got.Capabilities,
			DefaultLeaderLedCapabilities(RuntimeModeSingleProcess),
		)
	}
}

func TestBuildCoordinationPresetRejectsLeaderLedWithoutLead(t *testing.T) {
	if _, err := BuildCoordinationPreset(CoordinationPatternLeaderLed, RuntimeModeSingleProcess, ""); err == nil {
		t.Fatal("expected error when leadAgentId is missing")
	}
}

func TestBuildExecutionConfigSingleProcessForcesSerialized(t *testing.T) {
	got, err := BuildExecutionConfig(RuntimeModeSingleProcess, QueuePolicyBoundedParallel, 4)
	if err != nil {
		t.Fatalf("BuildExecutionConfig() error = %v", err)
	}

	if got.QueuePolicy != QueuePolicySerialized {
		t.Fatalf("queuePolicy = %q, want %q", got.QueuePolicy, QueuePolicySerialized)
	}
	if got.MaxConcurrentRuns != 1 {
		t.Fatalf("maxConcurrentRuns = %d, want 1", got.MaxConcurrentRuns)
	}
}

func TestBuildExecutionConfigBoundedParallelUsesMinimumConcurrency(t *testing.T) {
	got, err := BuildExecutionConfig(RuntimeModeMultiProcess, QueuePolicyBoundedParallel, 1)
	if err != nil {
		t.Fatalf("BuildExecutionConfig() error = %v", err)
	}

	if got.QueuePolicy != QueuePolicyBoundedParallel {
		t.Fatalf("queuePolicy = %q, want %q", got.QueuePolicy, QueuePolicyBoundedParallel)
	}
	if got.MaxConcurrentRuns != 2 {
		t.Fatalf("maxConcurrentRuns = %d, want 2", got.MaxConcurrentRuns)
	}
}

func TestValidateAcceptsIndependentBoundedParallelTeam(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeMultiProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternIndependent,
			ReportingMode: ReportingModeNone,
			MessagingMode: MessagingModeDisabled,
			Capabilities: Capabilities{
				ShowOrgContext:           false,
				InjectInbox:              false,
				AllowPeerTriggers:        false,
				ShowTaskBoardGuidance:    true,
				ShowDecisionLogGuidance:  true,
				ShowKnowledgeLogGuidance: true,
				RequireHandoff:           true,
			},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicyBoundedParallel,
			MaxConcurrentRuns: 2,
		},
		DecisionMode: DecisionModeYolo,
	}

	if err := Validate(contract); err != nil {
		t.Fatalf("expected valid contract, got error: %v", err)
	}
}

func TestValidateRejectsLeadOnIndependentTeam(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeMultiProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternIndependent,
			LeadAgentID:   "director",
			ReportingMode: ReportingModeNone,
			MessagingMode: MessagingModeDisabled,
			Capabilities:  Capabilities{},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicyBoundedParallel,
			MaxConcurrentRuns: 2,
		},
		DecisionMode: DecisionModeYolo,
	}

	if err := Validate(contract); err == nil {
		t.Fatal("expected validation error for independent team with leadAgentId")
	}
}

func TestValidateRejectsSingleProcessWithoutLeaderLedPattern(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeSingleProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternPeer,
			ReportingMode: ReportingModeOrgChart,
			MessagingMode: MessagingModeInSession,
			Capabilities: Capabilities{
				ShowOrgContext: true,
			},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicySerialized,
			MaxConcurrentRuns: 1,
		},
		DecisionMode: DecisionModeApproval,
	}

	if err := Validate(contract); err == nil {
		t.Fatal("expected validation error for single-process non-leader-led contract")
	}
}

func TestValidateRejectsInboxInjectionWithoutAsyncMessaging(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeMultiProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternPeer,
			ReportingMode: ReportingModeOrgChart,
			MessagingMode: MessagingModeDisabled,
			Capabilities: Capabilities{
				ShowOrgContext: true,
				InjectInbox:    true,
			},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicyBoundedParallel,
			MaxConcurrentRuns: 2,
		},
		DecisionMode: DecisionModeYolo,
	}

	if err := Validate(contract); err == nil {
		t.Fatal("expected validation error for injectInbox without async-inbox messaging")
	}
}

func TestValidateFindingsReportsIndependentDefectsTogether(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: "unsupported"},
		Coordination: Coordination{
			Pattern:       CoordinationPatternIndependent,
			LeadAgentID:   "director",
			ReportingMode: ReportingModeLeader,
			MessagingMode: MessagingModeInSession,
			Capabilities: Capabilities{
				InjectInbox:       true,
				AllowPeerTriggers: true,
			},
		},
		Execution:    Execution{QueuePolicy: QueuePolicySerialized, MaxConcurrentRuns: 2},
		DecisionMode: "unsupported",
	}
	findings := ValidateFindings(contract)
	if len(findings) < 6 {
		t.Fatalf("findings = %d, want multiple independent defects: %+v", len(findings), findings)
	}
	if err := Validate(contract); err == nil {
		t.Fatal("Validate must retain strict write-path rejection")
	}
}

func TestHelpersReflectResolvedPolicy(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeMultiProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternPeer,
			ReportingMode: ReportingModeOrgChart,
			MessagingMode: MessagingModeAsyncInbox,
			Capabilities: Capabilities{
				ShowOrgContext:           true,
				InjectInbox:              true,
				AllowPeerTriggers:        true,
				ShowTaskBoardGuidance:    true,
				ShowDecisionLogGuidance:  true,
				ShowKnowledgeLogGuidance: true,
				RequireHandoff:           true,
			},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicyBoundedParallel,
			MaxConcurrentRuns: 3,
		},
		DecisionMode: DecisionModeYolo,
	}

	if got := CoordinationSkillID(contract); got != "team-coordination-peer" {
		t.Fatalf("unexpected coordination skill id: %s", got)
	}
	if !MessagingEnabled(contract) {
		t.Fatal("expected messaging to be enabled")
	}
	if !ShouldInjectInbox(contract) {
		t.Fatal("expected inbox injection to be enabled")
	}
	if !ShouldShowOrgContext(contract) {
		t.Fatal("expected org context to be shown")
	}
	if !ShouldShowTaskBoardGuidance(contract) {
		t.Fatal("expected task board guidance")
	}
	if !ShouldShowDecisionLogGuidance(contract) {
		t.Fatal("expected decision log guidance")
	}
	if !ShouldShowKnowledgeLogGuidance(contract) {
		t.Fatal("expected knowledge log guidance")
	}
	if !RequiresHandoff(contract) {
		t.Fatal("expected handoff requirement")
	}
	if !AllowsPeerTriggers(contract) {
		t.Fatal("expected peer triggers to be allowed")
	}
	if UsesSingleProcessInterop(contract) {
		t.Fatal("did not expect single-process interop")
	}
	if TeamTriggerTargetsLead(contract) {
		t.Fatal("did not expect team trigger to target only the lead")
	}
	if UsesClaudeCodeRunner(contract) {
		t.Fatal("did not expect Claude Code runner for multi-process team")
	}
}

func TestHelpersReflectLeaderLedSingleProcessPolicy(t *testing.T) {
	contract := Contract{
		Runtime: Runtime{Mode: RuntimeModeSingleProcess},
		Coordination: Coordination{
			Pattern:       CoordinationPatternLeaderLed,
			LeadAgentID:   "director",
			ReportingMode: ReportingModeLeader,
			MessagingMode: MessagingModeInSession,
			Capabilities: Capabilities{
				ShowOrgContext:           true,
				InjectInbox:              false,
				AllowPeerTriggers:        false,
				ShowTaskBoardGuidance:    true,
				ShowDecisionLogGuidance:  true,
				ShowKnowledgeLogGuidance: true,
				RequireHandoff:           true,
			},
		},
		Execution: Execution{
			QueuePolicy:       QueuePolicySerialized,
			MaxConcurrentRuns: 1,
		},
		DecisionMode: DecisionModeApproval,
	}

	if err := Validate(contract); err != nil {
		t.Fatalf("expected valid single-process leader-led contract, got error: %v", err)
	}
	if got := CoordinationSkillID(contract); got != "team-coordination-leader-led" {
		t.Fatalf("unexpected coordination skill id: %s", got)
	}
	if !UsesSingleProcessInterop(contract) {
		t.Fatal("expected single-process interop")
	}
	if !TeamTriggerTargetsLead(contract) {
		t.Fatal("expected team trigger to target the lead")
	}
	if !UsesClaudeCodeRunner(contract) {
		t.Fatal("expected Claude Code runner")
	}
	if MessagingUsesAsyncInbox(contract) {
		t.Fatal("did not expect async inbox messaging")
	}
	if !MessagingUsesInSession(contract) {
		t.Fatal("expected in-session messaging")
	}
	if ShouldInjectInbox(contract) {
		t.Fatal("did not expect inbox injection")
	}
}
