package tuning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/envkit-go"
)

func TestBuildWidthDefaultDerivesFromCPU(t *testing.T) {
	resetCountCacheForTest()
	t.Setenv(EnvironmentVariable("BuildWidth"), "")
	want := min(4, max(1, runtime.NumCPU()/4))
	if got := BuildWidth(); got != want {
		t.Fatalf("BuildWidth() = %d, want %d for %d CPUs", got, want, runtime.NumCPU())
	}
}

func TestBuildWidthEnvOverrideAndInvalidFallsBack(t *testing.T) {
	fallback := defaultBuildWidth()
	cases := map[string]int{"2": 2, " 3 ": 3, "abc": fallback, "0": fallback, "-1": fallback}
	for raw, want := range cases {
		resetCountCacheForTest()
		t.Setenv(EnvironmentVariable("BuildWidth"), raw)
		if got := BuildWidth(); got != want {
			t.Errorf("override %q: BuildWidth() = %d, want %d", raw, got, want)
		}
	}
}

func TestEveryCountAccessorAppearsInRegistry(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "count.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse count.go: %v", err)
	}
	want := map[string]bool{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || !ast.IsExported(function.Name.Name) || function.Name.Name == "Count" || function.Type.Results == nil || len(function.Type.Results.List) != 1 || len(function.Type.Params.List) != 0 {
			continue
		}
		ident, ok := function.Type.Results.List[0].Type.(*ast.Ident)
		if !ok || ident.Name != "int" {
			continue
		}
		if function.Doc == nil || function.Doc.Text() == "" {
			t.Errorf("count accessor %s has no doc comment", function.Name.Name)
		}
		want[function.Name.Name] = true
	}
	got := map[string]bool{}
	for _, definition := range countDefinitions {
		if got[definition.Name] {
			t.Fatalf("duplicate registry entry %q", definition.Name)
		}
		if definition.Unit == "" {
			t.Errorf("count lever %s has no unit", definition.Name)
		}
		got[definition.Name] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("count accessor %s is absent from registry", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("registry entry %s has no exported count accessor", name)
		}
	}
}

func TestCountLeverRendersInDocumentation(t *testing.T) {
	resetCountCacheForTest()
	t.Setenv(EnvironmentVariable("BuildWidth"), "")
	doc := RenderDocumentation()
	if !strings.Contains(doc, "`BuildWidth` | count | `VROOLI_TUNING_BUILD_WIDTH` | `"+strconv.Itoa(defaultBuildWidth())+" processes`") {
		t.Fatalf("documentation lacks the BuildWidth row:\n%s", doc)
	}
}

func TestBuildWidthLeverSharesItsVariableWithEnvkit(t *testing.T) {
	if got := EnvironmentVariable("BuildWidth"); got != envkit.BuildWidthKey {
		t.Fatalf("lever variable %q, envkit reads %q", got, envkit.BuildWidthKey)
	}
	if defaultBuildWidth() != envkit.DefaultBuildWidth() {
		t.Fatalf("compiled default %d differs from envkit's %d", defaultBuildWidth(), envkit.DefaultBuildWidth())
	}
}
