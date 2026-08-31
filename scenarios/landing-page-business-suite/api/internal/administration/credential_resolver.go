package administration

import (
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/envx"
)

const credentialAuthorityIdentity = credentialauthority.Identity("vrooli/landing-page-business-suite")

var credentialWitness credentialauthority.Witness

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
	case "ADMIN_DEFAULT_PASSWORD":
		return "admin-default-password"
	case "STRIPE_SECRET_KEY":
		return "stripe-secret-key"
	case "STRIPE_WEBHOOK_SECRET":
		return "stripe-webhook-secret"
	}
	return field
}

// ResolveAuthorityCredential requires a value from the authority. The test
// fallback exists only for isolated package fixtures and still reports an
// empty value as ErrUnconfigured.
func ResolveAuthorityCredential(key string) (string, error) {
	if envx.Get("LPBS_TEST_CREDENTIAL_FALLBACK") == "1" {
		value := strings.TrimSpace(envx.Get(key))
		if value == "" {
			return "", fmt.Errorf("%w: %s", credentialauthority.ErrUnconfigured, key)
		}
		return value, nil
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("initialize credential authority: %w", err)
	}
	value, err := authority.Require(credentialAuthorityIdentity, AuthorityFieldForKey(key))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

// ResolveSecret preserves the optional configuration seam while routing every
// declared credential through the authority resolver.
func ResolveSecret(key string, log func(string, map[string]interface{})) string {
	switch key {
	case "SENDGRID_API_KEY", "ADMIN_DEFAULT_PASSWORD", "SESSION_SECRET", "SESSION_SECRET_PREVIOUS", "LPBS_" + "SERVICE_SECRET", "CONSUMER_AUTH_PRIVATE_KEY", "LPBS_API_KEY_ENCRYPTION_KEY", "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY", "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET":
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
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("initialize credential authority: %w", err)
	}
	field := AuthorityFieldForKey(key)
	switch field {
	case "session-secret", "service-secret", "consumer-auth-private-key", "api-key-encryption-key", "remote-profile-encryption-key":
		if envx.Get("VROOLI_ACCEPT_CREDENTIAL_LOSS") == "1" {
			return authority.ResolveOrMintWithCredentialLossOverride(credentialAuthorityIdentity, field, credentialWitness, mint)
		}
		return authority.ResolveOrMint(credentialAuthorityIdentity, field, credentialWitness, mint)
	default:
		return "", fmt.Errorf("%s is not a generated credential", key)
	}
}
