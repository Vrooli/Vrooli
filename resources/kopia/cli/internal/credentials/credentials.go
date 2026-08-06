// Package credentials defines the resource-kopia credential-authority seam.
package credentials

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

const (
	// S3AccessKeyIDField and S3SecretAccessKeyField are stored beside the
	// repository passphrase under the per-repository credential identity. They
	// deliberately use the same authority as filesystem repository credentials;
	// resource-kopia has no external secret-service dependency.
	S3AccessKeyIDField     = "s3-access-key-id"
	S3SecretAccessKeyField = "s3-secret-access-key"
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

// S3Credentials holds the credentials required by an S3-compatible repository.
// Values are only held transiently while a kopia child process is running.
type S3Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// Valid reports whether both S3 credential halves are present.
func (c S3Credentials) Valid() bool {
	return strings.TrimSpace(c.AccessKeyID) != "" && strings.TrimSpace(c.SecretAccessKey) != ""
}

// PutS3Credentials stores S3 credentials in the canonical per-repository
// authority identity. It never writes a provider-shaped path or uses an
// external secret-service path.
func PutS3Credentials(store Store, repo string, creds S3Credentials) error {
	if store == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	if !creds.Valid() {
		return fmt.Errorf("S3 credentials for repository %q are incomplete", repo)
	}
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return err
	}
	if err := store.Put(identity, S3AccessKeyIDField, creds.AccessKeyID); err != nil {
		return fmt.Errorf("store S3 access key for repository %q: %w", repo, err)
	}
	if err := store.Put(identity, S3SecretAccessKeyField, creds.SecretAccessKey); err != nil {
		_ = store.Delete(identity, S3AccessKeyIDField)
		return fmt.Errorf("store S3 secret access key for repository %q: %w", repo, err)
	}
	return nil
}

// S3CredentialsFor resolves S3 credentials from the canonical authority. A
// cleanly absent pair is reported as found=false; provider failures remain
// errors so callers cannot mistake an unavailable authority for a missing key.
func S3CredentialsFor(store Store, repo string) (S3Credentials, bool, error) {
	if store == nil {
		return S3Credentials{}, false, fmt.Errorf("credential authority is unavailable")
	}
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return S3Credentials{}, false, err
	}
	access, err := store.Resolve(identity, S3AccessKeyIDField)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return S3Credentials{}, false, nil
		}
		return S3Credentials{}, false, fmt.Errorf("read S3 access key for repository %q: %w", repo, err)
	}
	secret, err := store.Resolve(identity, S3SecretAccessKeyField)
	if err != nil {
		if errors.Is(err, credentialauthority.ErrUnconfigured) {
			return S3Credentials{}, false, nil
		}
		return S3Credentials{}, false, fmt.Errorf("read S3 secret access key for repository %q: %w", repo, err)
	}
	creds := S3Credentials{AccessKeyID: access, SecretAccessKey: secret}
	if !creds.Valid() {
		return S3Credentials{}, false, nil
	}
	return creds, true, nil
}

// DeleteS3Credentials removes only the per-repository S3 fields. Repository
// passphrase deletion remains an explicit separate operation.
func DeleteS3Credentials(store Store, repo string) error {
	if store == nil {
		return fmt.Errorf("credential authority is unavailable")
	}
	identity, err := kopiaregistry.PassphraseIdentity(repo)
	if err != nil {
		return err
	}
	if err := store.Delete(identity, S3AccessKeyIDField); err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
		return fmt.Errorf("delete S3 access key for repository %q: %w", repo, err)
	}
	if err := store.Delete(identity, S3SecretAccessKeyField); err != nil && !errors.Is(err, credentialauthority.ErrUnconfigured) {
		return fmt.Errorf("delete S3 secret access key for repository %q: %w", repo, err)
	}
	return nil
}
