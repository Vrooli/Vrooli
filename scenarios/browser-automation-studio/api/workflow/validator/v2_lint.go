package validator

import (
	"fmt"
	"strings"
	"time"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// ValidateV2 validates a V2 proto workflow definition.
// This provides semantic validation beyond proto unmarshalling.
func (v *Validator) ValidateV2(definition *basworkflows.WorkflowDefinitionV2) *Result {
	start := time.Now()
	result := &Result{
		SchemaVersion: "workflow_definition_v2",
		CheckedAt:     time.Now(),
		Stats:         Stats{},
	}

	if definition == nil {
		result.Errors = append(result.Errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_NIL",
			Message:  "Workflow definition is nil",
		})
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	nodes := definition.GetNodes()
	edges := definition.GetEdges()
	result.Stats.NodeCount = len(nodes)
	result.Stats.EdgeCount = len(edges)

	if len(nodes) == 0 {
		result.Errors = append(result.Errors, Issue{
			Severity: SeverityError,
			Code:     "WF_NODE_EMPTY",
			Message:  "Workflow must contain at least one node",
			Pointer:  "/nodes",
		})
	}

	if len(edges) == 0 && len(nodes) > 1 {
		result.Warnings = append(result.Warnings, Issue{
			Severity: SeverityWarning,
			Code:     "WF_EDGE_EMPTY",
			Message:  "Workflow does not contain any edges; execution order may be undefined",
			Pointer:  "/edges",
		})
	}

	// Track node IDs for edge validation
	nodeIndex := make(map[string]struct{}, len(nodes))

	for idx, node := range nodes {
		if node == nil {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_SHAPE",
				Message:  fmt.Sprintf("Node %d is nil", idx),
				Pointer:  fmt.Sprintf("/nodes/%d", idx),
			})
			continue
		}

		nodeID := node.GetId()
		if nodeID == "" {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_NODE_ID_MISSING",
				Message:  fmt.Sprintf("Node %d is missing an id", idx),
				Pointer:  fmt.Sprintf("/nodes/%d/id", idx),
			})
		} else {
			if _, exists := nodeIndex[nodeID]; exists {
				result.Errors = append(result.Errors, Issue{
					Severity: SeverityError,
					Code:     "WF_NODE_ID_DUPLICATE",
					Message:  fmt.Sprintf("Node id '%s' is duplicated", nodeID),
					NodeID:   nodeID,
					Pointer:  fmt.Sprintf("/nodes/%d/id", idx),
				})
			}
			nodeIndex[nodeID] = struct{}{}
		}

		action := node.GetAction()
		if action == nil {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_V2_ACTION_MISSING",
				Message:  fmt.Sprintf("Node '%s' is missing an action", nodeID),
				NodeID:   nodeID,
				Pointer:  fmt.Sprintf("/nodes/%d/action", idx),
			})
			continue
		}

		actionType := action.GetType()
		if actionType == basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_V2_ACTION_TYPE_MISSING",
				Message:  fmt.Sprintf("Node '%s' has unspecified action type", nodeID),
				NodeID:   nodeID,
				Pointer:  fmt.Sprintf("/nodes/%d/action/type", idx),
			})
			continue
		}

		// Action-specific validation
		switch actionType {
		case basactions.ActionType_ACTION_TYPE_LOOP:
			errors, warnings := lintLoopNodeV2(nodeID, idx, action)
			result.Errors = append(result.Errors, errors...)
			result.Warnings = append(result.Warnings, warnings...)

		case basactions.ActionType_ACTION_TYPE_SUBFLOW:
			errors, warnings := lintSubflowNodeV2(nodeID, idx, action)
			result.Errors = append(result.Errors, errors...)
			result.Warnings = append(result.Warnings, warnings...)

		case basactions.ActionType_ACTION_TYPE_CLICK,
			basactions.ActionType_ACTION_TYPE_HOVER,
			basactions.ActionType_ACTION_TYPE_FOCUS,
			basactions.ActionType_ACTION_TYPE_BLUR:
			errors, warnings := lintSelectorRequiredV2(nodeID, idx, action)
			result.Errors = append(result.Errors, errors...)
			result.Warnings = append(result.Warnings, warnings...)
		}

		// Check for label (warning only)
		metadata := action.GetMetadata()
		if metadata == nil || strings.TrimSpace(metadata.GetLabel()) == "" {
			result.Warnings = append(result.Warnings, Issue{
				Severity: SeverityWarning,
				Code:     "WF_NODE_LABEL_MISSING",
				Message:  "Node is missing a label; consider adding one for readability",
				NodeID:   nodeID,
				Pointer:  fmt.Sprintf("/nodes/%d/action/metadata/label", idx),
			})
		}
	}

	// Validate edges
	for idx, edge := range edges {
		if edge == nil {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_SHAPE",
				Message:  fmt.Sprintf("Edge %d is nil", idx),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
			})
			continue
		}

		source := strings.TrimSpace(edge.GetSource())
		target := strings.TrimSpace(edge.GetTarget())
		if source == "" || target == "" {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_ENDPOINT_MISSING",
				Message:  fmt.Sprintf("Edge %d must define a source and target", idx),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
			})
			continue
		}
		if source == target {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_CYCLE_SELF",
				Message:  fmt.Sprintf("Edge %d connects node '%s' to itself", idx, source),
				Pointer:  fmt.Sprintf("/edges/%d", idx),
				NodeID:   source,
			})
		}
		if _, ok := nodeIndex[source]; !ok {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_SOURCE_UNKNOWN",
				Message:  fmt.Sprintf("Edge %d references unknown source node '%s'", idx, source),
				Pointer:  fmt.Sprintf("/edges/%d/source", idx),
			})
		}
		if _, ok := nodeIndex[target]; !ok {
			result.Errors = append(result.Errors, Issue{
				Severity: SeverityError,
				Code:     "WF_EDGE_TARGET_UNKNOWN",
				Message:  fmt.Sprintf("Edge %d references unknown target node '%s'", idx, target),
				Pointer:  fmt.Sprintf("/edges/%d/target", idx),
			})
		}
	}

	result.Valid = len(result.Errors) == 0
	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

// lintLoopNodeV2 validates loop action parameters.
func lintLoopNodeV2(nodeID string, idx int, action *basactions.ActionDefinition) ([]Issue, []Issue) {
	var errors []Issue
	var warnings []Issue
	pointer := fmt.Sprintf("/nodes/%d/action/loop", idx)

	loopParams := action.GetLoop()
	if loopParams == nil {
		errors = append(errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_LOOP_PARAMS_MISSING",
			Message:  fmt.Sprintf("Loop node '%s' is missing loop parameters", nodeID),
			NodeID:   nodeID,
			Pointer:  pointer,
		})
		return errors, warnings
	}

	loopType := loopParams.GetLoopType()
	if loopType == basactions.LoopType_LOOP_TYPE_UNSPECIFIED {
		errors = append(errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_LOOP_TYPE_REQUIRED",
			Message:  fmt.Sprintf("Loop node '%s' must specify a loop_type", nodeID),
			NodeID:   nodeID,
			Field:    "loop_type",
			Pointer:  pointer + "/loop_type",
		})
		return errors, warnings
	}

	switch loopType {
	case basactions.LoopType_LOOP_TYPE_FOREACH:
		arraySource := strings.TrimSpace(loopParams.GetArraySource())
		if arraySource == "" {
			errors = append(errors, Issue{
				Severity: SeverityError,
				Code:     "WF_V2_LOOP_FOREACH_SOURCE_REQUIRED",
				Message:  fmt.Sprintf("Loop node '%s' (foreach) requires array_source", nodeID),
				NodeID:   nodeID,
				Field:    "array_source",
				Pointer:  pointer + "/array_source",
				Hint:     "Set array_source to a variable containing an array, e.g. ${items}",
			})
		}
		itemVar := strings.TrimSpace(loopParams.GetItemVariable())
		if itemVar == "" {
			warnings = append(warnings, Issue{
				Severity: SeverityWarning,
				Code:     "WF_V2_LOOP_FOREACH_ITEM_VAR",
				Message:  fmt.Sprintf("Loop node '%s' (foreach) should define item_variable", nodeID),
				NodeID:   nodeID,
				Field:    "item_variable",
				Pointer:  pointer + "/item_variable",
				Hint:     "Set item_variable to name the current item (default: 'item')",
			})
		}

	case basactions.LoopType_LOOP_TYPE_REPEAT:
		count := loopParams.GetCount()
		if count <= 0 {
			errors = append(errors, Issue{
				Severity: SeverityError,
				Code:     "WF_V2_LOOP_REPEAT_COUNT_REQUIRED",
				Message:  fmt.Sprintf("Loop node '%s' (repeat) requires count > 0", nodeID),
				NodeID:   nodeID,
				Field:    "count",
				Pointer:  pointer + "/count",
			})
		}

	case basactions.LoopType_LOOP_TYPE_WHILE:
		condition := loopParams.GetCondition()
		if condition == nil {
			errors = append(errors, Issue{
				Severity: SeverityError,
				Code:     "WF_V2_LOOP_WHILE_CONDITION_REQUIRED",
				Message:  fmt.Sprintf("Loop node '%s' (while) requires a condition", nodeID),
				NodeID:   nodeID,
				Field:    "condition",
				Pointer:  pointer + "/condition",
			})
		}
	}

	// Safety: warn if no max_iterations set
	maxIter := loopParams.GetMaxIterations()
	if maxIter <= 0 {
		warnings = append(warnings, Issue{
			Severity: SeverityWarning,
			Code:     "WF_V2_LOOP_MAX_ITERATIONS",
			Message:  fmt.Sprintf("Loop node '%s' has no max_iterations; consider setting a safety limit", nodeID),
			NodeID:   nodeID,
			Field:    "max_iterations",
			Pointer:  pointer + "/max_iterations",
			Hint:     "Set max_iterations to prevent infinite loops (recommended: 100-1000)",
		})
	}

	return errors, warnings
}

// lintSubflowNodeV2 validates subflow action parameters.
func lintSubflowNodeV2(nodeID string, idx int, action *basactions.ActionDefinition) ([]Issue, []Issue) {
	var errors []Issue
	var warnings []Issue
	pointer := fmt.Sprintf("/nodes/%d/action/subflow", idx)

	subflowParams := action.GetSubflow()
	if subflowParams == nil {
		errors = append(errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_SUBFLOW_PARAMS_MISSING",
			Message:  fmt.Sprintf("Subflow node '%s' is missing subflow parameters", nodeID),
			NodeID:   nodeID,
			Pointer:  pointer,
		})
		return errors, warnings
	}

	// Must have either workflow_id OR workflow_path
	workflowID := strings.TrimSpace(subflowParams.GetWorkflowId())
	workflowPath := strings.TrimSpace(subflowParams.GetWorkflowPath())

	if workflowID == "" && workflowPath == "" {
		errors = append(errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_SUBFLOW_TARGET_REQUIRED",
			Message:  fmt.Sprintf("Subflow node '%s' must define workflow_id or workflow_path", nodeID),
			NodeID:   nodeID,
			Pointer:  pointer,
			Hint:     "Set either workflow_id (UUID) or workflow_path (relative path like 'actions/login.json')",
		})
	}

	if workflowID != "" && workflowPath != "" {
		warnings = append(warnings, Issue{
			Severity: SeverityWarning,
			Code:     "WF_V2_SUBFLOW_TARGET_AMBIGUOUS",
			Message:  fmt.Sprintf("Subflow node '%s' has both workflow_id and workflow_path; workflow_id will be used", nodeID),
			NodeID:   nodeID,
			Pointer:  pointer,
		})
	}

	return errors, warnings
}

// lintSelectorRequiredV2 validates actions that require a selector.
func lintSelectorRequiredV2(nodeID string, idx int, action *basactions.ActionDefinition) ([]Issue, []Issue) {
	var errors []Issue
	var warnings []Issue

	actionType := action.GetType()
	var selector string
	var pointer string

	switch actionType {
	case basactions.ActionType_ACTION_TYPE_CLICK:
		if click := action.GetClick(); click != nil {
			selector = click.GetSelector()
			pointer = fmt.Sprintf("/nodes/%d/action/click/selector", idx)
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		if hover := action.GetHover(); hover != nil {
			selector = hover.GetSelector()
			pointer = fmt.Sprintf("/nodes/%d/action/hover/selector", idx)
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		if focus := action.GetFocus(); focus != nil {
			selector = focus.GetSelector()
			pointer = fmt.Sprintf("/nodes/%d/action/focus/selector", idx)
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		if blur := action.GetBlur(); blur != nil {
			selector = blur.GetSelector()
			pointer = fmt.Sprintf("/nodes/%d/action/blur/selector", idx)
		}
	}

	if strings.TrimSpace(selector) == "" {
		errors = append(errors, Issue{
			Severity: SeverityError,
			Code:     "WF_V2_SELECTOR_REQUIRED",
			Message:  fmt.Sprintf("Node '%s' (%s) requires a selector", nodeID, actionType.String()),
			NodeID:   nodeID,
			Field:    "selector",
			Pointer:  pointer,
		})
	}

	return errors, warnings
}
