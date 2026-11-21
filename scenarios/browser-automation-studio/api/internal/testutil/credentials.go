package testutil

import (
	"sync"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// CredentialStore keeps synthetic credentials in memory, including authority
// availability probes. It never resolves the host's credential provider.
type CredentialStore struct {
	mu     sync.Mutex
	values map[string]string
	Err    error
	Writes int
}

func (s *CredentialStore) Authority() (*credentialauthority.Authority, error) {
	return credentialauthority.NewAuthority(s)
}

func (s *CredentialStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return s.Err
	}
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[service+"/"+key] = value
	s.Writes++
	return nil
}

func (s *CredentialStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return "", s.Err
	}
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", credentialauthority.ErrNotFound
	}
	return value, nil
}

func (s *CredentialStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return s.Err
	}
	delete(s.values, service+"/"+key)
	return nil
}

// ProfileCredentialAuthority supplies the same synthetic key to independent
// repository instances. Fault/rotation tests inject their own CredentialStore.
func ProfileCredentialAuthority() (*credentialauthority.Authority, error) {
	authority, err := (&CredentialStore{}).Authority()
	if err != nil {
		return nil, err
	}
	err = authority.Put("vrooli/browser-automation-studio", "session-profile-keyring", `{"active":1,"keys":{"1":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}`)
	return authority, err
}
