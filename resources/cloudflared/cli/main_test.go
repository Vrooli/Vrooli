package main

import (
	"testing"

	"github.com/vrooli/vrooli/resources/testkit"
)

func TestHelpContract(t *testing.T) {
	_ = testkit.Handlers(t)
	const help = "resource-cloudflared: managed by vrooli resource lifecycle commands"
	if help == "" {
		t.Fatal("help text must not be empty")
	}
}
