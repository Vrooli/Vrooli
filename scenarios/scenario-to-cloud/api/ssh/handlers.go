package ssh

import (
	"context"
	"net/http"
	"time"

	"scenario-to-cloud/internal/httputil"
)

// requireKeyPath validates that key_path is present.
// Returns false and writes an error response if validation fails.
func requireKeyPath(w http.ResponseWriter, keyPath string) bool {
	if keyPath == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_key_path",
			Message: "key_path is required",
		})
		return false
	}
	return true
}

// requireHostAndKeyPath validates that both host and key_path are present.
// Returns false and writes an error response if validation fails.
func requireHostAndKeyPath(w http.ResponseWriter, host, keyPath string) bool {
	if host == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_host",
			Message: "host is required",
		})
		return false
	}
	return requireKeyPath(w, keyPath)
}

// HandleListKeys handles GET /api/v1/ssh/keys.
func HandleListKeys(w http.ResponseWriter, r *http.Request) {
	sshDir, err := GetSSHDir()
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "home_dir_error",
			Message: "Cannot determine SSH directory",
			Hint:    err.Error(),
		})
		return
	}

	keys, err := DiscoverKeys(sshDir)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "key_discovery_failed",
			Message: "Failed to discover SSH keys",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, ListKeysResponse{
		Keys:      keys,
		SSHDir:    sshDir,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleGenerateKey handles POST /api/v1/ssh/keys/generate.
func HandleGenerateKey(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[GenerateKeyRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate type
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_key_type",
			Message: "Key type must be 'ed25519' or 'rsa'",
			Hint:    "Ed25519 is recommended for modern systems",
		})
		return
	}

	// Set defaults
	if req.Type == KeyTypeRSA && req.Bits == 0 {
		req.Bits = 4096
	}
	if req.Filename == "" {
		if req.Type == KeyTypeEd25519 {
			req.Filename = "id_ed25519"
		} else {
			req.Filename = "id_rsa"
		}
	}

	key, err := GenerateKey(req)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "key_generate_failed",
			Message: "Failed to generate SSH key",
			Hint:    err.Error(),
		})
		return
	}

	// Clear password from request (defense in depth)
	req.Password = ""

	httputil.WriteJSON(w, http.StatusCreated, GenerateKeyResponse{
		Key:       key,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleGetPublicKey handles POST /api/v1/ssh/keys/public.
func HandleGetPublicKey(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[GetPublicKeyRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if !requireKeyPath(w, req.KeyPath) {
		return
	}

	publicKey, fingerprint, err := ReadPublicKey(req.KeyPath)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "key_not_found",
			Message: "Cannot read public key",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, GetPublicKeyResponse{
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleTestConnection handles POST /api/v1/ssh/test.
func HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[TestConnectionRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if !requireHostAndKeyPath(w, req.Host, req.KeyPath) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := TestConnection(ctx, req)
	httputil.WriteJSON(w, http.StatusOK, result)
}

// HandleCopyKey handles POST /api/v1/ssh/copy-key.
func HandleCopyKey(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[CopyKeyRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if !requireHostAndKeyPath(w, req.Host, req.KeyPath) {
		return
	}
	if req.Password == "" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_password",
			Message: "password is required for ssh-copy-id operation",
			Hint:    "Enter the SSH password for the target server",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := CopyKeyToServer(ctx, req)

	// Zero out password in memory (defense in depth)
	req.Password = ""

	httputil.WriteJSON(w, http.StatusOK, result)
}

// HandleDeleteKey handles DELETE /api/v1/ssh/keys.
func HandleDeleteKey(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[DeleteKeyRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if !requireKeyPath(w, req.KeyPath) {
		return
	}

	result := DeleteKey(req)
	status := http.StatusOK
	if !result.OK {
		status = http.StatusBadRequest
	}
	httputil.WriteJSON(w, status, result)
}
