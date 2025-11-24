package executor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
)

// BrowserlessPlanCompiler produces engine-agnostic plans from workflow
// definitions using the existing browserless compiler/runtime pipeline. The
// executor stays agnostic to the underlying engine.
type BrowserlessPlanCompiler struct{}

// Compile converts a workflow into the contract ExecutionPlan + compiled instructions.
func (c *BrowserlessPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	plan, err := compiler.CompileWorkflow(workflow)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}
	instructions, err := runtime.InstructionsFromPlan(ctx, plan)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	compiled := make([]contracts.CompiledInstruction, 0, len(instructions))
	for _, instr := range instructions {
		params := map[string]any{}
		raw, _ := json.Marshal(instr.Params)
		_ = json.Unmarshal(raw, &params)

		contextMap := map[string]any{}
		if len(instr.Context) > 0 {
			rawCtx, _ := json.Marshal(instr.Context)
			_ = json.Unmarshal(rawCtx, &contextMap)
		}

		compiled = append(compiled, contracts.CompiledInstruction{
			Index:       instr.Index,
			NodeID:      instr.NodeID,
			Type:        instr.Type,
			Params:      params,
			PreloadHTML: instr.PreloadHTML,
			Context:     contextMap,
			Metadata:    map[string]string{},
		})
	}

	return contracts.ExecutionPlan{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflow.ID,
		Instructions:   compiled,
		Graph:          toContractsGraph(plan),
		Metadata:       plan.Metadata,
		CreatedAt:      time.Now().UTC(),
	}, compiled, nil
}
