// This file composes the Swarm goal message delivered to one Agent Manager run.
package execution

import (
	"fmt"
	"strings"
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

// goalMessageInputForItem builds the composer input from a backlog item. The
// plan shape defaults to phased until Plan B ships the shape field.
func goalMessageInputForItem(item backlogItem, executionID string) GoalMessageInput {
	slug := ""
	if item.PlanRef != nil {
		slug = item.PlanRef.Slug
	}
	return GoalMessageInput{
		PlanShape:       "phased",
		PlanSlug:        slug,
		PlanExecutionID: executionID,
		AcceptanceAllow: item.AcceptanceAllow,
		ScopePolicy:     item.ScopePolicy,
		OperatorNote:    item.OperatorNote,
	}
}

// ComposeGoalMessage renders the harness-goal-authoring slots in order for a
// phased plan (Shape A) or a mandate (Shape B), then the operator note verbatim.
// It never truncates.
func ComposeGoalMessage(in GoalMessageInput) (string, error) {
	shape := strings.TrimSpace(in.PlanShape)
	if shape == "" {
		shape = "phased"
	}
	var b strings.Builder
	switch shape {
	case "mandate":
		fmt.Fprintf(&b, "/goal Every required setpoint row in %s-improve is in band on run %s.setpoint-read, read as often as the outcome contract requires, and the evidence audit passes; or the handoff names each out-of-band row with its blocker.\n\n", in.ScenarioName, in.ScenarioName)
		fmt.Fprintf(&b, "Authority: %s grants development inside the plan's acceptance_allow.\n", in.ProgramName)
	case "phased":
		fmt.Fprintf(&b, "/goal Plan %s is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.\n\n", in.PlanSlug)
	default:
		return "", fmt.Errorf("unknown plan shape %q (want phased or mandate)", in.PlanShape)
	}

	fmt.Fprintf(&b, "Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring. Then: plan-manager exec continue %s. The plan holds the design.\n", in.PlanSlug)
	b.WriteString("Proof: the phase validation commands named in the plan, output shown. Targeted checks by default.\n")
	boundary := strings.Join(in.AcceptanceAllow, ", ")
	scope := strings.TrimSpace(in.ScopePolicy)
	if scope == "" {
		scope = "fixed"
	}
	fmt.Fprintf(&b, "Boundary: %s. Scope policy: %s. Under extend-with-record run plan-manager exec boundary-extend %s before any out-of-scope edit.\n", boundary, scope, in.PlanExecutionID)
	b.WriteString("Adjacent defects: fix when you understand the cause and it blocks a phase; otherwise file with plan-manager log bug-add and continue.\n")
	b.WriteString("Quality: production code, no shims or dead code, docs updated with the code.\n")
	b.WriteString("Blocked means a decision, credential, or approval you lack. Name it in the handoff. Friction you can diagnose is not blocked.\n")
	budget := strings.TrimSpace(in.Budget)
	if budget == "" {
		budget = "the item's remaining charge allowance"
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
