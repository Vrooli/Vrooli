package hubsender

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/servicetoken"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
)

// Notice is one notice a scenario sends the operator through Notification Hub.
type Notice struct {
	Title string
	Body  string
	// Urgency is low, normal, high or critical.
	Urgency string
	// SensitivityLabel is public, private, sensitive or critical.
	SensitivityLabel string
	// IdempotencyKey makes a repeated send of the same notice a no-op at the Hub.
	// Keys are scoped to the sending scenario. Re-sending a key with different
	// terms is refused, so a key must name exactly one title, body and urgency.
	IdempotencyKey string
	// ContextURL is where a tap on the notice goes: a Hub-relative path
	// ("/asks/<id>") or an http(s) URL. The Hub refuses any other form.
	ContextURL string
}

// ErrSendFailed means the Hub did not accept the notice, or could not be reached.
// ErrNotIssued (see servicetoken) means the scenario has no sender credential yet.
var ErrSendFailed = errors.New("the Hub did not accept the notice")

// BaseURL resolves where Notification Hub's API is.
type BaseURL func(context.Context) (string, error)

// Client sends notices to Notification Hub as the source's sender principal.
type Client struct {
	tokens  *servicetoken.Tokens
	baseURL BaseURL
	http    *http.Client
}

// ClientOption customizes a Client.
type ClientOption func(*clientSettings)

type clientSettings struct {
	baseURL      BaseURL
	http         *http.Client
	tokenOptions []servicetoken.Option
}

// WithBaseURL replaces discovery of the Hub, for a fixed address.
func WithBaseURL(base BaseURL) ClientOption { return func(s *clientSettings) { s.baseURL = base } }

// WithHTTPClient replaces the HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(s *clientSettings) { s.http = client }
}

// WithTokenOptions customizes the client's token client, for example to replace
// the exchange at scenario-authenticator.
func WithTokenOptions(options ...servicetoken.Option) ClientOption {
	return func(s *clientSettings) { s.tokenOptions = append(s.tokenOptions, options...) }
}

// NewClient builds the sender for the scenario acting as source, with the
// credential the operator issues for its "<source>-notification-hub" principal.
// The credential is read when a notice is sent and a missing one is not
// remembered, so a scenario that started before the operator issued it sends its
// next notice with no restart.
func NewClient(source string, credential servicetoken.CredentialSource, options ...ClientOption) *Client {
	settings := clientSettings{
		baseURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
		},
		http: &http.Client{Timeout: 10 * time.Second},
	}
	for _, option := range options {
		option(&settings)
	}
	return &Client{tokens: NewTokens(source, credential, settings.tokenOptions...), baseURL: settings.baseURL, http: settings.http}
}

// Send sends one notice. The Hub attributes it to the client's own source.
func (c *Client) Send(ctx context.Context, notice Notice) error {
	if c == nil || c.tokens == nil {
		return fmt.Errorf("%w: no sender configured", ErrSendFailed)
	}
	// The notice is the scenario's own, under its sender token; the identity of
	// an agent run whose request led here must not be forwarded with it.
	ctx = provenance.WithoutCallerIdentity(ctx)
	authorization, err := c.tokens.Authorization(ctx)
	if err != nil {
		if errors.Is(err, servicetoken.ErrNotIssued) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrSendFailed, err)
	}
	base, err := c.baseURL(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return fmt.Errorf("%w: find notification-hub: %v", ErrSendFailed, err)
	}
	client := notificationsconnect.NewNotificationsServiceClient(c.http, strings.TrimRight(base, "/"))
	request := connect.NewRequest(&notificationsv1.SendRequest{
		Title: notice.Title, Body: notice.Body, Urgency: notice.Urgency,
		SensitivityLabel: notice.SensitivityLabel, IdempotencyKey: notice.IdempotencyKey, ContextUrl: notice.ContextURL,
	})
	request.Header().Set("Authorization", authorization)
	if _, err := client.Send(ctx, request); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	return nil
}
