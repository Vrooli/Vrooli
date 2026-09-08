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
	if !skillAvailable {
		return AgentRun{}, fmt.Errorf("agent mode refused: prompt-manager device-control skill is unavailable")
	}
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return AgentRun{}, fmt.Errorf("agent goal is required")
	}
	adapter, ok := s.strategyForFlow(deviceID, "")
	if !ok {
		return AgentRun{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	declaration, err := adapter.Describe(ctx)
	if err != nil {
		return AgentRun{}, err
	}
	allowed := map[string]bool{}
	for _, kind := range strategy.StepKinds(declaration) {
		allowed[kind] = true
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
			worldModel := internalflows.AgentWorld{Goal: goal, Capabilities: declaration.Capabilities, StepKinds: strategy.StepKinds(declaration), State: state, FrameOptional: declaration.Capabilities[strategy.CapScreenshot].Status == strategy.StatusAvailable, App: internalflows.AgentAppContext{ApplicationID: declaration.DeviceID, Revision: declaration.StrategyID}, Policy: policy}
			if promptHash == "" {
				promptHash, _ = internalflows.HashAgentPayload(worldModel)
			}
			appendChapter(fmt.Sprintf("world-model-%d", iteration), "passed", fmt.Sprintf("bounded world model captured; prompt_hash=%s policy_hash=%s", promptHash, policyHash))

			var plan internalflows.AgentPlan
			var planningErr error
			if planner != nil {
				plan, planningErr = planner.Plan(agentCtx, worldModel)
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
			if validationErr := internalflows.ValidateAgentPlan(plan, worldModel.StepKinds); validationErr != nil {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", validationErr.Error()+"; no actuation executed")
				result.Disposition = "failed"
				break
			}
			if planHash, hashErr := internalflows.HashAgentPayload(plan); hashErr == nil {
				planHashes = append(planHashes, planHash)
			}
			if plan.GoalMet {
				appendChapter(fmt.Sprintf("goal-%d", iteration), "passed", "planner reported the goal satisfied")
				completed = true
				break
			}
			if policyErr := internalflows.AgentPolicyViolation(policy, plan, confirmed); policyErr != nil {
				disposition := "failed"
				if strings.Contains(policyErr.Error(), "confirmation required") {
					disposition = "confirmation_required"
				}
				appendChapter(fmt.Sprintf("confirmation-%d", iteration), disposition, policyErr.Error()+"; no actuation executed")
				result.Disposition = disposition
				break
			}
			if !allowed[plan.StepKind] {
				appendChapter(fmt.Sprintf("planning-%d", iteration), "failed", fmt.Sprintf("planner proposed undeclared step kind %q; no actuation executed", plan.StepKind))
				result.Disposition = "failed"
				break
			}
			plannedSteps = append(plannedSteps, agentFlowStep(iteration, plan))
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
			if dryRun {
				appendChapter(fmt.Sprintf("iteration-%d", iteration), "passed", fmt.Sprintf("proposed=%s verdict=accepted actuation=dry-run without sending an actuation or writing an actuation audit", plan.StepKind))
			} else {
				direct := directActuationForAgentPlan(plan, declaration.Transport)
				audit, actuationErr := s.ActuateDevice(agentCtx, deviceID, actor, lease.LeaseToken, direct)
				if actuationErr != nil {
					iterationErr = actuationErr
				} else {
					state, iterationErr = s.ReadDeviceState(agentCtx, deviceID)
					if iterationErr == nil {
						afterHash, _ := internalflows.HashAgentPayload(state)
						appendChapter(fmt.Sprintf("iteration-%d", iteration), "passed", fmt.Sprintf("proposed=%s verdict=accepted actuation=completed causation_id=%s observed_state_hash=%s", plan.StepKind, audit.CausationID, afterHash))
					}
				}
				if iterationErr != nil {
					appendChapter(fmt.Sprintf("iteration-%d", iteration), "failed", fmt.Sprintf("proposed=%s verdict=accepted actuation=failed error=%v", plan.StepKind, iterationErr))
				}
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
	agent := AgentRun{ID: uuid.NewString(), Goal: goal, DeviceID: deviceID, Actor: actor, State: agentState, Skill: "prompt-manager/device-control", Result: result, CreatedAt: time.Now().UTC(), DryRun: dryRun, PlanningRole: internalflows.AgentPlanRole, PromptHash: promptHash, PolicyHash: policyHash, PlanHashes: append([]string(nil), planHashes...), PlannedSteps: plannedSteps}
	s.mu.Lock()
	s.agents[agent.ID] = agent
	s.mu.Unlock()
	return agent, nil
}

const (
	maxAgentIterations = 8
	maxAgentDuration   = 30 * time.Second
)

func agentDispositionForContext(err error) string {
	if errors.Is(err, context.Canceled) {
		return "aborted"
	}
	return "failed"
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

func agentFlowStep(iteration int, plan internalflows.AgentPlan) Step {
	step := Step{ID: fmt.Sprintf("iteration-%d", iteration), Kind: plan.StepKind, RequiredCapabilities: []string{capabilityForAgentStep(plan.StepKind)}, Arguments: map[string]any{}}
	switch {
	case plan.StepKind == "key":
		step.Target = fmt.Sprint(plan.Value)
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
	if a.State != "completed" || a.Result.Disposition != "passed" {
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
