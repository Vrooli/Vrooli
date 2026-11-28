package capture

import "testing"

func TestTerminalReadinessSelectorWaitsForDeclaredAsyncTerminalState(t *testing.T) {
	got := terminalReadinessSelector(`[data-testid="results"]`, "async", []string{"loading", "ready", "empty", "error"})
	want := `[data-testid="results"][data-experience-state="ready"], [data-testid="results"][data-experience-state="empty"], [data-testid="results"][data-experience-state="error"]`
	if got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
	if got := terminalReadinessSelector(`[data-testid="guidance"]`, "static", []string{"static"}); got != `[data-testid="guidance"]` {
		t.Fatalf("static selector = %q", got)
	}
	if got := terminalReadinessSelector(`[data-testid="component-detail-page"][data-preview-state="ready"]`, "async", []string{"loading", "ready"}); got != `[data-testid="component-detail-page"][data-preview-state="ready"]` {
		t.Fatalf("declared terminal selector = %q", got)
	}
}
