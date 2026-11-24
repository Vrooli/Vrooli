package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
)

type providerCredentialTestStore struct {
	mu     sync.Mutex
	values map[string]string
}

func (s *providerCredentialTestStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}
func (s *providerCredentialTestStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", errors.Join(credentialauthority.ErrNotFound, errors.New("missing synthetic value"))
	}
	return value, nil
}
func (s *providerCredentialTestStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, service+"/"+key)
	return nil
}

func TestProviderCredentialEndpointUsesVersionCASAndRedactedReceipts(t *testing.T) {
	store := &providerCredentialTestStore{}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	administration.SetCredentialAuthorityProvider(func() (*credentialauthority.Authority, error) { return authority, nil })
	t.Cleanup(func() { administration.SetCredentialAuthorityProvider(nil) })
	first, err := administration.PutAuthorityCredentialIfVersion("MAILGUN_API_KEY", "sentinel-first-secret", "")
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{}

	staleBody, _ := json.Marshal(map[string]string{"field": "mailgun-api-key", "value": "sentinel-stale-secret", "expected_version": "old-version"})
	stale := httptest.NewRecorder()
	server.adminProviderCredentialWrite(stale, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(staleBody)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale write status = %d, want conflict", stale.Code)
	}
	if strings.Contains(stale.Body.String(), "sentinel-") {
		t.Fatal("stale refusal leaked a secret sentinel")
	}

	read := httptest.NewRecorder()
	server.providerVerificationReport(read, httptest.NewRequest(http.MethodGet, "/", nil))
	var report struct {
		CredentialVersions []struct {
			Field, Version, Scope string
			Configured, Available bool
		} `json:"credential_versions"`
	}
	if err := json.NewDecoder(read.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	var observed string
	for _, item := range report.CredentialVersions {
		if item.Field == "mailgun-api-key" {
			observed = item.Version
			if !item.Configured || !item.Available || item.Scope != "mail:transactional:send" {
				t.Fatalf("version metadata mismatch: %+v", item)
			}
		}
	}
	if observed != first.Version {
		t.Fatalf("reported version does not match active metadata: %q", observed)
	}
	if strings.Contains(read.Body.String(), "sentinel-") {
		t.Fatal("metadata endpoint leaked a secret sentinel")
	}

	body, _ := json.Marshal(map[string]string{"field": "mailgun-api-key", "value": "sentinel-second-secret", "expected_version": observed})
	response := httptest.NewRecorder()
	server.adminProviderCredentialWrite(response, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("rotation status = %d: %s", response.Code, response.Body.String())
	}
	var receipt map[string]any
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["field"] != "mailgun-api-key" || receipt["identity"] != "vrooli/landing-page-business-suite" || receipt["scope"] != "mail:transactional:send" || receipt["version"] == observed {
		t.Fatalf("rotation receipt lacks new metadata: %#v", receipt)
	}
	encoded, _ := json.Marshal(receipt)
	if strings.Contains(string(encoded), "sentinel-") {
		t.Fatal("rotation receipt leaked a secret sentinel")
	}
	if _, ok := receipt["value"]; ok {
		t.Fatal("rotation receipt unexpectedly has a value field")
	}
}
