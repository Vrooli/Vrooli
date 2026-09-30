package tuning

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
	"time"
)

func TestEveryDurationAccessorAppearsInRegistry(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "timing.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse timing.go: %v", err)
	}
	want := map[string]bool{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil || !ast.IsExported(function.Name.Name) || function.Type.Results == nil || len(function.Type.Results.List) != 1 {
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
