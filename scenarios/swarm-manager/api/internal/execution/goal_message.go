// This file composes the Swarm goal message delivered to one Agent Manager run.
package execution

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// GoalMessageMaxChars is Agent Manager's `until` cap. The whole composed
// message is bounded by the same figure so it fits every harness.
const GoalMessageMaxChars = 2048

// GoalMessageInput carries the values the harness-goal-authoring slots need.
type GoalMessageInput struct {
	PlanShape       string // phased | mandate
	PlanSlug        string
	PlanExecutionID string
	ScenarioName    string
	ProgramName     string
	AcceptanceAllow []string
	ScopePolicy     string
	Budget          string
	OperatorNote    string
}

// GoalMessageTooLongError names the overflow so preflight can block with a count
// instead of silently truncating.
type GoalMessageTooLongError struct {
	Length int
	Limit  int
}

func (e *GoalMessageTooLongError) Error() string {
	return fmt.Sprintf("goal message is %d characters, over the %d-character limit by %d", e.Length, e.Limit, e.Length-e.Limit)
}

// setpointProgramPattern matches the scenario setpoint program an adaptive
// mandate names in its target, for example `audio-tools.setpoint-read`.
var setpointProgramPattern = regexp.MustCompile(`([a-z0-9][a-z0-9-]*)\.setpoint-read`)

// goalMessageInputForItem builds the composer input from a backlog item alone.
// It cannot see the linked plan, so it starts from the phased default and a
// slug-derived authority; Service.goalMessageInput refines the shape and target
// scenario from the accepted plan revision.
func goalMessageInputForItem(item backlogItem, executionID string) GoalMessageInput {
	slug := ""
	if item.PlanRef != nil {
		slug = strings.TrimSpace(item.PlanRef.Slug)
	}
	program := slug
	if program == "" {
		program = strings.TrimSpace(item.Name)
	}
	return GoalMessageInput{
		PlanShape:       "phased",
		PlanSlug:        slug,
		PlanExecutionID: executionID,
		ProgramName:     program,
		AcceptanceAllow: item.AcceptanceAllow,
		ScopePolicy:     item.ScopePolicy,
		OperatorNote:    item.OperatorNote,
	}
}

// goalMessageInput resolves the composer input for an item, deriving the plan
// shape and target scenario from the accepted canonical plan when one is linked.
// Plan Manager has no typed shape field yet, so the shape is derived from the
// plan the item is pinned to: a plan whose target, definition of done, technical
// approach or validation strategy names a scenario setpoint program is an
// adaptive mandate whose finish line is its setpoint board; every other plan is
// a phased plan whose finish line is its phase frontier. The derivation is
// recomputed from the same accepted revision on every compose, so it cannot
// drift from the plan.
func (s *Service) goalMessageInput(ctx context.Context, item backlogItem, executionID string) GoalMessageInput {
	input := goalMessageInputForItem(item, executionID)
	if !hasExecutionPlanRef(item) || s.planRenderer == nil {
		return input
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil || rendered.Plan == nil {
		return input
	}
	if shape, scenario := goalPlanShape(rendered.Plan); shape == "mandate" && scenario != "" {
		input.PlanShape = "mandate"
		input.ScenarioName = scenario
	}
	return input
}

// goalPlanShape classifies a linked plan and names its target scenario. A
// mandate is identified by the scenario setpoint program its contract points
// at; the captured scenario name is the same one Swarm uses to name the program
// board and the scenario improve skill.
func goalPlanShape(plan *sharedv1.Plan) (shape, scenario string) {
	if plan == nil {
		return "phased", ""
	}
	text := strings.Join([]string{
		plan.GetTargetOutcome(),
		plan.GetDefinitionOfDone(),
		plan.GetTechnicalApproach(),
		plan.GetValidationStrategy(),
	}, "\n")
	match := setpointProgramPattern.FindStringSubmatch(text)
	if len(match) >= 2 && strings.TrimSpace(match[1]) != "" {
		return "mandate", strings.TrimSpace(match[1])
	}
	return "phased", ""
}

// ComposeGoalMessage renders the harness-goal-authoring slots in order for a
// phased plan (Shape A) or a mandate (Shape B), then the operator note verbatim.
// It never truncates.
func ComposeGoalMessage(in GoalMessageInput) (string, error) {
	shape := strings.TrimSpace(in.PlanShape)
	if shape == "" {
		shape = "phased"
	}
	boundary := strings.Join(in.AcceptanceAllow, ", ")
	scope := strings.TrimSpace(in.ScopePolicy)
	if scope == "" {
		scope = "fixed"
	}
	budget := strings.TrimSpace(in.Budget)
	if budget == "" {
		budget = "the item's remaining charge allowance"
	}

	var b strings.Builder
	switch shape {
	case "mandate":
		fmt.Fprintf(&b, "/goal Every required setpoint row in %s-improve is in band on run %s.setpoint-read, read as often as the outcome contract requires, and the evidence audit passes; or the handoff names each out-of-band row with its blocker.\n\n", in.ScenarioName, in.ScenarioName)
		fmt.Fprintf(&b, "Authority: %s grants development inside %s.\n", in.ProgramName, boundary)
		fmt.Fprintf(&b, "Read first: prompt-manager skill read goal-loop scenario-improvement-campaign %s-improve. The scenario docs are the target; where the intended design is missing from them, write it there before the code.\n", in.ScenarioName)
		b.WriteString("Proof: paste the setpoint board after each intervention. Never move a row by editing its band or sensor.\n")
		b.WriteString("Iteration: one falsifiable intervention at a time, chosen from evidence, checkpointed through Plan Manager before the next.\n")
	case "phased":
		fmt.Fprintf(&b, "/goal Plan %s is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.\n\n", in.PlanSlug)
		fmt.Fprintf(&b, "Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring. Then: plan-manager exec continue %s. The plan holds the design.\n", in.PlanSlug)
		b.WriteString("Proof: the phase validation commands named in the plan, output shown. Targeted checks by default.\n")
	default:
		return "", fmt.Errorf("unknown plan shape %q (want phased or mandate)", in.PlanShape)
	}

	fmt.Fprintf(&b, "Boundary: %s. Scope policy: %s. Under extend-with-record run plan-manager exec boundary-extend %s before any out-of-scope edit.\n", boundary, scope, in.PlanExecutionID)
	if shape == "mandate" {
		b.WriteString("Adjacent defects in other scenarios: file them; repair at the owner only when the grant covers it.\n")
	} else {
		b.WriteString("Adjacent defects: fix when you understand the cause and it blocks a phase; otherwise file with plan-manager log bug-add and continue.\n")
	}
	b.WriteString("Quality: production code, no shims or dead code, docs updated with the code.\n")
	if shape == "mandate" {
		b.WriteString("Blocked means a decision, credential, or approval you lack. A row reading unavailable is journaled, not estimated.\n")
	} else {
		b.WriteString("Blocked means a decision, credential, or approval you lack. Name it in the handoff. Friction you can diagnose is not blocked.\n")
	}
	fmt.Fprintf(&b, "Budget: %s. An interruption (usage window, timeout, crash, session lost) is resumed by Swarm under continuation: until-allowance. A verdict (complete, blocked, abstained) is final.\n", budget)
	b.WriteString("Handoff: checkpoint through Plan Manager; final report: changed, verified, remaining, unverified.\n")
	b.WriteString("Non-goals: widen scope, loosen or delete tests, rerun unchanged validation for a greener result.\n")
	fmt.Fprintf(&b, "Operator note: %s", in.OperatorNote)

	message := b.String()
	if len([]rune(message)) > GoalMessageMaxChars {
		return "", &GoalMessageTooLongError{Length: len([]rune(message)), Limit: GoalMessageMaxChars}
	}
	return message, nil
}

// RenderFinishLine returns the finish line (the `until` text) that Agent Manager
// receives. It is the destination slot alone, rendered with real values.
func RenderFinishLine(in GoalMessageInput) string {
	shape := strings.TrimSpace(in.PlanShape)
	if shape == "mandate" {
		return fmt.Sprintf("Every required setpoint row in %s-improve is in band on run %s.setpoint-read, and the evidence audit passes; or the handoff names each out-of-band row with its blocker.", in.ScenarioName, in.ScenarioName)
	}
	return fmt.Sprintf("Plan %s is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.", in.PlanSlug)
}
