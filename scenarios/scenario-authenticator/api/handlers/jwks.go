package handlers

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"net/http"

	"scenario-authenticator/auth"
	"scenario-authenticator/utils"
)

// jwk is a single JSON Web Key. Only the fields required to verify an RS256
// signature are published; the private key never leaves this service.
type jwk struct {
	Kty string `json:"kty"` // key type — always "RSA" here
	Use string `json:"use"` // intended use — "sig" (signature verification)
	Alg string `json:"alg"` // algorithm — "RS256"
	Kid string `json:"kid"` // stable key id (fingerprint of the public key)
	N   string `json:"n"`   // modulus, base64url (no padding), big-endian
	E   string `json:"e"`   // public exponent, base64url (no padding), big-endian
}

// jwks is a JSON Web Key Set.
type jwks struct {
	Keys []jwk `json:"keys"`
}

// JWKSHandler publishes the RS256 public key as a JWK Set so consumers can
// verify owner JWTs locally (offline) instead of calling /validate on every
// request. This is the standard OIDC pattern: the signing key is public; only
// the private key (held here) can mint tokens.
//
// Served at both /.well-known/jwks.json (the discovery-standard path) and
// /api/v1/auth/jwks (the API-namespaced alias).
func JWKSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}

	pub := auth.PublicKey()
	if pub == nil {
		utils.SendError(w, "signing key not available", http.StatusServiceUnavailable)
		return
	}

	// Exponent as a minimal big-endian byte slice (RFC 7518 §6.3.1.2).
	eBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(eBytes, uint64(pub.E))
	eBytes = trimLeadingZeros(eBytes)

	set := jwks{Keys: []jwk{{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: publicKeyID(pub),
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}}}

	// The key is stable across restarts (persisted to disk); allow brief caching
	// so a consumer that fetches per process start does not hammer this service.
	w.Header().Set("Cache-Control", "public, max-age=300")
	utils.SendJSON(w, set, http.StatusOK)
}

// publicKeyID derives a stable, deterministic key id from the DER-encoded
// public key (a SHA-256 fingerprint, truncated). A consumer matches a token's
// "kid" header against this when present; with a single key it is informational
// but lets a future key rotation be expressed without breaking older clients.
func publicKeyID(pub interface{}) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "default"
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

func trimLeadingZeros(b []byte) []byte {
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return b[i:]
}
