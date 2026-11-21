package runannotations

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
)

// defaultCostLimit bounds one CEL evaluation. Deterministic label rules are
// small predicates; a pathological expression is refused rather than allowed
// to scan the host.
const defaultCostLimit = 10_000

// celEvaluator compiles and evaluates the pack's deterministic CEL. A zero
// CostLimit uses defaultCostLimit; tests lower it to prove enforcement.
type celEvaluator struct {
	CostLimit uint64
}

func featureEnv() (*cel.Env, error) {
	if toolRulesErr != nil {
		return nil, toolRulesErr
	}
	options := []cel.EnvOption{
		cel.CrossTypeNumericComparisons(true),
		cel.Variable("run_id", cel.StringType),
		cel.Variable("workload_kind", cel.StringType),
		cel.Variable("workload_key", cel.StringType),
		cel.Variable("harness_kind", cel.StringType),
		cel.Variable("run_mode", cel.StringType),
		cel.Variable("status", cel.StringType),
		cel.Variable("terminal_class", cel.StringType),
		cel.Variable("stop_reason", cel.StringType),
		cel.Variable("exit_code", cel.IntType),
		cel.Variable("duration_min", cel.DoubleType),
		cel.Variable("compaction_count", cel.IntType),
		cel.Variable("operator_message_count", cel.IntType),
		cel.Variable("human_message_count", cel.IntType),
		cel.Variable("brief_message_count", cel.IntType),
		cel.Variable("spawned_by_run", cel.BoolType),
		cel.Variable("imported_session", cel.BoolType),
		cel.Variable("agent_text_count", cel.IntType),
		cel.Variable("stepback_triggers", cel.ListType(cel.StringType)),
		cel.Variable("tool_findings", cel.ListType(cel.StringType)),
		cel.Variable("existing_signals", cel.MapType(cel.StringType, cel.DoubleType)),
		cel.Variable("last_operator_message", cel.StringType),
		cel.Variable("last_human_message", cel.StringType),
		cel.Variable("message_text", cel.StringType),
		cel.Variable("message_index", cel.IntType),
		cel.Variable("message_count", cel.IntType),
		cel.Variable("last_agent_text", cel.StringType),
		cel.Variable("last_tool_error", cel.StringType),
	}
	return cel.NewEnv(append(options, toolVariables()...)...)
}

func compositeEnvFor(base *cel.Env) (*cel.Env, error) {
	if base == nil {
		return nil, errors.New("feature environment is nil")
	}
	return base.Extend(
		cel.Variable("labels", cel.MapType(cel.StringType, cel.DoubleType)),
		cel.Variable("present", cel.MapType(cel.StringType, cel.BoolType)),
		// message_labels summarizes a run's message-scope labels: for each label
		// id, count, positives, max and mean over the run's human messages.
		cel.Variable("message_labels", cel.MapType(cel.StringType, cel.MapType(cel.StringType, cel.DoubleType))),
	)
}

func compileProgram(env *cel.Env, expr string, costLimit uint64) (cel.Program, error) {
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if costLimit == 0 {
		costLimit = defaultCostLimit
	}
	return env.Program(ast, cel.CostLimit(costLimit))
}

// evalBool evaluates a boolean predicate. A non-boolean result is an error so a
// doctrine mistake cannot silently mean false.
func (c celEvaluator) evalBool(program cel.Program, activation map[string]any) (bool, error) {
	if program == nil {
		return false, errors.New("no compiled program")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return false, err
	}
	if native, ok := value.Value().(bool); ok {
		return native, nil
	}
	return false, fmt.Errorf("expression returned %T, want bool", value.Value())
}

// evalString evaluates a string-producing rule.
func (c celEvaluator) evalString(program cel.Program, activation map[string]any) (string, bool, error) {
	if program == nil {
		return "", false, errors.New("no compiled program")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return "", false, err
	}
	if native, ok := value.Value().(string); ok {
		return strings.TrimSpace(native), native != "", nil
	}
	return "", false, fmt.Errorf("expression returned %T, want string", value.Value())
}

// evalNumber evaluates a numeric rule and converts every integer shape.
func (c celEvaluator) evalNumber(program cel.Program, activation map[string]any) (float64, error) {
	if program == nil {
		return 0, errors.New("no compiled program")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return 0, err
	}
	switch native := value.Value().(type) {
	case float64:
		return native, nil
	case int64:
		return float64(native), nil
	case uint64:
		return float64(native), nil
	default:
		return 0, fmt.Errorf("expression returned %T, want number", value.Value())
	}
}

// evalValue evaluates a rule whose result may be a string, bool or number.
// It is used by deterministic labels, whose packs declare different shapes.
func (c celEvaluator) evalValue(program cel.Program, activation map[string]any) (any, error) {
	if program == nil {
		return nil, errors.New("no compiled program")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return nil, err
	}
	switch native := value.Value().(type) {
	case string:
		return strings.TrimSpace(native), nil
	case bool:
		return native, nil
	case int64:
		return float64(native), nil
	case uint64:
		return float64(native), nil
	case float64:
		return native, nil
	default:
		return nil, fmt.Errorf("expression returned unsupported %T", value.Value())
	}
}

func (c celEvaluator) evalStrings(program cel.Program, activation map[string]any) ([]string, error) {
	if program == nil {
		return nil, errors.New("no compiled program")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return nil, err
	}
	native, err := value.ConvertToNative(reflect.TypeOf([]string(nil)))
	if err != nil {
		return nil, err
	}
	out, _ := native.([]string)
	return out, nil
}
