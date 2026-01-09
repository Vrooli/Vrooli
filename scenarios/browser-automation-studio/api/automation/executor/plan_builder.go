package executor

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

var (
	compilerRegistryMu sync.RWMutex
	compilerRegistry   = map[string]PlanCompiler{}
)

// PlanCompiler emits engine-agnostic execution plans and compiled instructions
// ready for orchestration.
type PlanCompiler interface {
	Compile(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error)
}

// PlanCompilerFunc is a function adapter that implements PlanCompiler.
// This allows using a simple function as a PlanCompiler without a wrapper struct.
type PlanCompilerFunc func(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error)

// Compile implements PlanCompiler by calling the underlying function.
func (f PlanCompilerFunc) Compile(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	return f(ctx, executionID, workflow)
}

// DefaultPlanCompiler supplies the contract-native compiler so multiple engines
// can share the same plan shape. It delegates to compiler.CompileWorkflowToContracts.
var DefaultPlanCompiler PlanCompiler = PlanCompilerFunc(autocompiler.CompileWorkflowToContracts)

// BuildContractsPlan compiles a workflow into the engine-agnostic plan +
// instructions expected by the executor path.
func BuildContractsPlan(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	return BuildContractsPlanWithCompiler(ctx, executionID, workflow, DefaultPlanCompiler)
}

// BuildContractsPlanWithCompiler allows callers to inject a custom compiler
// (e.g., desktop automation) without altering executor orchestration.
func BuildContractsPlanWithCompiler(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary, compiler PlanCompiler) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	if compiler == nil {
		compiler = DefaultPlanCompiler
	}
	return compiler.Compile(ctx, executionID, workflow)
}

// RegisterPlanCompiler associates an engine name with a plan compiler. Names are
// case-insensitive. Callers should register during init for custom engines.
func RegisterPlanCompiler(engineName string, compiler PlanCompiler) {
	name := strings.ToLower(strings.TrimSpace(engineName))
	if name == "" || compiler == nil {
		return
	}
	compilerRegistryMu.Lock()
	defer compilerRegistryMu.Unlock()
	compilerRegistry[name] = compiler
}

// PlanCompilerForEngine returns a compiler registered for the engine name, or
// the default compiler when none is registered.
func PlanCompilerForEngine(engineName string) PlanCompiler {
	name := strings.ToLower(strings.TrimSpace(engineName))
	compilerRegistryMu.RLock()
	if c, ok := compilerRegistry[name]; ok && c != nil {
		compilerRegistryMu.RUnlock()
		return c
	}
	compilerRegistryMu.RUnlock()
	return DefaultPlanCompiler
}

func init() {
	// Contract-native compiler is default; browserless runtime shaping was removed.
	// Both "browserless" and "browserless-contract" map to the same compiler since
	// there's only one compilation strategy now.
	RegisterPlanCompiler("browserless", DefaultPlanCompiler)
	RegisterPlanCompiler("browserless-contract", DefaultPlanCompiler)
}
