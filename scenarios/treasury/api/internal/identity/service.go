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

const (
	HeaderAgentIdentityToken  = "X-Agent-Identity-Token"
	HeaderAgentOperationToken = "X-Agent-Operation-Token"
)

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

// PairVerifier revalidates the run identity and its distinct operation token
// as one Agent Manager-owned pair. A verified operation claim carries the
// canonical operation grant used by Treasury's target evaluator.
type PairVerifier interface {
	VerifyPair(context.Context, string, string) (Claims, error)
}

func VerifyRequest(ctx context.Context, verifier Verifier, headers http.Header) (Claims, error) {
	if verifier == nil {
		return Claims{}, fmt.Errorf("%w: verifier is unavailable", ErrUnverifiable)
	}
	baseline := strings.TrimSpace(headers.Get(HeaderAgentIdentityToken))
	if baseline == "" {
		parts := strings.Fields(strings.TrimSpace(headers.Get("Authorization")))
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			baseline = parts[1]
		}
	}
	operation := strings.TrimSpace(headers.Get(HeaderAgentOperationToken))
	if operation == "" {
		return verifier.Verify(ctx, baseline)
	}
	pairVerifier, ok := verifier.(PairVerifier)
	if !ok {
		return Claims{}, fmt.Errorf("%w: paired operation verification is unavailable", ErrUnverifiable)
	}
	return pairVerifier.VerifyPair(ctx, baseline, operation)
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

func (v *HTTPVerifier) VerifyPair(ctx context.Context, baselineToken, operationToken string) (Claims, error) {
	baselineToken = strings.TrimSpace(baselineToken)
	operationToken = strings.TrimSpace(operationToken)
	if baselineToken == "" || operationToken == "" || baselineToken == operationToken {
		return Claims{}, fmt.Errorf("%w: identity token pair is incomplete", ErrUnverifiable)
	}
	payload, err := json.Marshal(map[string]string{"token": baselineToken, "operation_token": operationToken})
	if err != nil {
		return Claims{}, fmt.Errorf("%w: encode identity pair request: %v", ErrUnverifiable, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/api/v1/identity/verify", bytes.NewReader(payload))
	if err != nil {
		return Claims{}, fmt.Errorf("%w: build identity pair request: %v", ErrUnverifiable, err)
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
		return Claims{}, fmt.Errorf("%w: read identity pair response: %v", ErrUnverifiable, err)
	}
	var result struct {
		Valid     bool `json:"valid"`
		Operation *struct {
			Valid  bool    `json:"valid"`
			Claims *Claims `json:"claims"`
		} `json:"operation"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return Claims{}, fmt.Errorf("%w: invalid identity pair response", ErrUnverifiable)
	}
	if resp.StatusCode != http.StatusOK || !result.Valid || result.Operation == nil || !result.Operation.Valid || result.Operation.Claims == nil {
		cause := strings.TrimSpace(result.Error)
		if cause == "" {
			cause = "identity pair verification refused"
		}
		return Claims{}, fmt.Errorf("%w: %s", ErrUnverifiable, cause)
	}
	claims := *result.Operation.Claims
	claims.Subject = strings.TrimSpace(claims.Subject)
	if claims.Subject == "" || claims.RunID == "" || claims.ExpiresAt <= 0 {
		return Claims{}, fmt.Errorf("%w: operation claims omit run binding", ErrUnverifiable)
	}
	return claims, nil
}

var _ Verifier = (*HTTPVerifier)(nil)
