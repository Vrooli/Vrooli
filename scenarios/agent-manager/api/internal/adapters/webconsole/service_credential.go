package webconsole

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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

// ServiceCredentialIdentity and ServiceCredentialField address Agent Manager's
// operator-issued service-principal credential for the web-console audience.
// scenario-authenticator owns and writes the entry (`scenario-authenticator
// auth issue-service-principal --name agent-manager`); Agent Manager only reads
// it. Deleting the entry revokes the principal.
const (
	ServiceCredentialIdentity = "scenario-authenticator/service-principal/agent-manager/web-console"
	ServiceCredentialField    = "credential"
)

// refreshMargin renews a service token this long before it expires.
const refreshMargin = 2 * time.Minute

// ServiceTokenExchanger trades the stored credential for a short-lived access
// token and its expiry.
type ServiceTokenExchanger func(ctx context.Context, credential string) (string, time.Time, error)

// ServiceTokenSource caches Agent Manager's short-lived web-console service
// token. The credential is re-read on every exchange, so rotation and
// revocation take effect without restarting Agent Manager.
type ServiceTokenSource struct {
	credential func() (string, error)
	exchange   ServiceTokenExchanger
	now        func() time.Time

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewServiceTokenSource builds a token source from a credential reader and an
// exchanger. now may be nil.
func NewServiceTokenSource(credential func() (string, error), exchange ServiceTokenExchanger, now func() time.Time) *ServiceTokenSource {
	if now == nil {
		now = time.Now
	}
	return &ServiceTokenSource{credential: credential, exchange: exchange, now: now}
}

// Token returns a cached token or exchanges the current credential for one.
func (s *ServiceTokenSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.now().Add(refreshMargin).Before(s.expires) {
		return s.token, nil
	}
	s.token, s.expires = "", time.Time{}
	credential, err := s.credential()
	if err != nil {
		return "", fmt.Errorf("agent-manager service credential unavailable: %w", err)
	}
	token, expires, err := s.exchange(ctx, strings.TrimSpace(credential))
	if err != nil {
		return "", fmt.Errorf("agent-manager service credential exchange: %w", err)
	}
	if strings.TrimSpace(token) == "" || !s.now().Before(expires) {
		return "", errors.New("agent-manager service credential exchange returned no usable token")
	}
	s.token, s.expires = token, expires
	return token, nil
}

// serviceBearerTransport presents Agent Manager's service principal to Web
// Console. When no token can be obtained the request proceeds anonymously:
// read/drive calls keep their prior behaviour, while Web Console withholds a
// stable-create owner so the client still refuses reserved Create.
type serviceBearerTransport struct {
	base    http.RoundTripper
	source  *ServiceTokenSource
	logMu   sync.Mutex
	lastLog time.Time
}

func (t *serviceBearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	// Agent Manager speaks as itself, never as a forwarded run or caller.
	out.Header.Del("Authorization")
	out.Header.Del("X-Agent-Identity-Token")
	out.Header.Del("Cookie")
	token, err := t.source.Token(req.Context())
	if err != nil {
		t.logMu.Lock()
		if time.Since(t.lastLog) > time.Minute {
			t.lastLog = time.Now()
			log.Printf("interactive-runner: web-console calls are unauthenticated: %v", err)
		}
		t.logMu.Unlock()
	} else {
		out.Header.Set("Authorization", "Bearer "+token)
	}
	return t.base.RoundTrip(out)
}

// NewServicePrincipalHTTPClient returns the HTTP client Agent Manager uses for
// Web Console. Its transport is independent of the process default transport,
// which may forward an inbound agent identity.
func NewServicePrincipalHTTPClient(source *ServiceTokenSource) *http.Client {
	base := &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        32,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: &serviceBearerTransport{base: base, source: source}}
}

// AuthorityServiceCredential reads the credential from the credential
// authority. It never logs or returns the value except to the exchanger.
func AuthorityServiceCredential() (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(ServiceCredentialIdentity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, ServiceCredentialField)
}

// AuthenticatorServiceExchanger exchanges the credential at the discovered
// scenario-authenticator API.
func AuthenticatorServiceExchanger(httpClient connect.HTTPClient) ServiceTokenExchanger {
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
