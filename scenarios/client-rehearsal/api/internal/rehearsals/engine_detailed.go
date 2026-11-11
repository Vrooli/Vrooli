package rehearsals

import (
	"context"
	"fmt"
	"time"
)

// ExecuteDetailed returns the final synthetic graph output for local contract
// assertions. The output remains internal and is never serialized in the run
// record; trace summaries and assertion paths carry only bounded evidence.
func (SyntheticEngine) ExecuteDetailed(ctx context.Context, workflow Workflow, triggerNodeID string, simulator Simulator, injectFault bool, now time.Time) (ExecutionResult, error) {
	return executeDetailedWithInput(ctx, workflow, triggerNodeID, simulator, injectFault, now, nil)
}

func executeDetailedWithInput(ctx context.Context, workflow Workflow, triggerNodeID string, simulator Simulator, injectFault bool, now time.Time, supplied map[string]any) (ExecutionResult, error) {
	result := ExecutionResult{Trace: make([]TraceEvent, 0, len(workflow.Nodes)*2)}
	if len(workflow.EntryNodeIDs) == 0 {
		return result, fmt.Errorf("workflow %s has no externally invokable trigger", workflow.ID)
	}
	if triggerNodeID == "" && len(workflow.EntryNodeIDs) == 1 {
		triggerNodeID = workflow.EntryNodeIDs[0]
	}
	entry := ""
	for _, candidate := range workflow.EntryNodeIDs {
		if candidate == triggerNodeID {
			entry = candidate
			break
		}
	}
	if entry == "" {
		return result, ErrInvalid{Field: "trigger_node_id", Reason: "must identify one of the imported workflow's trigger nodes"}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	kind := workflow.NodesByID()[entry].Trigger
	var err error
	input := syntheticTriggerInput(kind)
	if supplied != nil {
		input = cloneMap(supplied)
	}
	result.Output, err = executeWorkflowEntry(ctx, workflow, entry, input, simulator, injectFault, now, &result.Trace, map[string]bool{}, 0)
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return result, err
}
