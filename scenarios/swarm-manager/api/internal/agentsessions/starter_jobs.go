package agentsessions

import "strings"

// starterJobs is server-owned so starter cards cannot smuggle arbitrary
// instructions into the stable job band. The operator's typed message remains
// the task band and always has higher priority.
var starterJobs = map[string]string{
	"operations-review":           "Review active goals and recommend the single next registered action that unblocks the most work. State what is moving, what is stalled, and what evidence is missing or contradictory.",
	"operations-decisions":        "Review the attached Plan Workshop. State what is decided, what is open, what it is waiting on, and the specific next operator action.",
	"operations-run":              "Review the attached failed or stale run. Separate its typed terminal reason from what actually happened, then recommend the registered recovery path.",
	"operations-goal":             "Assess the attached goal against its acceptance criteria and recommend its next registered transition. If your reading differs from the projection, explain the deviation.",
	"operations-triage-staleness": "Triage the attached work for staleness. Return keep, refresh, or supersede with evidence from intent, repository, and owning goal.",
	"operations-sweep-staleness":  "Find the stalest work using Swarm Manager's signal and walk through keep, refresh, or supersede one item at a time, waiting for the operator's decision.",
	"workflow-author-method":      "Classify the operator's described way of working as a session, declared workflow, skill change, or deterministic action, and recommend the specific disposition.",
	"workflow-author-friction":    "Locate the actual source of the described agent-work friction and recommend the smallest change that removes it, stating assumptions and misdiagnoses.",
	"workflow-author-transition":  "Review the existing transition's declaration, skill, and outcomes. Identify contract drift and recommend improve, replace, or leave alone with cost.",
	"workflow-author-scenario":    "Design how agents should handle the described kind of work end to end, selecting a skill change, existing transition improvement, new transition, or backlog work.",
	"meta-plan":                   "Shape the operator's idea against existing goals and backlog items. Recommend what to create or change and flag assumptions.",
	"meta-existing":               "Inspect existing goals, items, and scenarios around the operator's subject. Identify real gaps and recommend what to build next and why.",
	"meta-backlog":                "Inspect the attached backlog item and recommend independent follow-up work, dependencies, and sequencing.",
	"meta-image":                  "Read the attached image as source material, place it against existing goals and items, and recommend what to create without guessing at unclear details.",
	"proposal-split":              "Identify oversized work and propose independently reviewable splits. Return a mutation_list envelope and never apply it.",
	"proposal-merge":              "Identify tightly coupled work whose separation is harmful. Return a mutation_list envelope and never apply it.",
	"proposal-identify-missing":   "Identify missing work needed to satisfy the target outcome. Return a mutation_list envelope and never apply it.",
	"proposal-reconcile":          "Reconcile the target's recorded intent with repository evidence. Return a mutation_list envelope and never apply it.",
	"proposal-reframe":            "Reframe the target's scope and success criteria. Return a mutation_list envelope and never apply it.",
	"proposal-identify-followups": "Identify safe follow-up work for the target. Return a mutation_list envelope and never apply it.",
	"proposal-reframe-item":       "Reframe the target item scope and done condition. Return a mutation_list envelope and never apply it.",
	"proposal-reconcile-item":     "Reconcile the target item with related work and repository evidence. Return a mutation_list envelope and never apply it.",
}

func IsKnownStarterJob(id string) bool {
	_, ok := starterJobs[strings.TrimSpace(id)]
	return ok
}

func starterJobAllowedForKind(id string, kind Kind) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return true
	}
	switch kind {
	case KindSwarmOperations:
		return strings.HasPrefix(id, "operations-") || strings.HasPrefix(id, "proposal-")
	case KindWorkflowAuthoring:
		return strings.HasPrefix(id, "workflow-")
	case KindMetaOrchestration:
		return strings.HasPrefix(id, "meta-")
	default:
		return false
	}
}

func starterJobText(id string) (string, bool) {
	text, ok := starterJobs[strings.TrimSpace(id)]
	return text, ok
}
