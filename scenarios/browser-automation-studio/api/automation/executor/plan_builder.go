package executor

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

var (
	compilerRegistryMu sync.RWMutex
	compilerRegistry   = map[string]PlanCompiler{}
)

// PlanCompiler emits engine-agnostic execution plans and compiled instructions
// ready for orchestration.
type PlanCompiler interface {
	Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error)
}

// DefaultPlanCompiler supplies the legacy browserless-backed compiler while we
// bring additional engines online.
var DefaultPlanCompiler PlanCompiler = &ContractPlanCompiler{}

// BuildContractsPlan compiles a workflow into the engine-agnostic plan +
// instructions expected by the executor path.
func BuildContractsPlan(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	return BuildContractsPlanWithCompiler(ctx, executionID, workflow, DefaultPlanCompiler)
}

// BuildContractsPlanWithCompiler allows callers to inject a custom compiler
// (e.g., desktop automation) without altering executor orchestration.
func BuildContractsPlanWithCompiler(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow, compiler PlanCompiler) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
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
	if env := strings.ToLower(strings.TrimSpace(os.Getenv("BAS_PLAN_COMPILER"))); env != "" {
		switch env {
		case "contract":
			return &ContractPlanCompiler{}
		case "legacy", "runtime":
			return &BrowserlessPlanCompiler{}
		}
	}

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
	// Contract-native compiler is default; browserless-runtime stays available for legacy parity checks.
	RegisterPlanCompiler("browserless", DefaultPlanCompiler)
	RegisterPlanCompiler("browserless-contract", &ContractPlanCompiler{})
	RegisterPlanCompiler("browserless-runtime", &BrowserlessPlanCompiler{})
}
