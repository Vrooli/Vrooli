package vroolicli

import (
	"strings"
	"testing"
)

func TestCredentialKeyringRemedyNamesEachUnavailableState(t *testing.T) {
	for _, state := range []string{"locked", "unresponsive", "unavailable", "empty", "unsupported"} {
		remedies := credentialKeyringRemedy(state)
		if len(remedies) == 0 {
			t.Fatalf("state %q has no operator remedy", state)
		}
		joined := strings.Join(remedies, " ")
		if strings.Contains(joined, "grdctl rdp set-credentials") {
			t.Fatalf("state %q offered a credential-setting remedy: %q", state, joined)
		}
	}
}

func TestCredentialKeyringLockedRemedyUsesStdinUnlock(t *testing.T) {
	joined := strings.Join(credentialKeyringRemedy("locked"), " ")
	if !strings.Contains(joined, "credentials keyring unlock") {
		t.Fatalf("locked remedy = %q, want keyring unlock command", joined)
	}
	if strings.Contains(joined, "--password") || strings.Contains(joined, "PASSPHRASE=") {
		t.Fatalf("locked remedy exposes a passphrase channel: %q", joined)
	}
}
