package heartbeat

import (
	"context"
	"time"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// Prompt Manager presents an operator-issued service principal to Agent Manager
// so scheduled heartbeats create runs as a verified service identity. The
// operator-approved standing grant keyed to this subject is what authorizes the
// costly run.create; the credential here only proves "I am prompt-manager".
//
// scenario-authenticator owns and writes the credential entry
// (`scenario-authenticator auth issue-service-principal --name prompt-manager`);
// prompt-manager only reads it. Deleting the entry revokes the principal.
const (
	serviceCredentialIdentity = "scenario-authenticator/service-principal/prompt-manager/agent-manager"
	serviceCredentialField    = "credential"
)

// serviceRefreshMargin renews a token this long before it expires.
const serviceRefreshMargin = 2 * time.Minute

// newServiceTokens is Prompt Manager's agent-manager service-principal client.
// The credential is re-read on every exchange, so rotation and revocation take
// effect without a restart. When the credential has not been issued yet the
// client returns an error and the caller proceeds unqualified: the heartbeat is
// simply refused at the owner until the operator provisions it, never silently
// elevated. Agent Manager verifies the token itself, so no identity is expected
// here.
func newServiceTokens(credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Credential:    credential,
		RefreshMargin: serviceRefreshMargin,
		Describe:      servicetoken.DescribeWithCause("prompt-manager"),
	}, options...)
}

// authorityServiceCredential reads the credential from the credential authority.
// It never logs or returns the value except to the exchanger.
func authorityServiceCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(serviceCredentialIdentity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, serviceCredentialField)
}

// defaultServiceTokens is the production client reading prompt-manager's issued
// credential and exchanging it at scenario-authenticator.
func defaultServiceTokens() *servicetoken.Tokens {
	return newServiceTokens(authorityServiceCredential)
}
