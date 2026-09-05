package runtimecli

import (
	"bytes"
	"testing"
)

func TestRunDelegatesRuntimeHelp(t *testing.T) {
	var output bytes.Buffer
	ctx := &Context{Stdout: &output}
	if err := Run(&App{}, ctx, []string{"--help"}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if output.String() == "" {
		t.Fatal("runtime CLI help should be non-empty")
	}
}
