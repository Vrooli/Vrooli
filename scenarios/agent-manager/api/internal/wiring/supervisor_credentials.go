package wiring

import credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"

// The purpose has one declared canonical credential slot; no env/file fallback.
func provisionSupervisorCredential(token string) error {
	authority, err := credentialauthority.Default()
	if err != nil {
		return err
	}
	authority.Recheck()
	return authority.Put(credentialauthority.Identity("vrooli/prompt-manager/effort-supervision"), "dispatcher", token)
}
