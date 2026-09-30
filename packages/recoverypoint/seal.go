package recoverypoint

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialpolicy"
)

// sealPurposePrefix binds an envelope to the recovery point and binding it
// was sealed for, so an artifact cannot be replayed under another slot.
const sealPurposePrefix = "cloud-recovery-point"

const (
	sealKDF       = "pbkdf2-sha256"
	sealKeyBytes  = 32
	sealSaltBytes = 16
)

// ErrKeyUnavailable is returned by a KeyResolver when the referenced key
// cannot be obtained. Callers surface it as recovery_key_unavailable naming
// the reference as the blocker.
var ErrKeyUnavailable = errors.New("recovery key unavailable")

// KeyResolver obtains recovery key material from a reference. The reference
// is what a manifest records; the material never is. Production wiring
// resolves through the credential authority; tests use StaticKeys.
type KeyResolver interface {
	ResolveKey(ctx context.Context, keyRef string) ([]byte, error)
}

// StaticKeys is an in-memory resolver keyed by reference.
type StaticKeys map[string][]byte

// ResolveKey implements KeyResolver.
func (s StaticKeys) ResolveKey(_ context.Context, keyRef string) ([]byte, error) {
	material, ok := s[keyRef]
	if !ok || len(material) == 0 {
		return nil, ErrKeyUnavailable
	}
	return material, nil
}

// Sealer derives the artifact key from resolved key material. Iterations is
// the PBKDF2 cost; zero selects the credential policy default.
type Sealer struct {
	Iterations int
}

func (s Sealer) iterations() int {
	if s.Iterations > 0 {
		return s.Iterations
	}
	return credentialpolicy.RecoveryPBKDF2Iterations
}

// sealedEnvelope is the on-disk frame around credentialpolicy's AEAD.
type sealedEnvelope struct {
	Version    int    `json:"version"`
	Purpose    string `json:"purpose"`
	KDF        string `json:"kdf"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func sealPurpose(recoveryPointID, binding string) string {
	return sealPurposePrefix + ":" + recoveryPointID + ":" + binding
}

// Seal encrypts plaintext for one (recovery point, binding) slot.
func (s Sealer) Seal(material, plaintext []byte, recoveryPointID, binding string) ([]byte, error) {
	if len(material) == 0 {
		return nil, ErrKeyUnavailable
	}
	salt := make([]byte, sealSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, string(material), salt, s.iterations(), sealKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	envelope, err := credentialpolicy.Seal(key, plaintext, sealPurpose(recoveryPointID, binding), FormatVersion)
	if err != nil {
		return nil, err
	}
	return json.Marshal(sealedEnvelope{
		Version: envelope.Version, Purpose: envelope.Purpose, KDF: sealKDF, Iterations: s.iterations(),
		Salt: base64.StdEncoding.EncodeToString(salt), Nonce: base64.StdEncoding.EncodeToString(envelope.Nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(envelope.Ciphertext),
	})
}

// Open authenticates and decrypts a sealed artifact for its slot. Any
// framing or authentication failure is reported as corrupt; the plaintext is
// never returned on failure.
func (s Sealer) Open(material, sealed []byte, recoveryPointID, binding string) ([]byte, error) {
	if len(material) == 0 {
		return nil, ErrKeyUnavailable
	}
	var envelope sealedEnvelope
	if err := json.Unmarshal(sealed, &envelope); err != nil {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s is not a valid envelope", binding)
	}
	if envelope.Version != FormatVersion || envelope.KDF != sealKDF || envelope.Iterations <= 0 {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s has an unsupported envelope policy", binding)
	}
	if !strings.EqualFold(envelope.Purpose, sealPurpose(recoveryPointID, binding)) {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s was sealed for another slot", binding)
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s: bad salt", binding)
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s: bad nonce", binding)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s: bad ciphertext", binding)
	}
	key, err := pbkdf2.Key(sha256.New, string(material), salt, envelope.Iterations, sealKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	plain, err := credentialpolicy.Open(key, credentialpolicy.Envelope{Version: envelope.Version, Purpose: envelope.Purpose, Nonce: nonce, Ciphertext: ciphertext})
	if err != nil {
		return nil, newError(CodeRecoveryPointCorrupt, "sealed artifact %s failed authentication", binding)
	}
	return plain, nil
}
