package domain

import "fmt"

const WorkflowBudgetMeteredCancellation = "metered-cancellation"

// ValidateWorkflowBudgetPolicy is shared by catalog admission and execution.
// Opt-in metering currently qualifies sequential fresh runs. Continuations reuse
// a Run's event stream and need attempt-scoped metering before qualification;
// parallel and nested workflows need shared reservations, not independent caps.
func ValidateWorkflowBudgetPolicy(d WorkflowDefinition) error {
	if d.Budgets.Enforcement == "" {
		return nil // Existing revisions retain admission-only behavior.
	}
	if d.Budgets.Enforcement != WorkflowBudgetMeteredCancellation {
		return fmt.Errorf("budget enforcement %q is unsupported; hard ceilings require a qualified execution path", d.Budgets.Enforcement)
	}
	for _, n := range d.Nodes {
		switch n.Kind {
		case WorkflowNodeRun, WorkflowNodeWait, WorkflowNodeEnd:
		case WorkflowNodeBranch:
			if n.Branch == nil || n.Branch.Parallel {
				return fmt.Errorf("metered cancellation does not support parallel branches")
			}
		default:
			return fmt.Errorf("metered cancellation does not support node kind %q", n.Kind)
		}
		if n.Run != nil && n.Run.ResultSpec != nil && (n.Run.ResultSpec.SchemaRepairAttempts == nil || *n.Run.ResultSpec.SchemaRepairAttempts != 0) {
			return fmt.Errorf("metered cancellation requires explicit zero schema-repair continuations")
		}
	}
	return nil
}
