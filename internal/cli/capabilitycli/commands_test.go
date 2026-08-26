package capabilitycli

import (
	"bytes"
	"testing"
)

func TestRunCapabilityHelp(t *testing.T) {
	var output bytes.Buffer
	if err := Run(&App{}, &Context{Stdout: &output}, []string{"--help"}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if output.Len() == 0 {
		t.Fatal("capability help should be non-empty")
	}
}
