package securestore

import "testing"

// TestPassphraseComesFromTheNodeAgentWhenNoneWasSupplied is the headless reboot
// path: no operator typed a passphrase, and the Bridge agent holds the one the
// control plane escrowed, so the store's passphrase wrap can open.
func TestPassphraseComesFromTheNodeAgentWhenNoneWasSupplied(t *testing.T) {
	previousNode := nodeUnlockPassphrase
	nodeUnlockPassphrase = func() string { return "escrowed-by-control-plane" }
	t.Cleanup(func() { nodeUnlockPassphrase = previousNode })
	SetPassphrase("")
	t.Cleanup(func() { SetPassphrase("") })

	if got := passphraseSource(); got != "escrowed-by-control-plane" {
		t.Fatalf("passphraseSource() = %q, want the agent-held passphrase", got)
	}
	provider := passphraseProvider{source: passphraseSource}
	if _, err := provider.Available(); err != nil {
		t.Fatalf("passphrase wrap unavailable with an agent-held passphrase: %v", err)
	}
}

// An operator-supplied passphrase (store init / unlock) always wins, so a
// deliberate unlock is never silently replaced by the agent's answer.
func TestAnOperatorPassphraseTakesPrecedenceOverTheNodeAgent(t *testing.T) {
	previousNode := nodeUnlockPassphrase
	nodeUnlockPassphrase = func() string { return "escrowed-by-control-plane" }
	t.Cleanup(func() { nodeUnlockPassphrase = previousNode })
	SetPassphrase("typed-by-operator")
	t.Cleanup(func() { SetPassphrase("") })

	if got := passphraseSource(); got != "typed-by-operator" {
		t.Fatalf("passphraseSource() = %q, want the operator's passphrase", got)
	}
}

// With no agent and no passphrase, the wrap stays unavailable as before.
func TestNoAgentAndNoPassphraseLeavesTheWrapUnavailable(t *testing.T) {
	previousNode := nodeUnlockPassphrase
	nodeUnlockPassphrase = func() string { return "" }
	t.Cleanup(func() { nodeUnlockPassphrase = previousNode })
	SetPassphrase("")

	if _, err := (passphraseProvider{source: passphraseSource}).Available(); err == nil {
		t.Fatal("passphrase wrap reported available with no passphrase anywhere")
	}
}
