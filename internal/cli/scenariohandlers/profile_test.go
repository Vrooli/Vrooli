package scenariohandlers

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestProfileHelpDoesNotInspectHostOrRequireOutput(t *testing.T) {
	var output bytes.Buffer
	deps := rootcli.HandlerDeps[struct{}]{Stdout: func(struct{}) io.Writer { return &output }, Globals: func(struct{}) rootcli.GlobalOptions { return rootcli.GlobalOptions{} }}
	if err := profileHandler(deps)(struct{}{}, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "--out") || !strings.Contains(output.String(), "--seconds") {
		t.Fatalf("help=%s", output.String())
	}
}
