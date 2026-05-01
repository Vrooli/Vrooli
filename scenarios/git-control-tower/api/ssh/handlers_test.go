package ssh

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"git-control-tower/internal/testutil/httpx"
)

func TestSSHHandlersReturnExpectedStatuses(t *testing.T) {
	sshDir := t.TempDir()
	platform := &FakePlatform{SSHDirPath: sshDir, HomeDirPath: filepath.Dir(sshDir)}
	deps := SSHDeps{Platform: platform}

	server := httpx.NewServer(t, map[string]http.HandlerFunc{
		"/keys":     HandleListKeys(deps),
		"/generate": HandleGenerateKey(deps),
		"/public":   HandleGetPublicKey(deps),
		"/test":     HandleTestConnection(deps),
		"/delete":   HandleDeleteKey(deps),
	})

	t.Run("list keys succeeds with JSON body", func(t *testing.T) {
		res := requestJSON(t, http.MethodGet, server.URL+"/keys", "")
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
		}
		var body ListKeysResponse
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.SSHDir != sshDir {
			t.Fatalf("SSHDir = %q, want %q", body.SSHDir, sshDir)
		}
		if body.Keys == nil {
			t.Fatal("Keys = nil, want empty slice")
		}
	})

	t.Run("invalid JSON returns bad request", func(t *testing.T) {
		res := requestJSON(t, http.MethodPost, server.URL+"/generate", "{")
		defer res.Body.Close()

		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
		}
	})

	t.Run("service validation failure maps to unprocessable entity", func(t *testing.T) {
		res := requestJSON(t, http.MethodPost, server.URL+"/public", `{"key_path":""}`)
		defer res.Body.Close()

		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnprocessableEntity)
		}
		var body GetPublicKeyResponse
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Success {
			t.Fatal("Success = true, want false")
		}
	})

	t.Run("connection test returns ok for validation result", func(t *testing.T) {
		res := requestJSON(t, http.MethodPost, server.URL+"/test", `{"key_path":""}`)
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
		}
		var body TestConnectionResponse
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Status != "missing_key_path" {
			t.Fatalf("Status = %q, want missing_key_path", body.Status)
		}
	})

	t.Run("delete validation failure maps to unprocessable entity", func(t *testing.T) {
		res := requestJSON(t, http.MethodDelete, server.URL+"/delete", `{"key_path":""}`)
		defer res.Body.Close()

		if res.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusUnprocessableEntity)
		}
	})
}

func requestJSON(t *testing.T, method, url, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpx.TestClient().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return res
}
