package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
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

// serviceTokenExchanger trades the stored credential for a short-lived access
// token and its expiry.
type serviceTokenExchanger func(ctx context.Context, credential string) (string, time.Time, error)

// serviceTokenSource caches Prompt Manager's short-lived agent-manager service
// token. The credential is re-read on every exchange, so rotation and
// revocation take effect without a restart. When the credential has not been
// issued yet the source returns an error and the caller proceeds unqualified —
// the heartbeat is simply refused at the owner until the operator provisions it,
// never silently elevated.
type serviceTokenSource struct {
	credential func() (string, error)
	exchange   serviceTokenExchanger
	now        func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

func newServiceTokenSource(credential func() (string, error), exchange serviceTokenExchanger, now func() time.Time) *serviceTokenSource {
	if now == nil {
		now = time.Now
	}
	return &serviceTokenSource{credential: credential, exchange: exchange, now: now}
}

// Token returns a cached token or exchanges the current credential for one.
func (s *serviceTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.now().Add(serviceRefreshMargin).Before(s.expires) {
		return s.token, nil
	}
	s.token, s.expires = "", time.Time{}
	credential, err := s.credential()
	if err != nil {
		return "", fmt.Errorf("prompt-manager service credential unavailable: %w", err)
	}
	token, expires, err := s.exchange(ctx, strings.TrimSpace(credential))
	if err != nil {
		return "", fmt.Errorf("prompt-manager service credential exchange: %w", err)
	}
	if strings.TrimSpace(token) == "" || !s.now().Before(expires) {
		return "", errors.New("prompt-manager service credential exchange returned no usable token")
	}
	s.token, s.expires = token, expires
	return token, nil
}

// authorityServiceCredential reads the credential from the credential authority.
// It never logs or returns the value except to the exchanger.
func authorityServiceCredential() (string, error) {
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

// authenticatorServiceExchanger exchanges the credential at the discovered
// scenario-authenticator API.
func authenticatorServiceExchanger(httpClient connect.HTTPClient) serviceTokenExchanger {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return func(ctx context.Context, credential string) (string, time.Time, error) {
		base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		if err != nil {
			return "", time.Time{}, err
		}
		client := accountsconnect.NewAccountsServiceClient(httpClient, strings.TrimRight(base, "/"))
		resp, err := client.ExchangeServiceCredential(ctx, connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{Credential: credential}))
		if err != nil {
			return "", time.Time{}, err
		}
		return resp.Msg.GetAccessToken(), resp.Msg.GetAccessTokenExpiresAt().AsTime(), nil
	}
}

// defaultServiceTokenSource is the production source reading prompt-manager's
// issued credential and exchanging it at scenario-authenticator.
func defaultServiceTokenSource() *serviceTokenSource {
	return newServiceTokenSource(authorityServiceCredential, authenticatorServiceExchanger(nil), nil)
}
