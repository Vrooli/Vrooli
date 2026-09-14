package authn

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

// DefaultLocalAuthenticatorSocket returns the canonical local listener path.
// Ambient VROOLI_STORAGE_NAMESPACE belongs to the caller, not the authenticator.
// Custom listeners must be selected explicitly with VROOLI_AUTH_SOCKET.
func DefaultLocalAuthenticatorSocket() string {
	return filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
}

// ExchangeLocalMachinePrincipal exchanges the current process' Unix peer
// credential for a session. The authenticator owns verification and grants.
// Callers must require explicit operator intent; this is not an automatic
// agent-to-owner fallback. It never persists or prints credentials and never
// falls back to TCP or token files. Callers must not log the returned tokens.
func ExchangeLocalMachinePrincipal(ctx context.Context) (*accountsv1.LoginResponse, error) {
	socketPath := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SOCKET"))
	if socketPath == "" {
		socketPath = DefaultLocalAuthenticatorSocket()
	}
	machineID, err := os.Hostname()
	if err != nil || strings.TrimSpace(machineID) == "" {
		return nil, fmt.Errorf("resolve machine id: %w", err)
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
		return nil, err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Tokens == nil || strings.TrimSpace(resp.Msg.Tokens.AccessToken) == "" {
		return nil, fmt.Errorf("local exchange returned no access token")
	}
	return &accountsv1.LoginResponse{Account: resp.Msg.Account, Tokens: resp.Msg.Tokens}, nil
}
