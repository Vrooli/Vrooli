package administration

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const consumerSigningBundleFormat = "lpbs-consumer-signing/v1"
const ConsumerSigningJWKSCacheMaxAge = 5 * time.Minute
const ConsumerSigningDefaultClockSkew = 30 * time.Second

// ConsumerSigningCredential stores private signer material and the public
// overlap set as one authority value, so a rotation switches signer and JWKS
// overlap together. Previous keys are public only.
type ConsumerSigningCredential struct {
	Format                 string               `json:"format"`
	KeyID                  string               `json:"key_id"`
	PrivateKeyPEM          string               `json:"private_key_pem"`
	PreviousJWKS           string               `json:"previous_jwks,omitempty"`
	PreviousKeyExpirations map[string]time.Time `json:"previous_key_expirations,omitempty"`
}

type ConsumerSigningRotationOptions struct {
	Identity           credentialauthority.Identity
	Field              string
	Scope              string
	ExpectedVersion    string
	LegacyKeyID        string
	LegacyPreviousJWKS string
	Issuer             string
	AccessTTL          time.Duration
	MaxTokenTTL        time.Duration
	ClockSkew          time.Duration
	JWKSCacheMaxAge    time.Duration
	Now                time.Time
}

type ConsumerSigningRotationResult struct {
	Receipt ConsumerSigningRotationReceipt
	state   *consumerSigningRuntimeState
}

type consumerSigningRuntimeState struct {
	signer      *consumeridentity.Signer
	current     consumeridentity.PublicKey
	previous    []consumeridentity.PublicKey
	expirations map[string]time.Time
}

var consumerSigningRotationMu sync.Mutex

// ParseConsumerSigningCredential accepts the current bundled form and the
// pre-bundle PEM form. Legacy callers provide their existing public key ID and
// overlap JWKS; parsing never writes or migrates authority state.
func ParseConsumerSigningCredential(raw, legacyKeyID, legacyPreviousJWKS string) (ConsumerSigningCredential, error) {
	var bundle ConsumerSigningCredential
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		if err := json.Unmarshal([]byte(raw), &bundle); err != nil {
			return ConsumerSigningCredential{}, fmt.Errorf("parse consumer signing credential metadata: %w", err)
		}
		if bundle.Format != consumerSigningBundleFormat || strings.TrimSpace(bundle.KeyID) == "" || strings.TrimSpace(bundle.PrivateKeyPEM) == "" {
			return ConsumerSigningCredential{}, errors.New("consumer signing credential bundle is incomplete")
		}
		if _, err := consumeridentity.ParsePrivateKeyPEM(bundle.PrivateKeyPEM); err != nil {
			return ConsumerSigningCredential{}, fmt.Errorf("parse bundled consumer signing key: %w", err)
		}
		return bundle, nil
	}
	if strings.TrimSpace(raw) == "" {
		return ConsumerSigningCredential{}, errors.New("consumer signing key is empty")
	}
	if _, err := consumeridentity.ParsePrivateKeyPEM(raw); err != nil {
		return ConsumerSigningCredential{}, fmt.Errorf("parse legacy consumer signing key: %w", err)
	}
	legacyKeyID = strings.TrimSpace(legacyKeyID)
	if legacyKeyID == "" {
		legacyKeyID = "local-development"
	}
	return ConsumerSigningCredential{Format: consumerSigningBundleFormat, KeyID: legacyKeyID, PrivateKeyPEM: raw, PreviousJWKS: strings.TrimSpace(legacyPreviousJWKS)}, nil
}

// RotateConsumerSigningCredential rotates only a synthetic or explicitly
// authorized authority value supplied by the caller. It stages a candidate,
// checks the observed active version again, then activates the new private key
// and public overlap set atomically as one credential value. Receipt contains
// identifiers only; key material is never returned.
func RotateConsumerSigningCredential(authority *credentialauthority.Authority, opts ConsumerSigningRotationOptions) (ConsumerSigningRotationResult, error) {
	consumerSigningRotationMu.Lock()
	defer consumerSigningRotationMu.Unlock()
	if opts.AccessTTL <= 0 || strings.TrimSpace(opts.Issuer) == "" || opts.Identity != credentialAuthorityIdentity || strings.TrimSpace(opts.Field) != consumerSigningAuthorityField || strings.TrimSpace(opts.Scope) != ConsumerSigningAuthorityScope {
		return ConsumerSigningRotationResult{}, errors.New("consumer signing requires the LPBS signer binding, issuer, and positive TTL")
	}
	if opts.MaxTokenTTL < opts.AccessTTL {
		opts.MaxTokenTTL = opts.AccessTTL
	}
	if opts.ClockSkew <= 0 {
		opts.ClockSkew = ConsumerSigningDefaultClockSkew
	}
	if opts.JWKSCacheMaxAge <= 0 {
		opts.JWKSCacheMaxAge = ConsumerSigningJWKSCacheMaxAge
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	before := authority.Status(opts.Identity, opts.Field)
	if string(before.ProviderState) != "available" || !before.Configured {
		return ConsumerSigningRotationResult{}, errors.New("active consumer signing credential is unavailable")
	}
	if before.Version != opts.ExpectedVersion {
		return ConsumerSigningRotationResult{}, fmt.Errorf("%w: active consumer signing version changed", credentialauthority.ErrVersionConflict)
	}
	value, err := authority.Resolve(opts.Identity, opts.Field)
	if err != nil {
		switch {
		case errors.Is(err, credentialauthority.ErrUnconfigured):
			return ConsumerSigningRotationResult{}, errors.New("active consumer signing credential is not configured")
		case errors.Is(err, credentialauthority.ErrProviderUnavailable):
			return ConsumerSigningRotationResult{}, errors.New("credential provider is unavailable")
		case errors.Is(err, credentialauthority.ErrProviderAbsent):
			return ConsumerSigningRotationResult{}, errors.New("credential provider is absent")
		default:
			return ConsumerSigningRotationResult{}, errors.New("active consumer signing credential could not be resolved")
		}
	}
	after := authority.Status(opts.Identity, opts.Field)
	if string(after.ProviderState) != "available" || !after.Configured || after.Version != opts.ExpectedVersion {
		return ConsumerSigningRotationResult{}, fmt.Errorf("%w: active consumer signing version changed during resolution", credentialauthority.ErrVersionConflict)
	}
	current, err := ParseConsumerSigningCredential(value, opts.LegacyKeyID, opts.LegacyPreviousJWKS)
	if err != nil {
		return ConsumerSigningRotationResult{}, err
	}
	oldSigner, err := consumeridentity.ParsePrivateKeyPEM(current.PrivateKeyPEM)
	if err != nil {
		return ConsumerSigningRotationResult{}, errors.New("active consumer signing key could not be parsed")
	}
	newKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("generate candidate consumer signing key: %w", err)
	}
	newKeyID, err := newConsumerSigningKeyID()
	if err != nil {
		return ConsumerSigningRotationResult{}, err
	}
	privatePEM, err := consumeridentity.MarshalPrivateKeyPEM(newKey)
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("encode candidate consumer signing key: %w", err)
	}
	previous := consumeridentity.NewKeySet()
	// Consumer access tokens and entitlement leases share this signer. Keep
	// overlap until the longest token can no longer be valid, plus verifier
	// clock skew and the public JWKS cache max-age.
	retention := opts.MaxTokenTTL + opts.ClockSkew + opts.JWKSCacheMaxAge
	expiresAt := opts.Now.Add(retention).UTC()
	expirations := make(map[string]time.Time, len(current.PreviousKeyExpirations)+1)
	if strings.TrimSpace(current.PreviousJWKS) != "" {
		prior, parseErr := consumeridentity.ParseJWKS([]byte(current.PreviousJWKS))
		if parseErr != nil {
			return ConsumerSigningRotationResult{}, fmt.Errorf("parse existing public overlap set: %w", parseErr)
		}
		for id, key := range prior.Keys {
			keyExpiry := current.PreviousKeyExpirations[id]
			if keyExpiry.IsZero() {
				// Legacy public overlap has no timestamp; retain it for one bounded
				// maximum access-token lifetime from the migration/rotation point.
				keyExpiry = expiresAt
			}
			if !opts.Now.Before(keyExpiry) {
				continue
			}
			previous.Add(consumeridentity.PublicKey{ID: id, Key: key})
			expirations[id] = keyExpiry.UTC()
		}
	}
	previous.Add(consumeridentity.PublicKey{ID: current.KeyID, Key: &oldSigner.PublicKey})
	expirations[current.KeyID] = expiresAt
	previousJWKS, err := previous.JWKS()
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("encode public overlap set: %w", err)
	}
	bundle := ConsumerSigningCredential{Format: consumerSigningBundleFormat, KeyID: newKeyID, PrivateKeyPEM: string(privatePEM), PreviousJWKS: string(previousJWKS), PreviousKeyExpirations: expirations}
	serialized, err := json.Marshal(bundle)
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("encode candidate signing bundle: %w", err)
	}
	prepared, err := prepareConsumerSigningRuntime(bundle, opts.Issuer, opts.AccessTTL)
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("prepare runtime signer: %w", err)
	}
	if status := authority.Status(opts.Identity, opts.Field); !status.Configured || status.Version != opts.ExpectedVersion || string(status.ProviderState) != "available" {
		return ConsumerSigningRotationResult{}, fmt.Errorf("%w: active consumer signing credential changed before candidate activation", credentialauthority.ErrVersionConflict)
	}
	candidate, err := authority.PutCandidate(opts.Identity, opts.Field, string(serialized))
	if err != nil {
		return ConsumerSigningRotationResult{}, fmt.Errorf("stage consumer signing candidate: %w", err)
	}
	if status := authority.Status(opts.Identity, opts.Field); !status.Configured || status.Version != opts.ExpectedVersion || string(status.ProviderState) != "available" {
		_ = authority.RejectCandidate(candidate)
		return ConsumerSigningRotationResult{}, fmt.Errorf("%w: active consumer signing credential changed while candidate was staged", credentialauthority.ErrVersionConflict)
	}
	cleanupPending := false
	if err := authority.ActivateCandidateIfVersion(candidate, opts.ExpectedVersion); err != nil {
		if errors.Is(err, credentialauthority.ErrCandidateCleanupPending) {
			cleanupPending = true
		} else {
			return ConsumerSigningRotationResult{}, fmt.Errorf("activate consumer signing candidate: %w", err)
		}
	}
	receipt := ConsumerSigningRotationReceipt{PreviousKeyID: current.KeyID, ActiveKeyID: newKeyID, Version: candidate.Version, OverlapKeyCount: len(previous.Keys), PreviousKeysExpireAt: expiresAt, CleanupPending: cleanupPending}
	return ConsumerSigningRotationResult{Receipt: receipt, state: prepared}, nil
}

type ConsumerSigningRotationReceipt struct {
	PreviousKeyID        string    `json:"previous_key_id"`
	ActiveKeyID          string    `json:"active_key_id"`
	Version              string    `json:"version"`
	OverlapKeyCount      int       `json:"overlap_key_count"`
	PreviousKeysExpireAt time.Time `json:"previous_keys_expire_at"`
	CleanupPending       bool      `json:"candidate_cleanup_pending"`
}

func newConsumerSigningKeyID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate consumer signing key ID: %w", err)
	}
	return fmt.Sprintf("lpbs-%x", bytes[:]), nil
}

func cloneKeyExpirations(source map[string]time.Time) map[string]time.Time {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]time.Time, len(source))
	for keyID, expires := range source {
		clone[keyID] = expires
	}
	return clone
}

func prepareConsumerSigningRuntime(bundle ConsumerSigningCredential, issuer string, accessTTL time.Duration) (*consumerSigningRuntimeState, error) {
	privateKey, err := consumeridentity.ParsePrivateKeyPEM(bundle.PrivateKeyPEM)
	if err != nil {
		return nil, errors.New("candidate consumer signing key could not be parsed")
	}
	signer, err := consumeridentity.NewSigner(bundle.KeyID, privateKey, issuer, accessTTL)
	if err != nil {
		return nil, errors.New("candidate consumer signing key is invalid")
	}
	previous := make([]consumeridentity.PublicKey, 0)
	if strings.TrimSpace(bundle.PreviousJWKS) != "" {
		set, parseErr := consumeridentity.ParseJWKS([]byte(bundle.PreviousJWKS))
		if parseErr != nil {
			return nil, errors.New("candidate consumer overlap key set is invalid")
		}
		for id, key := range set.Keys {
			previous = append(previous, consumeridentity.PublicKey{ID: id, Key: key})
		}
	}
	return &consumerSigningRuntimeState{signer: signer, current: consumeridentity.PublicKey{ID: signer.KeyID, Key: &privateKey.PublicKey}, previous: previous, expirations: cloneKeyExpirations(bundle.PreviousKeyExpirations)}, nil
}

// ApplyTo switches the in-process signer and verifier set only after authority
// candidate activation has succeeded. The returned receipt is metadata-only.
func (r ConsumerSigningRotationResult) ApplyTo(service *UserAuthService) {
	if service == nil || r.state == nil {
		return
	}
	service.consumerReloadMu.Lock()
	defer service.consumerReloadMu.Unlock()
	service.applyConsumerSigningRuntimeLocked(r.Receipt.Version, r.state)
}

func (service *UserAuthService) applyConsumerSigningRuntimeLocked(version string, state *consumerSigningRuntimeState) {
	if service == nil || state == nil {
		return
	}
	service.consumerKeyMu.Lock()
	service.consumerSigner = state.signer
	service.consumerCurrentPublic = state.current
	service.consumerPreviousKeys = append([]consumeridentity.PublicKey(nil), state.previous...)
	service.consumerPreviousKeyExpirations = cloneKeyExpirations(state.expirations)
	service.consumerSigningVersion = version
	service.consumerKeys = service.consumerKeySetAtLocked(time.Now().UTC())
	service.consumerKeyMu.Unlock()
}
