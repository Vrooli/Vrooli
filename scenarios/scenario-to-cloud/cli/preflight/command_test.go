package preflight

import (
	"strings"
	"testing"
)

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := Run(nil, []string{"unknown-subcommand"})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRequirementsRejectsUnknownFlag(t *testing.T) {
	err := runRequirements(nil, []string{"--nope"})
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --nope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatSizeAndJoinPortsHelpers(t *testing.T) {
	if got := formatSize(1024); got != "1.0K" {
		t.Fatalf("formatSize(1024)=%q", got)
	}
	if got := formatSize(1024 * 1024); got != "1.0M" {
		t.Fatalf("formatSize(1MiB)=%q", got)
	}
	if got := joinPorts([]int{22, 80, 443}); got != "22, 80, 443" {
		t.Fatalf("joinPorts=%q", got)
	}
	if got := joinPorts(nil); got != "-" {
		t.Fatalf("joinPorts(nil)=%q", got)
	}
}
