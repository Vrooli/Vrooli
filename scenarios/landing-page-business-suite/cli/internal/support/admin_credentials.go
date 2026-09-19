package support

import (
	"errors"
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// LPBSCredentialIdentity is the credential-authority identity that owns the
// landing-page-business-suite credentials.
const LPBSCredentialIdentity = "vrooli/landing-page-business-suite"

// AdminDefaultPasswordField is the authority field for the seeded admin
// password. The email is deliberately not a credential; it is configuration.
const AdminDefaultPasswordField = "admin-default-password"

// DefaultAdminEmail mirrors the API's seeded administrator email.
const DefaultAdminEmail = "admin@localhost"

// ResolveAdminEmail chooses the administrator email for admin login. An
// explicit flag wins, then ADMIN_DEFAULT_EMAIL, then the seeded default. The
// email is not a secret and is intentionally not read from the authority.
func ResolveAdminEmail(explicit string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("ADMIN_DEFAULT_EMAIL")); value != "" {
		return value
	}
	return DefaultAdminEmail
}

// ResolveAdminPassword returns the admin password for login. An explicit
// --password (literal or @file) wins; otherwise it resolves the operator
// credential from the authority so admin login is a single command. A missing
// credential is reported as an actionable operator omission rather than
// silently falling back or prompting.
func ResolveAdminPassword(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		value, err := ResolveSecretArg(explicit)
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		if strings.TrimSpace(value) == "" {
			return "", errors.New("admin password is empty")
		}
		return value, nil
	}

	identity, err := credentialauthority.ParseIdentity(LPBSCredentialIdentity)
	if err != nil {
		return "", fmt.Errorf("parse credential identity %q: %w", LPBSCredentialIdentity, err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("open credential authority: %w", err)
	}
	value, err := authority.Require(identity, AdminDefaultPasswordField)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return "", fmt.Errorf(
				"admin password is not configured; provision %q for %q in vrooli-onboarding, or pass --password",
				AdminDefaultPasswordField, LPBSCredentialIdentity,
			)
		}
		return "", fmt.Errorf("resolve admin password from the credential authority: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("admin password credential %q is empty", AdminDefaultPasswordField)
	}
	return value, nil
}
