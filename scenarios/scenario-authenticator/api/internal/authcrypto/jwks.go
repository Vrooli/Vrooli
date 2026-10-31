package authcrypto

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
)

// JWK is a single JSON Web Key. Only the fields required to verify an RS256
// signature are published; the private key never leaves this service. Ported
// verbatim from the old handlers/jwks.go.
type JWK struct {
	Kty string `json:"kty"` // key type — always "RSA" here
	Use string `json:"use"` // intended use — "sig" (signature verification)
	Alg string `json:"alg"` // algorithm — "RS256"
	Kid string `json:"kid"` // stable key id (fingerprint of the public key)
	N   string `json:"n"`   // modulus, base64url (no padding), big-endian
	E   string `json:"e"`   // public exponent, base64url (no padding), big-endian
}

// JWKS is a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKS returns the public key set a relying party fetches to verify owner
// tokens locally (the OIDC pattern: signing key is public, only the private key
// mints). Carries the same `kid` the JWT header advertises.
func (k *Keys) JWKS() JWKS {
	return JWKS{Keys: []JWK{jwkFromRSA(k.public, k.kid)}}
}

func jwkFromRSA(pub *rsa.PublicKey, kid string) JWK {
	// Exponent as a minimal big-endian byte slice (RFC 7518 §6.3.1.2).
	eBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(eBytes, uint64(pub.E))
	eBytes = trimLeadingZeros(eBytes)
	return JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

func trimLeadingZeros(b []byte) []byte {
	i := 0
	for i < len(b)-1 && b[i] == 0 {
		i++
	}
	return b[i:]
}
