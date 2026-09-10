package credentialauthority

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/securestore"
)

type candidateActivationStore struct {
	values     map[string]string
	failActive bool
}

func (s *candidateActivationStore) Put(service, key, value string) error {
	if s.failActive && !strings.HasPrefix(key, candidateKeyPrefix) {
		return securestore.ErrUnavailable
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *candidateActivationStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}

func (s *candidateActivationStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func TestCandidateActivationNeverReplacesActiveOnProviderFailure(t *testing.T) {
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/candidate")
	if err := authority.Put(identity, "token", "active-value"); err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "candidate-value")
	if err != nil {
		t.Fatal(err)
	}
	store.failActive = true
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("failed activation = %v, want provider unavailable", err)
	}
	active, err := authority.Resolve(identity, "token")
	if err != nil || active != "active-value" {
		t.Fatalf("active value after failed activation = %q, %v", active, err)
	}
	store.failActive = false
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	active, err = authority.Resolve(identity, "token")
	if err != nil || active != "candidate-value" {
		t.Fatalf("active value after activation = %q, %v", active, err)
	}
}

func TestRejectCandidateLeavesActiveValueUntouched(t *testing.T) {
	store := &candidateActivationStore{}
	authority, _ := NewAuthority(store)
	identity, _ := ParseIdentity("vrooli/reject")
	if err := authority.Put(identity, "token", "active-value"); err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "discard-me")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.RejectCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	active, err := authority.Resolve(identity, "token")
	if err != nil || active != "active-value" {
		t.Fatalf("active value after rejection = %q, %v", active, err)
	}
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrUnconfigured) {
		t.Fatalf("removed candidate activation = %v, want unconfigured", err)
	}
}
