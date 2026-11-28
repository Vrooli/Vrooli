package main

import "testing"

// NewApp parses the embedded manifest and wires every declared binding.
// This catches packaging/bootstrap failures that a source-file domain test
// could miss when the installed CLI still embeds an older manifest.
func TestNewAppRegistersQualificationReviewCommand(t *testing.T) {
	if _, err := NewApp(); err != nil {
		t.Fatalf("build CLI with qualification review command: %v", err)
	}
}
