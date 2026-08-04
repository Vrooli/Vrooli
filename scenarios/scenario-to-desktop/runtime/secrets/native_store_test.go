package secrets

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

type nativeTestStore struct{ values map[string]string }

func (s *nativeTestStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *nativeTestStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		// A conforming Store distinguishes "no value" from "backend broken";
		// the desktop manager relies on that to keep a provider outage from
		// looking like an unset bundle secret.
		return "", fmt.Errorf("%w: %s/%s", securestore.ErrNotFound, service, key)
	}
	return value, nil
}

func (s *nativeTestStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

var _ securestore.Store = (*nativeTestStore)(nil)

// unreachableTestStore stands in for a desktop host whose platform credential
// service is running but unreachable.
type unreachableTestStore struct{}

func (unreachableTestStore) Put(string, string, string) error {
	return fmt.Errorf("%w: session unreachable", securestore.ErrUnavailable)
}

func (unreachableTestStore) Get(string, string) (string, error) {
	return "", fmt.Errorf("%w: session unreachable", securestore.ErrUnavailable)
}

func (unreachableTestStore) Delete(string, string) error {
	return fmt.Errorf("%w: session unreachable", securestore.ErrUnavailable)
}

// TestNativeManagerSurfacesProviderOutageInsteadOfSkippingSecrets proves the
// desktop manager no longer silently treats a broken platform store as a
// bundle whose secrets were never set. Skipping would hand the app an empty
// credential set and a confusing "please configure" prompt.
func TestNativeManagerSurfacesProviderOutageInsteadOfSkippingSecrets(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(unreachableTestStore{})
	if err != nil {
		t.Fatal(err)
	}
	bundle := &manifest.Manifest{App: manifest.App{Name: "Demo Desktop"}, Secrets: []manifest.Secret{{ID: "API_KEY"}}}
	manager, err := NewNativeManagerWithAuthority(bundle, authority)
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Load()
	if !errors.Is(err, credentialauthority.ErrProviderUnavailable) {
		t.Fatalf("Load() error = %v, want ErrProviderUnavailable", err)
	}
	if errors.Is(err, credentialauthority.ErrUnconfigured) {
		t.Fatalf("Load() error = %v must not read as an unset credential", err)
	}
}

func TestNativeManagerSurfacesAbsentProviderWithActionableError(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(securestore.Absent("test host has no native credential service"))
	if err != nil {
		t.Fatal(err)
	}
	bundle := &manifest.Manifest{App: manifest.App{Name: "Demo Desktop"}, Secrets: []manifest.Secret{{ID: "API_KEY"}}}
	manager, err := NewNativeManagerWithAuthority(bundle, authority)
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Load()
	if !errors.Is(err, credentialauthority.ErrProviderAbsent) {
		t.Fatalf("Load() error = %v, want ErrProviderAbsent", err)
	}
	if !strings.Contains(err.Error(), "configure a credential backend") {
		t.Fatalf("Load() error = %v, want an actionable backend instruction", err)
	}
}

func TestNativeManagerRoundTripNeverUsesAFile(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&nativeTestStore{})
	if err != nil {
		t.Fatal(err)
	}
	manifest := &manifest.Manifest{App: manifest.App{Name: "Demo Desktop"}, Secrets: []manifest.Secret{{ID: "API_KEY"}}}
	manager, err := NewNativeManagerWithAuthority(manifest, authority)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Persist(map[string]string{"API_KEY": "desktop-test-value"}); err != nil {
		t.Fatal(err)
	}
	values, err := manager.Load()
	if err != nil {
		t.Fatal(err)
	}
	if values["API_KEY"] != "desktop-test-value" {
		t.Fatalf("Load() = %q, want native value", values["API_KEY"])
	}
}
