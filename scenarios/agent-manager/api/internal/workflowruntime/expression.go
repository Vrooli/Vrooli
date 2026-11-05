package workflowruntime

import (
	"fmt"
	"sync"

	"agent-manager/internal/workflowexpr"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

type ExpressionContext struct {
	Input          any
	Journal        []any
	Status         string
	Iteration      int64
	EdgeTraversals map[string]int
	Budget         map[string]any
}

// ExpressionEvaluator evaluates branch edge conditions against the shared
// workflow CEL environment. Compiled programs are cached keyed by condition
// source so a hot execution loop does not recompile the same condition on every
// traversal. The environment is deterministic, so a given source always yields
// an equivalent program; the cache therefore preserves evaluation results while
// removing the per-evaluation compile cost.
type ExpressionEvaluator struct {
	env   *workflowexpr.Env
	mu    sync.RWMutex
	cache map[string]cel.Program
}

func NewExpressionEvaluator() (*ExpressionEvaluator, error) {
	env, err := workflowexpr.NewEnv()
	if err != nil {
		return nil, err
	}
	return &ExpressionEvaluator{env: env, cache: make(map[string]cel.Program)}, nil
}

func (e *ExpressionEvaluator) program(source string) (cel.Program, error) {
	e.mu.RLock()
	program, ok := e.cache[source]
	e.mu.RUnlock()
	if ok {
		return program, nil
	}
	ast, err := e.env.Compile(source)
	if err != nil {
		return nil, err
	}
	program, err = e.env.Program(ast)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	e.cache[source] = program
	e.mu.Unlock()
	return program, nil
}

func (e *ExpressionEvaluator) Evaluate(source string, ctx ExpressionContext) (bool, error) {
	if e == nil || e.env == nil {
		return false, fmt.Errorf("expression evaluator unavailable")
	}
	program, err := e.program(source)
	if err != nil {
		return false, err
	}
	value, _, err := program.Eval(map[string]any{"input": ctx.Input, "journal": ctx.Journal, "status": ctx.Status, "iteration": ctx.Iteration, "edge_traversals": ctx.EdgeTraversals, "budget": ctx.Budget})
	if err != nil {
		return false, err
	}
	typed, ok := value.(types.Bool)
	if !ok {
		return false, fmt.Errorf("expression returned %T", value)
	}
	return bool(typed), nil
}
