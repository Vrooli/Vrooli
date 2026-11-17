package buildinfo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func runtimeRebaselineProgressBoundaryViolations(file *ast.File) []string {
	var violations []string
	var run *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "run" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok && ident.Name == "rebaselineEffects" {
			run = fn
			break
		}
	}
	if run == nil || run.Body == nil {
		return []string{"rebaselineEffects.run missing"}
	}
	var ownerCalls int
	stateHelpers := map[string]bool{
		"readPreparedRebaselineProgress": true, "validatePreparedRebaselineProgress": true,
		"preparedProgressAllowsRollbackQualification": true, "selectWatchdog": true,
		"restorePreparedRebaselineWatchdog": true, "preparedProgressMatchesWatchdog": true,
		"preparedProgressNeedsFreshDecision": true, "preparedProgressContinuesCandidate": true,
		"preparePreparedRebaselineRecovery": true, "completePreparedRebaselineRecovery": true,
		"legacyPauseMatches": true, "pauseInEffect": true, "pause": true, "restore": true,
		"qualifyAndRetain": true,
	}
	ast.Inspect(run.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			if node.Name == "progressState" || node.Name == "progressRecord" {
				violations = append(violations, "durable progress values may only be interpreted in prepared_rebaseline.go")
			}
		case *ast.CallExpr:
			name := exprName(node.Fun)
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
				name = selector.Sel.Name
			}
			if stateHelpers[name] {
				violations = append(violations, "caller invokes prepared progress helper "+name+" outside the owner transaction")
			}
			if name == "runPreparedRebaseline" {
				ownerCalls++
				parent, ok := directParent(file, node)
				_, assignment := parent.(*ast.AssignStmt)
				if !ok || !assignment || !statementIsDirectChild(run.Body, parent) {
					violations = append(violations, "runPreparedRebaseline must be a direct unconditional statement in rebaselineEffects.run")
				}
				if len(node.Args) != 1 {
					violations = append(violations, fmt.Sprintf("runPreparedRebaseline must receive one typed input, got %d arguments", len(node.Args)))
				} else if literal, ok := node.Args[0].(*ast.CompositeLit); !ok || exprName(literal.Type) != "preparedRebaselineRunInput" {
					violations = append(violations, "runPreparedRebaseline must receive a preparedRebaselineRunInput literal")
				}
			}
		}
		return true
	})
	if ownerCalls != 1 {
		violations = append(violations, fmt.Sprintf("expected one runPreparedRebaseline owner call, got %d", ownerCalls))
	}
	return violations
}

func directParent(root ast.Node, target ast.Node) (ast.Node, bool) {
	var parent ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil || parent != nil {
			return true
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, child := range node.Rhs {
				if child == target {
					parent = node
					return false
				}
			}
		case *ast.ExprStmt:
			if node.X == target {
				parent = node
				return false
			}
		case *ast.ReturnStmt:
			for _, child := range node.Results {
				if child == target {
					parent = node
					return false
				}
			}
		}
		return true
	})
	if parent == nil {
		return nil, false
	}
	return parent, true
}

func statementIndex(body *ast.BlockStmt, statement ast.Node) int {
	for i, child := range body.List {
		if child == statement {
			return i
		}
	}
	return -1
}

func preparedRebaselinePlanEntryViolations(file *ast.File) []string {
	var violations []string
	var entry *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "runPreparedRebaseline" && fn.Body != nil {
			entry = fn
			break
		}
	}
	if entry == nil {
		return []string{"runPreparedRebaseline entry missing"}
	}
	planCalls, transactionCalls := 0, 0
	planStatementIndex, transactionStatementIndex := -1, -1
	wantTransactionFields := map[string]string{"preparedRebaselinePlan": "plan"}
	ast.Inspect(entry.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := exprName(call.Fun)
		switch name {
		case "preparePreparedRebaselinePlan":
			planCalls++
			if len(call.Args) != 1 || exprName(call.Args[0]) != "input" {
				violations = append(violations, "prepared plan must receive the typed owner input")
			}
			parent, ok := directParent(entry, call)
			_, assignment := parent.(*ast.AssignStmt)
			planStatementIndex = statementIndex(entry.Body, parent)
			if assign, ok := parent.(*ast.AssignStmt); ok && (len(assign.Lhs) != 2 || exprName(assign.Lhs[0]) != "plan" || exprName(assign.Lhs[1]) != "err") {
				violations = append(violations, "prepared plan and error results must remain bound before transaction setup")
			}
			if !ok || !assignment || !statementIsDirectChild(entry.Body, parent) {
				violations = append(violations, "progress plan must be an unconditional owner entry step")
			}
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "run" {
			transactionCalls++
			literal := unwrapCompositeLiteral(selector.X)
			fields := map[string]string{}
			if literal != nil {
				for _, element := range literal.Elts {
					kv, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if ok {
						fields[key.Name] = exprName(kv.Value)
					}
				}
			}
			for field, expected := range wantTransactionFields {
				if fields[field] != expected {
					violations = append(violations, fmt.Sprintf("transaction field %s changed: got %s, want %s", field, fields[field], expected))
				}
			}
			parent, ok := directParent(entry, call)
			_, returned := parent.(*ast.ReturnStmt)
			transactionStatementIndex = statementIndex(entry.Body, parent)
			if !ok || !returned || !statementIsDirectChild(entry.Body, parent) {
				violations = append(violations, "prepared transaction must be returned directly after the progress plan")
			}
		}
		return true
	})
	if planCalls != 1 {
		violations = append(violations, fmt.Sprintf("expected one prepared plan call, got %d", planCalls))
	}
	if transactionCalls != 1 {
		violations = append(violations, fmt.Sprintf("expected one prepared transaction call, got %d", transactionCalls))
	}
	if planStatementIndex < 0 || transactionStatementIndex <= planStatementIndex {
		violations = append(violations, "prepared plan must be reachable before the transaction call")
	}
	if len(entry.Body.List) != 3 || planStatementIndex != 0 || transactionStatementIndex != 2 || !preparedPlanErrorIsReturned(entry.Body.List) {
		violations = append(violations, "prepared plan error must be returned before transaction construction")
	}
	directReturns, directReturnIndex := 0, -1
	for i, statement := range entry.Body.List {
		if _, ok := statement.(*ast.ReturnStmt); ok {
			directReturns++
			directReturnIndex = i
		}
	}
	if directReturns != 1 || directReturnIndex != transactionStatementIndex {
		violations = append(violations, "transaction return must be the sole direct return after the prepared plan")
	}
	return violations
}

func unwrapCompositeLiteral(expr ast.Expr) *ast.CompositeLit {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	literal, _ := expr.(*ast.CompositeLit)
	return literal
}

func preparedPlanErrorIsReturned(statements []ast.Stmt) bool {
	if len(statements) < 2 {
		return false
	}
	branch, ok := statements[1].(*ast.IfStmt)
	if !ok || exprName(branch.Cond) != "err!=nil" || len(branch.Body.List) != 1 {
		return false
	}
	returned, ok := branch.Body.List[0].(*ast.ReturnStmt)
	return ok && len(returned.Results) == 2 && exprName(returned.Results[1]) == "err"
}

func statementIsDirectChild(body *ast.BlockStmt, statement ast.Node) bool {
	for _, child := range body.List {
		if child == statement {
			return true
		}
	}
	return false
}

func preparedProgressOrchestrationViolations(file *ast.File) []string {
	var violations []string
	var run *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "run" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		if ident, ok := fn.Recv.List[0].Type.(*ast.Ident); ok && ident.Name == "preparedRebaselineTransaction" {
			run = fn
			break
		}
	}
	if run == nil || run.Body == nil {
		return []string{"preparedRebaselineTransaction.run missing"}
	}
	positions := map[string][]token.Pos{}
	stepRunCount, stepsTableCount := 0, 0
	intentStatementIndex, loopStatementIndex, enrollmentStatementIndex, completionStatementIndex := -1, -1, -1, -1
	stepStageOrder := []string(nil)
	callName := func(expr ast.Expr) string {
		switch expr := expr.(type) {
		case *ast.Ident:
			return expr.Name
		case *ast.SelectorExpr:
			return expr.Sel.Name
		default:
			return ""
		}
	}
	shapeOf := func(expr ast.Expr) string {
		switch expr := expr.(type) {
		case *ast.Ident:
			return expr.Name
		case *ast.SelectorExpr:
			if base, ok := expr.X.(*ast.Ident); ok {
				return base.Name + "." + expr.Sel.Name
			}
		}
		return ""
	}
	callArgsMatch := func(call *ast.CallExpr, expected ...string) bool {
		if len(call.Args) != len(expected) {
			return false
		}
		for i := range expected {
			if shapeOf(call.Args[i]) != expected[i] {
				return false
			}
		}
		return true
	}
	var stack []ast.Node
	ast.Inspect(run.Body, func(node ast.Node) bool {
		if node == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}
		parent := ast.Node(nil)
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		switch node := node.(type) {
		case *ast.CallExpr:
			name := callName(node.Fun)
			qualifiedName := exprName(node.Fun)
			if qualifiedName == "request.BeforeEffects" {
				positions[qualifiedName] = append(positions[qualifiedName], node.Pos())
				if !callArgsMatch(node, "ctx", "tx.admission") {
					violations = append(violations, "authority callback arguments changed; it must bind the qualified admission")
				}
				guarded := false
				for _, ancestor := range stack {
					if branch, ok := ancestor.(*ast.IfStmt); ok && exprName(branch.Cond) == "request.BeforeEffects!=nil" {
						guarded = true
					}
				}
				if !guarded {
					violations = append(violations, "authority callback must remain guarded by request.BeforeEffects != nil")
				}
			}
			if qualifiedName == "e.enrollment" {
				if selector, ok := node.Fun.(*ast.SelectorExpr); ok && exprName(selector.X) == "e" {
					positions["final enrollment"] = append(positions["final enrollment"], node.Pos())
					if !callArgsMatch(node, "tx.home") {
						violations = append(violations, "final enrollment verification path changed")
					}
					_, assignment := parent.(*ast.AssignStmt)
					if !assignment || len(stack) != 2 || stack[0] != run.Body {
						violations = append(violations, "final enrollment verification must be an unconditional transaction step")
					}
					enrollmentStatementIndex = statementIndex(run.Body, parent)
				}
			}
			if name == "preparePreparedRebaselineRecovery" || name == "completePreparedRebaselineRecovery" {
				positions[name] = append(positions[name], node.Pos())
				_, assignment := parent.(*ast.AssignStmt)
				_, expression := parent.(*ast.ExprStmt)
				if (!assignment && !expression) || len(stack) != 2 || stack[0] != run.Body {
					violations = append(violations, name+" must be an unconditional top-level transaction step")
				}
				if name == "preparePreparedRebaselineRecovery" {
					intentStatementIndex = statementIndex(run.Body, parent)
				} else {
					completionStatementIndex = statementIndex(run.Body, parent)
				}
			}
			if name == "preparePreparedRebaselineRecovery" && !callArgsMatch(node, "ctx", "e", "request", "tx.home", "tx.admission", "tx.originalCandidate", "request.BundleSHA", "tx.stagedWatchdog", "tx.pendingWatchdog", "tx.progressPath", "tx.watchdogSHA", "record") {
				violations = append(violations, "intent helper arguments changed; review the prepared recovery binding")
			}
			if name == "completePreparedRebaselineRecovery" && !callArgsMatch(node, "e", "tx.home", "request.BundleSHA", "tx.progressPath", "record") {
				violations = append(violations, "completion helper arguments changed; review completion and bundle retirement")
			}
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "run" {
				if base, ok := selector.X.(*ast.Ident); ok && base.Name == "step" {
					inMutationLoop, directLoop := false, false
					for i, ancestor := range stack {
						loop, ok := ancestor.(*ast.RangeStmt)
						if !ok || loop.Body == nil || exprName(loop.Key) != "_" || exprName(loop.Value) != "step" || exprName(loop.X) != "steps" {
							continue
						}
						inMutationLoop = node.Pos() >= loop.Body.Pos() && node.End() <= loop.Body.End()
						directLoop = i > 0 && stack[i-1] == run.Body
						loopStatementIndex = statementIndex(run.Body, loop)
					}
					if !inMutationLoop || !directLoop {
						violations = append(violations, "step.run must execute within the top-level `for _, step := range steps` loop")
					} else {
						stepRunCount++
						positions["step.run"] = append(positions["step.run"], node.Pos())
					}
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name != "steps" || i >= len(node.Rhs) {
					continue
				}
				table, ok := node.Rhs[i].(*ast.CompositeLit)
				if !ok {
					continue
				}
				stepsTableCount++
				stepStageOrder = stepStageOrder[:0]
				for _, element := range table.Elts {
					row, ok := element.(*ast.CompositeLit)
					if !ok || len(row.Elts) < 3 {
						continue
					}
					stage, ok := row.Elts[1].(*ast.BasicLit)
					if !ok || stage.Kind != token.STRING {
						continue
					}
					if value, err := strconv.Unquote(stage.Value); err == nil {
						stepStageOrder = append(stepStageOrder, value)
					}
				}
			}
		}
		stack = append(stack, node)
		return true
	})
	for _, name := range []string{"request.BeforeEffects", "preparePreparedRebaselineRecovery", "step.run", "final enrollment", "completePreparedRebaselineRecovery"} {
		if len(positions[name]) != 1 {
			violations = append(violations, fmt.Sprintf("expected exactly one %s call, got %d", name, len(positions[name])))
		}
	}
	if stepRunCount != 1 {
		violations = append(violations, fmt.Sprintf("expected one step.run mutation loop, got %d", stepRunCount))
	}
	if intentStatementIndex != 3 {
		violations = append(violations, "intent must immediately follow transaction bindings, progress-record construction and authority verification")
	}
	if loopStatementIndex <= intentStatementIndex || enrollmentStatementIndex <= loopStatementIndex || completionStatementIndex <= enrollmentStatementIndex {
		violations = append(violations, "transaction order must be intent, mutation loop, final enrollment, then completion")
	}
	directReturns, directReturnIndex := 0, -1
	for i, statement := range run.Body.List {
		if _, ok := statement.(*ast.ReturnStmt); ok {
			directReturns++
			directReturnIndex = i
		}
	}
	if directReturns != 1 || directReturnIndex <= completionStatementIndex {
		violations = append(violations, "no top-level return may bypass intent, mutation, enrollment verification or completion")
	}
	wantStages := []string{"pause", "fence", "install", "watchdog", "supervisor", "verify", "companions", "restore"}
	if stepsTableCount != 1 || !slices.Equal(stepStageOrder, wantStages) {
		violations = append(violations, fmt.Sprintf("rebaseline mutation stages changed: got %v, want %v", stepStageOrder, wantStages))
	}
	order := []string{"request.BeforeEffects", "preparePreparedRebaselineRecovery", "step.run", "final enrollment", "completePreparedRebaselineRecovery"}
	var last token.Pos
	for _, name := range order {
		if len(positions[name]) != 1 {
			continue
		}
		if last.IsValid() && positions[name][0] <= last {
			violations = append(violations, fmt.Sprintf("%s must follow the prior prepared progress stage", name))
		}
		last = positions[name][0]
	}
	return violations
}

func TestPreparedProgressOrchestrationGuardRejectsReordering(t *testing.T) {
	const ordered = `package fixture
type preparedRebaselineTransaction struct{}
func (tx preparedRebaselineTransaction) run() {
e, request, ctx := tx.e, tx.request, tx.ctx
record := newPreparedRebaselineProgress(candidateSHA, watchdogSHA, request.BundleSHA, request.Source, request.HubDecision)
if request.BeforeEffects != nil { request.BeforeEffects(ctx, tx.admission) }
preparePreparedRebaselineRecovery(ctx, e, request, tx.home, tx.admission, tx.originalCandidate, request.BundleSHA, tx.stagedWatchdog, tx.pendingWatchdog, tx.progressPath, tx.watchdogSHA, record)
steps := []struct { name, stage string; run func() error }{
{"pause", "pause", nil}, {"fence", "fence", nil}, {"install", "install", nil}, {"watchdog", "watchdog", nil},
{"supervisor", "supervisor", nil}, {"verify", "verify", nil}, {"companions", "companions", nil}, {"restore", "restore", nil},
}
for _, step := range steps { step.run() }
enrollment, exists, err := e.enrollment(tx.home)
completePreparedRebaselineRecovery(e, tx.home, request.BundleSHA, tx.progressPath, record)
return result, nil
}
`
	parse := func(source string) *ast.File {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	if got := preparedProgressOrchestrationViolations(parse(ordered)); len(got) != 0 {
		t.Fatalf("ordered lifecycle rejected: %v", got)
	}
	intentCall := "preparePreparedRebaselineRecovery(ctx, e, request, tx.home, tx.admission, tx.originalCandidate, request.BundleSHA, tx.stagedWatchdog, tx.pendingWatchdog, tx.progressPath, tx.watchdogSHA, record)"
	completeCall := "completePreparedRebaselineRecovery(e, tx.home, request.BundleSHA, tx.progressPath, record)"
	authorityCall := "if request.BeforeEffects != nil { request.BeforeEffects(ctx, tx.admission) }"
	enrollmentCall := "enrollment, exists, err := e.enrollment(tx.home)"
	loopAndComplete := "for _, step := range steps { step.run() }\n" + enrollmentCall + "\n" + completeCall
	intentAfterLoop := strings.Replace(ordered, intentCall, "", 1)
	intentAfterLoop = strings.Replace(intentAfterLoop, loopAndComplete,
		"for _, step := range steps { step.run() }\n"+intentCall+"\n"+completeCall, 1)
	completionBeforeLoop := strings.Replace(ordered, loopAndComplete,
		completeCall+"\nfor _, step := range steps { step.run() }\n"+enrollmentCall, 1)
	authorityAfterIntent := strings.Replace(ordered, authorityCall+"\n"+intentCall, intentCall+"\n"+authorityCall, 1)
	completionBeforeEnrollment := strings.Replace(ordered, enrollmentCall+"\n"+completeCall, completeCall+"\n"+enrollmentCall, 1)
	for _, tc := range []struct{ name, source string }{
		{"intent after mutation loop", intentAfterLoop},
		{"completion before mutation loop", completionBeforeLoop},
		{"early return before intent", strings.Replace(ordered, authorityCall+"\n"+intentCall, authorityCall+"\nreturn result, err\n"+intentCall, 1)},
		{"early return before completion", strings.Replace(ordered, enrollmentCall+"\n"+completeCall, enrollmentCall+"\nreturn result, err\n"+completeCall, 1)},
		{"intent arguments changed", strings.Replace(ordered, "tx.admission, tx.originalCandidate", "otherAdmission, tx.originalCandidate", 1)},
		{"completion arguments changed", strings.Replace(ordered, "completePreparedRebaselineRecovery(e, tx.home", "completePreparedRebaselineRecovery(e, otherHome", 1)},
		{"authority callback arguments changed", strings.Replace(ordered, "request.BeforeEffects(ctx, tx.admission)", "request.BeforeEffects(otherContext, tx.admission)", 1)},
		{"final enrollment path changed", strings.Replace(ordered, "e.enrollment(tx.home)", "e.enrollment(otherHome)", 1)},
		{"authority after intent", authorityAfterIntent},
		{"authority callback removed", strings.Replace(ordered, authorityCall+"\n", "", 1)},
		{"completion before final enrollment", completionBeforeEnrollment},
		{"final enrollment removed", strings.Replace(ordered, enrollmentCall+"\n", "", 1)},
		{"intent helper bypassed", strings.Replace(ordered, intentCall+"\n", "", 1)},
		{"intent unreachable behind branch", strings.Replace(ordered, intentCall,
			"if false { "+intentCall+" }", 1)},
		{"completion unreachable behind branch", strings.Replace(ordered, completeCall,
			"if false { "+completeCall+" }", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.source == ordered {
				t.Fatal("regression mutation did not change its source fixture")
			}
			if got := preparedProgressOrchestrationViolations(parse(tc.source)); len(got) == 0 {
				t.Fatal("unsafe progress orchestration unexpectedly passed")
			}
		})
	}
}

func TestRuntimeProgressEntryGuardRejectsCallerBypass(t *testing.T) {
	const ordered = `package fixture
type rebaselineEffects struct{}
func (e rebaselineEffects) run() {
result, err = runPreparedRebaseline(preparedRebaselineRunInput{})
}
`
	parse := func(source string) *ast.File {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), "caller.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	if got := runtimeRebaselineProgressBoundaryViolations(parse(ordered)); len(got) != 0 {
		t.Fatalf("direct owner entry rejected: %v", got)
	}
	ownerHidden := strings.Replace(ordered, "result, err = runPreparedRebaseline", "if false { result, err = runPreparedRebaseline", 1)
	ownerHidden = strings.Replace(ownerHidden, "\n}\n", "\n}\n}\n", 1)
	mutations := []struct{ name, source string }{
		{"owner call hidden behind branch", ownerHidden},
		{"direct progress reader", strings.Replace(ordered, "result, err = runPreparedRebaseline", "progressState := readPreparedRebaselineProgress(progressPath)\nresult, err = runPreparedRebaseline", 1)},
		{"direct progress field", strings.Replace(ordered, "result, err = runPreparedRebaseline", "if progressState.Complete {}\nresult, err = runPreparedRebaseline", 1)},
		{"direct rollback state helper", strings.Replace(ordered, "result, err = runPreparedRebaseline", "_ = e.pauseInEffect(home, pausePath)\nresult, err = runPreparedRebaseline", 1)},
		{"input no longer explicit", strings.Replace(ordered, "runPreparedRebaseline(preparedRebaselineRunInput{})", "runPreparedRebaseline(input)", 1)},
		{"input type changed", strings.Replace(ordered, "preparedRebaselineRunInput{}", "otherInput{}", 1)},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			if tc.source == ordered {
				t.Fatal("regression mutation did not change its source fixture")
			}
			if got := runtimeRebaselineProgressBoundaryViolations(parse(tc.source)); len(got) == 0 {
				t.Fatal("unsafe caller boundary unexpectedly passed")
			}
		})
	}
}

func TestPreparedPlanEntryGuardRejectsUnreachableCalls(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "internal/app/runtime/prepared_rebaseline.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ordered := string(data)
	parse := func(source string) *ast.File {
		t.Helper()
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	if got := preparedRebaselinePlanEntryViolations(parse(ordered)); len(got) != 0 {
		t.Fatalf("ordered owner entry rejected: %v", got)
	}
	mutations := []struct{ name, old, replacement string }{
		{"plan input rebound", "preparePreparedRebaselinePlan(input)", "preparePreparedRebaselinePlan(otherInput)"},
		{"plan error ignored", "\tif err != nil {\n\t\treturn RebaselineResult{}, err\n\t}\n\treturn (preparedRebaselineTransaction{", "\treturn (preparedRebaselineTransaction{"},
		{"transaction plan rebound", "preparedRebaselinePlan: plan", "preparedRebaselinePlan: preparedRebaselinePlan{}"},
		{"return before plan", "\tplan, err := preparePreparedRebaselinePlan", "\treturn RebaselineResult{}, err\n\tplan, err := preparePreparedRebaselinePlan"},
		{"return before transaction", "\treturn (preparedRebaselineTransaction{", "\treturn RebaselineResult{}, err\n\treturn (preparedRebaselineTransaction{"},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(ordered, tc.old, tc.replacement, 1)
			if changed == ordered {
				t.Fatalf("mutation anchor %q missing", tc.old)
			}
			if got := preparedRebaselinePlanEntryViolations(parse(changed)); len(got) == 0 {
				t.Fatal("unsafe progress plan entry unexpectedly passed")
			}
		})
	}
}

func exprName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		base := exprName(expr.X)
		if base != "" {
			return base + "." + expr.Sel.Name
		}
	case *ast.BinaryExpr:
		if expr.Op == token.NEQ {
			return exprName(expr.X) + "!=" + exprName(expr.Y)
		}
	}
	return ""
}
