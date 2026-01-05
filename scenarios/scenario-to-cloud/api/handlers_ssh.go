package main

import (
	"context"
	"net/http"
	"time"
)

// requireKeyPath validates that key_path is present.
// Returns false and writes an error response if validation fails.
func requireKeyPath(w http.ResponseWriter, keyPath string) bool {
	if keyPath == "" {
		writeAPIError(w, http.StatusBadRequest, APIError{
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
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_host",
			Message: "host is required",
		})
		return false
	}
	return requireKeyPath(w, keyPath)
}

// handleListSSHKeys handles GET /api/v1/ssh/keys
func (s *Server) handleListSSHKeys(w http.ResponseWriter, r *http.Request) {
	sshDir, err := getSSHDir()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "home_dir_error",
			Message: "Cannot determine SSH directory",
			Hint:    err.Error(),
		})
		return
	}

	keys, err := DiscoverSSHKeys(sshDir)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "key_discovery_failed",
			Message: "Failed to discover SSH keys",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, ListSSHKeysResponse{
		Keys:      keys,
		SSHDir:    sshDir,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGenerateSSHKey handles POST /api/v1/ssh/keys/generate
func (s *Server) handleGenerateSSHKey(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[GenerateSSHKeyRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate type
	if req.Type != KeyTypeEd25519 && req.Type != KeyTypeRSA {
		writeAPIError(w, http.StatusBadRequest, APIError{
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

	key, err := GenerateSSHKey(req)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "key_generate_failed",
			Message: "Failed to generate SSH key",
			Hint:    err.Error(),
		})
		return
	}

	// Clear password from request (defense in depth)
	req.Password = ""

	writeJSON(w, http.StatusCreated, GenerateSSHKeyResponse{
		Key:       key,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGetPublicKey handles POST /api/v1/ssh/keys/public
func (s *Server) handleGetPublicKey(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[GetPublicKeyRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
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
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "key_not_found",
			Message: "Cannot read public key",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, GetPublicKeyResponse{
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleTestSSHConnection handles POST /api/v1/ssh/test
func (s *Server) handleTestSSHConnection(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[TestSSHConnectionRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
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

	result := TestSSHConnection(ctx, req)
	writeJSON(w, http.StatusOK, result)
}

// handleCopySSHKey handles POST /api/v1/ssh/copy-key
func (s *Server) handleCopySSHKey(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[CopySSHKeyRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
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
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "missing_password",
			Message: "password is required for ssh-copy-id operation",
			Hint:    "Enter the SSH password for the target server",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := CopySSHKeyToServer(ctx, req)

	// Zero out password in memory (defense in depth)
	req.Password = ""

	writeJSON(w, http.StatusOK, result)
}

// handleDeleteSSHKey handles DELETE /api/v1/ssh/keys
func (s *Server) handleDeleteSSHKey(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[DeleteSSHKeyRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if !requireKeyPath(w, req.KeyPath) {
		return
	}

	result := DeleteSSHKey(req)
	status := http.StatusOK
	if !result.OK {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, result)
}
