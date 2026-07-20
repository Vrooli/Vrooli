//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"resource-searxng/cli/internal/config"
)

func TestManifestImageBootsWithSeparateConfigAndCacheMounts(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker is required for integration test")
	}
	data, err := os.ReadFile(filepath.Join("..", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Runtime struct {
			Image string `json:"image"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	cacheDir := filepath.Join(root, "cache")
	if _, err := config.Apply(configDir, "http://localhost:8280", "SearXNG integration", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	name := fmt.Sprintf("searxng-integration-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
		// The upstream container initializes files as its runtime user. Restore
		// test-temp ownership through an isolated helper container so Go's test
		// cleanup can remove the temporary fixture.
		_ = exec.Command("docker", "run", "--rm", "-v", root+":/cleanup", "alpine", "chmod", "-R", "a+rwx", "/cleanup").Run()
	})
	cmd := exec.Command("docker", "run", "-d", "--name", name, "-p", fmt.Sprintf("127.0.0.1:%d:8080", port), "-v", configDir+":/etc/searxng", "-v", cacheDir+":/var/cache/searxng", manifest.Runtime.Image)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("start manifest image: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	url := fmt.Sprintf("http://127.0.0.1:%d/stats", port)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		if ctx.Err() != nil {
			t.Fatalf("manifest container did not become healthy: %v", ctx.Err())
		}
		time.Sleep(250 * time.Millisecond)
	}
}
