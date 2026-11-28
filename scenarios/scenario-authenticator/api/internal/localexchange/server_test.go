package localexchange

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestStartInjectsKernelPeerPrincipalOverRealSocket(t *testing.T) {
	want, err := localprincipal.Current()
	if errors.Is(err, localprincipal.ErrUnsupported) {
		repocontracttest.SkipPlatform(t, "platform has no local peer-credential implementation")
	}
	if err != nil {
		t.Fatalf("current principal: %v", err)
	}

	// A Unix socket address is limited to about 104-108 bytes; t.TempDir()
	// follows GOTMPDIR, which lifecycle-managed shells place deep under the
	// runtime home, so keep the socket in a short private directory.
	dir, err := os.MkdirTemp("", "sa")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "auth.sock")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PeerPrincipal(r.Context())
		if !ok {
			http.Error(w, "peer principal missing", http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, principal.String())
	})
	stop, err := Start(context.Background(), path, handler)
	if err != nil {
		t.Fatalf("start local exchange: %v", err)
	}
	t.Cleanup(stop)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	resp, err := client.Get("http://local-authenticator/peer")
	if err != nil {
		t.Fatalf("connect local exchange socket: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read peer response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %q", resp.StatusCode, body)
	}
	if strings.TrimSpace(string(body)) != want.String() {
		t.Fatalf("peer principal = %q, want %q", strings.TrimSpace(string(body)), want.String())
	}
}
