package infra

import (
	"context"
)

// CredentialModel identifies where GNOME Remote Desktop keeps its credentials.
// User-session credentials remain operator-managed; autoheal never mints them.
type CredentialModel string

const (
	CredentialModelSystem      CredentialModel = "system"
	CredentialModelUserSession CredentialModel = "user-session"
)

func (c *RDPCheck) gnomeRDPCredentialModel(_ context.Context) CredentialModel {
	if c.cachedServiceInfo != nil && !c.cachedServiceInfo.IsUserSession && c.cachedServiceInfo.Type == RDPTypeGnome {
		return CredentialModelSystem
	}
	return CredentialModelUserSession
}

func keyringModelRemedies() []string {
	return []string{
		"Disable GDM autologin in /etc/gdm3/custom.conf and log in interactively once, so pam_gnome_keyring unlocks the login keyring with the account password.",
		"Or migrate this host to the system-level gnome-remote-desktop.service credential store, where credentials do not depend on a user keyring and autoheal can repair the fault automatically.",
	}
}
