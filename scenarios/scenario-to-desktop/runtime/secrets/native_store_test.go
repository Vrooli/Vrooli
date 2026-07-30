package secrets

import (
	"errors"
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
		return "", errors.New("missing")
	}
	return value, nil
}

func (s *nativeTestStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

var _ securestore.Store = (*nativeTestStore)(nil)

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
