package uiselectors

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestConformance(t *testing.T) {
	raw, err := os.ReadFile("../../ui-selectors/conformance.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Manifest map[string]interface{} `json:"manifest"`
		Cases    []struct {
			Name, Key, Want string
			Params          map[string]interface{}
			Error           bool
		}
	}
	if err = json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	for _, row := range suite.Cases {
		t.Run(row.Name, func(t *testing.T) {
			got, err := Resolve(suite.Manifest, row.Key, row.Params)
			if row.Error {
				if err == nil {
					t.Fatalf("expected rejection, got %s", got)
				}
				return
			}
			if err != nil || got != row.Want {
				t.Fatalf("got %q, %v; want %q", got, err, row.Want)
			}
		})
	}
}

func TestDeferredParametersValidateAndEscapeAtExecution(t *testing.T) {
	raw, _ := os.ReadFile("../../ui-selectors/conformance.json")
	var suite struct{ Manifest map[string]interface{} }
	json.Unmarshal(raw, &suite)
	compiled := ResolveReference(`@selector/item(name="${@params/name}")`, suite.Manifest)
	if !strings.HasPrefix(compiled, deferredPrefix) {
		t.Fatal(compiled)
	}
	got, err := ResolveDeferred(compiled, func(string) (string, error) { return `a"b`, nil })
	if err != nil || got != `[data-name="a\22 b"]` {
		t.Fatalf("%q %v", got, err)
	}
	compiled = ResolveReference(`@selector/locale(code="${@params/code}")`, suite.Manifest)
	if _, err = ResolveDeferred(compiled, func(string) (string, error) { return "invalid", nil }); err == nil {
		t.Fatal("invalid enum accepted")
	}
}

func TestNamespacedReferenceAndExpressionEscaping(t *testing.T) {
	manifest := map[string]interface{}{"selectors": map[string]interface{}{
		"library.catalog:Button.root": map[string]interface{}{"selector": `[data-testid="button\2e root"]`},
	}}
	got := ResolveReference("@selector/library.catalog:Button.root", manifest)
	if got != `[data-testid="button\2e root"]` {
		t.Fatalf("namespaced reference: %q", got)
	}
	if got := EscapeExpressionSelector(got, '\''); got != `[data-testid="button\\2e root"]` {
		t.Fatalf("JavaScript escaping: %q", got)
	}
}
