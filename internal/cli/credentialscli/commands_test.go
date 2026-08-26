package credentialscli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestRunRendersCredentialsHelpThroughTheBoundary(t *testing.T) {
	var out bytes.Buffer
	ctx := &Context{Globals: rootcli.GlobalOptions{}, Stdout: &out, Stderr: &out}
	if err := Run(ctx, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "vrooli credentials doctor") {
		t.Fatalf("help = %q", out.String())
	}
}
