package clipolicy

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

func TestNewExternalCLIErrorWithArgs(t *testing.T) {
	err := NewExternalCLIError("prompt-manager", []string{"skill", "read", "swarm-manager-meta-orchestrator"})
	if vroolierr.Code(err, "") != CodeExternalCLIInvocation {
		t.Fatalf("expected code %q, got %q", CodeExternalCLIInvocation, vroolierr.Code(err, ""))
	}

	var buf bytes.Buffer
	PrintErrorWithContext(&buf, err)
	out := buf.String()

	for _, want := range []string{
		"'prompt-manager' is a scenario CLI",
		"Run it directly:",
		"prompt-manager skill read swarm-manager-meta-orchestrator",
		"'vrooli' wrapper",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestNewExternalCLIErrorNoArgs(t *testing.T) {
	err := NewExternalCLIError("prompt-manager", nil)
	var buf bytes.Buffer
	PrintErrorWithContext(&buf, err)
	out := buf.String()
	if !strings.Contains(out, "Run it directly:") {
		t.Errorf("expected 'Run it directly:' even with no args; got:\n%s", out)
	}
	if !strings.Contains(out, "  prompt-manager\n") {
		t.Errorf("expected bare 'prompt-manager' line; got:\n%s", out)
	}
}

func TestNewExternalCLIErrorQuotesWhitespaceArgs(t *testing.T) {
	err := NewExternalCLIError("prompt-manager", []string{"skill", "read", "name with spaces"})
	var buf bytes.Buffer
	PrintErrorWithContext(&buf, err)
	out := buf.String()
	if !strings.Contains(out, `"name with spaces"`) {
		t.Errorf("expected quoted whitespace arg; got:\n%s", out)
	}
}

func TestPrintErrorWithContextUnknownCommandPathUnchanged(t *testing.T) {
	err := NewUnknownCommandError("statsu", []string{"status"})
	var buf bytes.Buffer
	PrintErrorWithContext(&buf, err)
	out := buf.String()
	if !strings.Contains(out, "Unknown command: statsu") {
		t.Errorf("expected unchanged unknown-command rendering; got:\n%s", out)
	}
	if !strings.Contains(out, "Did you mean one of these?") {
		t.Errorf("expected suggestion section; got:\n%s", out)
	}
}
