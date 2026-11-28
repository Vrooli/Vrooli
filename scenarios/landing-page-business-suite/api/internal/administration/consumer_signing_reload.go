package administration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const consumerSigningAuthorityField = "consumer-auth-private-key"
const ConsumerSigningAuthorityScope = "signing:consumer-token"

const consumerSigningDefaultIssuer = "landing-page-business-suite"

// ConsumerSigningIssuer applies the same issuer default at startup and during
// admin rotation, preventing a rotation from creating keys for a different
// issuer than the running verifier.
func ConsumerSigningIssuer(configured string) string {
	if issuer := strings.TrimSpace(configured); issuer != "" {
		return issuer
	}
	return consumerSigningDefaultIssuer
}

// AuthorityConsumerSigningSource resolves one exact authority identity and
// scope. It is a process-local reader for the shared authority backend; each
// LPBS replica refreshes its own in-memory keyset before signing, validating,
// or publishing JWKS. The authority value never leaves this package.
type AuthorityConsumerSigningSource struct {
	authority      *credentialauthority.Authority
	identity       credentialauthority.Identity
	field          string
	scope          string
	legacyKeyID    string
	legacyPrevious string
	issuer         string
	accessTTL      time.Duration
}

// NewAuthorityConsumerSigningSource creates a reload source for the LPBS
// consumer-token signer. It accepts only the established identity, field,
// and signing scope so a caller cannot silently bind a different credential.
func NewAuthorityConsumerSigningSource(authority *credentialauthority.Authority, identity credentialauthority.Identity, field, scope, legacyKeyID, legacyPreviousJWKS, issuer string, accessTTL time.Duration) (*AuthorityConsumerSigningSource, error) {
	if authority == nil || identity != credentialAuthorityIdentity || strings.TrimSpace(field) != consumerSigningAuthorityField || strings.TrimSpace(scope) != ConsumerSigningAuthorityScope {
		return nil, errors.New("consumer signing reload requires the LPBS signer identity, field, and scope")
	}
	if strings.TrimSpace(issuer) == "" || accessTTL <= 0 {
		return nil, errors.New("consumer signing reload requires issuer and positive access-token TTL")
	}
	return &AuthorityConsumerSigningSource{
		authority: authority, identity: identity, field: field, scope: scope,
		legacyKeyID: legacyKeyID, legacyPrevious: legacyPreviousJWKS,
		issuer: issuer, accessTTL: accessTTL,
	}, nil
}

func (source *AuthorityConsumerSigningSource) load() (string, *consumerSigningRuntimeState, string, int, error) {
	if source == nil || source.authority == nil {
		return "", nil, "", 0, errors.New("consumer signing authority source is unavailable")
	}
	status := source.authority.Status(source.identity, source.field)
	if string(status.ProviderState) != "available" || !status.Configured || strings.TrimSpace(status.Version) == "" {
		return "", nil, "", 0, errors.New("consumer signing authority version is unavailable")
	}
	binding, err := ResolveBoundCredentialFromAuthority(source.authority, source.identity, source.field, source.scope)
	if err != nil {
		return "", nil, "", 0, errors.New("consumer signing authority value could not be resolved")
	}
	if binding.Version != status.Version {
		return "", nil, "", 0, fmt.Errorf("%w: consumer signing authority changed during reload", credentialauthority.ErrVersionConflict)
	}
	bundle, err := ParseConsumerSigningCredential(binding.Value, source.legacyKeyID, source.legacyPrevious)
	if err != nil {
		return "", nil, "", 0, errors.New("consumer signing authority bundle is invalid")
	}
	state, err := prepareConsumerSigningRuntime(bundle, source.issuer, source.accessTTL)
	if err != nil {
		return "", nil, "", 0, errors.New("consumer signing authority bundle could not be prepared")
	}
	return binding.Version, state, bundle.KeyID, len(state.previous), nil
}

func (source *AuthorityConsumerSigningSource) activeVersion() (string, error) {
	if source == nil || source.authority == nil {
		return "", errors.New("consumer signing authority source is unavailable")
	}
	status := source.authority.Status(source.identity, source.field)
	if string(status.ProviderState) != "available" || !status.Configured || strings.TrimSpace(status.Version) == "" {
		return "", errors.New("consumer signing authority version is unavailable")
	}
	return status.Version, nil
}

// ConsumerSigningReloadReceipt contains no secret material. A false Changed
// value means this replica already had the active authority version.
type ConsumerSigningReloadReceipt struct {
	Changed         bool   `json:"changed"`
	Version         string `json:"version,omitempty"`
	ActiveKeyID     string `json:"active_key_id,omitempty"`
	OverlapKeyCount int    `json:"overlap_key_count"`
}

// RefreshConsumerSigningKeyset loads and atomically applies a newer active
// signer bundle. The reload mutex also orders admin rotation's in-process
// ApplyTo call against concurrent reads, preventing a delayed stale snapshot
// from overwriting a newer local signer after authority activation.
func (s *UserAuthService) RefreshConsumerSigningKeyset() (ConsumerSigningReloadReceipt, error) {
	if s == nil || s.consumerSigningSource == nil {
		return ConsumerSigningReloadReceipt{}, errors.New("consumer signing reload is not configured")
	}
	s.consumerReloadMu.Lock()
	defer s.consumerReloadMu.Unlock()
	s.consumerKeyMu.RLock()
	activeVersion := s.consumerSigningVersion
	s.consumerKeyMu.RUnlock()
	observedVersion, err := s.consumerSigningSource.activeVersion()
	if err != nil {
		return ConsumerSigningReloadReceipt{}, err
	}
	if observedVersion == activeVersion {
		return ConsumerSigningReloadReceipt{Version: observedVersion}, nil
	}
	version, state, keyID, overlapCount, err := s.consumerSigningSource.load()
	if err != nil {
		return ConsumerSigningReloadReceipt{}, err
	}
	if version != observedVersion {
		return ConsumerSigningReloadReceipt{}, fmt.Errorf("%w: consumer signing authority changed while preparing reload", credentialauthority.ErrVersionConflict)
	}
	s.applyConsumerSigningRuntimeLocked(version, state)
	return ConsumerSigningReloadReceipt{Changed: true, Version: version, ActiveKeyID: keyID, OverlapKeyCount: overlapCount}, nil
}

func (s *UserAuthService) refreshConsumerSigningKeyset() error {
	if s == nil || s.consumerSigningSource == nil {
		return nil
	}
	_, err := s.RefreshConsumerSigningKeyset()
	return err
}
