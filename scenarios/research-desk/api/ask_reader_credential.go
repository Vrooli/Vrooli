package main

import (
	"context"

	"github.com/vrooli/api-core/askreader"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// askReaderCredential reads the desk's ask-reader credential, the one the
// operator issues with serviceprincipal.IssueCommand, from the credential authority
// entry scenario-authenticator writes on issuance. There is no fallback: until
// the credential is issued the shared client reports that and the desk answers
// in the desk instead of presenting a credential it did not read.
func askReaderCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(askreader.CredentialIdentity("research-desk"))
	if err != nil {
		return "", err
	}
	return authority.Require(identity, askreader.CredentialField)
}
