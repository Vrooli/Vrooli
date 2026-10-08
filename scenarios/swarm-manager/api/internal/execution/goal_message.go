// This file composes the Swarm goal message delivered to one Agent Manager run.
package execution

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// GoalUntilMaxChars is Agent Manager's completion-condition cap. It applies to
// the `until` field, not to the task prompt carrying the operating contract.
const GoalUntilMaxChars = 2048

// GoalPromptMaxChars bounds the durable task prompt independently from the
// completion condition. Agent Manager accepts a substantially larger prompt;
// keeping this guard prevents accidental unbounded operator notes while
// allowing adaptive mandates to carry their full operating model.
const GoalPromptMaxChars = 16384

// GoalMessageInput carries the values the harness-goal-authoring slots need.
type GoalMessageInput struct {
	PlanShape           string // phased | mandate
	PlanSlug            string
	PlanExecutionID     string
	ScenarioName        string
	ProgramName         string
	AcceptanceAllow     []string
	ScopePolicy         string
	Budget              string
	OperatorNote        string
	BlockerRepairPolicy string
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
		PlanShape:           "phased",
		PlanSlug:            slug,
		PlanExecutionID:     executionID,
		ProgramName:         program,
		AcceptanceAllow:     item.AcceptanceAllow,
		ScopePolicy:         item.ScopePolicy,
		OperatorNote:        item.OperatorNote,
		BlockerRepairPolicy: BlockerRepairInScopeOnly,
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
	blockerPolicy := strings.TrimSpace(in.BlockerRepairPolicy)
	if blockerPolicy == "" {
		blockerPolicy = BlockerRepairInScopeOnly
	}
	budget := strings.TrimSpace(in.Budget)
	if budget == "" {
		budget = "the item's remaining charge allowance"
	}

	var b strings.Builder
	switch shape {
	case "mandate":
		fmt.Fprintf(&b, "/goal Bring every required row in %s-improve in band on %s.setpoint-read.\n\n", in.ScenarioName, in.ScenarioName)
		fmt.Fprintf(&b, "Authority: %s grants development inside %s.\n", in.ProgramName, boundary)
		fmt.Fprintf(&b, "Read first: goal-loop, scenario-improvement-campaign, %s-improve, the accepted plan, target docs, and testing docs. Docs before code.\n", in.ScenarioName)
		b.WriteString("Intent: fully implement the target, not merely complete phases, produce a handoff, or satisfy a wrapper.\n")
		b.WriteString("Loop: inspect the authoritative board; choose the highest unmet target; state a falsifiable intervention; build or repair production code, tests, fixtures, evidence producers, qualification harnesses, sensors, receipt resolvers, joins, and docs; validate; checkpoint; reread the board.\n")
		b.WriteString("Missing or broken in-scope measurement infrastructure is work, never a blocker. Unknown or out-of-band results require repair and another measurement.\n")
		b.WriteString("Evidence: use authorized local/simulated paths; keep simulated, emulated, native, live, and paid evidence separate; never claim an outcome was measured when it was not. Never edit bands, thresholds, sensors, receipts, or board logic to manufacture success.\n")
		b.WriteString("Continue while any required row is unknown, stale, failed, out of band, missing a producer, or missing its protocol. Continue while useful in-scope work remains.\n")
		b.WriteString("Complete only when every required row is in band from fresh owner-backed evidence, required repetitions are satisfied, exclusions are explicit, and the evidence audit passes. Otherwise continue.\n")
		b.WriteString("Iterate one falsifiable intervention at a time and checkpoint Plan Manager.\n")
	case "phased":
		fmt.Fprintf(&b, "/goal Plan %s is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.\n\n", in.PlanSlug)
		fmt.Fprintf(&b, "Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring. Then: plan-manager exec continue %s. The plan holds the design.\n", in.PlanSlug)
		b.WriteString("Proof: the phase validation commands named in the plan, output shown. Targeted checks by default.\n")
	default:
		return "", fmt.Errorf("unknown plan shape %q (want phased or mandate)", in.PlanShape)
	}

	fmt.Fprintf(&b, "Boundary: %s. Scope policy: %s. Under extend-with-record run plan-manager exec boundary-extend %s before any out-of-scope edit.\n", boundary, scope, in.PlanExecutionID)
	fmt.Fprintf(&b, "External blocker policy: %s. %s\n", blockerPolicy, blockerRepairInstruction(blockerPolicy))
	if shape == "mandate" {
		b.WriteString("Isolation rule: when an authorized measurement requests a fresh or otherwise isolated execution context, unrelated active sessions and foreign leases are not blockers; use the owner's capacity/readiness signal and proceed while capacity exists. Do not wait for a zero-session state or modify a live foreign session merely to serialize work.\n")
		b.WriteString("Other-scenario defects: file; repair only if the grant covers the owner.\n")
		b.WriteString("Mandate lane: use local/simulated paths; excluded private data, devices, live keys, and paid access are not blockers. Build board receipts.\n")
	} else {
		b.WriteString("Adjacent defects: fix when you understand the cause and it blocks a phase; otherwise file with plan-manager log bug-add and continue.\n")
	}
	b.WriteString("Quality: production code; no shims/dead code; update docs.\n")
	if shape == "mandate" {
		b.WriteString("Blocked means only missing decision, credential, approval, or external access. In-boundary gaps are work: build, instrument, validate, continue. Journal unavailable rows, then repair them here.\n")
	} else {
		b.WriteString("Blocked means a decision, credential, or approval you lack. Name it in the handoff. Friction you can diagnose is not blocked.\n")
	}
	fmt.Fprintf(&b, "Budget: %s. An interruption (usage window, timeout, crash, session lost) is resumed by Swarm under continuation: until-allowance. A verdict (complete, blocked, abstained) is final.\n", budget)
	b.WriteString("Handoff: checkpoint through Plan Manager; final report: changed, verified, remaining, unverified.\n")
	b.WriteString("Non-goals: widen scope, loosen or delete tests, rerun unchanged validation for a greener result.\n")
	fmt.Fprintf(&b, "Operator note: %s", in.OperatorNote)

	message := b.String()
	if len([]rune(message)) > GoalPromptMaxChars {
		return "", &GoalMessageTooLongError{Length: len([]rune(message)), Limit: GoalPromptMaxChars}
	}
	return message, nil
}

func blockerRepairInstruction(policy string) string {
	switch policy {
	case BlockerRepairInvestigate:
		return "Investigate external blockers and preserve a falsifiable diagnosis; do not modify outside the accepted boundary without a recorded extension or operator decision."
	case BlockerRepairAndContinue:
		return "You are authorized and expected to investigate, root-cause, safely repair, and validate an external dependency or infrastructure blocker when necessary for the target. Treat missing or broken producers as work, use the owning path and record a boundary extension before edits, then continue the goal. Do not stop merely to report a repairable blocker. Stop only for missing authority, credentials, approval, unsafe, destructive, paid, private, or unresolved ambiguous actions."
	default:
		return "Stay within the accepted boundary; diagnose external blockers and report the exact evidence and repair decision needed."
	}
}

// RenderFinishLine returns the finish line (the `until` text) that Agent Manager
// receives. It is the destination slot alone, rendered with real values.
func RenderFinishLine(in GoalMessageInput) string {
	shape := strings.TrimSpace(in.PlanShape)
	if shape == "mandate" {
		return fmt.Sprintf("Every required row in %s.setpoint-read is in band from fresh owner-backed evidence; required protocols and exclusions are recorded; the evidence audit passes.", in.ScenarioName)
	}
	return fmt.Sprintf("Plan %s is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.", in.PlanSlug)
}
