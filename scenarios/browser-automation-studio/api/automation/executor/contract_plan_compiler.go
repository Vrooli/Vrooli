package executor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/database"
)

// ContractPlanCompiler produces contract-native plans without touching the
// browserless runtime instruction shaper. It keeps instruction params exactly
// as authored in the workflow definition so multiple engines can consume the
// same plan shape.
type ContractPlanCompiler struct{}

func (c *ContractPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	plan, err := compiler.CompileWorkflow(workflow)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	instructions := make([]contracts.CompiledInstruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		params := map[string]any{}
		for k, v := range step.Params {
			params[k] = v
		}
		instructions = append(instructions, contracts.CompiledInstruction{
			Index:       step.Index,
			NodeID:      step.NodeID,
			Type:        string(step.Type),
			Params:      params,
			PreloadHTML: "",
			Context:     map[string]any{},
			Metadata:    map[string]string{},
		})
	}

	contractPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflow.ID,
		Instructions:   instructions,
		Graph:          toContractsGraph(plan),
		Metadata:       plan.Metadata,
		CreatedAt:      time.Now().UTC(),
	}
	return contractPlan, instructions, nil
}
