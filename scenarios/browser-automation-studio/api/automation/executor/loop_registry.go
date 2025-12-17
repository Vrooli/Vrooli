// Package executor provides workflow execution capabilities.
package executor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/engine"
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
	State         *flowState
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
	stepParams := PlanStepParams(lctx.Step)

	desiredIterations := intValue(stepParams, "loopCount")
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
	stepParams := PlanStepParams(lctx.Step)

	items := extractLoopItems(stepParams, lctx.State)
	if len(items) == 0 {
		return result, nil
	}

	itemVar := stringValue(stepParams, "loopItemVariable")
	if itemVar == "" {
		itemVar = stringValue(stepParams, "itemVariable")
	}
	if itemVar == "" {
		itemVar = defaultLoopItemVar
	}
	indexVar := stringValue(stepParams, "loopIndexVariable")
	if indexVar == "" {
		indexVar = stringValue(stepParams, "indexVariable")
	}
	if indexVar == "" {
		indexVar = defaultLoopIndexVar
	}

	activeSession := lctx.Session
	upperBound := minInt(lctx.MaxIterations, len(items))
	executed := 0
	for i := 0; i < upperBound; i++ {
		lctx.State.set(itemVar, items[i])
		lctx.State.set(indexVar, i)

		control, nextSession, err := executor.executeGraphIteration(
			lctx.Ctx, lctx.Request, lctx.ExecCtx, lctx.Engine, lctx.Spec,
			activeSession, lctx.Step.Loop, lctx.State, lctx.ReuseMode,
		)
		if err != nil {
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
	stepParams := PlanStepParams(lctx.Step)

	activeSession := lctx.Session
	iterations := 0
	for iterations < lctx.MaxIterations {
		if !evaluateLoopCondition(stepParams, lctx.State) {
			break
		}

		control, nextSession, err := executor.executeGraphIteration(
			lctx.Ctx, lctx.Request, lctx.ExecCtx, lctx.Engine, lctx.Spec,
			activeSession, lctx.Step.Loop, lctx.State, lctx.ReuseMode,
		)
		if err != nil {
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

// =============================================================================
// REGISTRATION
// =============================================================================

func init() {
	// Register all built-in loop handlers
	RegisterLoopHandler("repeat", &repeatHandler{})
	RegisterLoopHandler("foreach", &forEachHandler{})
	RegisterLoopHandler("while", &whileHandler{})
}
