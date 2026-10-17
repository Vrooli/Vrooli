package cliutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

// ResolveAgentCredential consults the live authority for every request. A
// missing broker is compatible with an inherited credential; a responding
// broker's refusal is authoritative and never falls back to older authority.
func ResolveAgentCredential(ctx context.Context) (string, error) {
	return resolveAgentCredential(ctx, AgentCredentialSocket(), os.Getenv(EnvIdentityToken))
}

func resolveAgentCredential(ctx context.Context, socket, inherited string) (string, error) {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local-agent-manager/credential", nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Transport: transport, Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if _, directoryErr := os.Stat(filepath.Dir(socket)); errors.Is(directoryErr, os.ErrNotExist) {
				return inherited, nil
			}
		}

		return "", errors.New("agent credential authority unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return inherited, nil
	}
	if resp.StatusCode != http.StatusOK {
		var refusal struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&refusal)
		if refusal.Code == "" {
			refusal.Code = fmt.Sprintf("refused (HTTP %d)", resp.StatusCode)
		}
		return "", &AgentAuthorizationError{Code: refusal.Code}
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256*1024)).Decode(&body); err != nil || body.Token == "" {
		return "", errors.New("invalid agent credential authority response")
	}
	return body.Token, nil
}
