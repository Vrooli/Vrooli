package facts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCLIRootHandlerAliasResolvesSinglePackageAndAbstainsWithoutHandler(t *testing.T) {
	single := parseCLIRootAliasFixture(t, `package app
import handlers "github.com/vrooli/vrooli/internal/cli/resourcehandlers"
func factory() { handlers.RootHandler() }
`)
	alias, ok, ambiguous := cliRootHandlerAlias("factory", single)
	if !ok || ambiguous || alias != "handlers" {
		t.Fatalf("single package root handler = (%q,%t,%t)", alias, ok, ambiguous)
	}

	noHandler := parseCLIRootAliasFixture(t, `package app
func factory() { run() }
`)
	if alias, ok, ambiguous := cliRootHandlerAlias("factory", noHandler); ok || ambiguous || alias != "" {
		t.Fatalf("missing root handler must not resolve: (%q,%t,%t)", alias, ok, ambiguous)
	}
}

func TestCLIRootHandlerAliasMarksMultiplePackageFactoryAmbiguous(t *testing.T) {
	app := parseCLIRootAliasFixture(t, `package app
import (
  resource "github.com/vrooli/vrooli/internal/cli/resourcehandlers"
  scenario "github.com/vrooli/vrooli/internal/cli/scenariohandlers"
)
func factory(args []string) {
  if len(args) > 0 { resource.RootHandler() }
  scenario.RootHandler()
}
`)
	alias, ok, ambiguous := cliRootHandlerAlias("factory", app)
	if ok || !ambiguous || alias != "" {
		t.Fatalf("multiple package roots must remain ambiguous: (%q,%t,%t)", alias, ok, ambiguous)
	}

	fset := token.NewFileSet()
	unit := cliSourceUnit{path: "internal/cli/vroolicli/app.go", fset: fset, file: app, hash: strings.Repeat("a", 64)}
	rows := cliNestedGroupBindings(t.TempDir(), unit, strings.Repeat("a", 64), []cliCommandSpec{{name: "tools", handler: "tools"}}, map[string]cliHandlerBinding{"tools": {expression: "factory", line: 7}})
	if len(rows) != 1 || rows[0].Status != "unknown" || !strings.Contains(rows[0].ProviderExpression, "package ownership is ambiguous") {
		t.Fatalf("ambiguous factory must return explicit UNKNOWN evidence: %#v", rows)
	}
}

func parseCLIRootAliasFixture(t *testing.T, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "app.go", source, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
