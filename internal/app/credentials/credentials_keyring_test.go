package credentials

import (
	"strings"
	"testing"
	"time"

	keyring "github.com/vrooli/vrooli/internal/credentials"
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

func TestCredentialKeyringRepairOffersExplicitRetirementForOldBackups(t *testing.T) {
	report := keyring.RepairReport{File: &keyring.KeyringReport{Backups: []keyring.KeyringBackup{
		{Path: "/tmp/login.keyring.corrupt-backup", AgeSeconds: int64(48 * time.Hour / time.Second)},
		{Path: "/tmp/login.keyring.corrupt-backup.1", AgeSeconds: int64(2 * time.Hour / time.Second)},
	}}}
	offers := keyringRetirementOffers(report, 24*time.Hour)
	if len(offers) != 1 || !strings.Contains(offers[0], "login.keyring.corrupt-backup") || !strings.Contains(offers[0], "vrooli credentials keyring repair --retire-backup") {
		t.Fatalf("retirement offers = %v", offers)
	}
}

func TestCredentialKeyringLockedRemedyUsesInteractiveUnlock(t *testing.T) {
	joined := strings.Join(credentialKeyringRemedy("locked"), " ")
	if !strings.Contains(joined, "credentials keyring unlock") {
		t.Fatalf("locked remedy = %q, want keyring unlock command", joined)
	}
	if strings.Contains(joined, "--password") || strings.Contains(joined, "PASSPHRASE=") {
		t.Fatalf("locked remedy exposes a passphrase channel: %q", joined)
	}
}
