// Package authcrypto is the verbatim-ported, RS256-locked crypto core of the
// authenticator: the load-or-generate signing keypair, the JWT signer/validator
// (with the algorithm method-lock that kills none/HS confusion), the JWKS
// fingerprint, and the SHA-256 token-hash helpers.
//
// This is the highest-stakes boundary in the fleet. It is a VERBATIM PORT of
// the old scenario's auth/jwt.go + handlers/jwks.go, with exactly three
// deliberate corrections (DECISIONS #30, plan §8):
//
//  1. The key directory is resolved by the CALLER via the storage seam (an
//     absolute path under the scenario data root), not a relative CWD path —
//     so the keypair survives wherever the process runs.
//  2. A key WRITE failure is FATAL (returns an error) rather than logged-and-
//     continued — the old behaviour silently regenerated a fresh key on every
//     boot when the dir was unwritable, rotating the signing key and breaking
//     every relying party that had cached the old JWKS.
//  3. The JWT header carries `kid` (the same SHA-256-of-PKIX fingerprint JWKS
//     publishes) so rotation-aware verification is expressible later.
//
// Do not re-derive any of this. Port-with-corrections only.
package authcrypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// Keys holds the RS256 signing keypair plus its stable key id. It is
// constructed once at boot by LoadOrGenerate and shared read-only for the
// process lifetime.
type Keys struct {
	private *rsa.PrivateKey
	public  *rsa.PublicKey
	kid     string
}

// LoadOrGenerate loads the persisted RS256 keypair from keyDir, generating and
// persisting a fresh one if none exists. keyDir MUST be an absolute path the
// caller resolved via the storage seam.
//
// Ported from the old LoadJWTKeys/GenerateJWTKeys with the §8 corrections:
// persistence failure is fatal (never silently regenerate), and the kid is
// computed up front so it can ride the JWT header.
func LoadOrGenerate(keyDir string) (*Keys, error) {
	privateKeyPath := filepath.Join(keyDir, "private.pem")
	publicKeyPath := filepath.Join(keyDir, "public.pem")

	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read private key %s: %w", privateKeyPath, err)
		}
		return generateAndPersist(keyDir, privateKeyPath, publicKeyPath)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key PEM block in %s", privateKeyPath)
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format.
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err2)
		}
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
	}

	pub := &priv.PublicKey
	if publicKeyData, perr := os.ReadFile(publicKeyPath); perr == nil {
		pblock, _ := pem.Decode(publicKeyData)
		if pblock == nil {
			return nil, fmt.Errorf("failed to parse public key PEM block in %s", publicKeyPath)
		}
		pubInterface, err3 := x509.ParsePKIXPublicKey(pblock.Bytes)
		if err3 != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err3)
		}
		rsaPub, ok := pubInterface.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA public key")
		}
		pub = rsaPub
	}

	return &Keys{private: priv, public: pub, kid: fingerprint(pub)}, nil
}

// generateAndPersist creates a fresh 2048-bit RSA keypair and writes both PEMs
// to disk. CORRECTION (§8): a write failure is fatal — we refuse to boot rather
// than serve a key we cannot persist (which would rotate on the next restart).
func generateAndPersist(keyDir, privateKeyPath, publicKeyPath string) (*Keys, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	pub := &priv.PublicKey

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := &pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to prepare key directory %s: %w", keyDir, err)
	}
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(privateKeyPEM), 0o600); err != nil {
		return nil, fmt.Errorf("failed to persist private key %s: %w", privateKeyPath, err)
	}
	if err := os.WriteFile(publicKeyPath, pem.EncodeToMemory(publicKeyPEM), 0o644); err != nil {
		return nil, fmt.Errorf("failed to persist public key %s: %w", publicKeyPath, err)
	}

	return &Keys{private: priv, public: pub, kid: fingerprint(pub)}, nil
}

// NewKeysFromPair builds a Keys from an in-memory pair. Test-only helper that
// keeps the unexported fields encapsulated (mirrors the old SetTestKeys).
func NewKeysFromPair(priv *rsa.PrivateKey, pub *rsa.PublicKey) *Keys {
	return &Keys{private: priv, public: pub, kid: fingerprint(pub)}
}

// Public returns the RS256 public key used to verify tokens (published via
// JWKS). The private key is never exposed.
func (k *Keys) Public() *rsa.PublicKey { return k.public }

// KID returns the stable key id (SHA-256 fingerprint of the DER-encoded public
// key, truncated). Published in JWKS and carried in the JWT header.
func (k *Keys) KID() string { return k.kid }

// fingerprint derives a stable, deterministic key id from the DER-encoded
// public key (a SHA-256 fingerprint, truncated). Ported verbatim from the old
// handlers/jwks.go publicKeyID so the kid is byte-identical to what relying
// parties may already have observed.
func fingerprint(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "default"
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}
