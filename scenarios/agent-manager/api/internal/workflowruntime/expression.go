package workflowruntime

import (
	"fmt"

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

type ExpressionEvaluator struct{ env *cel.Env }

func NewExpressionEvaluator() (*ExpressionEvaluator, error) {
	env, err := cel.NewEnv(cel.Variable("input", cel.DynType), cel.Variable("journal", cel.ListType(cel.DynType)), cel.Variable("status", cel.StringType), cel.Variable("iteration", cel.IntType), cel.Variable("edge_traversals", cel.MapType(cel.StringType, cel.IntType)), cel.Variable("budget", cel.MapType(cel.StringType, cel.DynType)))
	if err != nil {
		return nil, err
	}
	return &ExpressionEvaluator{env: env}, nil
}

func (e *ExpressionEvaluator) Evaluate(source string, ctx ExpressionContext) (bool, error) {
	if e == nil || e.env == nil {
		return false, fmt.Errorf("expression evaluator unavailable")
	}
	ast, issues := e.env.Compile(source)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return false, fmt.Errorf("expression must return bool")
	}
	program, err := e.env.Program(ast, cel.CostLimit(10000))
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
