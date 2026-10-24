package infra

import (
	"context"
	"strings"

	sharedhost "github.com/vrooli/vrooli/internal/hostinventory"
)

// CredentialState classifies whether GNOME Remote Desktop holds RDP credentials.
type CredentialState string

const (
	CredentialStatePresent    CredentialState = "present"
	CredentialStateEmpty      CredentialState = "empty"
	CredentialStateUnreadable CredentialState = "unreadable"
)

// readGnomeRDPCredentialState never returns credential values. It only reports
// whether the daemon can authenticate clients from the calling session.
func (c *RDPCheck) readGnomeRDPCredentialState(ctx context.Context) CredentialState {
	env := sessionBusEnv(ctx, c.executor)
	if len(env) == 0 {
		return CredentialStateUnreadable
	}
	output, err := sharedhost.ProbeRemoteDesktopCredentials(ctx, remoteDesktopExecutor{executor: c.executor}, env)
	if err != nil && len(output) == 0 {
		return CredentialStateUnreadable
	}
	return classifyCredentialOutput(string(output))
}

func classifyCredentialOutput(output string) CredentialState {
	if strings.Contains(output, "Failed to read credentials") {
		return CredentialStateUnreadable
	}
	username, hasUsername := credentialFieldValue(output, "Username:")
	password, hasPassword := credentialFieldValue(output, "Password:")
	if !hasUsername || !hasPassword {
		return CredentialStateUnreadable
	}
	if isEmptyCredentialValue(username) || isEmptyCredentialValue(password) {
		return CredentialStateEmpty
	}
	return CredentialStatePresent
}

func credentialFieldValue(output, field string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, field) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, field)), true
		}
	}
	return "", false
}

func isEmptyCredentialValue(value string) bool {
	return value == "" || value == "(empty)" || value == "(null)"
}
