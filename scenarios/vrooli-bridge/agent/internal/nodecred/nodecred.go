// Package nodecred is the node-agent's half of mutual auth: it holds the
// per-node Ed25519 keypair generated at pairing (the private key never leaves
// the node) and signs every node→control-plane exchange so the control plane
// can verify the node's identity (api internal/nodeauth is the verifier).
//
// The wire format is the contract shared with internal/nodeauth and MUST match
// it byte-for-byte (like the channel ProtocolVersion):
//
//   - signing payload: "<node_id>\n<unix_seconds>"
//   - Connect-RPC headers: X-Bridge-Node, X-Bridge-Ts, X-Bridge-Sig(base64)
//   - SSE dial-out token (EventSource can't set headers): "<node_id>.<unix>.<b64sig>"
//
// nodecred_test.go cross-checks the encodings round-trip, but the authoritative
// definition lives in nodeauth; change both together.
package nodecred

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Header names carrying the node's auth proof on Connect-RPC calls. They mirror
// api internal/nodeauth.Header*.
const (
	HeaderNode = "X-Bridge-Node"
	HeaderTS   = "X-Bridge-Ts"
	HeaderSig  = "X-Bridge-Sig"
)

// Credential is the node's loaded Ed25519 keypair.
type Credential struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

// LoadOrCreate returns the node's keypair, generating and persisting a new one
// (0600 seed) the first time. The parent dir is created (0700) if missing. The
// public key is registered with the control plane at pairing; the private key
// stays here forever (rotation = re-pair).
func LoadOrCreate(path string) (*Credential, error) {
	if path == "" {
		return nil, errors.New("nodecred: empty credential path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("nodecred: create dir: %w", err)
	}
	seed, err := os.ReadFile(path) //nolint:gosec // path is the agent's own state dir, not user input
	switch {
	case err == nil:
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("nodecred: malformed key file %q", path)
		}
		return fromPrivate(ed25519.NewKeyFromSeed(seed)), nil
	case errors.Is(err, os.ErrNotExist):
		pub, priv, gErr := ed25519.GenerateKey(rand.Reader)
		if gErr != nil {
			return nil, fmt.Errorf("nodecred: generate: %w", gErr)
		}
		if wErr := os.WriteFile(path, priv.Seed(), 0o600); wErr != nil {
			return nil, fmt.Errorf("nodecred: write key: %w", wErr)
		}
		return &Credential{priv: priv, pub: pub}, nil
	default:
		return nil, fmt.Errorf("nodecred: read key: %w", err)
	}
}

func fromPrivate(priv ed25519.PrivateKey) *Credential {
	return &Credential{priv: priv, pub: priv.Public().(ed25519.PublicKey)}
}

// PublicKeyBase64 returns the standard-base64 public key the node hands to the
// control plane at pairing (the `--public-key` of `pair redeem`).
func (c *Credential) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(c.pub)
}

// signingPayload MUST match api internal/nodeauth.SigningPayload.
func signingPayload(nodeID string, ts time.Time) []byte {
	return []byte(nodeID + "\n" + strconv.FormatInt(ts.Unix(), 10))
}

// Sign returns the Ed25519 signature over the canonical payload.
func (c *Credential) Sign(nodeID string, ts time.Time) []byte {
	return ed25519.Sign(c.priv, signingPayload(nodeID, ts))
}

// Headers returns the X-Bridge-* proof headers for a Connect-RPC call at ts.
func (c *Credential) Headers(nodeID string, ts time.Time) map[string]string {
	return map[string]string{
		HeaderNode: nodeID,
		HeaderTS:   strconv.FormatInt(ts.Unix(), 10),
		HeaderSig:  base64.StdEncoding.EncodeToString(c.Sign(nodeID, ts)),
	}
}

// Token returns the SSE dial-out token for the EventSource `?token=` path.
func (c *Credential) Token(nodeID string, ts time.Time) string {
	return strings.Join([]string{
		nodeID,
		strconv.FormatInt(ts.Unix(), 10),
		base64.StdEncoding.EncodeToString(c.Sign(nodeID, ts)),
	}, ".")
}
