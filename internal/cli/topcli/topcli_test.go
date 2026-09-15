package topcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderMainHelpUsesPlainLabelsAndIncludesContract(t *testing.T) {
	var stdout bytes.Buffer
	RenderMainHelp(&stdout, CommandSpecs())

	output := stdout.String()
	if strings.Contains(output, "🚀") || strings.Contains(output, "📋") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "check-shared-drift") {
		t.Fatalf("retired check-shared-drift command appeared in help: %q", output)
	}
	if strings.Contains(output, "agent-policy") {
		t.Fatalf("removed agent-policy command appeared in help: %q", output)
	}
	if !strings.Contains(output, "Vrooli CLI - AI Platform Management Tool") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "contract") {
		t.Fatalf("output = %q", output)
	}
	for _, want := range []string{"--verbose", "Documentation: docs/"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output = %q", want, output)
		}
	}
}
