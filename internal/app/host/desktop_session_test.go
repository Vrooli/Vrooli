package hostapp

import (
	"bytes"
	"strings"
	"testing"
)

func TestDesktopSessionDispatchRejectsUnsafeIdentity(t *testing.T) {
	var output bytes.Buffer
	ctx := &CommandContext{Stdout: &output, Stderr: &output}
	err := (&App{}).Run(ctx, []string{"desktop-session", "--session-id", "../2", "--peer-pid", "42", "--json"})
	if err == nil || !strings.Contains(err.Error(), "invalid desktop session identity") {
		t.Fatalf("dispatch error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("invalid request produced facts: %s", output.String())
	}
}
