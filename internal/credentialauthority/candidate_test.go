package credentialauthority

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/securestore"
)

type candidateActivationStore struct {
	values      map[string]string
	failActive  bool
	failVersion bool
	failReads   bool
}

func (s *candidateActivationStore) Put(service, key, value string) error {
	if s.failActive && !strings.HasPrefix(key, candidateKeyPrefix) {
		return securestore.ErrUnavailable
	}
	if s.failVersion && strings.HasPrefix(key, "version/") {
		return securestore.ErrUnavailable
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *candidateActivationStore) Get(service, key string) (string, error) {
	if s.failReads && !strings.HasPrefix(key, candidateKeyPrefix) {
		return "", securestore.ErrUnavailable
	}
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
	status := authority.Status(identity, "token")
	if status.Version != candidate.Version {
		t.Fatalf("active version = %q, want candidate version %q", status.Version, candidate.Version)
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

func TestCandidateActivationRollsBackWhenVersionCommitFails(t *testing.T) {
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/rollback")
	if err := authority.Put(identity, "token", "active-value"); err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "candidate-value")
	if err != nil {
		t.Fatal(err)
	}
	store.failVersion = true
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("failed version commit = %v, want provider unavailable", err)
	}
	store.failVersion = false
	active, err := authority.Resolve(identity, "token")
	if err != nil || active != "active-value" {
		t.Fatalf("active value after rollback = %q, %v", active, err)
	}
}

func TestCandidateActivationFailsClosedWhenPriorStateCannotBeRead(t *testing.T) {
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/read-failure")
	candidate, err := authority.PutCandidate(identity, "token", "candidate-value")
	if err != nil {
		t.Fatal(err)
	}
	store.failReads = true
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("activation with unreadable prior state = %v, want provider unavailable", err)
	}
	store.failReads = false
	if _, err := authority.Resolve(identity, "token"); !errors.Is(err, ErrUnconfigured) {
		t.Fatalf("active value after fail-closed activation = %v, want unconfigured", err)
	}
}
