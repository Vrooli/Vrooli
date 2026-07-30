package secrets

import (
	"errors"
	"testing"
)

type authorityStore struct{ values map[string]string }

func (s *authorityStore) Put(service, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *authorityStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", errors.New("missing")
	}
	return value, nil
}
func (s *authorityStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func TestAuthorityStoresAndInjectsOnlyScopedValue(t *testing.T) {
	authority, err := NewAuthority(&authorityStore{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseIdentity("vrooli/openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(id, "api-key", "test-value"); err != nil {
		t.Fatal(err)
	}
	status := authority.Status(id, "api-key")
	if !status.Configured || status.Provider != "native-secure-store" {
		t.Fatalf("unsafe status: %+v", status)
	}
	env := map[string]string{}
	if err := authority.Inject(id, "api-key", "OPENROUTER_API_KEY", env); err != nil {
		t.Fatal(err)
	}
	if env["OPENROUTER_API_KEY"] != "test-value" {
		t.Fatal("scoped injection did not resolve value")
	}
	if err := authority.Delete(id, "api-key"); err != nil {
		t.Fatal(err)
	}
	if authority.Status(id, "api-key").Configured {
		t.Fatal("deleted credential remains configured")
	}
}

func TestAuthorityRejectsInvalidIdentityAndNeverFallsBack(t *testing.T) {
	if _, err := ParseIdentity("openrouter"); err == nil {
		t.Fatal("unnamespaced identity accepted")
	}
	if _, err := NewAuthority(nil); !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("error = %v", err)
	}
}
