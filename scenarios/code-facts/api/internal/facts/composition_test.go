package facts

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestDescribeCompositionReportsConcreteProductionAndUnknownTestBindings(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/app\n\ngo 1.22\n")
	write("wiring.go", `package app
type Store interface { Get() }
type MemStore struct{}
func (*MemStore) Get() {}
type Deps struct { Store Store }
func Build() { _ = Deps{Store: &MemStore{}} }
`)
	write("wiring_test.go", `package app
func TestBuild() { _ = Deps{} }
`)
	report, err := DescribeComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Bindings) != 2 {
		t.Fatalf("expected prod and test rows; got %#v", report.Bindings)
	}
	if !strings.Contains(report.AnalysisScope, "non-Go") || !strings.Contains(report.AnalysisScope, "type checking") {
		t.Fatalf("analysis limits not reported: %q", report.AnalysisScope)
	}
	prod, test := report.Bindings[0], report.Bindings[1]
	if prod.Environment != "production" || prod.Provider != "MemStore" || prod.ProviderExpression != "&MemStore{}" || prod.Status != "concrete" {
		t.Fatalf("unexpected prod binding: %#v", prod)
	}
	if prod.SourcePath != "wiring.go" || prod.Line != 6 || len(prod.SourceHash) != 64 || prod.Provenance == "" {
		t.Fatalf("missing provenance: %#v", prod)
	}
	if !strings.Contains(prod.Provenance, "type_check=UNKNOWN") || !strings.Contains(prod.Provenance, "entrypoint_reachability=PARTIAL") {
		t.Fatalf("unsupported type/reachability claims must stay explicit: %#v", prod)
	}
	if test.Environment != "test" || test.Status != "unknown" || test.SourcePath != "wiring.go" || test.Line < 1 || len(test.SourceHash) != 64 {
		t.Fatalf("unexpected test unknown: %#v", test)
	}
}

func TestDescribeCompositionDerivesSupportedMainConstructorChainAndKeepsTypecheckUnknown(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/chain\n\ngo 1.22\n")
	write("main.go", `package main
type Store interface { Get() }
type MemStore struct{}
func (*MemStore) Get() {}
type Deps struct { Store Store }
func NewServer(d Deps) {}
func Wire() { NewServer(Deps{Store: &MemStore{}}) }
func main() { Wire() }
`)
	report, err := DescribeComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Bindings) != 1 {
		t.Fatalf("expected one source binding: %#v", report.Bindings)
	}
	provenance := report.Bindings[0].Provenance
	if !strings.Contains(provenance, "entrypoint_reachability=DERIVED") || !strings.Contains(provenance, "main.go:") || !strings.Contains(provenance, "#sha256:") {
		t.Fatalf("missing source-backed entrypoint chain: %q", provenance)
	}
	if !strings.Contains(provenance, "type_check=UNKNOWN") {
		t.Fatalf("syntax chain must not imply compiler verification: %q", provenance)
	}
}

func TestDescribeCompositionMarksConflictingBindingsAmbiguous(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "deps.go"), []byte(`package app
type Store interface{ Get() }
type Deps struct{ Store Store }
func one(){ _ = Deps{Store: &A{}} }
func two(){ _ = Deps{Store: &B{}} }
type A struct{}; type B struct{}
`), 0600); err != nil {
		t.Fatal(err)
	}
	report, err := DescribeComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	var providers []string
	for _, binding := range report.Bindings {
		if binding.Environment == "production" {
			if binding.Status != "ambiguous" {
				t.Fatalf("expected ambiguity: %#v", binding)
			}
			providers = append(providers, binding.Provider)
		}
	}
	if strings.Join(providers, ",") != "A,B" {
		t.Fatalf("unexpected providers: %v", providers)
	}
}

func TestDescribeCompositionResolvesOnlyUniqueSameFunctionLocalLiteral(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/local\n\ngo 1.22\n")
	write("wiring.go", `package app
type Store interface { Get() }
type MemoryStore struct{}
func (*MemoryStore) Get() {}
type Deps struct { Store Store }
func Wire() {
	store := &MemoryStore{}
	_ = Deps{Store: store}
}

func WireIndirect() {
	store := NewStore()
	_ = Deps{Store: store}
}
func WireReassigned() {
	store := &MemoryStore{}
	store = NewStore()
	_ = Deps{Store: store}
}
func NewStore() Store { return &MemoryStore{} }
`)
	write("wiring_test.go", `package app
type TestStore struct{}
func (*TestStore) Get() {}
func TestWire() {
	store := &TestStore{}
	_ = Deps{Store: store}
}
`)
	report, err := DescribeComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	var productionConcrete, testConcrete, indirectUnknown int
	for _, binding := range report.Bindings {
		switch {
		case binding.Environment == "production" && binding.Provider == "MemoryStore" && binding.Status == "concrete":
			productionConcrete++
			if !strings.Contains(binding.Provenance, "resolved_local_initializer=wiring.go:") || !strings.Contains(binding.Provenance, "#sha256:") {
				t.Errorf("resolved row lacks per-hop provenance: %#v", binding)
			}
		case binding.Environment == "test" && binding.Provider == "TestStore" && binding.Status == "concrete":
			testConcrete++
		case binding.Environment == "production" && binding.ProviderExpression == "store" && binding.Status == "unknown":
			indirectUnknown++
		}
	}
	if productionConcrete != 1 || testConcrete != 1 || indirectUnknown != 2 {
		t.Fatalf("expected direct production/test locals concrete and indirect/reassigned locals unknown; concrete=%d test=%d unknown locals=%d rows=%#v", productionConcrete, testConcrete, indirectUnknown, report.Bindings)
	}
}

func TestDescribeCompositionResolvesOnlyUniqueDirectPackageReturnLiteral(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/returns\n\ngo 1.22\n")
	write("wiring.go", `package app
type Store interface { Get() }
type MemoryStore struct{}
func (*MemoryStore) Get() {}
type Deps struct { Store Store }
func Wire() { _ = Deps{Store: NewStore()} }
func WireWithArgument() { _ = Deps{Store: NewStoreWithLabel("cache")} }
func WireConditional() { _ = Deps{Store: NewMaybeStore()} }
`)
	write("constructors.go", `package app
func NewStore() Store { return &MemoryStore{} }
func NewStoreWithLabel(label string) Store { return &MemoryStore{} }
func NewMaybeStore() Store { if true { return &MemoryStore{} }; return &MemoryStore{} }
`)
	report, err := DescribeComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	foundConcrete, foundParameterizedUnknown, foundConditionalUnknown := false, false, false
	for _, binding := range report.Bindings {
		if binding.Environment != "production" {
			continue
		}
		switch binding.ProviderExpression {
		case "NewStore()":
			foundConcrete = binding.Status == "concrete" && binding.Provider == "MemoryStore" && strings.Contains(binding.Provenance, "resolved_call_return=constructors.go:") && strings.Contains(binding.Provenance, "#sha256:")
		case `NewStoreWithLabel("cache")`:
			foundParameterizedUnknown = binding.Status == "unknown" && binding.Provider == ""
		case "NewMaybeStore()":
			foundConditionalUnknown = binding.Status == "unknown" && binding.Provider == ""
		}
	}
	if !foundConcrete || !foundParameterizedUnknown || !foundConditionalUnknown {
		t.Fatalf("expected one direct zero-arg return literal and explicit parameterized/conditional unknowns; concrete=%t parameterized=%t conditional=%t rows=%#v", foundConcrete, foundParameterizedUnknown, foundConditionalUnknown, report.Bindings)
	}
	if !strings.Contains(report.AnalysisScope, "zero-argument same-package") || !strings.Contains(report.AnalysisScope, "conditional/indirect call returns") {
		t.Fatalf("supported and unsupported call-return scope missing: %q", report.AnalysisScope)
	}
}

func TestDescribeCompositionSearchHubRepositoryEvidence(t *testing.T) {
	root, err := compositionRepositoryRoot(t)
	if err != nil {
		t.Fatal(err)
	}
	report, err := DescribeComposition(filepath.Join(root, "scenarios", "search-hub"))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Bindings) == 0 {
		t.Fatal("expected composition evidence for Search Hub")
	}
	concrete := 0
	hasProduction, hasTest := false, false
	for _, binding := range report.Bindings {
		if binding.Environment == "production" {
			hasProduction = true
		}
		if binding.Environment == "test" {
			hasTest = true
		}
		if binding.Status == "concrete" {
			concrete++
			if binding.SourcePath == "" || binding.Line < 1 || len(binding.SourceHash) != 64 || binding.Provenance == "" {
				t.Fatalf("concrete evidence lacks provenance: %#v", binding)
			}
		}
	}
	if concrete == 0 {
		t.Fatalf("expected at least one observed concrete Search Hub binding; got %d rows", len(report.Bindings))
	}
	if !hasProduction || !hasTest {
		t.Fatalf("expected separate production/test evidence rows, production=%v test=%v", hasProduction, hasTest)
	}
}

func TestDescribeControlPlaneCLICompositionReportsRegisteredCommandHandlers(t *testing.T) {
	root, err := compositionRepositoryRoot(t)
	if err != nil {
		t.Fatal(err)
	}
	report, err := DescribeControlPlaneCLIComposition(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Bindings) < 20 {
		t.Fatalf("expected broad top-level command coverage, got %d", len(report.Bindings))
	}
	var resource *CompositionBinding
	var argsBinding *CompositionBinding
	registryFound, dispatchFound, earlyDispatchFound := false, false, false
	nestedBoundaryFound := false
	nestedFound := map[string]bool{}
	for i := range report.Bindings {
		if report.Bindings[i].Dependency == "resource" {
			resource = &report.Bindings[i]
		}
		if report.Bindings[i].Dependency == "supervision-set" {
			argsBinding = &report.Bindings[i]
		}
		registryFound = registryFound || report.Bindings[i].Dependency == "command registry"
		dispatchFound = dispatchFound || report.Bindings[i].Dependency == "normal CLI dispatch"
		if report.Bindings[i].Dependency == "early-dispatch:writeStandaloneVersion" {
			binding := report.Bindings[i]
			earlyDispatchFound = binding.Status == "concrete" && binding.SourcePath == "cmd/vrooli/main.go" && binding.Line > 0 && len(binding.SourceHash) == 64 && strings.Contains(binding.ProviderExpression, "matching args and runtime outcome are unknown")
		}
		if report.Bindings[i].Dependency == "resource:nested-scan-boundary" {
			nestedBoundaryFound = report.Bindings[i].Status == "unknown" && strings.Contains(report.Bindings[i].ProviderExpression, "other registration and dispatch forms remain UNKNOWN")
		}
		switch report.Bindings[i].Dependency {
		case "resource:archive", "resource:archive:gc", "resource:n8n:runtime", "resource:n8n:runtime:import-workflow", "package:build", "capacity:disk-reserve", "capacity:residency", "auth:status":
			nestedFound[report.Bindings[i].Dependency] = report.Bindings[i].Status == "concrete" && report.Bindings[i].SourcePath != "" && report.Bindings[i].Line > 0 && len(report.Bindings[i].SourceHash) == 64
		}
	}
	if resource == nil {
		t.Fatal("resource command was not reported")
	}
	if resource.Status != "concrete" || resource.Provider != "resourcehandlers.RootHandler" {
		t.Fatalf("unexpected resource command binding: %#v", resource)
	}
	if resource.SourcePath != "internal/cli/vroolicli/app.go" || resource.Line < 1 || len(resource.SourceHash) != 64 {
		t.Fatalf("command binding lacks source provenance: %#v", resource)
	}
	if argsBinding == nil || !strings.Contains(argsBinding.ProviderExpression, "Args declaration DECLARED_UNEVALUATED") || !strings.Contains(argsBinding.ProviderExpression, "commandtree.ArgSchema") || !strings.Contains(argsBinding.Provenance, "topcli/topcli.go:") {
		t.Fatalf("top-level Args declaration lacks expression/provenance: %#v", argsBinding)
	}
	if !strings.Contains(report.AnalysisScope, "unsupported forms degrade to UNKNOWN") || !strings.Contains(report.AnalysisScope, "nested/runtime argument dispatch and execution are not analyzed") {
		t.Fatalf("missing analysis boundary: %q", report.AnalysisScope)
	}
	if !registryFound || !dispatchFound || !earlyDispatchFound {
		t.Fatalf("missing registry/entrypoint evidence: registry=%t normal=%t early=%t", registryFound, dispatchFound, earlyDispatchFound)
	}
	if !nestedBoundaryFound {
		t.Fatal("missing explicit UNKNOWN boundary for unexpanded nested registration/dispatch forms")
	}
	for _, dependency := range []string{"resource:archive", "resource:archive:gc", "resource:n8n:runtime", "resource:n8n:runtime:import-workflow", "package:build", "capacity:disk-reserve", "capacity:residency", "auth:status"} {
		if !nestedFound[dependency] {
			t.Fatalf("missing nested/direct group evidence with provenance for %q: %#v", dependency, report.Bindings)
		}
	}
}

func TestCLIArgsDeclarationFailsClosedForAbsentDuplicateAndRenderedFields(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "args.go", `package sample
var _ = struct { Args any }{Args: schema}
var _ = struct { Args any }{Args: first, Args: second}
var _ = struct { Name string }{Name: "none"}
`, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	getFields := func(index int) []ast.Expr {
		decl := file.Decls[index].(*ast.GenDecl)
		value := decl.Specs[0].(*ast.ValueSpec).Values[0].(*ast.CompositeLit)
		return value.Elts
	}
	if expr, status, line := cliArgsDeclaration(getFields(0), fset); status != "DECLARED_UNEVALUATED" || expr != "schema" || line < 1 {
		t.Fatalf("unexpected declared Args result: expression=%q status=%q line=%d", expr, status, line)
	}
	if _, status, _ := cliArgsDeclaration(getFields(1), fset); status != "UNKNOWN_DUPLICATE_ARGS" {
		t.Fatalf("duplicate Args field did not fail closed: %q", status)
	}
	if _, status, line := cliArgsDeclaration(getFields(2), fset); status != "NOT_DECLARED" || line != 0 {
		t.Fatalf("absent Args field was not distinguished: status=%q line=%d", status, line)
	}
}

func TestNestedCommandSpecsPreserveArgsExpressionWithoutEvaluation(t *testing.T) {
	const source = `package sample
func CommandSpecs() []Spec {
	return []Spec{{Name: "probe", Handler: Probe, Args: probeArgs()}}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "commands.go", source, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	specs := cliCommandSpecsFromFunction(fn, nil, fset)
	if len(specs) != 1 || specs[0].argsExpression != "probeArgs()" || specs[0].argsStatus != "DECLARED_UNEVALUATED" {
		t.Fatalf("nested Args expression not retained as unevaluated source: %#v", specs)
	}
	specs[0].argsPath, specs[0].argsHash = "internal/cli/sample/commands.go", strings.Repeat("a", 64)
	row := cliArgsEvidence(specs[0])
	if !strings.Contains(row, "probeArgs()") || !strings.Contains(row, "commands.go:") || !strings.Contains(row, "runtime parsing and acceptance behavior unknown") {
		t.Fatalf("nested Args evidence lost source identity or runtime boundary: %q", row)
	}
	table := &cliNestedTable{function: "CommandSpecs", specName: "CommandSpecs", specs: specs, specPath: specs[0].argsPath, specHash: specs[0].argsHash, registries: map[string][]cliNestedRegistration{"Probe": {{key: "Probe", expr: ast.NewIdent("runProbe")}}}}
	rows := cliRenderNestedTables("sample", []*cliNestedTable{table}, nil, cliPackageFunction{})
	if len(rows) != 1 || !strings.Contains(rows[0].ProviderExpression, "probeArgs()") || !strings.Contains(rows[0].Provenance, "commands.go:") {
		t.Fatalf("nested rendered binding lost Args declaration provenance: %#v", rows)
	}
}

func TestControlPlaneNestedHandlerRegistrationReportsAmbiguityAndUnknown(t *testing.T) {
	const source = `package handlers
func table() {
	bindings := map[string]any{"go": first}
	bindings["go"] = second
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "handlers.go", source, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	unit := cliSourceUnit{path: "internal/cli/sample/handlers.go", fset: fset, file: file, hash: strings.Repeat("a", 64)}
	regs := cliNestedHandlerRegistrations(cliPackageFunction{unit: unit, node: fn}, ast.NewIdent("bindings"), &ast.CallExpr{Fun: ast.NewIdent("CommandSpecs")}, nil)
	if len(regs["go"]) != 2 || len(regs["Missing"]) != 0 {
		t.Fatalf("expected duplicate Go registrations and no Missing registration, got %#v", regs)
	}
	table := &cliNestedTable{function: "table", specName: "CommandSpecs", specs: []cliCommandSpec{{name: "go", handler: "go"}, {name: "missing", handler: "Missing"}}, specPath: unit.path, specHash: unit.hash, registries: regs}
	rows := cliRenderNestedTables("sample", []*cliNestedTable{table}, nil, cliPackageFunction{unit: unit, node: fn})
	statuses := map[string]string{}
	ambiguousSources := false
	for _, row := range rows {
		statuses[row.Dependency] = row.Status
		ambiguousSources = ambiguousSources || strings.Contains(row.ProviderExpression, "competing source registrations")
	}
	if statuses["sample:go"] != "ambiguous" || statuses["sample:missing"] != "unknown" || !ambiguousSources {
		t.Fatalf("expected explicit ambiguous and unknown rows, got %#v", rows)
	}
}

func TestControlPlaneEarlyDispatchExtractorRequiresConditionalExitShape(t *testing.T) {
	const source = `package main
func main() {
	if handled, code := first(args); handled { os.Exit(code) }
	if handled, code := nested(args); handled { return }
	if code := oneResult(args); code > 0 { os.Exit(code) }
}
func nested(args []string) (bool, int) { return false, 0 }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", source, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	hooks := cliEarlyDispatchHooks(file, fset)
	if len(hooks) != 1 || hooks[0].name != "first" || hooks[0].line != 3 {
		t.Fatalf("conditional terminating dispatch probes = %#v, want only first at line 3", hooks)
	}
}

func TestSearchCompositionControlPlaneQueryReturnsRegisteredHandler(t *testing.T) {
	response, err := NewService().SearchComposition(context.Background(), &factsv1.SearchCompositionRequest{
		Query: "vrooli root resource command handler", Scenario: "control-plane", Limit: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range response.GetResults() {
		if hit.GetId() == "code-facts:cli-composition:resource" {
			if hit.GetScope() != "global" || hit.GetAnalyzer() != "code-facts.cli-composition" || hit.GetPath() != "internal/cli/vroolicli/app.go" {
				t.Fatalf("unexpected CLI composition hit: %#v", hit)
			}
			return
		}
	}
	t.Fatalf("resource command handler not ranked: %#v", response.GetResults())
}

func TestSearchCompositionFindsNestedControlPlaneRowsAndRejectsNoise(t *testing.T) {
	service := NewService()
	for _, tc := range []struct {
		query string
		want  string
	}{
		{query: "vrooli resource archive gc handler", want: "code-facts:cli-composition:resource:archive:gc"},
		{query: "vrooli capacity residency handler", want: "code-facts:cli-composition:capacity:residency"},
	} {
		response, err := service.SearchComposition(context.Background(), &factsv1.SearchCompositionRequest{Query: tc.query, Scenario: "control-plane", Limit: 5})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, hit := range response.GetResults() {
			if hit.GetId() == tc.want {
				found = hit.GetScope() == "global" && hit.GetAnalyzer() == "code-facts.cli-composition" && hit.GetPath() != "" && hit.GetStartLine() > 0 && len(hit.GetSourceHash()) == 64
			}
		}
		if !found {
			t.Fatalf("nested source hit %q missing for query %q: %#v", tc.want, tc.query, response.GetResults())
		}
	}
	negative, err := service.SearchComposition(context.Background(), &factsv1.SearchCompositionRequest{Query: "lunar chlorophyll invoice", Scenario: "control-plane", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range negative.GetResults() {
		if hit.GetScore() > 0.2 {
			t.Fatalf("unrelated query produced a strong control-plane hit: %#v", hit)
		}
	}
}

func TestCompositionServiceReturnsTypedTargetAndBindings(t *testing.T) {
	root, err := compositionRepositoryRoot(t)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService()
	report, err := service.Composition(context.Background(), &factsv1.DescribeCompositionRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "search-hub", RepoRoot: root}})
	if err != nil {
		t.Fatal(err)
	}
	if report.GetTarget().GetScenario() != "search-hub" || len(report.GetBindings()) == 0 || !strings.Contains(report.GetAnalysisScope(), "non-Go") {
		t.Fatalf("unexpected service report: target=%v bindings=%d", report.GetTarget(), len(report.GetBindings()))
	}
	for _, binding := range report.GetBindings() {
		if binding.GetStatus() == "concrete" && (binding.GetSourcePath() == "" || binding.GetLine() < 1 || len(binding.GetSourceHash()) != 64) {
			t.Fatalf("concrete binding lacks typed provenance: %#v", binding)
		}
	}
}

func TestSearchCompositionRanksScopedEvidenceAndRejectsNoise(t *testing.T) {
	service := NewService()
	request := &factsv1.SearchCompositionRequest{Query: "production concrete binding", Scenario: "search-hub", Limit: 5}
	first, err := service.SearchComposition(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.GetResults()) == 0 {
		t.Fatal("expected positive composition query hits")
	}
	const expectedID = "code-facts:composition:301ef3075813c446debd5524"
	foundExpected := false
	for _, hit := range first.GetResults() {
		if hit.GetId() == expectedID {
			foundExpected = true
		}
		if !strings.HasPrefix(hit.GetId(), "code-facts:composition:") || hit.GetScore() <= 0 || hit.GetScope() != "scenario:search-hub" || hit.GetPath() == "" || hit.GetStartLine() < 1 || len(hit.GetSourceHash()) != 64 || hit.GetAnalyzer() != "code-facts.composition" {
			t.Fatalf("hit lacks search/provenance contract: %#v", hit)
		}
		if !strings.Contains(hit.GetText(), "status concrete") {
			t.Fatalf("binding status not preserved in text: %q", hit.GetText())
		}
	}
	if !foundExpected {
		t.Fatalf("owner-reviewed descriptor identity %q was not returned", expectedID)
	}
	second, err := service.SearchComposition(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.GetResults()[0].GetId() != second.GetResults()[0].GetId() || first.GetResults()[0].GetScore() != second.GetResults()[0].GetScore() {
		t.Fatalf("unstable hit identity/rank: %#v vs %#v", first.GetResults()[0], second.GetResults()[0])
	}
	negative, err := service.SearchComposition(context.Background(), &factsv1.SearchCompositionRequest{Query: "underwater penguin", Scenario: "search-hub"})
	if err != nil {
		t.Fatal(err)
	}
	if len(negative.GetResults()) != 0 {
		t.Fatalf("unrelated query unexpectedly matched: %#v", negative.GetResults())
	}
}

func compositionRepositoryRoot(t *testing.T) (string, error) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for current := cwd; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "scenarios", "search-hub")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
	}
}
