package session

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const (
	authSocketEnv          = "VROOLI_AUTH_SOCKET"
	bridgeTokenFileEnv     = "VROOLI_BRIDGE_TOKEN_FILE"
	authTokenFileEnv       = "VROOLI_AUTH_TOKEN_FILE"
	breakGlassTokenFileEnv = "VROOLI_BREAK_GLASS_TOKEN_FILE"
)

// ExchangeLocal asks the authenticator's local-only listener to exchange the
// current process' kernel peer credential for a normal short-lived owner
// session. The listener is deliberately a separate Unix socket: TCP callers
// cannot manufacture a local peer principal.
func ExchangeLocal(ctx context.Context) (token, refresh string, err error) {
	socketPath := strings.TrimSpace(os.Getenv(authSocketEnv))
	if socketPath == "" {
		socketPath = defaultAuthSocket()
	}
	machineID, err := os.Hostname()
	if err != nil || strings.TrimSpace(machineID) == "" {
		return "", "", fmt.Errorf("resolve machine id: %w", err)
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	defer transport.CloseIdleConnections()
	client := accountsconnect.NewAccountsServiceClient(&http.Client{Transport: transport, Timeout: 5 * time.Second}, "http://local-authenticator")
	resp, err := client.ExchangeMachinePrincipal(ctx, connect.NewRequest(&accountsv1.ExchangeMachinePrincipalRequest{MachineId: machineID}))
	if err != nil {
		return "", "", err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Tokens == nil || strings.TrimSpace(resp.Msg.Tokens.AccessToken) == "" {
		return "", "", fmt.Errorf("local exchange returned no access token")
	}
	return resp.Msg.Tokens.AccessToken, resp.Msg.Tokens.RefreshToken, nil
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
	// VROOLI_STORAGE_NAMESPACE is ambient process state belonging to whatever
	// scenario launched this CLI (often web-console), not to the authenticator
	// whose socket this client is acquiring. Using it here made an owner CLI
	// silently look for a web-console-namespaced socket. The live authenticator
	// listener has the canonical scenario-authenticator namespace; shadow or
	// custom instances must opt in explicitly with VROOLI_AUTH_SOCKET.
	return filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
}
