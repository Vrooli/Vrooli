package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrAgentIdentityRejected    = errors.New("agent identity was rejected")
	ErrAgentIdentityUnavailable = errors.New("agent identity verifier is unavailable")
)

// AttachedIdentity is the minimum verified identity needed by a System
// Monitor owner action. Session metadata comes from the signed Agent Manager
// claims and is accepted only after Agent Manager verifies its session binding.
type AttachedIdentity struct {
	RunID               string
	HarnessKind         string
	HarnessSession      string
	WebConsoleSessionID string
}

// VerifyAttachedIdentity verifies an Agent Manager run token and asks the
// authority to check the persisted Web Console session binding. The token is
// sent only in the request body to the local Agent Manager API.
func (c *Client) VerifyAttachedIdentity(ctx context.Context, token string) (*AttachedIdentity, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 16*1024 || strings.ContainsAny(token, " \t\r\n") {
		return nil, ErrAgentIdentityRejected
	}
	body, err := json.Marshal(struct {
		Token                string `json:"token"`
		VerifySessionBinding bool   `json:"verify_session_binding"`
	}{Token: token, VerifySessionBinding: true})
	if err != nil {
		return nil, fmt.Errorf("encode agent identity request: %w", err)
	}
	resp, err := c.verifyIdentityRequest(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentIdentityUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrAgentIdentityRejected
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: Agent Manager returned HTTP %d", ErrAgentIdentityUnavailable, resp.StatusCode)
	}
	const maxResponseBytes = 64 * 1024
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read verifier response: %v", ErrAgentIdentityUnavailable, err)
	}
	if len(raw) > maxResponseBytes {
		return nil, fmt.Errorf("%w: verifier response exceeded %d bytes", ErrAgentIdentityUnavailable, maxResponseBytes)
	}
	var result struct {
		Valid  bool `json:"valid"`
		Claims *struct {
			RunID string            `json:"run_id"`
			Meta  map[string]string `json:"meta"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("%w: decode verifier response: %v", ErrAgentIdentityUnavailable, err)
	}
	if !result.Valid || result.Claims == nil {
		return nil, ErrAgentIdentityRejected
	}
	identity := &AttachedIdentity{
		RunID:               strings.TrimSpace(result.Claims.RunID),
		HarnessKind:         strings.TrimSpace(result.Claims.Meta["harness_kind"]),
		HarnessSession:      strings.TrimSpace(result.Claims.Meta["harness_session"]),
		WebConsoleSessionID: strings.TrimSpace(result.Claims.Meta["verified_web_console_session_id"]),
	}
	if result.Claims.Meta["execution_mode"] != "attached" || identity.RunID == "" || identity.HarnessKind == "" || identity.HarnessSession == "" || identity.WebConsoleSessionID == "" {
		return nil, ErrAgentIdentityRejected
	}
	return identity, nil
}

// verifyIdentityRequest keeps the owner-specific identity verifier on REST:
// its request/response is not part of AgentManagerService. Redirects are
// rejected so the body token cannot be forwarded to another origin.
func (c *Client) verifyIdentityRequest(ctx context.Context, body []byte) (*http.Response, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/identity/verify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create identity verification request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := *c.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client.Do(request)
}
