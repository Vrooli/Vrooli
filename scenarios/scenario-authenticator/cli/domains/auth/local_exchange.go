package auth

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

func exchangeLocal(ctx context.Context) (*accountsv1.LoginResponse, error) {
	socketPath := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SOCKET"))
	if socketPath == "" {
		socketPath = defaultAuthSocket()
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
	return &accountsv1.LoginResponse{
		Account: resp.Msg.Account,
		Tokens:  resp.Msg.Tokens,
	}, nil
}

func defaultAuthSocket() string {
	// VROOLI_STORAGE_NAMESPACE is ambient process state belonging to whatever
	// scenario launched this CLI, not to the authenticator whose socket this
	// client is acquiring. The live listener uses the canonical authenticator
	// namespace; shadow or custom instances must opt in with VROOLI_AUTH_SOCKET.
	return filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
}
