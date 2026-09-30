package workflows

import (
	"fmt"

	actions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	workflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

// Automated repair preserves the acceptance contract. Changing an assertion or
// execution policy is a separate authoring decision, not a successful repair.
func validatePromotionChecks(before, after *workflows.WorkflowDefinitionV2) error {
	if after == nil {
		return fmt.Errorf("candidate definition required")
	}
	checks := map[string]*workflows.WorkflowNodeV2{}
	outgoing := map[string]bool{}
	for _, e := range after.Edges {
		outgoing[e.Source] = true
	}
	for _, n := range after.Nodes {
		if !outgoing[n.Id] && n.GetAction().GetType() != actions.ActionType_ACTION_TYPE_ASSERT {
			return fmt.Errorf("every terminal path must end in an outcome assertion")
		}
		if n.GetAction().GetType() == actions.ActionType_ACTION_TYPE_ASSERT {
			checks[n.Id] = n
		}
	}
	if len(checks) == 0 {
		return fmt.Errorf("promotion requires an explicit outcome assertion")
	}
	if before == nil {
		return nil
	}
	if !proto.Equal(before.Settings, after.Settings) || !proto.Equal(before.Metadata, after.Metadata) {
		return fmt.Errorf("automated repair must preserve workflow settings and metadata")
	}
	if len(before.Nodes) != len(after.Nodes) || len(before.Edges) != len(after.Edges) {
		return fmt.Errorf("automated repair must preserve graph topology")
	}
	for i, e := range before.Edges {
		if !proto.Equal(e, after.Edges[i]) {
			return fmt.Errorf("automated repair must preserve graph topology")
		}
	}
	ids := map[string]bool{}
	for _, n := range after.Nodes {
		ids[n.Id] = true
	}
	for _, n := range before.Nodes {
		if !ids[n.Id] {
			return fmt.Errorf("automated repair must preserve assertion and node identities")
		}
		if n.GetAction().GetType() == actions.ActionType_ACTION_TYPE_ASSERT {
			next := checks[n.Id]
			if next == nil || !proto.Equal(n.Action, next.Action) || !proto.Equal(n.ExecutionSettings, next.ExecutionSettings) {
				return fmt.Errorf("automated repair must preserve assertion %s", n.Id)
			}
		}
	}
	return nil
}
