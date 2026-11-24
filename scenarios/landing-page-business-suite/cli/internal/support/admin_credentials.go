package support

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
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

// ResolveLPBSServiceCredential returns the service token only when the
// authority has an available, configured, versioned LPBS bootstrap binding.
// It rechecks metadata after reading so a concurrent activation cannot make
// this caller use a value with an unknown version.
func ResolveLPBSServiceCredential() (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("initialize credential authority")
	}
	return ResolveLPBSServiceCredentialFromAuthority(authority)
}

// ResolveLPBSServiceCredentialFromAuthority is the same strict resolver for
// isolated synthetic authority fixtures.
func ResolveLPBSServiceCredentialFromAuthority(authority *credentialauthority.Authority) (string, error) {
	identity, err := credentialauthority.ParseIdentity(LPBSCredentialIdentity)
	if err != nil || authority == nil {
		return "", fmt.Errorf("LPBS credential authority binding is unavailable")
	}
	before := authority.Status(identity, "service-secret")
	if string(before.ProviderState) != "available" || !before.Configured || strings.TrimSpace(before.Version) == "" {
		return "", fmt.Errorf("LPBS service credential authority is unavailable or unconfigured")
	}
	value, err := authority.Resolve(identity, "service-secret")
	if err != nil {
		return "", fmt.Errorf("resolve LPBS service credential")
	}
	after := authority.Status(identity, "service-secret")
	if string(after.ProviderState) != "available" || !after.Configured || after.Version != before.Version {
		return "", fmt.Errorf("LPBS service credential changed during resolution")
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("LPBS service credential is empty")
	}
	return value, nil
}

// ResolveLPBSRecoveryAuthorization prefers the short-lived, narrowly scoped
// recovery principal. It falls back to the legacy shared bearer only when a
// healthy authority confirms the scoped principal has not yet been issued.
func ResolveLPBSRecoveryAuthorization(ctx context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("initialize LPBS credential authority")
	}
	exchanger := servicetoken.AuthenticatorExchanger(&http.Client{Timeout: 5 * time.Second}, func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	})
	return ResolveLPBSRecoveryAuthorizationFromAuthority(ctx, authority, exchanger)
}

// ResolveLPBSRecoveryAuthorizationFromAuthority is the isolated authority seam
// used by contract tests. It accepts only an exchanger that returns the exact
// declared subject, audience, and scope.
func ResolveLPBSRecoveryAuthorizationFromAuthority(ctx context.Context, authority *credentialauthority.Authority, exchanger servicetoken.Exchanger) (string, error) {
	if authority == nil {
		return "", fmt.Errorf("LPBS credential authority is unavailable")
	}
	identity, err := credentialauthority.ParseIdentity(lpbsidentity.LocalAdminRecovery.CredentialIdentity("landing-page-business-suite"))
	if err != nil {
		return "", fmt.Errorf("LPBS scoped recovery identity is invalid")
	}
	status := authority.Status(identity, "credential")
	if string(status.ProviderState) != "available" {
		return "", fmt.Errorf("LPBS scoped recovery identity authority is unavailable")
	}
	if status.Configured {
		if strings.TrimSpace(status.Version) == "" {
			return "", fmt.Errorf("LPBS scoped recovery identity has no active version")
		}
		if exchanger == nil {
			return "", fmt.Errorf("LPBS recovery token exchange is unavailable")
		}
		tokens := lpbsidentity.LocalAdminRecovery.NewTokens("landing-page-business-suite", func(ctx context.Context) (string, error) {
			return authority.Require(identity, "credential")
		}, servicetoken.WithExchanger(exchanger))
		authorization, err := tokens.Authorization(ctx)
		if err != nil {
			return "", fmt.Errorf("exchange LPBS scoped recovery identity")
		}
		return authorization, nil
	}
	legacy, err := ResolveLPBSServiceCredentialFromAuthority(authority)
	if err != nil {
		return "", err
	}
	return "Bearer " + strings.TrimSpace(legacy), nil
}
