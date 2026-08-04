// Package credentials defines the resource-kopia credential-authority seam.
package credentials

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// Store is the minimal authority surface resource-kopia needs. It never
// exposes a backend or accepts a passphrase through process arguments.
type Store interface {
	Put(credentialauthority.Identity, string, string) error
	Resolve(credentialauthority.Identity, string) (string, error)
	Delete(credentialauthority.Identity, string) error
}

// Default returns the platform credential authority.
func Default() (Store, error) { return credentialauthority.Default() }

// GeneratePassphrase creates a repository passphrase without writing it.
func GeneratePassphrase() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate passphrase: %w", err)
	}
	value := base64.RawURLEncoding.EncodeToString(buf)
	if len(value) < 32 {
		return "", fmt.Errorf("generated passphrase too short (%d)", len(value))
	}
	return value, nil
}

// ValidateStoredPassphrase confirms that a write reached the authority before
// kopia creates a repository that cannot later be opened.
func ValidateStoredPassphrase(store Store, identity credentialauthority.Identity, value string) error {
	if store == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	stored, err := store.Resolve(identity, kopiaregistry.PassphraseField)
	if err != nil {
		return fmt.Errorf("read back repository passphrase: %w", err)
	}
	if strings.TrimSpace(stored) == "" || stored != value {
		return fmt.Errorf("read back repository passphrase did not match")
	}
	return nil
}
