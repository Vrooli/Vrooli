package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentialStoreRefusalClassifiesRecovery(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		state    string
		recovery string
	}{
		{name: "uninitialized", output: "credential store is not initialized", state: "uninitialized", recovery: "credentials store init"},
		{name: "locked", output: "credential backend locked", state: "locked", recovery: "credentials store unlock"},
		{name: "unresponsive", output: "keyring timed out", state: "unresponsive", recovery: "credentials keyring repair"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			refusal, ok := credentialStoreRefusal(test.output)
			require.True(t, ok)
			require.Equal(t, test.state, refusal.State)
			require.Contains(t, refusal.Recovery, test.recovery)
		})
	}
}
