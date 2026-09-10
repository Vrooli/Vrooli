package credentialauthority

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/securestore"
)

const candidateKeyPrefix = "candidate/"

// CandidateRef is metadata for a protected value waiting for owner
// verification. The value itself never appears in this type or any response.
type CandidateRef struct {
	Identity Identity `json:"logical_id"`
	Field    string   `json:"field"`
	Version  string   `json:"version"`
}

func (r CandidateRef) Validate() error {
	if _, err := ParseIdentity(string(r.Identity)); err != nil {
		return err
	}
	if strings.TrimSpace(r.Field) == "" || strings.ContainsAny(r.Field, "/\\") {
		return errors.New("candidate field is required and cannot contain a path separator")
	}
	if len(r.Version) != 32 {
		return errors.New("candidate version is invalid")
	}
	if _, err := hex.DecodeString(r.Version); err != nil {
		return errors.New("candidate version is invalid")
	}
	return nil
}

// PutCandidate stores a protected value under an opaque candidate version.
// It never overwrites the active authority address.
func (a *Authority) PutCandidate(identity Identity, field, value string) (CandidateRef, error) {
	identity, field, err := normalizeCandidateAddress(identity, field)
	if err != nil {
		return CandidateRef{}, err
	}
	if strings.TrimSpace(value) == "" {
		return CandidateRef{}, errors.New("candidate credential value is required")
	}
	versionBytes := make([]byte, 16)
	if _, err := rand.Read(versionBytes); err != nil {
		return CandidateRef{}, fmt.Errorf("generate candidate version: %w", err)
	}
	ref := CandidateRef{Identity: identity, Field: field, Version: hex.EncodeToString(versionBytes)}
	if err := ref.Validate(); err != nil {
		return CandidateRef{}, err
	}
	if a == nil || a.store == nil {
		return CandidateRef{}, fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	a.mu.Lock()
	err = classifyStoreError(a.store.Put(credentialService, candidateKey(ref), value))
	a.mu.Unlock()
	if err != nil {
		return CandidateRef{}, err
	}
	return ref, nil
}

// ActivateCandidate makes one verified candidate the active value. The active
// value is written before candidate cleanup; a cleanup error is reported but
// never causes the previously active value to be deleted.
func (a *Authority) ActivateCandidate(ref CandidateRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	a.mu.Lock()
	value, err := a.store.Get(credentialService, candidateKey(ref))
	if err == nil && strings.TrimSpace(value) == "" {
		err = securestore.ErrNotFound
	}
	if err == nil {
		err = a.store.Put(credentialService, storeKey(ref.Identity, ref.Field), value)
	}
	if err == nil {
		err = a.store.Delete(credentialService, candidateKey(ref))
	}
	a.mu.Unlock()
	return classifyStoreError(err)
}

// RejectCandidate removes an unaccepted candidate and leaves the active
// authority value unchanged.
func (a *Authority) RejectCandidate(ref CandidateRef) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	a.mu.Lock()
	err := a.store.Delete(credentialService, candidateKey(ref))
	a.mu.Unlock()
	if errors.Is(err, securestore.ErrNotFound) {
		return nil
	}
	return classifyStoreError(err)
}

func normalizeCandidateAddress(identity Identity, field string) (Identity, string, error) {
	parsed, err := ParseIdentity(string(identity))
	if err != nil {
		return "", "", err
	}
	field = strings.TrimSpace(field)
	if field == "" || strings.ContainsAny(field, "/\\") {
		return "", "", errors.New("candidate field is required and cannot contain a path separator")
	}
	return parsed, field, nil
}

func candidateKey(ref CandidateRef) string {
	return candidateKeyPrefix + storeKey(ref.Identity, ref.Field) + ":" + ref.Version
}
