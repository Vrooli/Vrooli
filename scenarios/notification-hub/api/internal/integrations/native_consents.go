package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"notification-hub/internal/hub"
)

type consentURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// AgentManagerConsentReader resolves the lifecycle's local API, never a URL
// from a notification. Redirects are refused and no owner credential is sent.
type AgentManagerConsentReader struct {
	Resolver consentURLResolver
	Client   *http.Client
}

func (r AgentManagerConsentReader) ReadConsent(ctx context.Context, id string) (hub.NativeConsentStatus, error) {
	var out hub.NativeConsentStatus
	if r.Resolver == nil {
		return out, errors.New("consent source unavailable")
	}
	base, err := r.Resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return out, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return out, err
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "http" || (u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return out, errors.New("consent source must resolve to local API")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/authorizations/" + url.PathEscape(id) + "/status"
	client := http.Client{Timeout: 2 * time.Second}
	if r.Client != nil {
		client = *r.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return out, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("consent source returned %d", resp.StatusCode)
	}
	var wire struct {
		AuthorizationID string `json:"authorization_id"`
		State           string `json:"state"`
		ReviewExpiresAt string `json:"review_expires_at"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&wire); err != nil {
		return out, err
	}
	return hub.NativeConsentStatus{AuthorizationID: wire.AuthorizationID, State: wire.State, ReviewExpiresAt: wire.ReviewExpiresAt}, nil
}
