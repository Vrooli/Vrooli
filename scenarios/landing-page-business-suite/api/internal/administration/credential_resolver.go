package administration

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/envx"
)

const credentialAuthorityIdentity = credentialauthority.Identity("vrooli/landing-page-business-suite")

var credentialWitness credentialauthority.Witness

// CredentialBinding is the non-secret identity/version evidence paired with
// a resolved value. Scope is policy metadata declared alongside the consumer
// in service.json; it is never inferred from the value itself.
type CredentialBinding struct {
	Identity credentialauthority.Identity
	Field    string
	Version  string
	Scope    string
	Value    string
}

var authorityProvider = struct {
	sync.RWMutex
	open func() (*credentialauthority.Authority, error)
}{open: credentialauthority.Default}

// SetCredentialAuthorityProvider replaces the default host authority only at
// the composition boundary. Local rehearsal builds use this to share one
// encrypted, synthetic authority with Treasury; normal builds retain Default.
func SetCredentialAuthorityProvider(open func() (*credentialauthority.Authority, error)) {
	authorityProvider.Lock()
	defer authorityProvider.Unlock()
	if open == nil {
		authorityProvider.open = credentialauthority.Default
		return
	}
	authorityProvider.open = open
}

func openCredentialAuthority() (*credentialauthority.Authority, error) {
	authorityProvider.RLock()
	open := authorityProvider.open
	authorityProvider.RUnlock()
	return open()
}

// OpenCredentialAuthority returns the configured authority composition for
// operator workflows that must perform a candidate transaction. Callers must
// never expose resolved values; the normal consumer path remains the scoped
// binding resolver below.
func OpenCredentialAuthority() (*credentialauthority.Authority, error) {
	return openCredentialAuthority()
}

// SetCredentialWitness installs the data-owned witness used by generated
// credential resolution. Composition owns the database-backed witness; this
// package owns the resolver policy.
func SetCredentialWitness(witness credentialauthority.Witness) {
	credentialWitness = witness
}

// AuthorityFieldForKey maps legacy environment names to the stable authority
// field identity used by manifests, recovery, and deployment tooling.
func AuthorityFieldForKey(key string) string {
	field := strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(strings.TrimSpace(key)))
	switch key {
	case "SESSION_SECRET":
		return "session-secret"
	case "SESSION_SECRET_PREVIOUS":
		return "session-secret-previous"
	case "LPBS_" + "SERVICE_SECRET":
		return "service-secret"
	case "CONSUMER_AUTH_PRIVATE_KEY":
		return "consumer-auth-private-key"
	case "LPBS_API_KEY_ENCRYPTION_KEY":
		return "api-key-encryption-key"
	case "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY":
		return "remote-profile-encryption-key"
	case "LPBS_ADMIN_MFA_ENCRYPTION_KEY":
		return "admin-mfa-encryption-key"
	case "ADMIN_DEFAULT_PASSWORD":
		return "admin-default-password"
	case "SENDGRID_WEBHOOK_PUBLIC_KEY":
		return "sendgrid-webhook-public-key"
	case "MAILGUN_API_KEY":
		return "mailgun-api-key"
	case "SMTP_PASSWORD":
		return "smtp-password"
	}
	return field
}

func AuthorityScopeForField(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-test-publishable-key":
		return "stripe:test:payments"
	case "stripe-live-secret-key", "stripe-live-webhook-secret", "stripe-live-publishable-key":
		return "stripe:live:payments"
	case "delivery-s3-access-key-id", "delivery-s3-secret-access-key", "delivery-s3-session-token":
		return "s3:delivery:object-read-write"
	case "smtp-password", "sendgrid-api-key", "mailgun-api-key":
		return "mail:transactional:send"
	case "google-play-oauth-token":
		return "google-play:receipt-verify"
	case "sendgrid-webhook-public-key":
		return "mail:transactional:webhook-verify"
	case "session-secret", "session-secret-previous":
		return "signing:session-cookie"
	case "consumer-auth-private-key":
		return "signing:consumer-token"
	case "service-secret":
		return "service:lpbs-bootstrap"
	case "metrics-reader-token":
		return "service:metrics-read"
	case "admin-default-password":
		return "service:admin-bootstrap"
	case "admin-mfa-encryption-key":
		return "signing:admin-mfa-encryption"
	case "api-key-encryption-key":
		return "signing:provider-key-encryption"
	case "remote-profile-encryption-key":
		return "signing:remote-profile-encryption"
	default:
		return ""
	}
}

func isRetiredStripeCredential(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "stripe-secret-key", "stripe-webhook-secret", "stripe-publishable-key":
		return true
	default:
		return false
	}
}

// ResolveAuthorityCredential requires a value from the authority. The test
// fallback exists only for isolated package fixtures and still reports an
// empty value as ErrUnconfigured.
func ResolveAuthorityCredential(key string) (string, error) {
	binding, err := ResolveCredentialBinding(key)
	if err != nil {
		return "", err
	}
	return binding.Value, nil
}

// ResolveCredentialBinding binds a value to an active authority version.
// Version metadata is checked on both sides of the read; a concurrent
// rotation fails closed instead of returning an un-attributed credential.
func ResolveCredentialBinding(key string) (CredentialBinding, error) {
	field := AuthorityFieldForKey(key)
	scope := AuthorityScopeForField(field)
	if scope == "" {
		return CredentialBinding{}, fmt.Errorf("credential %s has no declared consumer scope", field)
	}
	if envx.Get("LPBS_TEST_CREDENTIAL_FALLBACK") == "1" {
		value := strings.TrimSpace(envx.Get(key))
		if value == "" {
			return CredentialBinding{}, fmt.Errorf("%w: %s", credentialauthority.ErrUnconfigured, key)
		}
		return CredentialBinding{Identity: credentialAuthorityIdentity, Field: field, Version: "isolated-test-fixture", Scope: scope, Value: value}, nil
	}
	return ResolveBoundCredential(credentialAuthorityIdentity, field, scope)
}

// ResolveBoundCredential resolves an authority credential owned by LPBS or
// issued to an LPBS service principal. Callers must supply the exact scope
// declared for that identity/field pair.
func ResolveBoundCredential(identity credentialauthority.Identity, field, scope string) (CredentialBinding, error) {
	field = strings.TrimSpace(field)
	if identity == "" || field == "" || strings.TrimSpace(scope) == "" {
		return CredentialBinding{}, fmt.Errorf("credential identity, field, and consumer scope are required")
	}
	if isRetiredStripeCredential(field) {
		return CredentialBinding{}, fmt.Errorf("legacy Stripe credential %s is retired; select the mode-specific Stripe field", field)
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return CredentialBinding{}, fmt.Errorf("initialize credential authority: %w", err)
	}
	return ResolveBoundCredentialFromAuthority(authority, identity, field, scope)
}

// ResolveBoundCredentialFromAuthority is for isolated compositions such as
// the tagged rehearsal. It keeps the same identity/version checks while
// allowing the caller to supply its explicitly scoped fake authority.
func ResolveBoundCredentialFromAuthority(authority *credentialauthority.Authority, identity credentialauthority.Identity, field, scope string) (CredentialBinding, error) {
	field = strings.TrimSpace(field)
	if authority == nil || identity == "" || field == "" || strings.TrimSpace(scope) == "" {
		return CredentialBinding{}, fmt.Errorf("credential authority, identity, field, and consumer scope are required")
	}
	if isRetiredStripeCredential(field) {
		return CredentialBinding{}, fmt.Errorf("legacy Stripe credential %s is retired; select the mode-specific Stripe field", field)
	}
	before := authority.Status(identity, field)
	if string(before.ProviderState) != "available" {
		return CredentialBinding{}, fmt.Errorf("credential authority status for %s: %s", field, before.ProviderDetail)
	}
	if !before.Configured {
		return CredentialBinding{}, fmt.Errorf("%w: %s", credentialauthority.ErrUnconfigured, field)
	}
	if strings.TrimSpace(before.Version) == "" {
		return CredentialBinding{}, fmt.Errorf("credential %s has no active version metadata; activate a versioned candidate", field)
	}
	value, err := authority.Resolve(identity, field)
	if err != nil {
		switch {
		case errors.Is(err, credentialauthority.ErrUnconfigured):
			return CredentialBinding{}, fmt.Errorf("%w: %s", credentialauthority.ErrUnconfigured, field)
		case errors.Is(err, credentialauthority.ErrProviderUnavailable):
			return CredentialBinding{}, fmt.Errorf("credential provider unavailable for %s", field)
		case errors.Is(err, credentialauthority.ErrProviderAbsent):
			return CredentialBinding{}, fmt.Errorf("credential provider absent for %s", field)
		default:
			return CredentialBinding{}, fmt.Errorf("resolve credential %s: %w", field, err)
		}
	}
	after := authority.Status(identity, field)
	if string(after.ProviderState) != "available" || !after.Configured || after.Version != before.Version {
		return CredentialBinding{}, fmt.Errorf("credential %s changed or became unavailable during resolution", field)
	}
	return CredentialBinding{Identity: identity, Field: field, Version: before.Version, Scope: scope, Value: strings.TrimSpace(value)}, nil
}

// PutAuthorityCredential is the write side for operator-supplied credentials
// that are accepted by an owning settings surface. The settings layer may
// retain non-secret SMTP metadata, but the protected value always goes through
// the shared authority.
func PutAuthorityCredential(key, value string) error {
	status, err := AuthorityCredentialStatus(key)
	if err != nil {
		return err
	}
	_, err = PutAuthorityCredentialIfVersion(key, value, status.Version)
	return err
}

// AuthorityCredentialReceipt exposes only the identity/version/scope result
// of an authority write. It intentionally has no field capable of carrying
// the credential value.
type AuthorityCredentialReceipt struct {
	Identity       credentialauthority.Identity `json:"identity"`
	Field          string                       `json:"field"`
	Version        string                       `json:"version"`
	Scope          string                       `json:"scope"`
	CleanupPending bool                         `json:"cleanup_pending"`
}

// AuthorityCredentialStatus provides the redacted metadata needed for an
// operator to confirm the version they intend to replace.
func AuthorityCredentialStatus(key string) (credentialauthority.Status, error) {
	field := AuthorityFieldForKey(key)
	if AuthorityScopeForField(field) == "" || isRetiredStripeCredential(field) {
		return credentialauthority.Status{}, fmt.Errorf("credential %s has no writable declared scope", field)
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return credentialauthority.Status{}, fmt.Errorf("initialize credential authority")
	}
	status := authority.Status(credentialAuthorityIdentity, field)
	if string(status.ProviderState) != "available" {
		return credentialauthority.Status{}, fmt.Errorf("credential authority is unavailable")
	}
	return status, nil
}

// PutAuthorityCredentialIfVersion stages and activates a provider candidate
// only when the active version still matches the operator's observed version.
// It is safe for both first provisioning (expectedVersion empty) and rotation.
func PutAuthorityCredentialIfVersion(key, value, expectedVersion string) (AuthorityCredentialReceipt, error) {
	field := AuthorityFieldForKey(key)
	if isRetiredStripeCredential(field) {
		return AuthorityCredentialReceipt{}, fmt.Errorf("legacy Stripe credential %s is retired; provision a mode-specific Stripe field", field)
	}
	scope := AuthorityScopeForField(field)
	if scope == "" {
		return AuthorityCredentialReceipt{}, fmt.Errorf("credential %s has no declared consumer scope", field)
	}
	if strings.TrimSpace(value) == "" {
		return AuthorityCredentialReceipt{}, fmt.Errorf("credential value is required")
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return AuthorityCredentialReceipt{}, fmt.Errorf("initialize credential authority")
	}
	status := authority.Status(credentialAuthorityIdentity, field)
	if string(status.ProviderState) != "available" {
		return AuthorityCredentialReceipt{}, fmt.Errorf("credential authority unavailable for %s", field)
	}
	if status.Version != expectedVersion {
		return AuthorityCredentialReceipt{}, fmt.Errorf("%w: active credential version changed", credentialauthority.ErrVersionConflict)
	}
	candidate, err := authority.PutCandidate(credentialAuthorityIdentity, field, value)
	if err != nil {
		return AuthorityCredentialReceipt{}, fmt.Errorf("stage credential candidate %s: %w", field, err)
	}
	current := authority.Status(credentialAuthorityIdentity, field)
	if string(current.ProviderState) != "available" || current.Version != expectedVersion {
		_ = authority.RejectCandidate(candidate)
		return AuthorityCredentialReceipt{}, fmt.Errorf("%w: active credential changed while candidate was staged", credentialauthority.ErrVersionConflict)
	}
	cleanupPending := false
	if err := authority.ActivateCandidateIfVersion(candidate, expectedVersion); err != nil {
		if !errors.Is(err, credentialauthority.ErrCandidateCleanupPending) {
			return AuthorityCredentialReceipt{}, fmt.Errorf("activate credential candidate %s: %w", field, err)
		}
		cleanupPending = true
	}
	return AuthorityCredentialReceipt{Identity: credentialAuthorityIdentity, Field: field, Version: candidate.Version, Scope: scope, CleanupPending: cleanupPending}, nil
}

// DeleteAuthorityCredential removes an operator-supplied credential from the
// same authority used for resolution.
func DeleteAuthorityCredential(key string) error {
	authority, err := openCredentialAuthority()
	if err != nil {
		return fmt.Errorf("initialize credential authority: %w", err)
	}
	return authority.Delete(credentialAuthorityIdentity, AuthorityFieldForKey(key))
}

// ResolveSecret preserves the optional configuration seam while routing every
// declared credential through the authority resolver.
func ResolveSecret(key string, log func(string, map[string]interface{})) string {
	switch key {
	case "SENDGRID_API_KEY", "SENDGRID_WEBHOOK_PUBLIC_KEY", "MAILGUN_API_KEY", "SMTP_PASSWORD",
		"ADMIN_DEFAULT_PASSWORD", "SESSION_SECRET", "SESSION_SECRET_PREVIOUS", "LPBS_" + "SERVICE_SECRET",
		"CONSUMER_AUTH_PRIVATE_KEY", "LPBS_API_KEY_ENCRYPTION_KEY", "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY", "LPBS_ADMIN_MFA_ENCRYPTION_KEY",
		"STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_PUBLISHABLE_KEY",
		"STRIPE_TEST_SECRET_KEY", "STRIPE_TEST_WEBHOOK_SECRET", "STRIPE_TEST_PUBLISHABLE_KEY",
		"STRIPE_LIVE_SECRET_KEY", "STRIPE_LIVE_WEBHOOK_SECRET", "STRIPE_LIVE_PUBLISHABLE_KEY",
		"DELIVERY_S3_ACCESS_KEY_ID", "DELIVERY_S3_SECRET_ACCESS_KEY", "DELIVERY_S3_SESSION_TOKEN":
		value, err := ResolveAuthorityCredential(key)
		if err != nil {
			if log != nil {
				log("credential_authority_unavailable", map[string]interface{}{"level": "warn", "key": key, "error": err.Error()})
			}
			return ""
		}
		return value
	default:
		return ResolveConfig(key)
	}
}

func ResolveConfig(key string) string { return strings.TrimSpace(envx.Get(key)) }

// ResolveGeneratedSecret performs the authority-backed resolve-or-mint policy
// for generated credentials. A truthful missing value is gated by the
// data-owned witness so a restored or rebuilt deployment cannot silently mint
// a replacement.
func ResolveGeneratedSecret(key string, mint func() (string, error)) (string, error) {
	if envx.Get("LPBS_TEST_CREDENTIAL_FALLBACK") == "1" {
		return strings.TrimSpace(envx.Get(key)), nil
	}
	authority, err := openCredentialAuthority()
	if err != nil {
		return "", fmt.Errorf("initialize credential authority: %w", err)
	}
	field := AuthorityFieldForKey(key)
	switch field {
	case "session-secret", "service-secret", "metrics-reader-token", "consumer-auth-private-key", "api-key-encryption-key", "remote-profile-encryption-key", "admin-mfa-encryption-key":
		if envx.Get("VROOLI_ACCEPT_CREDENTIAL_LOSS") == "1" {
			return authority.ResolveOrMintWithCredentialLossOverride(credentialAuthorityIdentity, field, credentialWitness, mint)
		}
		return authority.ResolveOrMint(credentialAuthorityIdentity, field, credentialWitness, mint)
	default:
		return "", fmt.Errorf("%s is not a generated credential", key)
	}
}
