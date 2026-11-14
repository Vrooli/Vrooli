package ledger

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const (
	TreasuryServiceCredentialIdentity = "scenario-authenticator/service-principal/treasury/money-ledger"
	TreasuryNoticeCredentialIdentity  = "scenario-authenticator/service-principal/treasury/notification-hub"
)

type TokenSource interface {
	Token(context.Context) (string, error)
}

type CredentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

// ServiceTokenSource consumes an operator-issued scenario-authenticator service
// principal. It cannot mint one, and a caller's agent or human token is never
// forwarded as Treasury identity. The credential owner remains authoritative
// for issuance, rotation and revocation.
type ServiceTokenSource struct {
	credentials        CredentialReader
	credentialIdentity string
	client             accountsconnect.AccountsServiceClient
	now                func() time.Time
	mu                 sync.Mutex
	token              string
	expires            time.Time
}

func NewServiceTokenSource(credentials CredentialReader, authenticatorURL string, client *http.Client) (*ServiceTokenSource, error) {
	return NewServiceTokenSourceFor(credentials, authenticatorURL, client, "treasury", "money-ledger")
}

// NewServiceTokenSourceFor consumes one operator-issued service principal
// bound to the supplied service name and resource. Separate resources must
// use separate credentials and audiences.
func NewServiceTokenSourceFor(credentials CredentialReader, authenticatorURL string, client *http.Client, serviceName, resource string) (*ServiceTokenSource, error) {
	if credentials == nil || strings.TrimSpace(authenticatorURL) == "" {
		return nil, errors.New("Treasury service credential authority and authenticator are required")
	}
	serviceName, resource = strings.TrimSpace(serviceName), strings.TrimSpace(resource)
	if serviceName == "" || resource == "" {
		return nil, errors.New("Treasury service principal name and resource are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	identity := "scenario-authenticator/service-principal/" + serviceName + "/" + resource
	return &ServiceTokenSource{credentials: credentials, credentialIdentity: identity, client: accountsconnect.NewAccountsServiceClient(client, strings.TrimRight(authenticatorURL, "/")), now: time.Now}, nil
}

func (s *ServiceTokenSource) Token(ctx context.Context) (string, error) {
	if s == nil || s.credentials == nil || s.client == nil {
		return "", errors.New("Treasury service identity is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.now().Add(time.Minute).Before(s.expires) {
		return s.token, nil
	}
	s.token, s.expires = "", time.Time{}
	identity := strings.TrimSpace(s.credentialIdentity)
	if identity == "" {
		identity = TreasuryServiceCredentialIdentity
	}
	credential, err := s.credentials.Resolve(ctx, identity, "credential")
	if err != nil || strings.TrimSpace(credential) == "" {
		return "", errors.New("Treasury service credential is unavailable")
	}
	response, err := s.client.ExchangeServiceCredential(ctx, connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{Credential: credential}))
	if err != nil {
		return "", errors.New("Treasury service credential exchange failed")
	}
	if response.Msg == nil || strings.TrimSpace(response.Msg.GetAccessToken()) == "" || response.Msg.GetAccessTokenExpiresAt() == nil || response.Msg.GetAccessTokenExpiresAt().CheckValid() != nil || !s.now().Before(response.Msg.GetAccessTokenExpiresAt().AsTime()) {
		return "", errors.New("Treasury service credential exchange returned no current token")
	}
	s.token, s.expires = response.Msg.GetAccessToken(), response.Msg.GetAccessTokenExpiresAt().AsTime()
	return s.token, nil
}
