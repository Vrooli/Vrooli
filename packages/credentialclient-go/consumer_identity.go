package credentialclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxConsumerIdentityResponse = 32 << 10

// ResolveConsumerIdentity asks the LPBS authority to identify a short-lived
// consumer access token. The token is sent only to HTTPS authorities or a
// loopback development authority, and redirects are rejected to prevent
// forwarding it to a different endpoint.
func ResolveConsumerIdentity(ctx context.Context, baseURL, accessToken string, client *http.Client) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.TrimSpace(accessToken) == "" {
		return "", ErrSubscriptionUnauthorized
	}
	if err := validateSubscriptionAuthority(base); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/auth/me", nil)
	if err != nil {
		return "", fmt.Errorf("create consumer identity request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))

	if client == nil {
		client = &http.Client{Timeout: credentialHTTPTimeout}
	}
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &copy
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("consumer identity authority unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return "", ErrSubscriptionUnauthorized
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("consumer identity authority returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxConsumerIdentityResponse+1))
	if err != nil {
		return "", fmt.Errorf("read consumer identity response: %w", err)
	}
	if len(body) > maxConsumerIdentityResponse {
		return "", fmt.Errorf("consumer identity response is too large")
	}
	var payload struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode consumer identity response: %w", err)
	}
	identity := strings.ToLower(strings.TrimSpace(payload.User.Email))
	if identity == "" || len(identity) > 320 || strings.ContainsAny(identity, "\r\n\x00") {
		return "", fmt.Errorf("consumer identity authority returned no valid identity")
	}
	return identity, nil
}

func validateSubscriptionAuthority(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("consumer identity authority URL is invalid")
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		return nil
	}
	if !strings.EqualFold(parsed.Scheme, "http") {
		return fmt.Errorf("consumer identity authority must use HTTPS or loopback HTTP")
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("consumer identity authority must use HTTPS or loopback HTTP")
	}
	return nil
}
