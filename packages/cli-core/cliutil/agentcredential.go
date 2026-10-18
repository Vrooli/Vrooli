package cliutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AgentCredentialSocket is shared by the local authority and CLI clients.
// The directory is private to the host account; peer credentials, rather than
// harness environment variables or a caller-supplied PID, bind each request.
func AgentCredentialSocket() string {
	return filepath.Join(os.TempDir(), "vrooli-agent-manager-"+strconv.Itoa(os.Getuid()), "identity.sock")
}

// AgentAuthorizationError is the local authority's refusal to issue this
// session's credential. The API may be healthy; retrying or starting the
// scenario cannot help, so callers report it as an authorization problem.
type AgentAuthorizationError struct{ Code string }

func (e *AgentAuthorizationError) Error() string {
	return "agent authorization " + e.Code + "; inspect with vrooli agent authorization <session-or-run>"
}

// ResolveAgentCredential consults the live authority for every request. An
// unavailable broker preserves a structurally ordinary inherited run token;
// grant-only credentials remain unavailable until the broker can perform its
// current-grant selection. A responding broker's refusal never falls back.
func ResolveAgentCredential(ctx context.Context) (string, error) {
	credentials, err := ResolveAgentCredentials(ctx)
	if err != nil {
		return "", err
	}
	if credentials.BaselineToken != "" {
		return credentials.BaselineToken, nil
	}
	return credentials.LegacyToken, nil
}

// AgentCredentials keeps ordinary run identity separate from a currently
// approved operation grant. Legacy grant-only callers use LegacyToken and
// remain bounded by the existing identity-token policy.
type AgentCredentials struct {
	BaselineToken    string
	OperationToken   string
	LegacyToken      string
	OperationRefusal string
}

// ResolveAgentCredentials consults the live authority for each request. An
// unavailable broker preserves only a structurally ordinary inherited run
// token; a responding broker is authoritative and never falls back to an
// older grant.
func ResolveAgentCredentials(ctx context.Context) (AgentCredentials, error) {
	return resolveAgentCredentials(ctx, AgentCredentialSocket(), os.Getenv(EnvIdentityToken))
}

func resolveAgentCredentials(ctx context.Context, socket, inherited string) (AgentCredentials, error) {
	inherited = strings.TrimSpace(inherited)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local-agent-manager/credential", nil)
	if err != nil {
		return AgentCredentials{}, err
	}
	if inherited != "" {
		req.Header.Set(HeaderAgentIdentityToken, inherited)
	}
	resp, err := (&http.Client{Transport: transport, Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		if _, directoryErr := os.Stat(filepath.Dir(socket)); errors.Is(directoryErr, os.ErrNotExist) {
			return inheritedBaselineOnUnavailableBroker(inherited)
		}
		return inheritedBaselineOnUnavailableBroker(inherited)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return AgentCredentials{BaselineToken: inherited}, nil
	}
	if resp.StatusCode != http.StatusOK {
		var refusal struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&refusal)
		if refusal.Code == "" {
			refusal.Code = fmt.Sprintf("refused (HTTP %d)", resp.StatusCode)
		}
		return AgentCredentials{}, &AgentAuthorizationError{Code: refusal.Code}
	}
	var body struct {
		Token            string `json:"token"`
		BaselineToken    string `json:"baseline_token"`
		OperationToken   string `json:"operation_token"`
		OperationRefusal string `json:"operation_refusal"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256*1024)).Decode(&body); err != nil || body.Token == "" {
		if err != nil {
			return AgentCredentials{}, errors.New("invalid agent credential authority response")
		}
	}
	credentials := AgentCredentials{
		BaselineToken: strings.TrimSpace(body.BaselineToken), OperationToken: strings.TrimSpace(body.OperationToken),
		LegacyToken: strings.TrimSpace(body.Token), OperationRefusal: strings.TrimSpace(body.OperationRefusal),
	}
	if (credentials.BaselineToken == "") == (credentials.LegacyToken == "") ||
		(credentials.LegacyToken != "" && credentials.OperationToken != "") ||
		(credentials.BaselineToken != "" && inherited == "") ||
		(credentials.BaselineToken != "" && credentials.BaselineToken != inherited) {
		return AgentCredentials{}, errors.New("invalid agent credential authority response")
	}
	return credentials, nil
}

// inheritedBaselineOnUnavailableBroker preserves only a well-formed ordinary
// run token when the local authority cannot be reached. This is a restrictive
// role classifier, not token verification: the receiving API still verifies
// the signature, expiry and retained run. A grant token must not bypass the
// broker's current-grant selection or revocation checks.
func inheritedBaselineOnUnavailableBroker(inherited string) (AgentCredentials, error) {
	if inherited == "" {
		return AgentCredentials{}, nil
	}
	parts := strings.Split(inherited, ".")
	if len(parts) != 2 {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != 32 {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	var claims struct {
		Purpose   string            `json:"purpose,omitempty"`
		RunID     string            `json:"run_id"`
		IssuedAt  int64             `json:"iat"`
		ExpiresAt int64             `json:"exp"`
		Meta      map[string]string `json:"meta"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Purpose != "" || strings.TrimSpace(claims.RunID) == "" || claims.IssuedAt <= 0 || claims.ExpiresAt <= 0 || claims.Meta["authorization_id"] != "" {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	return AgentCredentials{BaselineToken: inherited, OperationRefusal: "authority_unavailable"}, nil
}

// resolveAgentCredential preserves the legacy single-token test/caller seam.
func resolveAgentCredential(ctx context.Context, socket, inherited string) (string, error) {
	credentials, err := resolveAgentCredentials(ctx, socket, inherited)
	if err != nil {
		return "", err
	}
	if credentials.BaselineToken != "" {
		return credentials.BaselineToken, nil
	}
	return credentials.LegacyToken, nil
}
