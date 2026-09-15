// Package executor provides workflow execution capabilities.
package executor

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/state"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// LoopContext bundles all state needed for loop execution.
// This reduces parameter passing and makes the interface cleaner.
type LoopContext struct {
	Ctx           context.Context
	Request       Request
	ExecCtx       executionContext
	Engine        engine.AutomationEngine
	Spec          engine.SessionSpec
	Session       engine.EngineSession
	Step          contracts.PlanStep
	State         *state.ExecutionState
	ReuseMode     engine.SessionReuseMode
	MaxIterations int
}

// LoopHandler defines the interface for loop execution strategies.
// Each loop type (repeat, foreach, while) implements this interface.
type LoopHandler interface {
	// Execute runs the loop with the given context.
	// The executor is passed to allow calling executeGraphIteration.
	Execute(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error)
}

// LoopHandlerFunc is an adapter to allow ordinary functions to be used as LoopHandlers.
type LoopHandlerFunc func(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error)

// Execute implements LoopHandler.
func (f LoopHandlerFunc) Execute(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error) {
	return f(executor, lctx)
}

var (
	loopRegistryMu sync.RWMutex
	loopRegistry   = map[string]LoopHandler{}
)

// RegisterLoopHandler registers a handler for a specific loop type.
// Names are case-insensitive and whitespace-trimmed.
func RegisterLoopHandler(loopType string, handler LoopHandler) {
	name := normalizeLoopType(loopType)
	if name == "" || handler == nil {
		return
	}
	loopRegistryMu.Lock()
	defer loopRegistryMu.Unlock()
	loopRegistry[name] = handler
}

// GetLoopHandler returns the registered handler for a loop type.
// Returns nil if no handler is registered.
func GetLoopHandler(loopType string) LoopHandler {
	name := normalizeLoopType(loopType)
	loopRegistryMu.RLock()
	defer loopRegistryMu.RUnlock()
	return loopRegistry[name]
}

// SupportedLoopTypes returns a list of all registered loop type names.
func SupportedLoopTypes() []string {
	loopRegistryMu.RLock()
	defer loopRegistryMu.RUnlock()
	types := make([]string, 0, len(loopRegistry))
	for t := range loopRegistry {
		types = append(types, t)
	}
	return types
}

func normalizeLoopType(loopType string) string {
	return strings.ToLower(strings.TrimSpace(loopType))
}

// =============================================================================
// LOOP HANDLER IMPLEMENTATIONS
// =============================================================================

// repeatHandler implements the repeat loop (fixed count iterations).
type repeatHandler struct{}

func (h *repeatHandler) Execute(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error) {
	result := loopExecutionResult{session: lctx.Session}
	loop := lctx.Step.Action.GetLoop()
	if loop == nil {
		return result, fmt.Errorf("loop node %s is missing typed loop params", lctx.Step.NodeID)
	}
	desiredIterations := int(loop.GetCount())
	if desiredIterations <= 0 {
		return result, fmt.Errorf("loop node %s repeat requires loopCount > 0", lctx.Step.NodeID)
	}

	clampedIterations := minInt(desiredIterations, lctx.MaxIterations)
	if clampedIterations == 0 {
		return result, fmt.Errorf("loop node %s has zero iterations after clamping", lctx.Step.NodeID)
	}

	activeSession := lctx.Session
	for i := 0; i < clampedIterations; i++ {
		control, nextSession, err := executor.executeGraphIteration(
			lctx.Ctx, lctx.Request, lctx.ExecCtx, lctx.Engine, lctx.Spec,
			activeSession, lctx.Step.Loop, lctx.State, lctx.ReuseMode,
		)
		if err != nil {
			result.session = nextSession
			return result, err
		}
		activeSession = nextSession
		result.lastOutcome = control.LastOutcome
		if control.Break {
			break
		}
	}

	result.iterations = clampedIterations
	result.session = activeSession
	return result, nil
}

// forEachHandler implements the foreach loop (iterate over items).
type forEachHandler struct{}

func (h *forEachHandler) Execute(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error) {
	result := loopExecutionResult{session: lctx.Session}
	loop := lctx.Step.Action.GetLoop()
	if loop == nil {
		return result, fmt.Errorf("loop node %s is missing typed loop params", lctx.Step.NodeID)
	}
	items := loopItems(loop, lctx.State)
	if len(items) == 0 {
		return result, nil
	}

	itemVar := loop.GetItemVariable()
	if itemVar == "" {
		itemVar = defaultLoopItemVar
	}
	indexVar := loop.GetIndexVariable()
	if indexVar == "" {
		indexVar = defaultLoopIndexVar
	}

	activeSession := lctx.Session
	upperBound := minInt(lctx.MaxIterations, len(items))
	executed := 0
	for i := 0; i < upperBound; i++ {
		lctx.State.Set(itemVar, items[i])
		lctx.State.Set(indexVar, i)

		control, nextSession, err := executor.executeGraphIteration(
			lctx.Ctx, lctx.Request, lctx.ExecCtx, lctx.Engine, lctx.Spec,
			activeSession, lctx.Step.Loop, lctx.State, lctx.ReuseMode,
		)
		if err != nil {
			result.session = nextSession
			return result, err
		}
		activeSession = nextSession
		executed++
		result.lastOutcome = control.LastOutcome
		if control.Break {
			break
		}
	}

	result.iterations = executed
	result.session = activeSession
	return result, nil
}

// whileHandler implements the while loop (condition-based iteration).
type whileHandler struct{}

func (h *whileHandler) Execute(executor *SimpleExecutor, lctx LoopContext) (loopExecutionResult, error) {
	result := loopExecutionResult{session: lctx.Session}
	loop := lctx.Step.Action.GetLoop()
	if loop == nil {
		return result, fmt.Errorf("loop node %s is missing typed loop params", lctx.Step.NodeID)
	}

	activeSession := lctx.Session
	iterations := 0
	for iterations < lctx.MaxIterations {
		if !evaluateLoopCondition(loop.GetCondition(), lctx.State) {
			break
		}

		control, nextSession, err := executor.executeGraphIteration(
			lctx.Ctx, lctx.Request, lctx.ExecCtx, lctx.Engine, lctx.Spec,
			activeSession, lctx.Step.Loop, lctx.State, lctx.ReuseMode,
		)
		if err != nil {
			result.session = nextSession
			return result, err
		}
		activeSession = nextSession
		iterations++
		result.lastOutcome = control.LastOutcome
		if control.Break {
			break
		}
	}

	result.iterations = iterations
	result.session = activeSession
	return result, nil
}

func loopItems(loop *basactions.LoopParams, execState *state.ExecutionState) []any {
	if loop == nil || execState == nil || strings.TrimSpace(loop.GetArraySource()) == "" {
		return nil
	}
	value, ok := execState.Get(loop.GetArraySource())
	if !ok || value == nil {
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil
	}
	items := make([]any, rv.Len())
	for i := range items {
		items[i] = rv.Index(i).Interface()
	}
	return items
}

func evaluateLoopCondition(condition *basactions.LoopCondition, execState *state.ExecutionState) bool {
	if condition == nil || execState == nil {
		return false
	}
	switch condition.GetType() {
	case basactions.LoopConditionType_LOOP_CONDITION_TYPE_VARIABLE:
		name := strings.TrimSpace(condition.GetVariable())
		if name == "" {
			return false
		}
		current, ok := execState.Get(name)
		if !ok {
			return false
		}
		op := strings.ToLower(strings.TrimPrefix(condition.GetOperator().String(), "LOOP_CONDITION_OPERATOR_"))
		return state.CompareValues(current, typeconv.JsonValueToAny(condition.GetValue()), op)
	case basactions.LoopConditionType_LOOP_CONDITION_TYPE_EXPRESSION:
		result, ok := state.NewInterpolator(execState).EvaluateExpression(strings.TrimSpace(condition.GetExpression()))
		return ok && result
	default:
		return false
	}
}

// =============================================================================
// REGISTRATION
// =============================================================================

func init() {
	// Register all built-in loop handlers
	RegisterLoopHandler("repeat", &repeatHandler{})
	RegisterLoopHandler("foreach", &forEachHandler{})
	RegisterLoopHandler("while", &whileHandler{})
}
