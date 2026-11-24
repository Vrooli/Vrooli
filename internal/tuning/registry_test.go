package tuning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEveryDurationAccessorAppearsInRegistry(t *testing.T) {
	// The invariant is package-wide: an accessor may live in any non-test file
	// of this package, so the scan follows the package rather than one filename.
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), source, nil, parser.ParseComments)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", source, parseErr)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || !ast.IsExported(function.Name.Name) || function.Type.Results == nil || len(function.Type.Results.List) != 1 {
				continue
			}
			// Duration is the generic resolver every accessor delegates to, not a
			// lever of its own.
			if function.Name.Name == "Duration" {
				continue
			}
			selector, ok := function.Type.Results.List[0].Type.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Duration" {
				continue
			}
			if function.Doc == nil || function.Doc.Text() == "" {
				t.Errorf("duration accessor %s has no doc comment", function.Name.Name)
			}
			want[function.Name.Name] = true
		}
	}
	got := map[string]bool{}
	for _, definition := range durationDefinitions {
		if got[definition.Name] {
			t.Fatalf("duplicate registry entry %q", definition.Name)
		}
		got[definition.Name] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("duration accessor %s is absent from registry", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("registry entry %s has no exported duration accessor", name)
		}
	}
}

func TestLeversExposeDefaultsResolutionAndSource(t *testing.T) {
	resetDurationCacheForTest()
	t.Setenv(EnvironmentVariable("HealthCheckTimeout"), "9s")
	levers := Levers()
	if len(levers) != len(durationDefinitions)+len(countDefinitions) {
		t.Fatalf("Levers count = %d, want %d", len(levers), len(durationDefinitions)+len(countDefinitions))
	}
	var health, runtime Lever
	for _, lever := range levers {
		if lever.Description == "" || lever.Environment == "" {
			t.Errorf("incomplete lever metadata: %+v", lever)
		}
		switch lever.Name {
		case "HealthCheckTimeout":
			health = lever
		case "ScenarioActionTimeout":
			runtime = lever
		}
	}
	if health.CompiledDefault != (3*time.Second).String() || health.ResolvedValue != "9s" || health.Source != "environment" {
		t.Errorf("HealthCheckTimeout = %+v", health)
	}
	if runtime.CompiledDefault != callerProvidedValue || runtime.ResolvedValue != callerProvidedValue || runtime.Source != "default" {
		t.Errorf("ScenarioActionTimeout = %+v", runtime)
	}
}
