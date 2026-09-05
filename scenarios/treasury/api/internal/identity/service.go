// Package identity verifies agent-manager claims and fails closed.
package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"treasury/internal/httpc"
)

const HeaderAgentIdentityToken = "X-Agent-Identity-Token"

var ErrUnverifiable = errors.New("agent identity unverifiable")

type Claims struct {
	RunID      string            `json:"run_id"`
	TaskID     string            `json:"task_id"`
	Subject    string            `json:"subject"`
	Scopes     []string          `json:"scopes"`
	ProfileKey string            `json:"profile_key"`
	ScopePath  string            `json:"scope_path"`
	IssuedAt   int64             `json:"iat"`
	ExpiresAt  int64             `json:"exp"`
	Meta       map[string]string `json:"meta"`
}

type Verifier interface {
	Verify(context.Context, string) (Claims, error)
}

type UnavailableVerifier struct{ Cause error }

func (v UnavailableVerifier) Verify(context.Context, string) (Claims, error) {
	if v.Cause == nil {
		return Claims{}, fmt.Errorf("%w: authority is not configured", ErrUnverifiable)
	}
	return Claims{}, fmt.Errorf("%w: %v", ErrUnverifiable, v.Cause)
}

type HTTPVerifier struct {
	baseURL string
	doer    httpc.Doer
}

func NewHTTPVerifier(baseURL string, doer httpc.Doer) (*HTTPVerifier, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: agent-manager base URL must be an absolute HTTP(S) origin", ErrUnverifiable)
	}
	if doer == nil {
		return nil, fmt.Errorf("%w: HTTP client is required", ErrUnverifiable)
	}
	return &HTTPVerifier{baseURL: baseURL, doer: doer}, nil
}

func (v *HTTPVerifier) Verify(ctx context.Context, token string) (Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Claims{}, fmt.Errorf("%w: identity token is absent", ErrUnverifiable)
	}
	payload, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return Claims{}, fmt.Errorf("%w: encode request: %v", ErrUnverifiable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/api/v1/identity/verify", bytes.NewReader(payload))
	if err != nil {
		return Claims{}, fmt.Errorf("%w: build request: %v", ErrUnverifiable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.doer.Do(req)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: authority unavailable: %v", ErrUnverifiable, err)
	}
	if resp == nil || resp.Body == nil {
		return Claims{}, fmt.Errorf("%w: authority returned no response", ErrUnverifiable)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Claims{}, fmt.Errorf("%w: read response: %v", ErrUnverifiable, err)
	}
	var result struct {
		Valid  bool    `json:"valid"`
		Claims *Claims `json:"claims"`
		Error  string  `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return Claims{}, fmt.Errorf("%w: invalid authority response", ErrUnverifiable)
	}
	if resp.StatusCode != http.StatusOK || !result.Valid || result.Claims == nil {
		cause := strings.TrimSpace(result.Error)
		if cause == "" {
			cause = fmt.Sprintf("verification refused with HTTP %d", resp.StatusCode)
		}
		return Claims{}, fmt.Errorf("%w: %s", ErrUnverifiable, cause)
	}
	result.Claims.Subject = strings.TrimSpace(result.Claims.Subject)
	if result.Claims.Subject == "" {
		return Claims{}, fmt.Errorf("%w: verified claims omit subject", ErrUnverifiable)
	}
	return *result.Claims, nil
}

var _ Verifier = (*HTTPVerifier)(nil)
