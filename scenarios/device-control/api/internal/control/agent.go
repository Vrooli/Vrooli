package control

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	internalflows "device-control/internal/flows"
	"device-control/strategy"

	"github.com/google/uuid"
)

// StartAgent keeps the public entry point compatible while routing through the
// capability/state planner. A frame is an optional observation, never a gate.
func (s *Service) StartAgent(ctx context.Context, goal, deviceID, actor string, skillAvailable bool) (AgentRun, error) {
	return s.StartAgentWithOptions(ctx, goal, deviceID, actor, skillAvailable, false)
}

func (s *Service) StartAgentWithOptions(ctx context.Context, goal, deviceID, actor string, skillAvailable, dryRun bool) (AgentRun, error) {
	return s.StartAgentWithPolicy(ctx, goal, deviceID, actor, skillAvailable, dryRun, internalflows.DefaultAgentPolicy(), false)
}

// StartAgentWithPolicy is the policy-aware entry point. Confirmation is an
// operator decision supplied outside the model response; it cannot be granted
// by a planner output.
func (s *Service) StartAgentWithPolicy(ctx context.Context, goal, deviceID, actor string, skillAvailable, dryRun bool, policy internalflows.AgentPolicy, confirmed bool) (AgentRun, error) {
	return s.StartAgentWithOutcome(ctx, goal, deviceID, actor, skillAvailable, dryRun, policy, confirmed, "")
}

// StartAgentWithOutcome accepts an operator-supplied, independently observable
// completion condition. The planner cannot change this condition.
func (s *Service) StartAgentWithOutcome(ctx context.Context, goal, deviceID, actor string, skillAvailable, dryRun bool, policy internalflows.AgentPolicy, confirmed bool, expectSelected string) (AgentRun, error) {
	startedAt := time.Now().UTC()
	if !skillAvailable {
		return AgentRun{}, fmt.Errorf("agent mode refused: prompt-manager device-control skill is unavailable")
	}
	goal = strings.TrimSpace(goal)
	expectSelected = strings.TrimSpace(expectSelected)
	if len(expectSelected) > 256 {
		return AgentRun{}, fmt.Errorf("selected-control expectation exceeds 256 bytes")
	}
	if goal == "" {
		return AgentRun{}, fmt.Errorf("agent goal is required")
	}
	adapter, ok := s.strategyForFlow(deviceID, "")
	if !ok {
		return AgentRun{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	declaration, err := adapter.Describe(ctx)
	selectedTransport := ""
	if record, found := s.devices.Get(deviceID); found {
		selectedTransport = record.Transport
	}
	if err != nil || declaration.Status != strategy.StatusAvailable {
		if usb, ok := s.knownSerialAndroidUSB(deviceID); ok {
			if usbDeclaration, probeErr := usb.Describe(ctx); probeErr == nil && usbDeclaration.Status == strategy.StatusAvailable {
				declaration = usbDeclaration
				selectedTransport = "usb"
				err = nil
			}
		}
	}
	if (err != nil || declaration.Status != strategy.StatusAvailable) && s.recoverWirelessBeforeUse(ctx, deviceID) {
		if wireless, ok := s.strategyForFlow(deviceID, "wireless"); ok {
			if recovered, probeErr := wireless.Describe(ctx); probeErr == nil && recovered.Status == strategy.StatusAvailable {
				declaration, selectedTransport, err = recovered, "wireless", nil
			}
		}
	}
	if err != nil {
		return AgentRun{}, err
	}
	if selectedTransport != "" {
		declaration.Transport = selectedTransport
	}
	allowed := map[string]bool{}
	for _, kind := range strategy.StepKinds(declaration) {
		if agentPlannerCanExecute(kind) {
			allowed[kind] = true
		}
	}
	result := RunResult{RunID: uuid.NewString(), Disposition: "passed", Chapters: []Chapter{}}
	appendChapter := func(id, disposition, message string) {
		result.Chapters = append(result.Chapters, Chapter{ID: id, Title: "Agent " + id, Disposition: disposition, Message: message})
	}
	duration := maxAgentDuration
	if policy.MaxDurationMS > 0 && time.Duration(policy.MaxDurationMS)*time.Millisecond < duration {
		duration = time.Duration(policy.MaxDurationMS) * time.Millisecond
	}
	agentCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	s.mu.Lock()
	planner := s.agentPlanner
	s.mu.Unlock()
	plannedSteps := []Step{}
	plannerCalls := 0
	inferenceCalls := []internalflows.AgentInferenceEvidence{}
	actionAttempts := 0
	actionCompletions := 0
	goalVerified := false
	var proposedStep *AgentProposedStep
	history := []internalflows.AgentHistory{}
	planHashes := []string{}
	promptHash := ""
	policyHash, _ := internalflows.HashAgentPayload(policy)
	state, stateErr := s.ReadDeviceState(agentCtx, deviceID)
	if stateErr != nil && declaration.Capabilities[strategy.CapScreenshot].Status != strategy.StatusAvailable {
		appendChapter("world-model-1", "failed", "typed device state unavailable: "+stateErr.Error())
		result.Disposition = "failed"
	} else {
		if stateErr != nil {
			appendChapter("world-model-state-1", "passed", "typed state unavailable; frame modality remains available")
		}
		completed := false
		iterationBudget := maxAgentIterations
		if policy.MaxIterations > 0 && policy.MaxIterations < iterationBudget {
			iterationBudget = policy.MaxIterations
		}
		for iteration := 1; iteration <= iterationBudget && result.Disposition == "passed"; iteration++ {
			if err := agentCtx.Err(); err != nil {
				result.Disposition = agentDispositionForContext(err)
				appendChapter(fmt.Sprintf("budget-%d", iteration), "failed", err.Error())
				break
			}
			worldModel := internalflows.AgentWorld{Goal: goal, Capabilities: declaration.Capabilities, StepKinds: strategy.StepKinds(declaration), State: state, FrameAvailable: false, App: internalflows.AgentAppContext{ApplicationID: declaration.DeviceID, Revision: declaration.StrategyID}, Policy: policy}
			if planner != nil && declaration.Capabilities[strategy.CapSemanticTree].Status == strategy.StatusAvailable {
				snapshot, observeErr := s.ObserveSemantics(agentCtx, deviceID, actor, "", declaration.Transport)
				if observeErr == nil {
					worldModel.Semantics = &snapshot
					if len(history) > 0 {
						last := &history[len(history)-1]
						last.AfterSnapshotHash, _ = internalflows.HashAgentPayload(snapshot.Elements)
						if last.BeforeSnapshotHash != "" && last.AfterSnapshotHash != "" {
							changed := last.BeforeSnapshotHash != last.AfterSnapshotHash
							last.SnapshotChanged = &changed
						}
					}
				} else {
					worldModel.SemanticUnavailable = "semantic observation unavailable"
				}
			}
			worldModel.History = append([]internalflows.AgentHistory(nil), history...)
			// The planner receives typed state and semantics, not captured image
			// bytes. Offering observe would let it repeatedly request screenshots
			// whose contents it cannot actually inspect.
			filtered := worldModel.StepKinds[:0]
			for _, kind := range worldModel.StepKinds {
				if !agentPlannerCanExecute(kind) || planner != nil && kind == "observe" {
					continue
				}
				filtered = append(filtered, kind)
			}
			worldModel.StepKinds = filtered
			if worldModel.Semantics == nil || worldModel.Semantics.Truncated || declaration.Capabilities[strategy.CapScreenshot].Status != strategy.StatusAvailable {
				filtered := worldModel.StepKinds[:0]
				for _, kind := range worldModel.StepKinds {
					if kind != "semantic-target" {
						filtered = append(filtered, kind)
					}
				}
				worldModel.StepKinds = filtered
			}
			iterationPromptHash, _ := internalflows.HashAgentPayload(worldModel)
			if promptHash == "" {
				promptHash = iterationPromptHash
			}
			appendChapter(fmt.Sprintf("world-model-%d", iteration), "passed", fmt.Sprintf("bounded world model captured; prompt_hash=%s policy_hash=%s", iterationPromptHash, policyHash))

			var plan internalflows.AgentPlan
			var planningErr error
			if planner != nil {
				plannerCalls++
				if measured, ok := planner.(internalflows.AgentPlannerWithEvidence); ok {
					var evidence *internalflows.AgentInferenceEvidence
					plan, evidence, planningErr = measured.PlanWithEvidence(agentCtx, worldModel)
					if evidence != nil {
						inferenceCalls = append(inferenceCalls, *evidence)
					}
				} else {
					plan, planningErr = planner.Plan(agentCtx, worldModel)
				}
			} else {
				// Test and offline operation can use the deterministic capability
				// planner; production wires GatewayPlanner through ai-gateway.
				plan.StepKind, plan.Action, plan.Value = chooseAgentStep(goal, allowed)
			}
			if planningErr != nil {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", planningErr.Error())
				result.Disposition = "failed"
				break
			}
			// Validate plan shape against the device's declared step vocabulary
			// first. This keeps contradictory goal_met responses from being
			// misreported as confirmation requests.
			if validationErr := internalflows.ValidateAgentPlan(plan, strategy.StepKinds(declaration)); validationErr != nil {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", validationErr.Error()+"; no actuation executed")
				result.Disposition = "failed"
				break
			}
			// Classify the requested effect before checking device support so
			// risky unsupported choices still fail closed through the operator
			// confirmation path, without being advertised or actuated.
			if policyErr := internalflows.AgentPolicyViolation(policy, plan, confirmed); policyErr != nil {
				disposition := "failed"
				if strings.Contains(policyErr.Error(), "confirmation required") {
					disposition = "confirmation_required"
				}
				appendChapter(fmt.Sprintf("confirmation-%d", iteration), disposition, policyErr.Error()+"; no actuation executed")
				result.Disposition = disposition
				break
			}
			if validationErr := internalflows.ValidateAgentPlan(plan, worldModel.StepKinds); validationErr != nil {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", validationErr.Error()+"; no actuation executed")
				result.Disposition = "failed"
				break
			}
			if planHash, hashErr := internalflows.HashAgentPayload(plan); hashErr == nil {
				planHashes = append(planHashes, planHash)
			}
			if plan.StepKind == "semantic-target" {
				if shouldBlockRepeatedSemanticAction(history, plan) {
					history[len(history)-1].RepeatBlocked = true
					appendChapter(fmt.Sprintf("planning-%d", iteration), "passed", "repeated semantic action skipped because the observed snapshot did not change")
					continue
				}
				if err := validateAgentSemanticTarget(worldModel.Semantics, plan.Action); err != nil {
					appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", err.Error()+"; no actuation executed")
					result.Disposition = "failed"
					break
				}
			}
			if plan.GoalMet {
				if expectSelected != "" {
					if err := verifySelectedSemanticControl(worldModel.Semantics, expectSelected); err != nil {
						appendChapter(fmt.Sprintf("goal-%d", iteration), "failed", err.Error())
						result.Disposition = "failed"
						break
					}
					goalVerified = true
					appendChapter(fmt.Sprintf("goal-%d", iteration), "passed", "operator-specified selected control independently observed")
				} else {
					appendChapter(fmt.Sprintf("goal-%d", iteration), "passed", "planner reported the goal satisfied")
				}
				completed = true
				break
			}
			if plan.StepKind == "semantic-target" {
				if policyErr := semanticTargetPolicyViolation(policy, worldModel.Semantics, plan.Action, confirmed); policyErr != nil {
					disposition := "failed"
					if strings.Contains(policyErr.Error(), "confirmation required") {
						disposition = "confirmation_required"
					}
					appendChapter(fmt.Sprintf("confirmation-%d", iteration), disposition, policyErr.Error()+"; no actuation executed")
					result.Disposition = disposition
					break
				}
			}
			if !allowed[plan.StepKind] {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", fmt.Sprintf("planner proposed undeclared step kind %q; no actuation executed", plan.StepKind))
				result.Disposition = "failed"
				break
			}
			plannedSteps = append(plannedSteps, agentFlowStep(iteration, plan))
			if dryRun {
				// No future state can be observed without performing this step.
				// Return the proposal instead of repeatedly planning against an
				// unchanged screen. Text and arbitrary values stay private.
				preview := AgentProposedStep{Kind: plan.StepKind}
				if plan.StepKind == "semantic-target" || plan.StepKind == "key" {
					preview.Target = plannedSteps[len(plannedSteps)-1].Target
				}
				proposedStep = &preview
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "planned", fmt.Sprintf("proposed=%s; no actuation was sent", plan.StepKind))
				result.Disposition = "planned"
				break
			}
			entry := internalflows.AgentHistory{StepKind: plan.StepKind, Outcome: "completed"}
			if plan.StepKind == "semantic-target" || strings.HasPrefix(plan.StepKind, "property-") || strings.HasPrefix(plan.StepKind, "media-") {
				entry.Target = plan.Action
			}
			if plan.StepKind == "key" {
				entry.Target = fmt.Sprint(plan.Value)
			}
			if worldModel.Semantics != nil {
				entry.BeforeSnapshotHash, _ = internalflows.HashAgentPayload(worldModel.Semantics.Elements)
				entry.VisibleControls, entry.VisibleControlsTruncated = agentVisibleControls(worldModel.Semantics)
			}
			history = append(history, entry)
			if plan.StepKind == "observe" {
				_, observeErr := s.Run(agentCtx, Flow{ID: uuid.NewString(), Name: "agent-observe", Steps: []Step{{ID: "observe", Kind: "observe", TimeoutMS: 5000}}}, deviceID, actor)
				if observeErr != nil {
					appendChapter(fmt.Sprintf("iteration-%d", iteration), "failed", fmt.Sprintf("proposed=%s verdict=accepted observation=%v", plan.StepKind, observeErr))
					result.Disposition = "failed"
					break
				}
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "passed", fmt.Sprintf("proposed=%s verdict=accepted actuation=observation completed", plan.StepKind))
				if planner == nil {
					completed = true
					break
				}
				continue
			}
			if plan.StepKind == "property-get" {
				propertyName := strings.TrimSpace(plan.Action)
				property, propertyOK := state.Properties[propertyName]
				if !propertyOK || property.Status != strategy.StatusAvailable {
					appendChapter(fmt.Sprintf("iteration-%d", iteration), "failed", fmt.Sprintf("proposed=%s verdict=accepted property=%q is unavailable", plan.StepKind, propertyName))
					result.Disposition = "failed"
					break
				}
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "passed", fmt.Sprintf("proposed=%s verdict=accepted observed=%s", plan.StepKind, fmt.Sprint(property.Value)))
				if planner == nil {
					completed = true
					break
				}
				continue
			}
			lease, leaseErr := s.AcquireContext(agentCtx, deviceID, actor, 2*time.Minute)
			if leaseErr != nil {
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "failed", fmt.Sprintf("proposed=%s verdict=accepted lease=%v", plan.StepKind, leaseErr))
				result.Disposition = "failed"
				break
			}
			var iterationErr error
			causation := ""
			if plan.StepKind == "semantic-target" || isAgentAppLifecycleStep(plan.StepKind) {
				actionAttempts++
				flow := Flow{ID: uuid.NewString(), Transport: declaration.Transport, Steps: []Step{agentFlowStep(iteration, plan)}}
				flowResult, flowErr := s.RunWithLease(agentCtx, flow, deviceID, actor, lease.LeaseToken)
				if flowErr != nil {
					iterationErr = flowErr
				} else if flowResult.Disposition != "passed" {
					detail := ""
					if len(flowResult.Chapters) > 0 {
						detail = flowResult.Chapters[len(flowResult.Chapters)-1].Message
					}
					iterationErr = fmt.Errorf("agent flow action did not pass: %s: %s", flowResult.Disposition, detail)
				} else {
					actionCompletions++
					if plan.StepKind == "launch" {
						launched, stateErr := s.ReadDeviceState(agentCtx, deviceID)
						if stateErr != nil {
							iterationErr = fmt.Errorf("read state after launch: %w", stateErr)
						} else if launched.ForegroundPackage != plan.Action || launched.Unavailable["foreground_package"] != "" {
							iterationErr = fmt.Errorf("launched package %q was not observed in the foreground", plan.Action)
						}
					}
				}
			} else {
				actionAttempts++
				direct := directActuationForAgentPlan(plan, declaration.Transport)
				audit, actuationErr := s.ActuateDevice(agentCtx, deviceID, actor, lease.LeaseToken, direct)
				iterationErr = actuationErr
				causation = audit.CausationID
				if actuationErr == nil {
					actionCompletions++
				}
			}
			if iterationErr == nil {
				state, iterationErr = s.ReadDeviceState(agentCtx, deviceID)
				if iterationErr == nil {
					afterHash, _ := internalflows.HashAgentPayload(state)
					appendChapter(fmt.Sprintf("iteration-%d", iteration), "passed", fmt.Sprintf("proposed=%s verdict=accepted actuation=completed causation_id=%s observed_state_hash=%s", plan.StepKind, causation, afterHash))
				}
			}
			if iterationErr != nil {
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "failed", fmt.Sprintf("proposed=%s verdict=accepted actuation=failed error=%v", plan.StepKind, iterationErr))
			}
			_, _ = s.ReleaseContext(agentCtx, lease.ID)
			if iterationErr != nil {
				result.Disposition = "failed"
				break
			}
			// The deterministic fallback is intentionally one-step. A real
			// planner must explicitly return goal_met or continue planning.
			if planner == nil {
				completed = true
				break
			}
		}
		if result.Disposition == "passed" && !completed {
			appendChapter("budget", "failed", fmt.Sprintf("agent step budget exhausted after %d iterations", iterationBudget))
			result.Disposition = "failed"
		}
	}
	if result.Disposition == "passed" && len(result.Chapters) == 0 {
		result.Disposition = "failed"
	}
	agentState := "completed"
	if result.Disposition != "passed" {
		agentState = "failed"
	}
	if result.Disposition == "confirmation_required" {
		agentState = "awaiting_confirmation"
	}
	if result.Disposition == "planned" {
		agentState = "planned"
	}
	completedAt := time.Now().UTC()
	agent := AgentRun{ID: uuid.NewString(), Goal: goal, DeviceID: deviceID, Actor: actor, State: agentState, Skill: "prompt-manager/device-control", Result: result, CreatedAt: startedAt, CompletedAt: completedAt, DurationMS: completedAt.Sub(startedAt).Milliseconds(), PlannerCalls: plannerCalls, ActionAttempts: actionAttempts, ActionCompletions: actionCompletions, InferenceCalls: inferenceCalls, DryRun: dryRun, GoalVerified: goalVerified, PlanningRole: internalflows.AgentPlanRole, PromptHash: promptHash, PolicyHash: policyHash, PlanHashes: append([]string(nil), planHashes...), PlannedSteps: plannedSteps, ProposedStep: proposedStep}
	s.mu.Lock()
	s.agents[agent.ID] = agent
	s.mu.Unlock()
	return agent, nil
}

const (
	maxAgentIterations = 8
	maxAgentDuration   = 2 * time.Minute
)

func agentDispositionForContext(err error) string {
	if errors.Is(err, context.Canceled) {
		return "aborted"
	}
	return "failed"
}

// Keep a short local-only summary of the previous screen so the planner can
// distinguish an unproductive action from returning to an earlier surface.
func agentVisibleControls(snapshot *strategy.SemanticSnapshot) ([]string, bool) {
	if snapshot == nil {
		return nil, false
	}
	seen := map[string]bool{}
	var controls []string
	for _, element := range snapshot.Elements {
		if !element.Enabled || element.Editable || (!element.Clickable && !element.Focusable) {
			continue
		}
		label := element.Text
		if label == "" {
			label = element.Description
		}
		if label == "" {
			label = element.ResourceID
		}
		label = strings.TrimSpace(label)
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		if len(controls) == 12 {
			return controls, true
		}
		controls = append(controls, label)
	}
	return controls, snapshot.Truncated
}

func shouldBlockRepeatedSemanticAction(history []internalflows.AgentHistory, plan internalflows.AgentPlan) bool {
	if len(history) == 0 || plan.StepKind != "semantic-target" {
		return false
	}
	last := history[len(history)-1]
	return last.StepKind == plan.StepKind && last.Target == plan.Action && last.SnapshotChanged != nil && !*last.SnapshotChanged
}

func validateAgentSemanticTarget(snapshot *strategy.SemanticSnapshot, target string) error {
	if snapshot == nil || snapshot.Truncated {
		return fmt.Errorf("complete semantic observation is unavailable")
	}
	matches := 0
	for _, item := range snapshot.Elements {
		if (item.Text == target || item.Description == target || item.ResourceID == target) && item.Enabled && !item.Editable && (item.Clickable || item.Focusable) {
			matches++
		}
	}
	if matches != 1 {
		return fmt.Errorf("semantic target must match exactly one enabled actionable non-editable control")
	}
	return nil
}

func verifySelectedSemanticControl(snapshot *strategy.SemanticSnapshot, target string) error {
	if snapshot == nil || snapshot.Truncated {
		return fmt.Errorf("selected-control outcome cannot be verified without a complete semantic observation")
	}
	var matched *strategy.SemanticElement
	for i := range snapshot.Elements {
		item := &snapshot.Elements[i]
		if item.Text == target || item.Description == target || item.ResourceID == target {
			if matched != nil {
				return fmt.Errorf("selected-control outcome is ambiguous")
			}
			matched = item
		}
	}
	if matched == nil || matched.Selected == nil || !*matched.Selected {
		return fmt.Errorf("selected-control outcome was not observed")
	}
	return nil
}

// Check every visible identifier on the selected control: a harmless resource
// ID must not hide an explicitly destructive or sensitive label.
func semanticTargetPolicyViolation(policy internalflows.AgentPolicy, snapshot *strategy.SemanticSnapshot, target string, confirmed bool) error {
	if err := validateAgentSemanticTarget(snapshot, target); err != nil {
		return err
	}
	for _, item := range snapshot.Elements {
		if (item.Text == target || item.Description == target || item.ResourceID == target) && item.Enabled && !item.Editable && (item.Clickable || item.Focusable) {
			for _, label := range []string{item.Text, item.Description, item.ResourceID} {
				if err := internalflows.AgentPolicyViolation(policy, internalflows.AgentPlan{StepKind: "semantic-target", Action: label}, confirmed); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func directActuationForAgentPlan(plan internalflows.AgentPlan, transport string) DirectActuation {
	direct := DirectActuation{Action: plan.Action, Value: plan.Value, Transport: transport}
	switch {
	case plan.StepKind == "key":
		direct.Key, direct.Action, direct.Value = fmt.Sprint(plan.Value), "", nil
	case strings.HasPrefix(plan.StepKind, "media-"):
		direct.Media = strings.TrimPrefix(plan.StepKind, "media-")
		direct.Action = ""
	case plan.StepKind == "property-set":
		direct.Property = plan.Action
		direct.Action = ""
	case plan.StepKind == "property-get":
		direct.Property = plan.Action
		direct.Action = ""
	}
	return direct
}

func isAgentAppLifecycleStep(stepKind string) bool {
	switch stepKind {
	case "install", "launch", "focus", "close", "minimize", "restore", "stop", "uninstall", "clear-data", "home-set", "package-state":
		return true
	default:
		return false
	}
}

// agentPlannerCanExecute keeps the advertised action contract aligned with
// the owner paths that StartAgent can actually execute. Visual gestures are
// omitted because the planner receives no screenshot pixels or coordinates.
func agentPlannerCanExecute(stepKind string) bool {
	if isAgentAppLifecycleStep(stepKind) || stepKind == "semantic-target" || stepKind == "key" || stepKind == "observe" {
		return true
	}
	if strings.HasPrefix(stepKind, "media-") {
		return true
	}
	return stepKind == "property-get" || stepKind == "property-set"
}

func agentFlowStep(iteration int, plan internalflows.AgentPlan) Step {
	step := Step{ID: fmt.Sprintf("iteration-%d", iteration), Kind: plan.StepKind, RequiredCapabilities: []string{capabilityForAgentStep(plan.StepKind)}, Arguments: map[string]any{}}
	switch {
	case plan.StepKind == "key":
		step.Target = fmt.Sprint(plan.Value)
	case plan.StepKind == "semantic-target":
		step.Target = plan.Action
	case isAgentAppLifecycleStep(plan.StepKind):
		step.Target = plan.Action
	case strings.HasPrefix(plan.StepKind, "media-"):
		if plan.Value != nil {
			step.Arguments["value"] = plan.Value
		}
	case strings.HasPrefix(plan.StepKind, "property-"):
		step.Arguments["name"] = plan.Action
		if plan.Value != nil {
			step.Arguments["value"] = plan.Value
		}
	}
	if step.RequiredCapabilities[0] == "" {
		step.RequiredCapabilities = nil
	}
	return step
}

func capabilityForAgentStep(stepKind string) string {
	switch {
	case stepKind == "key", stepKind == "text":
		return strategy.CapInput
	case strings.HasPrefix(stepKind, "media-"):
		return strategy.CapMedia
	case strings.HasPrefix(stepKind, "property-"):
		return strategy.CapProperty
	case stepKind == "observe":
		return strategy.CapScreenshot
	case stepKind == "semantic-target":
		return strategy.CapSemanticTree
	default:
		return ""
	}
}

func chooseAgentStep(goal string, allowed map[string]bool) (kind, action string, value any) {
	lower := strings.ToLower(goal)
	if strings.Contains(lower, "pause") && allowed["media-pause"] {
		return "media-pause", "pause", nil
	}
	if strings.Contains(lower, "play") && allowed["media-play"] {
		return "media-play", "play", nil
	}
	if strings.Contains(lower, "next") && allowed["media-next"] {
		return "media-next", "next", nil
	}
	if strings.Contains(lower, "previous") && allowed["media-previous"] {
		return "media-previous", "previous", nil
	}
	if strings.Contains(lower, "volume") && allowed["media-volume"] {
		return "media-volume", "volume", valueFromGoal(lower)
	}
	if strings.Contains(lower, "down") && allowed["key"] {
		return "key", "", "DPAD_DOWN"
	}
	if allowed["observe"] {
		return "observe", "", nil
	}
	return "", "", nil
}

func valueFromGoal(goal string) any {
	for _, token := range strings.Fields(goal) {
		value, err := strconv.ParseFloat(strings.Trim(token, ",;:."), 64)
		if err == nil && value >= 0 && value <= 1 {
			return value
		}
	}
	return nil
}

func (s *Service) AbortAgent(id, reason string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	a.State = "aborted"
	a.Result.Disposition = "aborted"
	a.Result.Chapters = append(a.Result.Chapters, Chapter{ID: "abort", Title: "Agent abort", Disposition: "passed", Message: reason})
	s.agents[id] = a
	return a, nil
}

func (s *Service) PromoteAgent(id string) (AgentRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[id]
	if !ok {
		return AgentRun{}, fmt.Errorf("agent run %q not found", id)
	}
	if a.State != "completed" || a.Result.Disposition != "passed" || a.Result.Incomplete {
		return AgentRun{}, fmt.Errorf("agent run %q is not eligible for promotion", id)
	}
	if a.DryRun {
		return AgentRun{}, fmt.Errorf("agent run %q was a dry-run and has no executable actuation history", id)
	}
	if len(a.PlannedSteps) == 0 {
		return AgentRun{}, fmt.Errorf("agent run %q has no executable steps to promote", id)
	}
	flowID := "agent-flow-" + a.ID
	flow := Flow{ID: flowID, Name: "Promoted agent goal: " + a.Goal, Steps: append([]Step(nil), a.PlannedSteps...)}
	if err := ValidatePromotionCandidate(flow); err != nil {
		return AgentRun{}, err
	}
	// Model completion and agent narrative chapters do not prove that a
	// reusable flow's assertions ran. Require actual matching flow chapters.
	for _, step := range flow.Steps {
		verified := false
		for _, chapter := range a.Result.Chapters {
			if chapter.ID == step.ID && chapter.Title == step.Kind && chapter.Disposition == "passed" {
				verified = true
				break
			}
		}
		if !verified {
			return AgentRun{}, fmt.Errorf("promotion requires a verified flow chapter for step %q", step.ID)
		}
	}
	if !a.GoalVerified {
		return AgentRun{}, fmt.Errorf("promotion requires independently verified goal completion")
	}
	s.runs[flowID] = RunResult{RunID: flowID, Disposition: "passed", Chapters: append([]Chapter(nil), a.Result.Chapters...)}
	s.flowRuns[flowID] = flow
	a.PromotedFlowID = flowID
	a.State = "promoted"
	s.agents[id] = a
	return a, nil
}

func (s *Service) ListAgents() []AgentRun {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AgentRun, 0, len(s.agents))
	for _, a := range s.agents {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}
