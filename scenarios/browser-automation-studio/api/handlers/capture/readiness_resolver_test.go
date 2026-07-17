package capture

import "testing"

func TestContainsRouteMatchesDeclaredLiteralAndParameterizedRoutes(t *testing.T) {
	if !containsRoute([]string{"/", "/results"}, "/results") {
		t.Fatal("declared route was not matched")
	}
	if !containsRoute([]string{"/assets/:id"}, "/assets/react-component-library%3AButton?tab=preview") {
		t.Fatal("declared parameterized route was not matched")
	}
	if containsRoute([]string{"/results"}, "/results/other") {
		t.Fatal("undeclared route must retain generic fallback")
	}
	if containsRoute([]string{"/assets/:id"}, "/assets/Button/history") {
		t.Fatal("parameterized route must not match an undeclared subroute")
	}
}

func TestTerminalReadinessSelectorWaitsForDeclaredAsyncTerminalState(t *testing.T) {
	got := terminalReadinessSelector(`[data-testid="results"]`, "async", []string{"loading", "ready", "empty", "error"})
	want := `[data-testid="results"][data-experience-state="ready"], [data-testid="results"][data-experience-state="empty"], [data-testid="results"][data-experience-state="error"]`
	if got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
	if got := terminalReadinessSelector(`[data-testid="guidance"]`, "static", []string{"static"}); got != `[data-testid="guidance"]` {
		t.Fatalf("static selector = %q", got)
	}
}
