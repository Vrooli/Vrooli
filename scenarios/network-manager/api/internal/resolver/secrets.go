package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type Credentials struct {
	Username string
	Password string
}

type SecretResolver interface {
	ResolveAdGuardCredentials(ctx context.Context, cfg BackendConfig) (Credentials, error)
}

var ErrSecretMissing = errors.New("adguard home credential is missing from the credential authority")

// CredentialAuthorityResolver resolves only the canonical authority identity
// stored in BackendConfig. It never invokes a resource CLI or reads a secret
// file, keeping Network Manager portable across supported host key services.
type CredentialAuthorityResolver struct {
	Authority credentialStore
}

type credentialStore interface {
	Resolve(credentialauthority.Identity, string) (string, error)
}

func NewCredentialAuthorityResolver() *CredentialAuthorityResolver {
	authority, err := credentialauthority.Default()
	if err != nil {
		return &CredentialAuthorityResolver{Authority: unavailableCredentialStore{err: err}}
	}
	return &CredentialAuthorityResolver{Authority: authority}
}

func (r *CredentialAuthorityResolver) ResolveAdGuardCredentials(ctx context.Context, cfg BackendConfig) (Credentials, error) {
	_ = ctx
	identity, err := credentialauthority.ParseIdentity(normalizeCredentialRef(cfg.CredentialRef))
	if err != nil {
		return Credentials{}, err
	}
	store := r.Authority
	if store == nil {
		return Credentials{}, fmt.Errorf("credential authority is not configured")
	}
	password, err := resolveAuthorityField(store, identity, "password")
	if err != nil {
		return Credentials{}, fmt.Errorf("read AdGuard Home password from credential authority: %w", err)
	}
	if password == "" {
		return Credentials{}, fmt.Errorf("%w: %s/password", ErrSecretMissing, identity)
	}
	username := strings.TrimSpace(cfg.Username)
	if username == "" {
		username, err = resolveAuthorityField(store, identity, "username")
		if err != nil {
			return Credentials{}, fmt.Errorf("read AdGuard Home username from credential authority: %w", err)
		}
	}
	if username == "" {
		username = "admin"
	}
	return Credentials{Username: username, Password: password}, nil
}

func resolveAuthorityField(store credentialStore, identity credentialauthority.Identity, field string) (string, error) {
	value, err := store.Resolve(identity, field)
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

type unavailableCredentialStore struct{ err error }

func (s unavailableCredentialStore) Resolve(credentialauthority.Identity, string) (string, error) {
	return "", s.err
}

func normalizeCredentialRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "vrooli/adguard-home"
	}
	return ref
}

func redactedRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if len(ref) <= 12 {
		return "[redacted]"
	}
	return ref[:6] + "..." + ref[len(ref)-4:]
}
