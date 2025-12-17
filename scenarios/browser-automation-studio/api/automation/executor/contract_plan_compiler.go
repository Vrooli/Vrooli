package executor

import (
	"context"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// ContractPlanCompiler produces contract-native plans without touching the
// browserless runtime instruction shaper. It keeps instruction params exactly
// as authored in the workflow definition so multiple engines can consume the
// same plan shape.
//
// This type delegates to compiler.CompileWorkflowToContracts which centralizes
// the conversion from compiler types to contracts types.
type ContractPlanCompiler struct{}

func (c *ContractPlanCompiler) Compile(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	// Delegate to the centralized conversion function in the compiler package.
	// This avoids duplicating the conversion logic between compiler/ and executor/.
	return autocompiler.CompileWorkflowToContracts(ctx, executionID, workflow)
}
