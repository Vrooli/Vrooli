package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	providerDecisionPrincipal          = "ai-gateway"
	providerDecisionResource           = "notification-hub-provider-decision"
	providerDecisionAudience           = "scenario-authenticator:notification-hub-provider-decision"
	providerDecisionScope              = "notification-hub:provider-decision"
	providerDecisionCredentialIdentity = "scenario-authenticator/service-principal/ai-gateway/notification-hub-provider-decision"
	providerDecisionCredentialField    = "credential"
	providerDecisionPublishPath        = "/api/v1/internal/provider-decisions/publish"
	providerDecisionLookupPath         = "/api/v1/internal/provider-decisions/lookup"
)

const providerDecisionTokenRefreshMargin = 30 * time.Second

// errProviderDecisionIdentityMismatch is what a token that is not exactly the
// provider-decision identity (or is expired or about to be) reports.
var errProviderDecisionIdentityMismatch = errors.New("scenario-authenticator returned an unexpected provider-decision identity")

// NewProviderDecisionIdentity exchanges the Credential Authority value for a
// short-lived Hub-only service token. The credential is reread for every
// exchange so Authority rotation/revocation takes effect without a Gateway
// restart. Missing custody, exchange failures, and any mismatched
// subject/audience/capability fail closed, and neither credential nor access
// token is included in errors.
func NewProviderDecisionIdentity(credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Credential:    credential,
		Expect:        servicetoken.ExpectIdentity(providerDecisionPrincipal, providerDecisionAudience, providerDecisionScope),
		RefreshMargin: providerDecisionTokenRefreshMargin,
		MinRemaining:  providerDecisionTokenRefreshMargin,
		Describe:      describeProviderDecisionFailure,
	}, options...)
}

func describeProviderDecisionFailure(kind, _ error) error {
	switch kind {
	case servicetoken.ErrNotIssued:
		return servicetoken.Kind(kind, errors.New("provider-decision service credential is unavailable"))
	case servicetoken.ErrExchangeFailed:
		return servicetoken.Kind(kind, errors.New("provider-decision service credential exchange failed"))
	case servicetoken.ErrNoUsableToken, servicetoken.ErrIdentityMismatch:
		return servicetoken.Kind(kind, errProviderDecisionIdentityMismatch)
	}
	return servicetoken.Kind(kind, errors.New("provider-decision service identity is unavailable"))
}

// authorityProviderDecisionCredential reads the service principal credential
// from the existing Credential Authority owner path.
func authorityProviderDecisionCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(providerDecisionCredentialIdentity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, providerDecisionCredentialField)
}

// ProviderDecisionLookup is the exact immutable ask information returned by
// Notification Hub's read-only provider-decision route.
type ProviderDecisionLookup struct {
	AskID                    string `json:"ask_id"`
	Source                   string `json:"source"`
	EventType                string `json:"event_type"`
	Requester                string `json:"requester"`
	EventID                  string `json:"event_id"`
	Digest                   string `json:"canonical_terms_sha256"`
	Scope                    string `json:"scope"`
	State                    string `json:"state"`
	Answer                   string `json:"answer,omitempty"`
	AnsweredByIdentityKind   string `json:"answered_by_identity_kind"`
	AnsweredByIdentitySource string `json:"answered_by_identity_source,omitempty"`
	CreatedAt                string `json:"created_at"`
	AnsweredAt               string `json:"answered_at"`
	Deadline                 string `json:"deadline"`
}

// ProviderDecisionHubClient applies the verified service token to the two
// fixed Hub routes. It never copies caller-supplied Authorization headers.
type ProviderDecisionHubClient struct {
	baseURL    func(context.Context) (string, error)
	identity   *servicetoken.Tokens
	httpClient *http.Client
}

func NewProviderDecisionHubClient(identity *servicetoken.Tokens, httpClient *http.Client) *ProviderDecisionHubClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &ProviderDecisionHubClient{
		baseURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
		},
		identity: identity, httpClient: httpClient,
	}
}

func (c *ProviderDecisionHubClient) Lookup(ctx context.Context, requesterRef, digest, scope string) (ProviderDecisionLookup, error) {
	var result ProviderDecisionLookup
	err := c.post(ctx, providerDecisionLookupPath, map[string]string{
		"requester_ref": requesterRef, "canonical_terms_sha256": digest, "scope": scope,
	}, &result)
	return result, err
}

func (c *ProviderDecisionHubClient) Publish(ctx context.Context, eventID string, payload json.RawMessage) error {
	return c.post(ctx, providerDecisionPublishPath, struct {
		EventID string          `json:"event_id"`
		Payload json.RawMessage `json:"payload"`
	}{EventID: eventID, Payload: payload}, nil)
}

func (c *ProviderDecisionHubClient) post(ctx context.Context, path string, input, output any) error {
	req, err := c.request(ctx, path, input)
	if err != nil {
		return err
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification-hub provider-decision request returned HTTP %d", resp.StatusCode)
	}
	return decodeProviderDecisionResponse(resp.Body, output)
}

func (c *ProviderDecisionHubClient) request(ctx context.Context, path string, input any) (*http.Request, error) {
	if c == nil || c.identity == nil || c.baseURL == nil || c.httpClient == nil {
		return nil, errors.New("provider-decision Hub client is unavailable")
	}
	token, err := c.identity.Token(ctx)
	if err != nil {
		return nil, err
	}
	base, err := c.baseURL(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, errors.New("notification-hub endpoint is unavailable")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, errors.New("provider-decision request could not be encoded")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("provider-decision request could not be created")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req, nil
}

func (c *ProviderDecisionHubClient) send(req *http.Request) (*http.Response, error) {
	// Service credentials must never follow a redirect, even when callers
	// inject an HTTP client with a permissive redirect policy. Clone the client
	// so the security rule applies without mutating a shared caller-owned client.
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.New("notification-hub provider-decision request failed")
	}
	return resp, nil
}

func decodeProviderDecisionResponse(body io.Reader, output any) error {
	if output == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(body, 64<<10))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("notification-hub provider-decision response was invalid")
	}
	return nil
}

func NewDefaultProviderDecisionHubClient() *ProviderDecisionHubClient {
	identity := NewProviderDecisionIdentity(authorityProviderDecisionCredential)
	return NewProviderDecisionHubClient(identity, nil)
}
