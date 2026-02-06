package deploy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// staticResolver returns a BaseURLResolver that always returns the given URL.
func staticResolver(baseURL string) BaseURLResolver {
	return func(_ context.Context) (string, error) {
		return baseURL, nil
	}
}

// newTestClient creates an LPBSClient pointed at an httptest.Server.
func newTestClient(server *httptest.Server) *LPBSClient {
	return NewLPBSClientWithResolver(
		staticResolver(server.URL),
		server.Client(),
		"test-service-token",
	)
}

func TestListRemoteProfiles(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/admin/remote-profiles" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-service-token" {
			t.Errorf("unexpected auth: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]RemoteProfile{
			{ID: 1, Tag: "prod", APIBase: "https://prod.example.com/api/v1", Status: "active"},
			{ID: 2, Tag: "staging", APIBase: "https://staging.example.com/api/v1", Status: "active"},
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	profiles, err := client.ListRemoteProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListRemoteProfiles: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].Tag != "prod" {
		t.Errorf("expected tag 'prod', got %q", profiles[0].Tag)
	}
}

func TestTestRemoteProfile(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			json.NewEncoder(w).Encode([]RemoteProfile{
				{ID: 42, Tag: "prod", APIBase: "https://prod.example.com/api/v1"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/42/test" && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	err := client.TestRemoteProfile(context.Background(), "prod")
	if err != nil {
		t.Fatalf("TestRemoteProfile: %v", err)
	}
}

func TestTestRemoteProfileNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]RemoteProfile{})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	err := client.TestRemoteProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestProxyRequest(t *testing.T) {
	var capturedBody map[string]interface{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			json.NewEncoder(w).Encode([]RemoteProfile{
				{ID: 10, Tag: "prod"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/10/proxy" && r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.Write([]byte(`{"result":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.ProxyRequest(context.Background(), "prod", "POST",
		"/admin/download-artifacts/presign-upload",
		map[string]string{"filename": "test.exe"})
	if err != nil {
		t.Fatalf("ProxyRequest: %v", err)
	}

	// Verify response
	var result map[string]string
	json.Unmarshal(resp, &result)
	if result["result"] != "ok" {
		t.Errorf("unexpected response: %s", string(resp))
	}

	// Verify proxy payload structure
	if capturedBody["method"] != "POST" {
		t.Errorf("expected method POST, got %v", capturedBody["method"])
	}
	if capturedBody["path"] != "/admin/download-artifacts/presign-upload" {
		t.Errorf("unexpected path: %v", capturedBody["path"])
	}
	if capturedBody["body"] == nil {
		t.Error("expected body in proxy payload")
	}
}

func TestUploadArtifact(t *testing.T) {
	// Create a temp file to upload
	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "test-app.exe")
	if err := os.WriteFile(artifactPath, []byte("fake-binary-content"), 0o644); err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	// Track which S3 upload URL was used
	s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("S3: expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "fake-binary-content" {
			t.Errorf("S3: unexpected body: %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer s3Server.Close()

	var proxyCallCount int

	lpbsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			json.NewEncoder(w).Encode([]RemoteProfile{
				{ID: 5, Tag: "prod", APIBase: "https://prod.example.com/api/v1"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/5/proxy" && r.Method == "POST":
			proxyCallCount++
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)

			path, _ := payload["path"].(string)
			switch {
			case strings.Contains(path, "presign-upload"):
				json.NewEncoder(w).Encode(presignResponse{
					UploadURL: s3Server.URL + "/bucket/object",
					Bucket:    "test-bucket",
					ObjectKey: "uploads/test-app.exe",
				})
			case strings.Contains(path, "commit"):
				w.Write([]byte(`{"id":99}`))
			case strings.Contains(path, "apply"):
				w.Write([]byte(`{"ok":true}`))
			default:
				t.Errorf("unexpected proxy path: %s", path)
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	lpbsServer := httptest.NewServer(lpbsHandler)
	defer lpbsServer.Close()

	// Use the LPBS server's client for admin requests, but need a custom HTTP client
	// that can reach both the LPBS server and S3 server
	client := NewLPBSClientWithResolver(
		staticResolver(lpbsServer.URL),
		http.DefaultClient,
		"test-token",
	)

	result, err := client.UploadArtifact(context.Background(), &UploadRequest{
		RemoteProfile:  "prod",
		AppKey:         "test-app",
		Platform:       "windows",
		FilePath:       artifactPath,
		ReleaseVersion: "1.0.0",
		ReleaseNotes:   "Initial release",
	})
	if err != nil {
		t.Fatalf("UploadArtifact: %v", err)
	}

	if result.ArtifactID != 99 {
		t.Errorf("expected artifact ID 99, got %d", result.ArtifactID)
	}
	if result.Platform != "windows" {
		t.Errorf("expected platform 'windows', got %q", result.Platform)
	}
	// 4 proxy calls: list profiles (for resolve) + presign + list profiles (for resolve) + commit + list profiles (for resolve) + apply
	// Actually: resolveProfileID calls ListRemoteProfiles each time, so:
	// presign: 1 list + 1 proxy = 2 admin requests
	// commit: 1 list + 1 proxy = 2 admin requests
	// apply: 1 list + 1 proxy = 2 admin requests
	// = 3 proxy calls to the proxy endpoint
	if proxyCallCount != 3 {
		t.Errorf("expected 3 proxy calls, got %d", proxyCallCount)
	}
}

func TestDeriveUpdateURL(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]RemoteProfile{
			{ID: 1, Tag: "prod", APIBase: "https://prod.example.com/api/v1"},
			{ID: 2, Tag: "staging", APIBase: "https://staging.example.com/api/v1/"},
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)

	// Test prod
	url, err := client.DeriveUpdateURL(context.Background(), "prod", "my-app")
	if err != nil {
		t.Fatalf("DeriveUpdateURL: %v", err)
	}
	if url != "https://prod.example.com/api/v1/updates/my-app" {
		t.Errorf("unexpected URL: %s", url)
	}

	// Test staging (with trailing slash in APIBase)
	url, err = client.DeriveUpdateURL(context.Background(), "staging", "my-app")
	if err != nil {
		t.Fatalf("DeriveUpdateURL staging: %v", err)
	}
	if url != "https://staging.example.com/api/v1/updates/my-app" {
		t.Errorf("unexpected URL: %s", url)
	}

	// Test not found
	_, err = client.DeriveUpdateURL(context.Background(), "nonexistent", "my-app")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestInferContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"app.exe", "application/octet-stream"},
		{"app.msi", "application/octet-stream"},
		{"app.dmg", "application/x-apple-diskimage"},
		{"app.zip", "application/zip"},
		{"app.deb", "application/vnd.debian.binary-package"},
		{"app.rpm", "application/x-rpm"},
		{"app.AppImage", "application/octet-stream"},
		{"latest.yml", "text/yaml"},
		{"latest.yaml", "text/yaml"},
		{"unknown.bin", "application/octet-stream"},
	}
	for _, tc := range tests {
		got := inferContentType(tc.filename)
		if got != tc.expected {
			t.Errorf("inferContentType(%q) = %q, want %q", tc.filename, got, tc.expected)
		}
	}
}

func TestAdminRequestHTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	_, err := client.ListRemoteProfiles(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got: %v", err)
	}
}

func TestProxyRequestWithoutPayload(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/admin/remote-profiles" && r.Method == "GET":
			json.NewEncoder(w).Encode([]RemoteProfile{
				{ID: 1, Tag: "prod"},
			})
		case r.URL.Path == "/api/v1/admin/remote-profiles/1/proxy" && r.Method == "POST":
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)
			// Without payload, body and headers should be absent
			if _, ok := payload["body"]; ok {
				t.Error("expected no body in proxy payload when payload is nil")
			}
			w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := newTestClient(server)
	resp, err := client.ProxyRequest(context.Background(), "prod", "GET", "/admin/some-endpoint", nil)
	if err != nil {
		t.Fatalf("ProxyRequest without payload: %v", err)
	}
	if !strings.Contains(string(resp), "ok") {
		t.Errorf("unexpected response: %s", string(resp))
	}
}
