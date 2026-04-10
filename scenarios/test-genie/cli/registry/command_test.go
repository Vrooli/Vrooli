package registry

import (
	"strings"
	"testing"
)

func TestRunHelpAndNoArgsReturnUsage(t *testing.T) {
	if err := Run(nil); err != nil {
		t.Fatalf("expected no-arg registry invocation to print usage, got %v", err)
	}
	if err := Run([]string{"help"}); err != nil {
		t.Fatalf("expected explicit help to succeed, got %v", err)
	}
}

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := Run([]string{"unknown"})
	if err == nil {
		t.Fatal("expected unknown registry subcommand to fail")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("expected unknown-subcommand error, got %v", err)
	}
}
