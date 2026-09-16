package authn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	return LocalAuthenticatorSocket("scenario-authenticator")
}

// LocalAuthenticatorSocket returns the listener path the authenticator binds
// for a storage namespace. The authenticator and every client derive the path
// here, so the two sides cannot disagree.
//
// The path lives in the per-user temporary directory. On macOS that directory
// is /var/folders/<id>/T/ (49 bytes), and the canonical name pushes the path
// past the 104-byte sun_path limit, so bind fails with EINVAL. A path that
// does not fit is replaced by a short name derived from the canonical one; a
// path that fits is unchanged.
func LocalAuthenticatorSocket(namespace string) string {
	return localAuthenticatorSocketIn(os.TempDir(), namespace, runtime.GOOS)
}

func localAuthenticatorSocketIn(dir, namespace, goos string) string {
	name := "vrooli-scenario-authenticator"
	if namespace = strings.TrimSpace(namespace); namespace != "" {
		name += "-" + strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return '-'
		}, namespace)
	}
	canonical := filepath.Join(dir, name+".sock")
	if len(canonical) <= maxUnixSocketPath(goos) {
		return canonical
	}
	sum := sha256.Sum256([]byte(name))
	return filepath.Join(dir, "vrooli-auth-"+hex.EncodeToString(sum[:6])+".sock")
}

// maxUnixSocketPath is the longest socket path the platform accepts: sun_path
// holds 104 bytes on Darwin and the BSDs and 108 on Linux, NUL included.
func maxUnixSocketPath(goos string) int {
	switch goos {
	case "darwin", "freebsd", "netbsd", "openbsd", "dragonfly":
		return 103
	default:
		return 107
	}
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
