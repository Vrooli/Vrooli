package executor

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	autoworkflow "github.com/vrooli/browser-automation-studio/automation/workflow"
	"github.com/vrooli/browser-automation-studio/database"
)

// ContractPlanCompiler produces contract-native plans without touching the
// browserless runtime instruction shaper. It keeps instruction params exactly
// as authored in the workflow definition so multiple engines can consume the
// same plan shape.
type ContractPlanCompiler struct{}

func (c *ContractPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	plan, err := autocompiler.CompileWorkflow(workflow)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	instructions := make([]contracts.CompiledInstruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		params := map[string]any{}
		for k, v := range step.Params {
			params[k] = v
		}

		instr := contracts.CompiledInstruction{
			Index:       step.Index,
			NodeID:      step.NodeID,
			Type:        string(step.Type),
			Params:      params,
			PreloadHTML: "",
			Context:     map[string]any{},
			Metadata:    map[string]string{},
		}

		// MIGRATION: Populate typed Action field from V1 type/params
		// This enables protovalidate and typed param access in playwright-driver
		action, actionErr := autoworkflow.V1DataToActionDefinition(string(step.Type), step.Params)
		if actionErr != nil {
			// Log but don't fail - Action is optional during migration
			log.Printf("[COMPILER] Warning: failed to build ActionDefinition for step %d (%s): %v", step.Index, step.Type, actionErr)
		} else {
			instr.Action = action
		}

		instructions = append(instructions, instr)
	}

	contractPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflow.ID,
		Instructions:   instructions,
		Graph:          toContractsGraphFromAutomation(plan),
		Metadata:       plan.Metadata,
		CreatedAt:      time.Now().UTC(),
	}
	return contractPlan, instructions, nil
}
