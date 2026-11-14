// Package cpkeys owns the control plane's long-lived Ed25519 identity keypair
// (SECURITY.md boundary 2, DECISIONS.md 2026-06-18). The control plane proves
// it is the legitimate coordinator by signing its node-facing pushes with this
// key; a node pins the matching public key at bootstrap (returned by the
// pairing domain) and rejects any push that does not verify — so a node never
// executes a job from an impostor control plane.
//
// The PRIVATE key is genuinely secret: it is generated once, persisted to the
// storage root with 0600 permissions, and never leaves the control plane. The
// PUBLIC key is handed to every paired node and is safe to disclose. Loading is
// idempotent (load-or-generate) so the identity is stable across restarts — a
// rotated key would orphan every paired node, so rotation is a deliberate,
// re-pair-all operation, never an accident.
package cpkeys

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// keyFileName is the on-disk file the control plane's Ed25519 seed is persisted
// to inside the provided directory.
const keyFileName = "control_plane_ed25519.key"

// Keypair is the control plane's loaded identity. It exposes signing and the
// shareable public key; the private key never escapes the struct.
type Keypair struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

// LoadOrCreate returns the control plane's keypair, generating and persisting a
// new one (0600) the first time. dir is created (0700) if missing. Subsequent
// calls return the same stable identity.
func LoadOrCreate(dir string) (*Keypair, error) {
	if dir == "" {
		return nil, errors.New("cpkeys: empty key directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("cpkeys: create key dir: %w", err)
	}
	path := filepath.Join(dir, keyFileName)

	seed, err := os.ReadFile(path) //nolint:gosec // path is server-controlled (storage root), not user input
	switch {
	case err == nil:
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("cpkeys: malformed key file %q: want %d-byte seed, got %d", path, ed25519.SeedSize, len(seed))
		}
		priv := ed25519.NewKeyFromSeed(seed)
		return fromPrivate(priv), nil
	case errors.Is(err, os.ErrNotExist):
		return generate(path)
	default:
		return nil, fmt.Errorf("cpkeys: read key file: %w", err)
	}
}

func generate(path string) (*Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cpkeys: generate key: %w", err)
	}
	// Persist the 32-byte seed (not the 64-byte expanded private key) with
	// owner-only permissions; the seed deterministically regenerates the key.
	if err := os.WriteFile(path, priv.Seed(), 0o600); err != nil {
		return nil, fmt.Errorf("cpkeys: write key file: %w", err)
	}
	return &Keypair{priv: priv, pub: pub}, nil
}

func fromPrivate(priv ed25519.PrivateKey) *Keypair {
	return &Keypair{priv: priv, pub: priv.Public().(ed25519.PublicKey)}
}

// Sign returns the Ed25519 signature over msg using the control plane's private
// key.
func (k *Keypair) Sign(msg []byte) []byte {
	return ed25519.Sign(k.priv, msg)
}

// PublicKey returns the raw 32-byte public key.
func (k *Keypair) PublicKey() ed25519.PublicKey {
	return k.pub
}

// PublicKeyBase64 returns the standard-base64 public key the pairing domain
// hands to nodes to pin.
func (k *Keypair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(k.pub)
}
