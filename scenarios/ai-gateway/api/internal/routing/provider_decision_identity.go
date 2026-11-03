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
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
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

var errProviderDecisionIdentityMismatch = errors.New("scenario-authenticator returned an unexpected provider-decision identity")

type serviceCredentialReader func() (string, error)
type serviceCredentialExchanger func(context.Context, string) (ExchangedServiceIdentity, error)

// ExchangedServiceIdentity is the metadata returned by Scenario Authenticator.
// AccessToken is held only in memory and must never be logged or persisted.
type ExchangedServiceIdentity struct {
	AccessToken string
	ExpiresAt   time.Time
	Subject     string
	Audience    string
	Scopes      []string
}

// ProviderDecisionIdentity exchanges the Credential Authority value for a
// short-lived Hub-only service token. The credential is reread for every
// exchange so Authority rotation/revocation takes effect without a Gateway
// restart. Neither credential nor access token is included in errors.
type ProviderDecisionIdentity struct {
	credential serviceCredentialReader
	exchange   serviceCredentialExchanger
	now        func() time.Time
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

func NewProviderDecisionIdentity(credential serviceCredentialReader, exchange serviceCredentialExchanger, now func() time.Time) *ProviderDecisionIdentity {
	if now == nil {
		now = time.Now
	}
	return &ProviderDecisionIdentity{credential: credential, exchange: exchange, now: now}
}

// BearerToken returns a cached, verified service token or exchanges the
// current credential. Missing custody, exchange failures, and any mismatched
// subject/audience/capability fail closed.
func (s *ProviderDecisionIdentity) BearerToken(ctx context.Context) (string, error) {
	if s == nil || s.credential == nil || s.exchange == nil {
		return "", errors.New("provider-decision service identity is unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.now().Add(providerDecisionTokenRefreshMargin).Before(s.expiresAt) {
		return s.token, nil
	}
	s.token, s.expiresAt = "", time.Time{}
	credential, err := s.credential()
	if err != nil || strings.TrimSpace(credential) == "" {
		return "", errors.New("provider-decision service credential is unavailable")
	}
	identity, err := s.exchange(ctx, strings.TrimSpace(credential))
	if err != nil {
		return "", errors.New("provider-decision service credential exchange failed")
	}
	if strings.TrimSpace(identity.AccessToken) == "" || !s.now().Add(providerDecisionTokenRefreshMargin).Before(identity.ExpiresAt) ||
		identity.Subject != providerDecisionPrincipal || identity.Audience != providerDecisionAudience || !exactProviderDecisionScope(identity.Scopes) {
		return "", errProviderDecisionIdentityMismatch
	}
	s.token, s.expiresAt = identity.AccessToken, identity.ExpiresAt
	return s.token, nil
}

func exactProviderDecisionScope(scopes []string) bool {
	return len(scopes) == 1 && scopes[0] == providerDecisionScope
}

// authorityProviderDecisionCredential reads the service principal credential
// from the existing Credential Authority owner path.
func authorityProviderDecisionCredential() (string, error) {
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

// authenticatorProviderDecisionExchanger trades the stored credential through
// Scenario Authenticator's existing Connect exchange API.
func authenticatorProviderDecisionExchanger(httpClient connect.HTTPClient) serviceCredentialExchanger {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return func(ctx context.Context, credential string) (ExchangedServiceIdentity, error) {
		base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		if err != nil {
			return ExchangedServiceIdentity{}, err
		}
		client := accountsconnect.NewAccountsServiceClient(httpClient, strings.TrimRight(base, "/"))
		response, err := client.ExchangeServiceCredential(ctx, connect.NewRequest(&accountsv1.ExchangeServiceCredentialRequest{Credential: credential}))
		if err != nil {
			return ExchangedServiceIdentity{}, err
		}
		if response.Msg.GetAccessTokenExpiresAt() == nil {
			return ExchangedServiceIdentity{}, errors.New("exchange response has no expiry")
		}
		return ExchangedServiceIdentity{
			AccessToken: response.Msg.GetAccessToken(),
			ExpiresAt:   response.Msg.GetAccessTokenExpiresAt().AsTime(),
			Subject:     response.Msg.GetSubject(), Audience: response.Msg.GetAudience(),
			Scopes: append([]string(nil), response.Msg.GetScopes()...),
		}, nil
	}
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
	identity   *ProviderDecisionIdentity
	httpClient *http.Client
}

func NewProviderDecisionHubClient(identity *ProviderDecisionIdentity, httpClient *http.Client) *ProviderDecisionHubClient {
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
	if c == nil || c.identity == nil || c.baseURL == nil || c.httpClient == nil {
		return errors.New("provider-decision Hub client is unavailable")
	}
	token, err := c.identity.BearerToken(ctx)
	if err != nil {
		return err
	}
	base, err := c.baseURL(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return errors.New("notification-hub endpoint is unavailable")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return errors.New("provider-decision request could not be encoded")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+path, bytes.NewReader(body))
	if err != nil {
		return errors.New("provider-decision request could not be created")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	// Service credentials must never follow a redirect, even when callers
	// inject an HTTP client with a permissive redirect policy. Clone the client
	// so the security rule applies without mutating a shared caller-owned client.
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.New("notification-hub provider-decision request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification-hub provider-decision request returned HTTP %d", resp.StatusCode)
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("notification-hub provider-decision response was invalid")
	}
	return nil
}

func NewDefaultProviderDecisionHubClient() *ProviderDecisionHubClient {
	identity := NewProviderDecisionIdentity(authorityProviderDecisionCredential, authenticatorProviderDecisionExchanger(nil), nil)
	return NewProviderDecisionHubClient(identity, nil)
}
