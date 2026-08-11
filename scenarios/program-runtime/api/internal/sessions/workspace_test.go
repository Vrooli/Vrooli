package sessions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/discovery"
)

func TestDiscoveryWorkspaceResolverResolvesWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sandbox-123/workspace" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"path": root})
	}))
	defer server.Close()

	resolver := NewDiscoveryWorkspaceResolver(discovery.NewStaticResolver(server.URL), server.Client())
	got, err := resolver.Resolve(context.Background(), "sandbox-123")
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("resolved=%q want=%q", got, root)
	}
}

func TestDiscoveryWorkspaceResolverLocalFallbackValidatesPath(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	resolver := NewDiscoveryWorkspaceResolver(nil, nil)
	got, err := resolver.Resolve(context.Background(), root)
	if err != nil || got != root {
		t.Fatalf("resolved=%q err=%v want=%q", got, err, root)
	}
}
