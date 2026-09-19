package signing

import (
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// CredentialAuthority is the narrow custody seam for signing key material.
// Production wraps the native credential authority; tests inject a fake so the
// custody contract can be exercised without touching the live store.
type CredentialAuthority interface {
	Put(identity credentialauthority.Identity, field, value string) error
	Resolve(identity credentialauthority.Identity, field string) (string, error)
	Status(identity credentialauthority.Identity, field string) credentialauthority.Status
	Availability() error
	Provider() string
}

// openCredentialAuthority is a variable rather than a direct call so tests can
// substitute a fake. Production never reassigns it.
var openCredentialAuthority = func() (CredentialAuthority, error) {
	return credentialauthority.Default()
}

// managedIdentity returns the credential-authority identity for a scenario's
// generated signing key when no explicit identity is supplied.
func managedIdentity(scenario string) (credentialauthority.Identity, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", fmt.Errorf("scenario name is required for a managed signing key")
	}
	return credentialauthority.ParseIdentity("vrooli/scenario-to-desktop/" + scenario)
}

// resolveManagedIdentity selects the custody identity for a managed signing
// key. An explicit logical ID lets one publisher key serve many scenarios;
// otherwise custody defaults to the scenario's own namespace.
func resolveManagedIdentity(scenario, logicalID string) (credentialauthority.Identity, error) {
	if logicalID = strings.TrimSpace(logicalID); logicalID != "" {
		identity, err := credentialauthority.ParseIdentity(logicalID)
		if err != nil {
			return "", fmt.Errorf("invalid managed signing identity %q: %w", logicalID, err)
		}
		return identity, nil
	}
	return managedIdentity(scenario)
}
