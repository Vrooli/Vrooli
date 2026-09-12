// Package jwks serves the RS256 public key as a JWK Set at the OIDC-standard
// /.well-known/jwks.json path so relying parties (device-sync-hub) verify owner
// tokens locally instead of calling Validate per request. This is the only P0
// REST endpoint — a web standard (RFC 7517) that cannot be expressed as a
// Connect RPC. Ported from the old handlers/jwks.go.
package jwks

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"scenario-authenticator/internal/authcrypto"
	"scenario-authenticator/internal/module"
)

// Path is the OIDC discovery-standard JWKS path. FROZEN: device-sync-hub fetches
// exactly this path.
const Path = "/.well-known/jwks.json"

// Handler serves the JWK Set for a keypair.
type Handler struct {
	keys *authcrypto.Keys
}

// NewHandler constructs a JWKS handler over the loaded keypair.
func NewHandler(keys *authcrypto.Keys) *Handler { return &Handler{keys: keys} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if h.keys == nil {
		http.Error(w, `{"code":"unavailable","message":"signing key not available"}`, http.StatusServiceUnavailable)
		return
	}
	// The key is stable across restarts (persisted); allow brief caching so a
	// consumer fetching per process start does not hammer this service.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(h.keys.JWKS())
}

// Module mounts the JWKS REST edge.
func Module(keys *authcrypto.Keys) module.Module {
	h := NewHandler(keys)
	return module.Module{
		Name: "jwks",
		Mount: func(r *mux.Router) {
			r.Handle(Path, h).Methods(http.MethodGet, http.MethodOptions)
		},
		Endpoints: Endpoints,
	}
}

// Endpoints documents the JWKS REST edge. It is a deliberate REST exception: the
// /.well-known path and JWK Set shape are an external standard, not a Connect
// RPC.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "jwks",
		Path:        Path,
		Method:      "GET",
		Summary:     "JSON Web Key Set",
		Description: "Publishes the RS256 public key (kid, n, e) so relying parties verify owner tokens locally. Cache-Control: public, max-age=300.",
		Category:    "auth",
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"keys": "array<JWK{kty,use,alg,kid,n,e}>",
		}},
		Examples: []module.Example{{Name: "Fetch JWKS", Curl: "curl http://localhost:${API_PORT}/.well-known/jwks.json"}},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonThirdPartyShape,
			Note:   "The /.well-known/jwks.json path and JWK Set shape are an external standard (RFC 7517 / OIDC), consumed by relying parties that verify tokens offline. Not expressible as a Connect RPC.",
		},
	},
}
