package webconsole

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
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

// serviceRefreshMargin renews a service token this long before it expires.
const serviceRefreshMargin = 2 * time.Minute

// NewServiceTokens is Agent Manager's web-console service-principal client. The
// credential is re-read on every exchange, so rotation and revocation take
// effect without restarting Agent Manager. It presents whatever token the
// authenticator returns: Web Console verifies the token itself, and a token it
// does not verify only withholds a stable-create owner.
func NewServiceTokens(credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Credential:    credential,
		RefreshMargin: serviceRefreshMargin,
		Describe:      servicetoken.DescribeWithCause("agent-manager"),
	}, options...)
}

// serviceBearerTransport presents Agent Manager's service principal to Web
// Console. When no token can be obtained the request proceeds anonymously:
// read/drive calls keep their prior behaviour, while Web Console withholds a
// stable-create owner so the client still refuses reserved Create.
type serviceBearerTransport struct {
	base    http.RoundTripper
	source  *servicetoken.Tokens
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
func NewServicePrincipalHTTPClient(source *servicetoken.Tokens) *http.Client {
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
func AuthorityServiceCredential(context.Context) (string, error) {
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
