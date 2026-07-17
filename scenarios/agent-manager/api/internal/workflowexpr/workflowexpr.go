// Package workflowexpr holds the single definition of the two expression
// languages a workflow definition embeds: the CEL environment used for branch
// edge conditions and the Go text/template dialect used for prompt rendering.
//
// It is a leaf package (it imports neither workflowruntime nor workflowcatalog)
// so both the engine (which evaluates these expressions at runtime) and the
// catalog validator (which compiles and parses them at registration time) can
// share it. That sharing is the point: an expression a definition is validated
// against is exactly the expression the engine will later evaluate, so the two
// surfaces cannot diverge.
package workflowexpr

import (
	"fmt"
	"reflect"
	"text/template"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
)

// celCostLimit bounds evaluation cost. It is applied when a program is built so
// the runtime and any pre-flight program construction share one ceiling.
const celCostLimit = 10000

// Env wraps the workflow CEL environment. The declared variable set here is the
// whole runtime contract: input, journal, status, iteration, edge_traversals,
// and budget.
type Env struct{ env *cel.Env }

// NewEnv builds the shared CEL environment. It fails only on a programmer error
// in the variable declarations, never on user input.
//
// Alongside the declared variables it registers the journal-traversal helpers
// (latest, count) so a definition can query the projected journal without
// re-deriving the "newest successful structured result" incantation by hand.
// The helpers read the enriched journal projection the engine feeds at runtime
// (each entry carries nodeId and kind alongside its payload fields), so a
// condition that compiles here is one the engine can evaluate.
func NewEnv() (*Env, error) {
	env, err := cel.NewEnv(
		cel.Variable("input", cel.DynType),
		cel.Variable("journal", cel.ListType(cel.DynType)),
		cel.Variable("status", cel.StringType),
		cel.Variable("iteration", cel.IntType),
		cel.Variable("edge_traversals", cel.MapType(cel.StringType, cel.IntType)),
		cel.Variable("budget", cel.MapType(cel.StringType, cel.DynType)),
		// latest(journal): the newest successful structured result across every
		// node, as its projected entry map (carrying .value/.status). Returns an
		// empty map when no successful structured result exists yet, so callers
		// can guard with has(latest(journal).value).
		cel.Function("latest",
			cel.Overload("latest_journal", []*cel.Type{cel.ListType(cel.DynType)}, cel.MapType(cel.StringType, cel.DynType),
				cel.UnaryBinding(latestGlobal)),
			// latest(journal, nodeId): the newest result-bearing entry for one
			// node (a successful structured result carrying .value, or a child
			// workflow completion carrying .output). Empty map when the node has
			// produced none yet.
			cel.Overload("latest_journal_node", []*cel.Type{cel.ListType(cel.DynType), cel.StringType}, cel.MapType(cel.StringType, cel.DynType),
				cel.BinaryBinding(latestForNode)),
		),
		// count(journal, nodeId): how many times a node has been attempted
		// (traversal count), i.e. the number of node_attempt journal entries for
		// that node.
		cel.Function("count",
			cel.Overload("count_journal_node", []*cel.Type{cel.ListType(cel.DynType), cel.StringType}, cel.IntType,
				cel.BinaryBinding(countForNode)),
		),
	)
	if err != nil {
		return nil, err
	}
	return &Env{env: env}, nil
}

// journalEntries converts the CEL journal list into native projected maps. A
// non-map element (there should be none in the engine projection) is skipped.
func journalEntries(list ref.Val) []map[string]any {
	lister, ok := list.(traits.Lister)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0)
	for it := lister.Iterator(); it.HasNext() == types.True; {
		elem := it.Next()
		native, err := elem.ConvertToNative(reflect.TypeOf(map[string]any{}))
		if err != nil {
			continue
		}
		if m, ok := native.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func isSuccessStructured(m map[string]any) bool {
	if _, ok := m["value"]; !ok {
		return false
	}
	status, ok := m["status"]
	if !ok {
		return false
	}
	s, ok := status.(string)
	return ok && s == "success"
}

func hasOutput(m map[string]any) bool {
	_, ok := m["output"]
	return ok
}

func nodeIDEquals(m map[string]any, nodeID string) bool {
	id, ok := m["nodeId"].(string)
	return ok && id == nodeID
}

func mapValue(m map[string]any) ref.Val {
	return types.DefaultTypeAdapter.NativeToValue(m)
}

// latestGlobal implements latest(journal): the last (newest) journal entry that
// is a successful structured result, regardless of which node produced it.
func latestGlobal(list ref.Val) ref.Val {
	var found map[string]any
	for _, m := range journalEntries(list) {
		if isSuccessStructured(m) {
			found = m
		}
	}
	if found == nil {
		return mapValue(map[string]any{})
	}
	return mapValue(found)
}

// latestForNode implements latest(journal, nodeId): the last result-bearing
// entry for a specific node.
func latestForNode(list, nodeID ref.Val) ref.Val {
	id, ok := nodeID.Value().(string)
	if !ok {
		return types.NewErr("latest: nodeId must be a string")
	}
	var found map[string]any
	for _, m := range journalEntries(list) {
		if nodeIDEquals(m, id) && (isSuccessStructured(m) || hasOutput(m)) {
			found = m
		}
	}
	if found == nil {
		return mapValue(map[string]any{})
	}
	return mapValue(found)
}

// countForNode implements count(journal, nodeId): the number of node attempts
// (traversals) recorded for a node.
func countForNode(list, nodeID ref.Val) ref.Val {
	id, ok := nodeID.Value().(string)
	if !ok {
		return types.NewErr("count: nodeId must be a string")
	}
	n := 0
	for _, m := range journalEntries(list) {
		if kind, ok := m["kind"].(string); ok && kind == "node_attempt" && nodeIDEquals(m, id) {
			n++
		}
	}
	return types.Int(n)
}

// Compile compiles a branch edge condition and enforces the bool-output rule the
// engine requires. It returns the checked AST so callers that also need an
// executable program do not compile twice.
func (e *Env) Compile(source string) (*cel.Ast, error) {
	if e == nil || e.env == nil {
		return nil, fmt.Errorf("expression environment unavailable")
	}
	ast, issues := e.env.Compile(source)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("expression must return bool")
	}
	return ast, nil
}

// Program builds an executable program from a previously compiled AST under the
// shared cost limit.
func (e *Env) Program(ast *cel.Ast) (cel.Program, error) {
	return e.env.Program(ast, cel.CostLimit(celCostLimit))
}

// Check compiles a condition and enforces the bool-output rule without building
// a program. Registration-time validation uses it so a definition is rejected
// for exactly the compile errors the engine would hit at evaluation time.
func (e *Env) Check(source string) error {
	_, err := e.Compile(source)
	return err
}

// ParsePrompt parses a prompt template with the exact options the runtime uses
// to render it (missingkey=error), so a template that parses here is one the
// binding renderer can execute and vice versa.
func ParsePrompt(source string) (*template.Template, error) {
	return template.New("prompt").Option("missingkey=error").Parse(source)
}
