package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"program-runtime/internal/programs"
)

func TestDeclaredPrivateBindingNamesMatchKernelNamespace(t *testing.T) {
	names := declaredPrivateBindingNames([]programs.BindingSpec{{Scenario: "client-rehearsal", Group: "bounty-cases", Command: "resolve"}})
	require.Equal(t, []string{"client_rehearsal.bounty_cases.resolve"}, names)
}
