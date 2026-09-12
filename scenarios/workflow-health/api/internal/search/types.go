package search

import "workflow-health/internal/workflows"

const (
	LeafTypeFlow     = "workflow.flow"
	LeafTypeTest     = "workflow.test"
	LeafTypeFragment = "workflow.fragment"
)

type Options struct {
	Query            string
	Types            []string
	IncludeFragments bool
	Limit            int
}

type Result struct {
	ID                   string                  `json:"id"`
	LeafType             string                  `json:"leaf_type"`
	Asset                workflows.WorkflowAsset `json:"asset"`
	Title                string                  `json:"title"`
	Snippet              string                  `json:"snippet"`
	Score                float64                 `json:"score"`
	Runnable             bool                    `json:"runnable"`
	SafetySummary        string                  `json:"safety_summary"`
	Guardrails           []string                `json:"guardrails,omitempty"`
	RequirementIDs       []string                `json:"requirement_ids,omitempty"`
	SelectorRefs         []string                `json:"selector_refs,omitempty"`
	RouteRefs            []string                `json:"route_refs,omitempty"`
	LabelPairs           []string                `json:"label_pairs,omitempty"`
	DependencyPaths      []string                `json:"dependency_paths,omitempty"`
	RequiresConfirmation bool                    `json:"requires_confirmation"`
	RequiresIsolation    bool                    `json:"requires_isolation"`
	Mutating             bool                    `json:"mutating"`
}
