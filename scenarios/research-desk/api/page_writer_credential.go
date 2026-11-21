package main

import (
	"context"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// The desk's Agent Manager identity, for the decision-page writer's runs: the
// `research-desk-agent-manager` service principal (resource agent-manager,
// scope agent-manager:write) that .vrooli/service-principals.json declares.
// The operator issues it (`scenario-authenticator service-principals sync
// --apply`); an agent run cannot. The credential only proves "I am the desk";
// what lets it create runs is the operator's standing run.create grant keyed
// to this subject. Without that grant Agent Manager asks the operator for one
// (202 approval_pending) and the writer retries after its cool-down.
const (
	pageWriterPrincipal          = "research-desk-agent-manager"
	pageWriterCredentialIdentity = "scenario-authenticator/service-principal/" + pageWriterPrincipal + "/agent-manager"
	pageWriterCredentialField    = "credential"
	pageWriterAudience           = "scenario-authenticator:agent-manager"
)

// newPageWriterTokens is the desk's Agent Manager token client over a
// credential source. It presents only a token that is exactly the desk's
// principal for Agent Manager.
func newPageWriterTokens(credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Name:         pageWriterPrincipal,
		Credential:   credential,
		Expect:       servicetoken.ExpectIdentity(pageWriterPrincipal, pageWriterAudience, "agent-manager:write"),
		MinRemaining: servicetoken.DefaultRefreshMargin,
	}, options...)
}

// authorityPageWriterCredential reads the credential from the credential
// authority entry scenario-authenticator writes on issuance.
func authorityPageWriterCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(pageWriterCredentialIdentity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, pageWriterCredentialField)
}
