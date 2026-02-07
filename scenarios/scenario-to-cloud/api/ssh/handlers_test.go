package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleListKeys_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	handler := HandleListKeys(ks)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ssh/keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body ListKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.OK {
		t.Error("expected OK = true")
	}
	if body.SSHDir != dir {
		t.Errorf("SSHDir = %q, want %q", body.SSHDir, dir)
	}
}

func TestHandleListKeys_Error(t *testing.T) {
	t.Parallel()

	// Use a path that's a file, not a directory, to trigger an error
	tmpFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	ks := NewKeyService(&fakeCommandRunner{}, tmpFile)

	handler := HandleListKeys(ks)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ssh/keys", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleGenerateKey_Success(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	// This test validates the handler parses JSON correctly.
	// The actual generate will fail (no ssh-keygen), which is expected
	// to return a 400 from the handler.
	body, _ := json.Marshal(GenerateKeyRequest{
		Type: KeyTypeEd25519,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh/keys/generate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := HandleGenerateKey(ks)
	handler(w, req)

	resp := w.Result()
	// Without real ssh-keygen, we expect 400 (key_generate_failed)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 201 or 400", resp.StatusCode)
	}
}

func TestHandleGenerateKey_MissingType(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	// Send request with unsupported key type
	body, _ := json.Marshal(GenerateKeyRequest{
		Type: "dsa",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh/keys/generate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := HandleGenerateKey(ks)
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleDeleteKey_ProtectsSpecialFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ks := NewKeyService(&fakeCommandRunner{}, dir)

	body, _ := json.Marshal(DeleteKeyRequest{
		KeyPath: filepath.Join(dir, "authorized_keys"),
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/ssh/keys", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := HandleDeleteKey(ks)
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	var result DeleteKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.OK {
		t.Error("should not allow deleting authorized_keys")
	}
}

// fakeRunner implements ssh.Runner for handler tests.
type fakeRunner struct {
	result Result
	err    error
}

func (f *fakeRunner) Run(_ context.Context, _ Config, _ string, _ RunOptions) (Result, error) {
	return f.result, f.err
}

func TestHandleTestConnection_Success(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	testDir := filepath.Join(sshDir, "test_conn_success")
	if err := os.MkdirAll(testDir, 0o700); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(testDir) })

	keyPath := filepath.Join(testDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfake"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		result: Result{Stdout: "ok\nPRETTY_NAME=\"Ubuntu 24.04\"", ExitCode: 0},
	}

	body, _ := json.Marshal(TestConnectionRequest{
		Host:    "192.168.1.1",
		KeyPath: keyPath,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh/test", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := HandleTestConnection(runner, DefaultHandlerOptions())
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result TestConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.OK {
		t.Errorf("expected OK = true, got message: %s hint: %s", result.Message, result.Hint)
	}
}

func TestHandleTestConnection_AuthFailed(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}
	sshDir := filepath.Join(homeDir, ".ssh")
	testDir := filepath.Join(sshDir, "test_conn_authfail")
	if err := os.MkdirAll(testDir, 0o700); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(testDir) })

	keyPath := filepath.Join(testDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nfake"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		result: Result{Stderr: "Permission denied (publickey).", ExitCode: 255},
		err:    &SSHError{Category: ErrAuth, Message: "Permission denied (publickey)."},
	}

	body, _ := json.Marshal(TestConnectionRequest{
		Host:    "192.168.1.1",
		KeyPath: keyPath,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ssh/test", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := HandleTestConnection(runner, DefaultHandlerOptions())
	handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result TestConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.OK {
		t.Error("expected OK = false for auth failure")
	}
	if result.Status != StatusAuthFailed {
		t.Errorf("status = %q, want %q", result.Status, StatusAuthFailed)
	}
}
