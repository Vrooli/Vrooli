package ssh

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// httpResponse provides consistent response handling for SSH handlers.
type httpResponse struct {
	w http.ResponseWriter
}

func newResponse(w http.ResponseWriter) *httpResponse {
	return &httpResponse{w: w}
}

func (r *httpResponse) json(status int, data interface{}) {
	r.w.Header().Set("Content-Type", "application/json")
	r.w.WriteHeader(status)
	_ = json.NewEncoder(r.w).Encode(data)
}

func (r *httpResponse) ok(data interface{}) {
	r.json(http.StatusOK, data)
}

func (r *httpResponse) created(data interface{}) {
	r.json(http.StatusCreated, data)
}

func (r *httpResponse) badRequest(message string) {
	r.json(http.StatusBadRequest, map[string]string{"error": message})
}

func (r *httpResponse) internalError(message string) {
	r.json(http.StatusInternalServerError, map[string]string{"error": message})
}

// parseJSONBody decodes a JSON request body into the target struct.
func parseJSONBody(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	resp := newResponse(w)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		resp.badRequest("invalid request body: " + err.Error())
		return false
	}
	return true
}

// HandleListKeys handles GET /api/v1/ssh/keys.
func HandleListKeys(deps SSHDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := newResponse(w)

		result, err := ListKeys(ctx, deps)
		if err != nil {
			resp.internalError(err.Error())
			return
		}

		resp.ok(result)
	}
}

// HandleGenerateKey handles POST /api/v1/ssh/keys/generate.
func HandleGenerateKey(deps SSHDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		resp := newResponse(w)

		var req GenerateKeyRequest
		if !parseJSONBody(w, r, &req) {
			return
		}

		result, err := GenerateKeyService(ctx, deps, req)
		if err != nil {
			resp.internalError(err.Error())
			return
		}

		if !result.Success {
			resp.json(http.StatusUnprocessableEntity, result)
			return
		}

		resp.created(result)
	}
}

// HandleGetPublicKey handles POST /api/v1/ssh/keys/public.
func HandleGetPublicKey(deps SSHDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := newResponse(w)

		var req GetPublicKeyRequest
		if !parseJSONBody(w, r, &req) {
			return
		}

		result, err := GetPublicKeyService(ctx, deps, req)
		if err != nil {
			resp.internalError(err.Error())
			return
		}

		if !result.Success {
			resp.json(http.StatusUnprocessableEntity, result)
			return
		}

		resp.ok(result)
	}
}

// HandleTestConnection handles POST /api/v1/ssh/keys/test.
func HandleTestConnection(deps SSHDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		resp := newResponse(w)

		var req TestConnectionRequest
		if !parseJSONBody(w, r, &req) {
			return
		}

		result, err := TestGitHubConnectionService(ctx, deps, req)
		if err != nil {
			resp.internalError(err.Error())
			return
		}

		// Always return 200 for test results - success/failure is in the response body
		resp.ok(result)
	}
}

// HandleDeleteKey handles DELETE /api/v1/ssh/keys.
func HandleDeleteKey(deps SSHDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := newResponse(w)

		var req DeleteKeyRequest
		if !parseJSONBody(w, r, &req) {
			return
		}

		result, err := DeleteKeyService(ctx, deps, req)
		if err != nil {
			resp.internalError(err.Error())
			return
		}

		if !result.Success {
			resp.json(http.StatusUnprocessableEntity, result)
			return
		}

		resp.ok(result)
	}
}
