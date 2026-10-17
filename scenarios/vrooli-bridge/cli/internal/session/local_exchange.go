package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/authn"
)

const (
	bridgeTokenFileEnv     = "VROOLI_" + "BRIDGE_" + "TOKEN_FILE"
	authTokenFileEnv       = "VROOLI_" + "AUTH_" + "TOKEN_FILE"
	breakGlassTokenFileEnv = "VROOLI_" + "BREAK_GLASS_" + "TOKEN_FILE"
)

// ExchangeLocal asks the authenticator's local-only listener to exchange the
// current process' kernel peer credential for a normal short-lived owner
// session. The listener is deliberately a separate Unix socket: TCP callers
// cannot manufacture a local peer principal.
func ExchangeLocal(ctx context.Context) (token, refresh string, err error) {
	resp, err := authn.ExchangeLocalMachinePrincipal(ctx)
	if err != nil {
		return "", "", err
	}
	return resp.Tokens.AccessToken, resp.Tokens.RefreshToken, nil
}

// TokenFile reads the platform-agnostic owner-token fallback. The file must
// be owner-only; unlike a peer credential, a token file can be copied to
// another machine and replayed, so this path is intentionally explicit.
func TokenFile() (string, error) {
	path := strings.TrimSpace(os.Getenv(bridgeTokenFileEnv))
	if path == "" {
		path = strings.TrimSpace(os.Getenv(authTokenFileEnv))
	}
	if path == "" {
		return "", nil
	}
	return readTokenFile(path)
}

// BreakGlassTokenFile reads the explicit offline credential file. It remains
// separate from TokenFile because the HTTP authorization scheme is different.
func BreakGlassTokenFile() (string, error) {
	path := strings.TrimSpace(os.Getenv(breakGlassTokenFileEnv))
	if path == "" {
		return "", nil
	}
	return readTokenFile(path)
}

func readTokenFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat token file: %w", err)
	}
	if info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("token file must be owner-only: %s", filepath.Base(path))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read token file: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

func defaultAuthSocket() string {
	return authn.DefaultLocalAuthenticatorSocket()
}
