package ledger

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/servicetoken"
)

// Treasury calls three peers, and holds one service principal per peer: a
// principal name is unique (it is the token subject and part of the credential's
// address), and one subject per credential lets each peer accept exactly its own
// principal, so a token minted for one peer can never satisfy another. The
// operator issues them with `scenario-authenticator service-principals sync
// --apply` from .vrooli/service-principals.json; TestTreasuryDeclaresExactlyTheServicePrincipalsItUses holds that
// file to the names below.
const (
	// MoneyLedgerPrincipal is Treasury's principal at Money Ledger, which accepts
	// only this subject (and journals the actor as "treasury").
	MoneyLedgerPrincipal = "treasury-money-ledger"
	MoneyLedgerResource  = "money-ledger"
	// AgentManagerPrincipal is Treasury's principal at Agent Manager, for run-bound
	// research-budget evidence. It is also the subject of the one-time grant an
	// operator approves.
	AgentManagerPrincipal = "treasury-agent-manager"
	AgentManagerResource  = "agent-manager"
	// NoticePrincipal is Treasury's sender principal at Notification Hub: the Hub
	// reads the source "treasury" off the name (api-core hubsender).
	NoticePrincipal = "treasury-notification-hub"
	NoticeResource  = "notification-hub"

	TreasuryServiceCredentialIdentity      = "scenario-authenticator/service-principal/" + MoneyLedgerPrincipal + "/" + MoneyLedgerResource
	TreasuryAgentManagerCredentialIdentity = "scenario-authenticator/service-principal/" + AgentManagerPrincipal + "/" + AgentManagerResource
	TreasuryNoticeCredentialIdentity       = "scenario-authenticator/service-principal/" + NoticePrincipal + "/" + NoticeResource
)

type TokenSource interface {
	Token(context.Context) (string, error)
}

type CredentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

// NewServiceTokenSource consumes Treasury's operator-issued money-ledger service
// principal. It cannot mint one, and a caller's agent or human token is never
// forwarded as Treasury identity. The credential owner remains authoritative
// for issuance, rotation and revocation.
func NewServiceTokenSource(credentials CredentialReader, authenticatorURL string, client *http.Client, options ...servicetoken.Option) (*servicetoken.Tokens, error) {
	return NewServiceTokenSourceFor(credentials, authenticatorURL, client, MoneyLedgerPrincipal, MoneyLedgerResource, options...)
}

// NewServiceTokenSourceFor consumes one operator-issued service principal
// bound to the supplied principal name and resource. Separate resources must
// use separate credentials and audiences. Each peer verifies the token it is
// given, so no identity is expected here, and the errors are deliberately
// terse: they never carry the credential authority's or authenticator's reason.
func NewServiceTokenSourceFor(credentials CredentialReader, authenticatorURL string, client *http.Client, principal, resource string, options ...servicetoken.Option) (*servicetoken.Tokens, error) {
	if credentials == nil || strings.TrimSpace(authenticatorURL) == "" {
		return nil, errors.New("Treasury service credential authority and authenticator are required")
	}
	principal, resource = strings.TrimSpace(principal), strings.TrimSpace(resource)
	if principal == "" || resource == "" {
		return nil, errors.New("Treasury service principal name and resource are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	identity := "scenario-authenticator/service-principal/" + principal + "/" + resource
	exchanger := servicetoken.AuthenticatorExchanger(client, func(context.Context) (string, error) {
		return strings.TrimRight(authenticatorURL, "/"), nil
	})
	return servicetoken.New(servicetoken.Config{
		Credential: func(ctx context.Context) (string, error) {
			return credentials.Resolve(ctx, identity, "credential")
		},
		RefreshMargin: time.Minute,
		Describe:      describeTreasuryServiceFailure,
	}, append([]servicetoken.Option{servicetoken.WithExchanger(exchanger)}, options...)...), nil
}

func describeTreasuryServiceFailure(kind, _ error) error {
	switch kind {
	case servicetoken.ErrNotIssued:
		return servicetoken.Kind(kind, errors.New("Treasury service credential is unavailable"))
	case servicetoken.ErrExchangeFailed:
		return servicetoken.Kind(kind, errors.New("Treasury service credential exchange failed"))
	case servicetoken.ErrNoUsableToken:
		return servicetoken.Kind(kind, errors.New("Treasury service credential exchange returned no current token"))
	}
	return servicetoken.Kind(kind, errors.New("Treasury service identity is unavailable"))
}
