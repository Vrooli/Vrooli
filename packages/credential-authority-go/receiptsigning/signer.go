// Package credentialauthoritysigning implements trusted receipt signing over
// Vrooli's backend-neutral credential authority. The private key is held as a
// credential-authority value; this package never writes key material to a
// scenario file, environment variable, or Vault.
package credentialauthoritysigning

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	defaultField     = "key-ring"
	keyRingVersion   = 1
	keyIDSeparator   = ":"
	keyVersionPrefix = "v"
	signingDomain    = "vrooli.receipt-signing.ed25519.v1\x00"
	providerName     = "credential-authority-ed25519"
)

var allowedPurposes = []receiptsigning.Purpose{
	receiptsigning.PurposeExperimentAuditReceipt,
	receiptsigning.PurposeExperimentHoldoutReceipt,
}

// Store is the smallest authority surface required by the signer. It keeps
// the signer straightforward to test without allowing callers to substitute a
// plaintext or environment-backed implementation in production code.
type Store interface {
	Resolve(credentialauthority.Identity, string) (string, error)
	Put(credentialauthority.Identity, string, string) error
}

type Config struct {
	Identity        credentialauthority.Identity
	Field           string
	Store           Store
	AllowedPurposes []receiptsigning.Purpose
}

// Signer signs new receipts with the active Ed25519 key and verifies any
// retained key version in the authority key ring. Rotation appends a key; it
// never destroys old public verifiers, so existing evidence remains valid.
type Signer struct {
	identity credentialauthority.Identity
	field    string
	store    Store
	allowed  map[receiptsigning.Purpose]struct{}
	rotateMu sync.Mutex
}

type keyRing struct {
	Version int       `json:"version"`
	Active  string    `json:"active"`
	Keys    []keyPair `json:"keys"`
}

type keyPair struct {
	ID         string `json:"id"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	CreatedAt  string `json:"createdAt"`
}

func New(config Config) (*Signer, error) {
	identity, err := credentialauthority.ParseIdentity(string(config.Identity))
	if err != nil {
		return nil, err
	}
	field := strings.TrimSpace(config.Field)
	if field == "" {
		field = defaultField
	}
	if strings.ContainsAny(field, "/\\") {
		return nil, fmt.Errorf("receipt signing authority field cannot contain a path separator")
	}
	if config.Store == nil {
		return nil, fmt.Errorf("receipt signing credential authority is unavailable")
	}
	if len(config.AllowedPurposes) == 0 {
		config.AllowedPurposes = allowedPurposes
	}
	allowed := make(map[receiptsigning.Purpose]struct{}, len(config.AllowedPurposes))
	for _, purpose := range config.AllowedPurposes {
		if !purpose.Valid() {
			return nil, fmt.Errorf("unsupported receipt signing purpose %q", purpose)
		}
		allowed[purpose] = struct{}{}
	}
	return &Signer{identity: identity, field: field, store: config.Store, allowed: allowed}, nil
}

func NewDefault(identity string, field string) (*Signer, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return nil, err
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, fmt.Errorf("create credential authority: %w", err)
	}
	return New(Config{Identity: parsed, Field: field, Store: authority})
}

func (s *Signer) Sign(_ context.Context, purpose receiptsigning.Purpose, canonical []byte) (receiptsigning.SignatureEnvelope, error) {
	if err := s.checkPurpose(purpose); err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	ring, err := s.load()
	if err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	active, err := ring.activeKey()
	if err != nil {
		return receiptsigning.SignatureEnvelope{}, err
	}
	privateKey, err := decodeKey(active.PrivateKey, ed25519.PrivateKeySize)
	if err != nil {
		return receiptsigning.SignatureEnvelope{}, fmt.Errorf("decode receipt signing private key: %w", err)
	}
	digest := receiptsigning.Digest(canonical)
	signature := ed25519.Sign(ed25519.PrivateKey(privateKey), signingMessage(purpose, digest))
	return receiptsigning.SignatureEnvelope{
		Version:   receiptsigning.EnvelopeVersionV1,
		Purpose:   purpose,
		Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519,
		KeyID:     s.keyID(active.ID),
		Digest:    digest,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

func (s *Signer) Verify(_ context.Context, envelope receiptsigning.SignatureEnvelope, canonical []byte) error {
	if err := envelope.Validate(); err != nil {
		return err
	}
	if envelope.Algorithm != receiptsigning.AlgorithmCredentialAuthorityEd25519 {
		return fmt.Errorf("receipt signer cannot verify algorithm %q", envelope.Algorithm)
	}
	if err := s.checkPurpose(envelope.Purpose); err != nil {
		return err
	}
	digest := receiptsigning.Digest(canonical)
	if envelope.Digest != digest {
		return fmt.Errorf("receipt digest does not match canonical content")
	}
	version, ok := strings.CutPrefix(envelope.KeyID, s.identityString()+keyIDSeparator)
	if !ok || version == "" {
		return fmt.Errorf("receipt signature key ID does not belong to this authority identity")
	}
	ring, err := s.load()
	if err != nil {
		return err
	}
	var key *keyPair
	for i := range ring.Keys {
		if ring.Keys[i].ID == version {
			key = &ring.Keys[i]
			break
		}
	}
	if key == nil {
		return fmt.Errorf("receipt signature key version %q is not retained", version)
	}
	publicKey, err := decodeKey(key.PublicKey, ed25519.PublicKeySize)
	if err != nil {
		return fmt.Errorf("decode receipt signing public key: %w", err)
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("receipt signature is not a valid Ed25519 signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), signingMessage(envelope.Purpose, digest), signature) {
		return fmt.Errorf("receipt signature verification failed")
	}
	return nil
}

func (s *Signer) Health(_ context.Context) (receiptsigning.Health, error) {
	ring, err := s.load()
	if err != nil {
		return receiptsigning.Health{Provider: providerName, Production: true, RotationOK: true}, err
	}
	active, err := ring.activeKey()
	if err != nil {
		return receiptsigning.Health{Provider: providerName, Production: true, RotationOK: true}, err
	}
	return receiptsigning.Health{
		Ready:       true,
		Provider:    providerName,
		KeyID:       s.keyID(active.ID),
		Production:  true,
		RotationOK:  true,
		Description: "Ed25519 receipt key is retained by the Vrooli credential authority",
	}, nil
}

// Rotate creates the initial key when the authority field is absent, or
// appends a new version while retaining every historical verifier.
func (s *Signer) Rotate(_ context.Context) (receiptsigning.Health, error) {
	s.rotateMu.Lock()
	defer s.rotateMu.Unlock()

	ring, err := s.loadOptional()
	if err != nil {
		return receiptsigning.Health{}, err
	}
	next := 1
	for _, key := range ring.Keys {
		if version, parseErr := strconv.Atoi(strings.TrimPrefix(key.ID, keyVersionPrefix)); parseErr == nil && version >= next {
			next = version + 1
		}
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return receiptsigning.Health{}, fmt.Errorf("generate receipt signing key: %w", err)
	}
	version := keyVersionPrefix + strconv.Itoa(next)
	ring.Version = keyRingVersion
	ring.Active = version
	ring.Keys = append(ring.Keys, keyPair{
		ID:         version,
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err := validateRing(ring); err != nil {
		return receiptsigning.Health{}, err
	}
	encoded, err := json.Marshal(ring)
	if err != nil {
		return receiptsigning.Health{}, fmt.Errorf("encode receipt signing key ring: %w", err)
	}
	if err := s.store.Put(s.identity, s.field, string(encoded)); err != nil {
		return receiptsigning.Health{}, fmt.Errorf("store receipt signing key ring: %w", err)
	}
	return s.Health(context.Background())
}

func (s *Signer) checkPurpose(purpose receiptsigning.Purpose) error {
	if _, ok := s.allowed[purpose]; !ok {
		return fmt.Errorf("receipt signing purpose %q is not allowed", purpose)
	}
	return nil
}

func (s *Signer) load() (keyRing, error) {
	ring, err := s.loadOptional()
	if err != nil {
		return keyRing{}, err
	}
	if len(ring.Keys) == 0 {
		return keyRing{}, credentialauthority.ErrUnconfigured
	}
	if err := validateRing(ring); err != nil {
		return keyRing{}, err
	}
	return ring, nil
}

func (s *Signer) loadOptional() (keyRing, error) {
	encoded, err := s.store.Resolve(s.identity, s.field)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return keyRing{Version: keyRingVersion}, nil
		}
		return keyRing{}, fmt.Errorf("resolve receipt signing key ring: %w", err)
	}
	var ring keyRing
	if err := json.Unmarshal([]byte(encoded), &ring); err != nil {
		return keyRing{}, fmt.Errorf("parse receipt signing key ring: %w", err)
	}
	return ring, nil
}

func (s *Signer) identityString() string { return string(s.identity) }

func (s *Signer) keyID(version string) string { return s.identityString() + keyIDSeparator + version }

func (ring keyRing) activeKey() (*keyPair, error) {
	for i := range ring.Keys {
		if ring.Keys[i].ID == ring.Active {
			return &ring.Keys[i], nil
		}
	}
	return nil, fmt.Errorf("receipt signing key ring has no active key")
}

func validateRing(ring keyRing) error {
	if ring.Version != keyRingVersion || strings.TrimSpace(ring.Active) == "" || len(ring.Keys) == 0 {
		return fmt.Errorf("receipt signing key ring is invalid")
	}
	seen := make(map[string]struct{}, len(ring.Keys))
	for _, key := range ring.Keys {
		if key.ID == "" {
			return fmt.Errorf("receipt signing key ring contains an empty key ID")
		}
		if _, exists := seen[key.ID]; exists {
			return fmt.Errorf("receipt signing key ring contains duplicate key ID %q", key.ID)
		}
		seen[key.ID] = struct{}{}
		privateKey, err := decodeKey(key.PrivateKey, ed25519.PrivateKeySize)
		if err != nil {
			return fmt.Errorf("receipt signing key %q has invalid private key: %w", key.ID, err)
		}
		publicKey, err := decodeKey(key.PublicKey, ed25519.PublicKeySize)
		if err != nil {
			return fmt.Errorf("receipt signing key %q has invalid public key: %w", key.ID, err)
		}
		derived := ed25519.PrivateKey(privateKey).Public().(ed25519.PublicKey)
		if string(derived) != string(publicKey) {
			return fmt.Errorf("receipt signing key %q has mismatched public key", key.ID)
		}
	}
	if _, ok := seen[ring.Active]; !ok {
		return fmt.Errorf("receipt signing key ring active key %q is not retained", ring.Active)
	}
	return nil
}

func decodeKey(value string, size int) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) != size {
		return nil, fmt.Errorf("expected base64 key of %d bytes", size)
	}
	return decoded, nil
}

func signingMessage(purpose receiptsigning.Purpose, digest string) []byte {
	return []byte(signingDomain + string(purpose) + "\x00" + digest)
}

// CompatibleSigner uses the authority signer for all new signatures and
// local verification, while routing explicitly marked historical Vault Transit
// envelopes to an operator-supplied legacy verifier when one is configured.
// No Vault client or lifecycle dependency is created unless that compatibility
// configuration is explicitly present.
type CompatibleSigner struct {
	Primary receiptsigning.ReceiptSigner
	Legacy  receiptsigning.ReceiptSigner
}

func (s CompatibleSigner) Sign(ctx context.Context, purpose receiptsigning.Purpose, canonical []byte) (receiptsigning.SignatureEnvelope, error) {
	return s.Primary.Sign(ctx, purpose, canonical)
}

func (s CompatibleSigner) Verify(ctx context.Context, envelope receiptsigning.SignatureEnvelope, canonical []byte) error {
	if envelope.Algorithm == receiptsigning.AlgorithmVaultTransit {
		if s.Legacy == nil {
			return fmt.Errorf("historical Vault Transit receipt verification is not configured")
		}
		return s.Legacy.Verify(ctx, envelope, canonical)
	}
	return s.Primary.Verify(ctx, envelope, canonical)
}

func (s CompatibleSigner) Health(ctx context.Context) (receiptsigning.Health, error) {
	return s.Primary.Health(ctx)
}

func (s CompatibleSigner) Rotate(ctx context.Context) (receiptsigning.Health, error) {
	rotator, ok := s.Primary.(interface {
		Rotate(context.Context) (receiptsigning.Health, error)
	})
	if !ok {
		return receiptsigning.Health{}, fmt.Errorf("active receipt signer does not support rotation")
	}
	return rotator.Rotate(ctx)
}

// NewSignerFromRuntimeConfig constructs the authority signer and, when
// explicitly requested, attaches the old Vault Transit verifier for historical
// envelopes. The active signing path remains authority-only.
func NewSignerFromRuntimeConfig(config receiptsigning.RuntimeConfig) (receiptsigning.ReceiptSigner, bool, error) {
	if err := config.Validate(); err != nil {
		return nil, false, err
	}
	if config.Mode != receiptsigning.ModeCredentialAuthorityEd25519 {
		return nil, false, fmt.Errorf("credential-authority binding requires mode %q", receiptsigning.ModeCredentialAuthorityEd25519)
	}
	authorityConfig := config.CredentialAuthority
	identity, err := credentialauthority.ParseIdentity(authorityConfig.Identity)
	if err != nil {
		return nil, false, err
	}
	primary, err := NewDefault(string(identity), authorityConfig.Field)
	if err != nil {
		return nil, false, err
	}
	if config.LegacyVaultTransit == nil {
		return primary, true, nil
	}
	legacy := config.LegacyVaultTransit
	legacySigner, err := receiptsigning.NewVaultTransitSigner(receiptsigning.VaultTransitConfig{
		Address:         legacy.Address,
		KeyName:         legacy.KeyName,
		Credentials:     receiptsigning.FileCredentialSource{Path: legacy.CredentialFile},
		AllowedPurposes: allowedPurposes,
	})
	if err != nil {
		return nil, false, err
	}
	return CompatibleSigner{Primary: primary, Legacy: legacySigner}, true, nil
}
