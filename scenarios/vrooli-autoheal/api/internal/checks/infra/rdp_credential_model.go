package infra

import (
	"context"
	"strings"
	"time"
)

// CredentialModel identifies where GNOME Remote Desktop keeps its credentials.
// User-session credentials remain operator-managed; autoheal never mints them.
type CredentialModel string

const (
	CredentialModelSystem      CredentialModel = "system"
	CredentialModelUserSession CredentialModel = "user-session"
)

const modelProbeTimeout = 10 * time.Second

func (c *RDPCheck) gnomeRDPCredentialModel(ctx context.Context) CredentialModel {
	ctx, cancel := context.WithTimeout(ctx, modelProbeTimeout)
	defer cancel()
	output, err := c.executor.Output(ctx, "systemctl", "is-enabled", "gnome-remote-desktop.service")
	if err != nil {
		return CredentialModelUserSession
	}
	status := strings.TrimSpace(string(output))
	if status != "enabled" && status != "static" {
		return CredentialModelUserSession
	}
	active, err := c.executor.Output(ctx, "systemctl", "is-active", "gnome-remote-desktop.service")
	if err != nil || strings.TrimSpace(string(active)) != "active" {
		return CredentialModelUserSession
	}
	return CredentialModelSystem
}

func keyringModelRemedies() []string {
	return []string{
		"Disable GDM autologin in /etc/gdm3/custom.conf and log in interactively once, so pam_gnome_keyring unlocks the login keyring with the account password.",
		"Or migrate this host to the system-level gnome-remote-desktop.service credential store, where credentials do not depend on a user keyring and autoheal can repair the fault automatically.",
	}
}
