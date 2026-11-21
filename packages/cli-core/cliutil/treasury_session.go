package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// ResolveTreasurySessionKey asks the local Agent Manager authority for the
// current run-owned Treasury key. A non-empty rotationRequest atomically
// rotates the active key and makes a same-request retry return that key.
// The value travels only over the permission-restricted Unix socket.
func ResolveTreasurySessionKey(ctx context.Context, inherited, rotationRequest string) (string, error) {
	return resolveTreasurySessionKeyAt(ctx, AgentCredentialSocket(), inherited, rotationRequest)
}

func resolveTreasurySessionKeyAt(ctx context.Context, socket, inherited, rotationRequest string) (string, error) {
	inherited = strings.TrimSpace(inherited)
	if inherited == "" {
		return "", errors.New("ordinary Agent Manager run identity is required for Treasury session key access")
	}
	query := url.Values{"purpose": []string{"treasury_session_key"}}
	if rotationRequest != "" {
		query.Set("rotation_request", strings.TrimSpace(rotationRequest))
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local-agent-manager/credential?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set(HeaderAgentIdentityToken, inherited)
	response, err := (&http.Client{Transport: transport, Timeout: 5e9}).Do(request)
	if err != nil {
		return "", fmt.Errorf("resolve run-owned Treasury session key from Agent Manager: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		var refusal struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(response.Body).Decode(&refusal)
		if refusal.Code == "" {
			refusal.Code = fmt.Sprintf("refused (HTTP %d)", response.StatusCode)
		}
		return "", fmt.Errorf("Agent Manager refused Treasury session key: %s", refusal.Code)
	}
	var body struct {
		Key string `json:"treasury_session_private_key"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || len(body.Key) != 64 {
		return "", ErrInvalidAuthorityResponse
	}
	return strings.ToLower(body.Key), nil
}
