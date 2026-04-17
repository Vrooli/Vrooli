// DOC: docs/reference/api-endpoints.md#ssh-key-management — HTTP endpoint documentation
package ssh

import (
	"context"
	"net/http"
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

// HandleListKeys returns a handler for GET /api/v1/ssh/keys.
func HandleListKeys(ks *KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sshDir, err := ks.getSSHDir()
		if err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "home_dir_error",
				Message: "Cannot determine SSH directory",
				Hint:    err.Error(),
			})
			return
		}

		keys, err := ks.DiscoverKeys()
		if err != nil {
			httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
				Code:    "key_discovery_failed",
				Message: "Failed to discover SSH keys",
				Hint:    err.Error(),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, ListKeysResponse{
			Outcome: Outcome{
				OK:        true,
				Status:    StatusSuccess,
				Timestamp: nowTimestamp(),
			},
			Keys:   keys,
			SSHDir: sshDir,
		})
	}
}

// HandleGenerateKey returns a handler for POST /api/v1/ssh/keys/generate.
func HandleGenerateKey(ks *KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := httputil.DecodeJSON[GenerateKeyRequest](r.Body, 1<<20)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Request body must be valid JSON",
				Hint:    err.Error(),
			})
			return
		}

		key, err := ks.GenerateKey(req)
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
			Outcome: Outcome{
				OK:        true,
				Status:    StatusSuccess,
				Timestamp: nowTimestamp(),
			},
			Key: key,
		})
	}
}

// HandleGetPublicKey returns a handler for POST /api/v1/ssh/keys/public.
func HandleGetPublicKey(ks *KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		publicKey, fingerprint, err := ks.ReadPublicKey(req.KeyPath)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "key_not_found",
				Message: "Cannot read public key",
				Hint:    err.Error(),
			})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, GetPublicKeyResponse{
			Outcome: Outcome{
				OK:        true,
				Status:    StatusSuccess,
				Timestamp: nowTimestamp(),
			},
			PublicKey:   publicKey,
			Fingerprint: fingerprint,
		})
	}
}

// HandleTestConnection returns a handler for POST /api/v1/ssh/test.
// The runner is passed through to TestConnection for testability.
func HandleTestConnection(runner Runner, opts HandlerOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ctx, cancel := context.WithTimeout(r.Context(), opts.TestConnectionTimeout)
		defer cancel()

		result := TestConnection(ctx, runner, req)
		httputil.WriteJSON(w, http.StatusOK, result)
	}
}

// HandleCopyKey returns a handler for POST /api/v1/ssh/copy-key.
func HandleCopyKey(copier KeyCopier, opts HandlerOptions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		ctx, cancel := context.WithTimeout(r.Context(), opts.CopyKeyTimeout)
		defer cancel()

		result := copier.CopyKey(ctx, req)

		// Zero out password in memory (defense in depth)
		req.Password = ""

		httputil.WriteJSON(w, http.StatusOK, result)
	}
}

// HandleDeleteKey returns a handler for DELETE /api/v1/ssh/keys.
func HandleDeleteKey(ks *KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		result := ks.DeleteKey(req)
		status := http.StatusOK
		if !result.OK {
			status = http.StatusBadRequest
		}
		httputil.WriteJSON(w, status, result)
	}
}
