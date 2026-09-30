package remotedesktopaccess

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func credentialCommand() string {
	return credentialProvisionCommand("username")
}

func setupCommand() string {
	return "vrooli setup --include-optional --maintenance-window --sudo-mode=ask"
}

func credentialProvisionCommand(field string) string {
	return "vrooli credentials provision --identity " + remoteDesktopID + " --field " + field
}

func credentialProvisionRequired(status *hostreqkit.ItemStatus, field string) hostreqkit.ItemStatus {
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.BlockingReason = hostreqkit.BlockingManual
	status.Command = credentialProvisionCommand(field)
	status.Notes = append(status.Notes, "remote-desktop credential is not configured in Vrooli's encrypted authority; run "+credentialProvisionCommand(field)+" and retry")
	return *status
}

func resolveRemoteDesktopCredential(identity, field string) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return "", err
	}
	return authority.Require(parsed, field)
}

func remoteDesktopCredentialsReady() bool {
	authority, err := credentialauthority.Default()
	if err != nil {
		return false
	}
	identity, err := credentialauthority.ParseIdentity(remoteDesktopID)
	if err != nil {
		return false
	}
	for _, field := range []string{"username", "password"} {
		if !authority.Status(identity, field).Configured {
			return false
		}
	}
	return true
}

func credentialStoreBlock(status *hostreqkit.ItemStatus) bool {
	var reason hostreqkit.BlockingReason
	var remedy string
	switch status.CredentialStoreState {
	case "locked":
		reason = hostreqkit.BlockingCredentialStoreLocked
		remedy = "run `vrooli credentials keyring unlock`; if the login keyring is intentionally password-protected, opt in to login_keyring_unlock before retrying"
	case "unresponsive":
		reason = hostreqkit.BlockingCredentialStoreUnresponsive
		remedy = "run `vrooli credentials keyring status` and restore the session bus before retrying"
	case "unavailable", "unsupported":
		reason = hostreqkit.BlockingCredentialStoreUnavailable
		remedy = "make the active user's credential store available, then rerun `vrooli credentials keyring status`"
	default:
		return false
	}
	status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	status.BlockingReason = reason
	status.Command = ""
	status.Notes = append(status.Notes, "credential store state is "+status.CredentialStoreState+"; "+remedy)
	return true
}
