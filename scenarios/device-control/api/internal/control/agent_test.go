package control

import (
	"context"
	"encoding/json"
	"testing"

	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestAgentRefusesWithoutSkillAndDoesNotPromoteUnassertedRun(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", false)
	require.ErrorContains(t, err, "prompt-manager device-control skill is unavailable")

	run, err := svc.StartAgent(context.Background(), "observe the screen", "fake", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	_, err = svc.PromoteAgent(run.ID)
	require.ErrorContains(t, err, "terminal outcome assertion", "a successful observation alone does not verify a reusable outcome")
}

func TestPromoteAgentRejectsUnverifiedModelAssertion(t *testing.T) {
	svc, _ := testService(t)
	svc.agents["unverified"] = AgentRun{ID: "unverified", State: "completed", PlannedSteps: []Step{{ID: "iteration-1", Kind: "assert-foreground", Target: "com.example"}}, Result: RunResult{Disposition: "passed", Chapters: []Chapter{{ID: "iteration-1", Title: "Agent iteration-1", Disposition: "passed", Message: "planner reported success"}}}}
	_, err := svc.PromoteAgent("unverified")
	require.ErrorContains(t, err, "verified flow chapter")
	candidate := svc.agents["unverified"]
	candidate.Result.Chapters[0].Title = "assert-foreground"
	svc.agents["unverified"] = candidate
	_, err = svc.PromoteAgent("unverified")
	require.ErrorContains(t, err, "independently verified goal")
}

func TestFramelessAgentUsesTypedStateAndDryRunDoesNotActuate(t *testing.T) {
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("frameless", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "frameless", StrategyID: "frameless", Transport: "frameless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})

	run, err := svc.StartAgentWithOptions(context.Background(), "pause whatever is playing", "frameless", "operator", true, true)
	require.NoError(t, err)
	require.Equal(t, "planned", run.State)
	require.Equal(t, "planned", run.Result.Disposition)
	require.True(t, run.DryRun)
	require.Empty(t, strategyUnderTest.commands)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "no actuation was sent")
	require.Equal(t, "media-pause", run.ProposedStep.Kind)
}

func TestAgentPlannerReceivesSemanticSnapshotAndActionHistory(t *testing.T) {
	fixture := &semanticAgentFixture{semanticSnapshotFixture: &semanticSnapshotFixture{Strategy: fakes.New("semantic-agent", strategy.StatusAvailable, strategy.CapSemanticTree, strategy.CapScreenshot, strategy.CapInput)}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	planner := &capturingAgentPlanner{}
	svc.SetAgentPlanner(planner)
	run, err := svc.StartAgentWithOptions(context.Background(), "open settings", "target", "test", true, false)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.False(t, run.GoalVerified)
	require.Equal(t, "unverified", run.Result.Chapters[len(run.Result.Chapters)-1].Disposition)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "no independent terminal outcome assertion")
	require.Equal(t, 2, run.PlannerCalls)
	require.Equal(t, 1, run.ActionAttempts)
	require.Equal(t, 1, run.ActionCompletions)
	require.NotZero(t, run.CreatedAt)
	require.NotZero(t, run.CompletedAt)
	require.GreaterOrEqual(t, run.DurationMS, int64(0))
	require.Len(t, run.InferenceCalls, 2)
	require.Equal(t, "fixture-model", run.InferenceCalls[0].Model)
	require.Equal(t, int64(41), *run.InferenceCalls[0].InputTokens)
	require.Equal(t, "USAGE_PROVENANCE_GATEWAY_ESTIMATE", run.InferenceCalls[0].CostProvenance)
	encoded, err := json.Marshal(run)
	require.NoError(t, err)
	var response map[string]any
	require.NoError(t, json.Unmarshal(encoded, &response))
	require.Equal(t, float64(2), response["planner_calls"])
	require.Equal(t, float64(1), response["action_attempts"])
	require.Equal(t, "fixture-model", response["inference_calls"].([]any)[0].(map[string]any)["model"])
	require.Len(t, planner.worlds, 2)
	require.NotNil(t, planner.worlds[0].Semantics)
	require.NotContains(t, planner.worlds[0].StepKinds, "observe", "the planner receives no image to inspect")
	require.False(t, planner.worlds[0].FrameAvailable)
	require.Equal(t, "Settings", planner.worlds[0].Semantics.Elements[0].Text)
	require.Len(t, planner.worlds[1].History, 1)
	require.Equal(t, "key", planner.worlds[1].History[0].StepKind)
	require.Equal(t, "HOME", planner.worlds[1].History[0].Target)
	require.Equal(t, "completed", planner.worlds[1].History[0].Outcome)
	require.NotEmpty(t, planner.worlds[1].History[0].BeforeSnapshotHash)
	require.Equal(t, planner.worlds[1].History[0].BeforeSnapshotHash, planner.worlds[1].History[0].AfterSnapshotHash)
	require.Equal(t, []string{"Settings"}, planner.worlds[1].History[0].VisibleControls)
	require.NotNil(t, planner.worlds[1].History[0].SnapshotChanged)
	require.False(t, *planner.worlds[1].History[0].SnapshotChanged)
	require.Empty(t, svc.ListLiveSessions())
}

func TestAgentSemanticTargetRunsThroughFlowOwner(t *testing.T) {
	fixture := &semanticAgentFixture{semanticSnapshotFixture: &semanticSnapshotFixture{Strategy: fakes.New("semantic-agent", strategy.StatusAvailable, strategy.CapSemanticTree, strategy.CapScreenshot, strategy.CapInput)}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "semantic-target", Action: "Settings"}, {GoalMet: true}}})
	run, err := svc.StartAgent(context.Background(), "open settings", "target", "test", true)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State, "%+v", run.Result.Chapters)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Len(t, fixture.Actuations, 1)
	require.NotNil(t, fixture.Actuations[0].Pointer)
	require.Equal(t, "semantic-target", fixture.Actuations[0].Pointer.Kind)
	require.Equal(t, "Settings", run.PlannedSteps[0].Target)
	_, err = svc.PromoteAgent(run.ID)
	require.ErrorContains(t, err, "not eligible for promotion")
}

func TestAgentLaunchUsesTypedAppLifecycleAndVerifiesForeground(t *testing.T) {
	const packageName = "com.google.android.calculator"
	fixture := &agentLifecycleStateFixture{Strategy: fakes.New("agent-launch", strategy.StatusAvailable, strategy.CapAppLifecycle), foregroundOnLaunch: packageName}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "agent-launch", StrategyID: "agent-launch", Transport: "agent-launch", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "launch", Action: packageName}, {GoalMet: true}}})

	run, err := svc.StartAgent(context.Background(), "open the calculator", "agent-launch", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Len(t, run.PlannedSteps, 1)
	require.Equal(t, packageName, run.PlannedSteps[0].Target)
	require.Len(t, fixture.requests, 1)
	require.Equal(t, "launch", fixture.requests[0].Operation)
	require.Equal(t, packageName, fixture.requests[0].Package)
	require.Equal(t, 1, run.ActionCompletions)
}

func TestAgentLaunchFailsWhenForegroundPackageDoesNotMatch(t *testing.T) {
	const packageName = "com.google.android.calculator"
	fixture := &agentLifecycleStateFixture{Strategy: fakes.New("agent-launch-mismatch", strategy.StatusAvailable, strategy.CapAppLifecycle), foregroundOnLaunch: "com.example.other"}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "agent-launch-mismatch", StrategyID: "agent-launch-mismatch", Transport: "agent-launch-mismatch", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "launch", Action: packageName}, {GoalMet: true}}})

	run, err := svc.StartAgent(context.Background(), "open the calculator", "agent-launch-mismatch", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "failed", run.State)
	require.Equal(t, "failed", run.Result.Disposition)
	require.Equal(t, 1, run.ActionAttempts)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "was not observed in the foreground")
}

func TestAgentVerifiesOperatorForegroundPackageExpectation(t *testing.T) {
	const packageName = "com.vrooli.generated.app.e7candidate"
	fixture := &agentLifecycleStateFixture{Strategy: fakes.New("agent-assert-foreground", strategy.StatusAvailable, strategy.CapAppLifecycle), foregroundOnLaunch: packageName}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "agent-assert-foreground", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "launch", Action: packageName}, {GoalMet: true}}})

	run, err := svc.StartAgentWithAssertions(context.Background(), "launch and verify the app", "agent-assert-foreground", "operator", true, false, internalflows.DefaultAgentPolicy(), false, AgentOutcomeAssertions{ForegroundPackage: packageName})
	require.NoError(t, err)
	require.Equal(t, "completed", run.State)
	require.Equal(t, "passed", run.Result.Disposition)
	require.True(t, run.GoalVerified)
	require.Equal(t, packageName, run.ExpectedForegroundPackage)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, packageName)
}

func TestAgentFailsWhenOperatorForegroundPackageExpectationIsNotObserved(t *testing.T) {
	const expected = "com.vrooli.generated.app.e7candidate"
	fixture := &agentLifecycleStateFixture{Strategy: fakes.New("agent-assert-foreground-mismatch", strategy.StatusAvailable, strategy.CapAppLifecycle), foreground: "com.example.other"}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "agent-assert-foreground-mismatch", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{GoalMet: true}}})

	run, err := svc.StartAgentWithAssertions(context.Background(), "launch and verify the app", "agent-assert-foreground-mismatch", "operator", true, false, internalflows.DefaultAgentPolicy(), false, AgentOutcomeAssertions{ForegroundPackage: expected})
	require.NoError(t, err)
	require.Equal(t, "failed", run.State)
	require.Equal(t, "failed", run.Result.Disposition)
	require.False(t, run.GoalVerified)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "does not match expected package")
}

func TestAgentRejectsInvalidOrAmbiguousOutcomeAssertions(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.StartAgentWithAssertions(context.Background(), "verify app", "fake", "operator", true, false, internalflows.DefaultAgentPolicy(), false, AgentOutcomeAssertions{ForegroundPackage: "not a package"})
	require.ErrorContains(t, err, "not a valid Android package identity")
	_, err = svc.StartAgentWithAssertions(context.Background(), "verify app", "fake", "operator", true, false, internalflows.DefaultAgentPolicy(), false, AgentOutcomeAssertions{SelectedControl: "Home", ForegroundPackage: "com.example.app"})
	require.ErrorContains(t, err, "only one independently observed terminal outcome assertion")
}

func TestAgentPlannerOnlyReceivesExecutableStepKinds(t *testing.T) {
	fixture := fakes.New("agent-step-contract", strategy.StatusAvailable, strategy.CapInput, strategy.CapMedia, strategy.CapAppLifecycle, strategy.CapProperty, strategy.CapScreenshot)
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "agent-step-contract", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	planner := &sequenceAgentPlanner{plans: []internalflows.AgentPlan{{GoalMet: true}}}
	svc.SetAgentPlanner(planner)

	run, err := svc.StartAgent(context.Background(), "check current state", "agent-step-contract", "operator", true)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Len(t, planner.worlds, 1)
	for _, supported := range []string{"key", "media-pause", "launch", "property-get", "property-set"} {
		require.Contains(t, planner.worlds[0], supported)
	}
	for _, unsupported := range []string{"tap", "swipe", "text", "observe"} {
		require.NotContains(t, planner.worlds[0], unsupported)
	}
}

func TestAgentDryRunReturnsFirstProposalWithoutActuation(t *testing.T) {
	fixture := &semanticAgentFixture{semanticSnapshotFixture: &semanticSnapshotFixture{Strategy: fakes.New("semantic-agent-dry", strategy.StatusAvailable, strategy.CapSemanticTree, strategy.CapScreenshot, strategy.CapInput)}}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	planner := &sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "semantic-target", Action: "Settings"}, {GoalMet: true}}}
	svc.SetAgentPlanner(planner)
	run, err := svc.StartAgentWithOptions(context.Background(), "open settings", "target", "test", true, true)
	require.NoError(t, err)
	require.Equal(t, "planned", run.State)
	require.False(t, run.GoalVerified)
	require.Equal(t, 1, planner.calls)
	require.Equal(t, "Settings", run.ProposedStep.Target)
	require.Empty(t, fixture.Actuations)
	require.Empty(t, svc.ListLiveSessions())
}

func TestAgentVerifiesOperatorSelectedControlExpectation(t *testing.T) {
	selected, unselected := true, false
	for _, tc := range []struct {
		name     string
		selected *bool
		verified bool
	}{
		{name: "selected", selected: &selected, verified: true},
		{name: "not selected", selected: &unselected},
		{name: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := &semanticAgentFixture{semanticSnapshotFixture: &semanticSnapshotFixture{Strategy: fakes.New("selected-agent", strategy.StatusAvailable, strategy.CapSemanticTree, strategy.CapInput), selected: tc.selected}}
			svc := New(strategyregistry.New(fixture))
			svc.devices.Upsert(devicedomain.Record{ID: "target", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
			svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{GoalMet: true}}})
			run, err := svc.StartAgentWithOutcome(context.Background(), "open settings", "target", "test", true, false, internalflows.DefaultAgentPolicy(), false, "Settings")
			require.NoError(t, err)
			require.Equal(t, tc.verified, run.GoalVerified)
			if tc.verified {
				require.Equal(t, "completed", run.State)
			} else {
				require.Equal(t, "failed", run.State)
			}
		})
	}
	snapshot := &strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{{Text: "Settings", Selected: &selected}, {Text: "Settings", Selected: &selected}}}
	require.ErrorContains(t, verifySelectedSemanticControl(snapshot, "Settings"), "ambiguous")
	snapshot.Truncated = true
	require.ErrorContains(t, verifySelectedSemanticControl(snapshot, "Settings"), "complete")
}

func TestAgentBlocksImmediateRepeatedSemanticActionWithoutObservedChange(t *testing.T) {
	unchanged, changed := false, true
	plan := internalflows.AgentPlan{StepKind: "semantic-target", Action: "Settings"}
	history := []internalflows.AgentHistory{{StepKind: "semantic-target", Target: "Settings", SnapshotChanged: &unchanged}}
	require.True(t, shouldBlockRepeatedSemanticAction(history, plan))
	history[0].SnapshotChanged = &changed
	require.False(t, shouldBlockRepeatedSemanticAction(history, plan))
	history[0].SnapshotChanged = nil
	require.False(t, shouldBlockRepeatedSemanticAction(history, plan))
	plan.StepKind = "key"
	require.False(t, shouldBlockRepeatedSemanticAction(history, plan))
}

func TestAgentSemanticTargetRequiresOneActionableUntruncatedControl(t *testing.T) {
	snapshot := &strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{{Text: "Settings", Enabled: true, Clickable: true}}}
	require.NoError(t, validateAgentSemanticTarget(snapshot, "Settings"))
	snapshot.Elements = append(snapshot.Elements, snapshot.Elements[0])
	require.Error(t, validateAgentSemanticTarget(snapshot, "Settings"))
	snapshot.Elements = snapshot.Elements[:1]
	snapshot.Truncated = true
	require.Error(t, validateAgentSemanticTarget(snapshot, "Settings"))
	snapshot.Truncated = false
	snapshot.Elements[0].Editable = true
	require.Error(t, validateAgentSemanticTarget(snapshot, "Settings"))
}

func TestAgentSemanticTargetPolicyChecksControlLabelsWhenSelectedByID(t *testing.T) {
	snapshot := &strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{{Text: "Uninstall app", ResourceID: "app:id/button1", Enabled: true, Clickable: true}}}
	policy := internalflows.DefaultAgentPolicy()
	require.ErrorContains(t, semanticTargetPolicyViolation(policy, snapshot, "app:id/button1", false), "destructive")
	policy.AllowDestructiveActions = true
	policy.RequireConfirmationFor = []string{"destructive"}
	require.ErrorContains(t, semanticTargetPolicyViolation(policy, snapshot, "app:id/button1", false), "confirmation required")
	require.NoError(t, semanticTargetPolicyViolation(policy, snapshot, "app:id/button1", true))
}

func TestAgentUsesBoundedGoalLoopAndRecordsEachWorldModel(t *testing.T) {
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("frameless-loop", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "frameless-loop", StrategyID: "frameless-loop", Transport: "frameless-loop", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})
	planner := &sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "media-pause", Action: "pause"}, {GoalMet: true}}}
	svc.SetAgentPlanner(planner)

	run, err := svc.StartAgentWithOptions(context.Background(), "pause and verify", "frameless-loop", "operator", true, false)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Equal(t, 2, planner.calls)
	require.Contains(t, run.Result.Chapters[0].ID, "world-model-1")
	require.Contains(t, run.Result.Chapters[1].ID, "iteration-1")
	require.Contains(t, run.Result.Chapters[2].ID, "world-model-2")
	require.Contains(t, run.Result.Chapters[3].ID, "goal-2")
	require.Len(t, strategyUnderTest.commands, 1)
}

func TestAgentTerminatesAtBoundedStepBudget(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-FLOW-07]
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("budget", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "budget", StrategyID: "budget", Transport: "budget", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})
	plans := make([]internalflows.AgentPlan, maxAgentIterations+1)
	for i := range plans {
		plans[i] = internalflows.AgentPlan{StepKind: "media-pause", Action: "pause"}
	}
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: plans})

	run, err := svc.StartAgentWithOptions(context.Background(), "unfamiliar task", "budget", "operator", true, false)
	require.NoError(t, err)
	require.Equal(t, "failed", run.State)
	require.Equal(t, "failed", run.Result.Disposition)
	require.Equal(t, "budget", run.Result.Chapters[len(run.Result.Chapters)-1].ID)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "step budget exhausted")
	require.Len(t, run.PlannedSteps, maxAgentIterations)
	require.Len(t, strategyUnderTest.commands, maxAgentIterations)
}

func TestAgentRejectsUndeclaredPlannerStepBeforeActuation(t *testing.T) { // [REQ:DEVICECONTROL-EVERYWHERE-FLOW-10]
	strategyUnderTest := &framelessAgentStrategy{Strategy: fakes.New("binding-guard", strategy.StatusAvailable, strategy.CapMedia)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "binding-guard", StrategyID: "binding-guard", Transport: "binding-guard", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapMedia, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "undeclared-operation", Action: "host-shell"}}})

	run, err := svc.StartAgentWithOptions(context.Background(), "run undeclared operation", "binding-guard", "operator", true, false)
	require.NoError(t, err)
	require.Equal(t, "failed", run.State)
	require.Equal(t, "failed", run.Result.Disposition)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "no actuation executed")
	require.Empty(t, strategyUnderTest.commands)
}

func TestAgentRequiresOutOfBandConfirmationForRiskyPlannerStep(t *testing.T) {
	strategyUnderTest := &lifecycleAgentStrategy{Strategy: fakes.New("confirmation-guard", strategy.StatusAvailable, strategy.CapAppLifecycle)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "confirmation-guard", StrategyID: "confirmation-guard", Transport: "confirmation-guard", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "share", Action: "send"}}})

	run, err := svc.StartAgentWithPolicy(context.Background(), "share this", "confirmation-guard", "operator", true, false, internalflows.DefaultAgentPolicy(), false)
	require.NoError(t, err)
	require.Equal(t, "awaiting_confirmation", run.State)
	require.Equal(t, "confirmation_required", run.Result.Disposition)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "no actuation executed")
	require.Empty(t, run.PlannedSteps)
}

func TestAgentRejectsMalformedGoalMetBeforeRiskClassification(t *testing.T) {
	strategyUnderTest := &lifecycleAgentStrategy{Strategy: fakes.New("malformed-goal-risk", strategy.StatusAvailable, strategy.CapAppLifecycle)}
	svc := New(strategyregistry.New(strategyUnderTest))
	svc.devices.Upsert(devicedomain.Record{ID: "malformed-goal-risk", StrategyID: "malformed-goal-risk", Transport: "malformed-goal-risk", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{GoalMet: true, StepKind: "share", Action: "send"}}})

	run, err := svc.StartAgentWithPolicy(context.Background(), "share this", "malformed-goal-risk", "operator", true, false, internalflows.DefaultAgentPolicy(), false)
	require.NoError(t, err)
	require.Equal(t, "failed", run.State)
	require.Equal(t, "failed", run.Result.Disposition)
	require.Contains(t, run.Result.Chapters[len(run.Result.Chapters)-1].Message, "goal_met cannot include")
	require.Empty(t, run.PlannedSteps)
}

func TestAgentUsesKnownUSBWhenPreferredWirelessIsUnavailable(t *testing.T) {
	base := &serialScopedAgentFixture{serialScopedFlowFixture: &serialScopedFlowFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput)}}
	wireless := fakes.New("wireless", strategy.StatusUnavailable, strategy.CapInput)
	svc := New(strategyregistry.New(base))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: base.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{
		{Name: "wireless", StrategyID: base.ID(), Health: strategy.StatusUnavailable, Capabilities: map[string]strategy.Capability{"input": {Name: "input", Status: strategy.StatusUnavailable}}},
		{Name: "usb", StrategyID: base.ID(), Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{"input": {Name: "input", Status: strategy.StatusAvailable}}},
	}})
	svc.transportStrategies["phone"] = wireless
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "key", Value: "DPAD_DOWN"}, {GoalMet: true}}})
	run, err := svc.StartAgent(context.Background(), "move down", "phone", "test", true)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State, "%+v", run.Result.Chapters)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Len(t, base.Calls(), 1)
	require.Empty(t, wireless.Calls())
	require.Equal(t, "usb", svc.Audit()[0].Transport)
}

func TestAgentKeepsImplicitUSBFallbackWhenSavedWirelessAdapterIsMissing(t *testing.T) {
	base := &serialScopedAgentFixture{serialScopedFlowFixture: &serialScopedFlowFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput)}}
	svc := New(strategyregistry.New(base))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: base.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{
		{Name: "wireless", StrategyID: base.ID(), Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{"input": {Name: "input", Status: strategy.StatusAvailable}}},
		{Name: "usb", StrategyID: base.ID(), Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{"input": {Name: "input", Status: strategy.StatusAvailable}}},
	}})
	svc.SetAgentPlanner(&sequenceAgentPlanner{plans: []internalflows.AgentPlan{{StepKind: "key", Value: "DPAD_DOWN"}, {GoalMet: true}}})

	run, err := svc.StartAgent(context.Background(), "move down", "phone", "test", true)
	require.NoError(t, err)
	require.Equal(t, "unverified", run.State, "%+v", run.Result.Chapters)
	require.Equal(t, "unverified", run.Result.Disposition)
	require.Len(t, base.Calls(), 1)
	require.Equal(t, "usb", svc.Audit()[0].Transport)
}

func TestAgentRecoversKnownWirelessBeforePlanning(t *testing.T) {
	fixture := &recoveringWirelessStateFixture{Strategy: fakes.New("android-adb", strategy.StatusAvailable, strategy.CapInput)}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "phone", Kind: "physical", Serial: "serial-1", StrategyID: fixture.ID(), Transport: "wireless", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{{Name: "wireless", StrategyID: fixture.ID()}}})
	svc.transportStrategies["phone"] = fixture
	run, err := svc.StartAgentWithOptions(context.Background(), "move down", "phone", "test", true, true)
	require.NoError(t, err)
	require.Equal(t, "planned", run.State)
	require.Equal(t, 1, fixture.reconnects)
	require.Empty(t, fixture.Calls())
}
