package support

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type serviceCredentialTestStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *serviceCredentialTestStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *serviceCredentialTestStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[service+"/"+key]
	if !ok {
		return "", fmt.Errorf("%w: missing", credentialauthority.ErrNotFound)
	}
	return v, nil
}
func (s *serviceCredentialTestStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, service+"/"+key)
	return nil
}

func TestResolveAdminEmail(t *testing.T) {
	t.Run("explicit wins", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "env@example.test")
		if got := ResolveAdminEmail("flag@example.test"); got != "flag@example.test" {
			t.Fatalf("ResolveAdminEmail = %q, want explicit value", got)
		}
	})
	t.Run("environment fallback", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "env@example.test")
		if got := ResolveAdminEmail(""); got != "env@example.test" {
			t.Fatalf("ResolveAdminEmail = %q, want environment value", got)
		}
	})
	t.Run("seeded default", func(t *testing.T) {
		t.Setenv("ADMIN_DEFAULT_EMAIL", "")
		if got := ResolveAdminEmail(""); got != DefaultAdminEmail {
			t.Fatalf("ResolveAdminEmail = %q, want %q", got, DefaultAdminEmail)
		}
	})
}

func TestResolveAdminPasswordExplicit(t *testing.T) {
	t.Run("literal", func(t *testing.T) {
		got, err := ResolveAdminPassword("hunter2")
		if err != nil {
			t.Fatalf("ResolveAdminPassword returned error: %v", err)
		}
		if got != "hunter2" {
			t.Fatalf("ResolveAdminPassword = %q, want literal", got)
		}
	})
	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "password.txt")
		if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
			t.Fatalf("write password file: %v", err)
		}
		got, err := ResolveAdminPassword("@" + path)
		if err != nil {
			t.Fatalf("ResolveAdminPassword returned error: %v", err)
		}
		if got != "file-secret" {
			t.Fatalf("ResolveAdminPassword = %q, want trimmed file value", got)
		}
	})
}

func TestResolveLPBSServiceCredentialRequiresVersionedBindingAndFailsClosed(t *testing.T) {
	identity, err := credentialauthority.ParseIdentity(LPBSCredentialIdentity)
	if err != nil {
		t.Fatal(err)
	}
	store := &serviceCredentialTestStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "service-secret", "sentinel-service-token")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveLPBSServiceCredentialFromAuthority(authority)
	if err != nil || got != "sentinel-service-token" {
		t.Fatalf("resolver returned expected token:%t err:%v", got == "sentinel-service-token", err)
	}

	unavailable, err := credentialauthority.Unavailable("synthetic outage")
	if err != nil {
		t.Fatal(err)
	}
	if value, err := ResolveLPBSServiceCredentialFromAuthority(unavailable); err == nil || value != "" || strings.Contains(err.Error(), "sentinel-service-token") {
		t.Fatalf("unavailable authority returned value:%t error leaked secret:%t", value != "", err != nil && strings.Contains(err.Error(), "sentinel-service-token"))
	}
	if _, err := ResolveLPBSServiceCredentialFromAuthority(nil); err == nil || errors.Is(err, credentialauthority.ErrNotFound) {
		t.Fatalf("nil authority error = %v", err)
	}
}

func TestResolveLPBSRecoveryAuthorizationUsesScopedTokenAndLimitedLegacyFallback(t *testing.T) {
	roleIdentity, err := credentialauthority.ParseIdentity(lpbsidentity.LocalAdminRecovery.CredentialIdentity("landing-page-business-suite"))
	if err != nil {
		t.Fatal(err)
	}
	store := &serviceCredentialTestStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	roleCredential, err := authority.PutCandidate(roleIdentity, "credential", "synthetic-role-credential")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(roleCredential); err != nil {
		t.Fatal(err)
	}
	exchanger := func(_ context.Context, credential string) (servicetoken.Exchanged, error) {
		if credential != "synthetic-role-credential" {
			t.Fatal("unexpected credential passed to exchange")
		}
		return servicetoken.Exchanged{
			AccessToken: "synthetic-scoped-token", ExpiresAt: time.Now().Add(time.Minute),
			Subject:  lpbsidentity.LocalAdminRecovery.Principal("landing-page-business-suite"),
			Audience: lpbsidentity.LocalAdminRecovery.Audience,
			Scopes:   []string{lpbsidentity.LocalAdminRecovery.Scope},
		}, nil
	}
	header, err := ResolveLPBSRecoveryAuthorizationFromAuthority(context.Background(), authority, exchanger)
	if err != nil || header != "Bearer synthetic-scoped-token" || strings.Contains(header, "synthetic-role-credential") {
		t.Fatalf("scoped recovery authorization = %q, %v", header, err)
	}

	legacyStore := &serviceCredentialTestStore{}
	legacyAuthority, err := credentialauthority.NewAuthority(legacyStore)
	if err != nil {
		t.Fatal(err)
	}
	legacyIdentity, err := credentialauthority.ParseIdentity(LPBSCredentialIdentity)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := legacyAuthority.PutCandidate(legacyIdentity, "service-secret", "sentinel-old-service-secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := legacyAuthority.ActivateCandidate(legacy); err != nil {
		t.Fatal(err)
	}
	header, err = ResolveLPBSRecoveryAuthorizationFromAuthority(context.Background(), legacyAuthority, nil)
	if err != nil || header != "Bearer sentinel-old-service-secret" {
		t.Fatalf("legacy overlap authorization = %q, %v", header, err)
	}

	unavailable, err := credentialauthority.Unavailable("synthetic outage")
	if err != nil {
		t.Fatal(err)
	}
	header, err = ResolveLPBSRecoveryAuthorizationFromAuthority(context.Background(), unavailable, nil)
	if err == nil || header != "" || strings.Contains(err.Error(), "sentinel-old-service-secret") {
		t.Fatalf("unavailable authority result = %q, %v", header, err)
	}
}
