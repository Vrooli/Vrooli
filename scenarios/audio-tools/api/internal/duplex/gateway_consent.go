package duplex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GatewayConsentVerifier uses AI Gateway's owner-authenticated consent authority.
// It intentionally has no local cache so revocation takes effect on the next
// admission check and on periodic checks for active GPT-Live sessions.
type GatewayConsentVerifier struct {
	endpoint   string
	resolveURL func(context.Context) (string, error)
	client     *http.Client
}

func NewGatewayConsentVerifier(baseURL string, client *http.Client) (*GatewayConsentVerifier, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, errors.New("duplex: AI Gateway URL is invalid")
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &GatewayConsentVerifier{endpoint: strings.TrimRight(u.String(), "/") + "/api/v1/ai/privacy-consent/verify", client: consentHTTPClient(client)}, nil
}

// NewGatewayConsentVerifierWithResolver resolves AI Gateway when consent is
// checked, so service discovery remains current across scenario restarts.
func NewGatewayConsentVerifierWithResolver(resolve func(context.Context) (string, error), client *http.Client) (*GatewayConsentVerifier, error) {
	if resolve == nil {
		return nil, errors.New("duplex: AI Gateway resolver is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &GatewayConsentVerifier{resolveURL: resolve, client: consentHTTPClient(client)}, nil
}

// consentHTTPClient copies the caller's transport settings while preventing a
// redirect from forwarding the receipt body to another endpoint. A consent
// receipt is itself a bearer authorization artifact, so the verifier must
// treat redirects as an invalid authority response.
func consentHTTPClient(client *http.Client) *http.Client {
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &copy
}

func (v *GatewayConsentVerifier) Verify(ctx context.Context, _ string, receipt, source string) error {
	token, ok := humanBearer(ctx)
	if v == nil || !ok || strings.TrimSpace(receipt) == "" || strings.TrimSpace(source) == "" {
		return ErrConsentInvalid
	}
	body, err := json.Marshal(map[string]string{"receipt": receipt, "source": source})
	if err != nil {
		return ErrConsentInvalid
	}
	endpoint := v.endpoint
	if v.resolveURL != nil {
		base, err := v.resolveURL(ctx)
		if err != nil {
			return ErrConsentInvalid
		}
		u, err := url.Parse(strings.TrimSpace(base))
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return ErrConsentInvalid
		}
		endpoint = strings.TrimRight(u.String(), "/") + "/api/v1/ai/privacy-consent/verify"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ErrConsentInvalid
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return ErrConsentInvalid
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ErrConsentInvalid
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1024)).Decode(&result) != nil || !result.Valid {
		return ErrConsentInvalid
	}
	return nil
}
