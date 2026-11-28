package credentialauthority

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/securestore"
)

var (
	ErrVersionConflict         = errors.New("credential active version conflict")
	ErrCandidateCleanupPending = errors.New("credential candidate cleanup pending after activation")
)

// CandidateCleanupPendingError means the active record was committed but the
// duplicate candidate could not be removed. Callers must treat the new active
// version as committed and surface this cleanup state separately.
type CandidateCleanupPendingError struct {
	Version string
	Cause   error
}

func (e *CandidateCleanupPendingError) Error() string {
	if e == nil {
		return ErrCandidateCleanupPending.Error()
	}
	return fmt.Sprintf("%s at version %s: %v", ErrCandidateCleanupPending, e.Version, e.Cause)
}

func (e *CandidateCleanupPendingError) Is(target error) bool {
	return target == ErrCandidateCleanupPending
}

func (e *CandidateCleanupPendingError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

const candidateKeyPrefix = "candidate/"

type candidateLegacySnapshot struct {
	Value         string `json:"value"`
	VersionMarker string `json:"version_marker,omitempty"`
}

// CandidateRef is metadata for a protected value waiting for owner
// verification. The value itself never appears in this type or any response.
type CandidateRef struct {
	Identity        Identity `json:"logical_id"`
	Field           string   `json:"field"`
	Version         string   `json:"version"`
	ExpectedVersion string   `json:"expected_active_version,omitempty"`
	// ExpectedLegacy marks a candidate staged to replace an ambiguous legacy
	// inline value. Its expected source snapshot stays inside the secure store.
	ExpectedLegacy bool `json:"expected_legacy,omitempty"`
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
	if r.ExpectedVersion != "" {
		if len(r.ExpectedVersion) != 32 {
			return errors.New("candidate expected active version is invalid")
		}
		if _, err := hex.DecodeString(r.ExpectedVersion); err != nil {
			return errors.New("candidate expected active version is invalid")
		}
	}
	if r.ExpectedLegacy && r.ExpectedVersion != "" {
		return errors.New("candidate cannot bind both a version and an ambiguous legacy value")
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
	version, err := newCredentialVersion()
	if err != nil {
		return CandidateRef{}, fmt.Errorf("generate candidate version: %w", err)
	}
	ref := CandidateRef{Identity: identity, Field: field, Version: version}
	if err := ref.Validate(); err != nil {
		return CandidateRef{}, err
	}
	if a == nil || a.store == nil {
		return CandidateRef{}, fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	release, err := acquireCredentialStoreLock()
	if err != nil {
		return CandidateRef{}, fmt.Errorf("lock credential candidate staging: %w", err)
	}
	defer release()
	a.mu.Lock()
	var legacySnapshot string
	if currentVersion, versionErr := a.activeVersion(identity, field); versionErr == nil {
		ref.ExpectedVersion = currentVersion
	} else if errors.Is(versionErr, ErrAmbiguousLegacyActiveRecord) {
		var snapshot candidateLegacySnapshot
		snapshot.Value, err = a.store.Get(credentialService, storeKey(identity, field))
		if err != nil {
			a.mu.Unlock()
			return CandidateRef{}, classifyStoreError(err)
		}
		snapshot.VersionMarker, err = a.store.Get(credentialService, versionKey(identity, field))
		if errors.Is(err, securestore.ErrNotFound) {
			err = nil
			snapshot.VersionMarker = ""
		}
		if err != nil {
			a.mu.Unlock()
			return CandidateRef{}, classifyStoreError(err)
		}
		var snapshotJSON []byte
		snapshotJSON, err = json.Marshal(snapshot)
		if err != nil {
			a.mu.Unlock()
			return CandidateRef{}, err
		}
		legacySnapshot = string(snapshotJSON)
		ref.ExpectedLegacy = true
	} else if !errors.Is(versionErr, ErrUnconfigured) {
		a.mu.Unlock()
		return CandidateRef{}, fmt.Errorf("read active version before candidate staging: %w", versionErr)
	}
	err = classifyStoreError(a.store.Put(credentialService, candidateKey(ref), value))
	if err == nil && ref.ExpectedLegacy {
		err = classifyStoreError(a.store.Put(credentialService, candidateExpectationKey(ref), legacySnapshot))
		if err != nil {
			_ = a.store.Delete(credentialService, candidateKey(ref))
		}
	}
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
	return a.ActivateCandidateIfVersion(ref, ref.ExpectedVersion)
}

// ActivateCandidateIfVersion commits the candidate only if the active value
// still has expectedVersion. An empty expectedVersion asserts that no
// versioned value is active. Store writes share an OS-backed artifact lock,
// so separate Vrooli processes cannot pass the comparison and overwrite each
// other. The global lock also protects read-modify-write store adapters from
// cross-address lost updates.
func (a *Authority) ActivateCandidateIfVersion(ref CandidateRef, expectedVersion string) error {
	if expectedVersion != ref.ExpectedVersion {
		return fmt.Errorf("%w: candidate was staged against a different expected version", ErrVersionConflict)
	}
	return a.activateCandidate(ref, &expectedVersion)
}

func (a *Authority) activateCandidate(ref CandidateRef, expectedVersion *string) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	release, err := acquireCredentialStoreLock()
	if err != nil {
		return fmt.Errorf("lock credential candidate activation: %w", err)
	}
	defer release()
	a.mu.Lock()
	defer a.mu.Unlock()
	if expectedVersion != nil {
		actualVersion, versionErr := a.activeVersion(ref.Identity, ref.Field)
		if errors.Is(versionErr, ErrAmbiguousLegacyActiveRecord) && ref.ExpectedLegacy {
			legacy, readErr := a.store.Get(credentialService, storeKey(ref.Identity, ref.Field))
			if readErr != nil {
				return classifyStoreError(readErr)
			}
			legacyMarker, markerErr := a.store.Get(credentialService, versionKey(ref.Identity, ref.Field))
			if errors.Is(markerErr, securestore.ErrNotFound) {
				markerErr = nil
				legacyMarker = ""
			}
			if markerErr != nil {
				return classifyStoreError(markerErr)
			}
			expectedLegacy, expectedErr := a.store.Get(credentialService, candidateExpectationKey(ref))
			if expectedErr != nil {
				return fmt.Errorf("read candidate legacy source witness: %w", classifyStoreError(expectedErr))
			}
			var snapshot candidateLegacySnapshot
			if err := json.Unmarshal([]byte(expectedLegacy), &snapshot); err != nil {
				return errors.New("candidate legacy source witness is invalid")
			}
			if legacy != snapshot.Value || legacyMarker != snapshot.VersionMarker {
				return fmt.Errorf("%w: ambiguous legacy credential changed after candidate staging", ErrVersionConflict)
			}
			actualVersion = ""
		} else if errors.Is(versionErr, ErrUnconfigured) {
			actualVersion = ""
		} else if versionErr != nil {
			return fmt.Errorf("read active version before candidate activation: %w", versionErr)
		}
		if actualVersion != *expectedVersion {
			return fmt.Errorf("%w: expected %q, found %q", ErrVersionConflict, *expectedVersion, actualVersion)
		}
	}
	value, err := a.store.Get(credentialService, candidateKey(ref))
	if err == nil && strings.TrimSpace(value) == "" {
		err = securestore.ErrNotFound
	}
	if err == nil {
		// Verify any legacy active item is reachable before staging the new
		// immutable value and switching the version pointer.
		_, previousErr := a.store.Get(credentialService, storeKey(ref.Identity, ref.Field))
		if previousErr != nil && !errors.Is(previousErr, securestore.ErrNotFound) {
			err = previousErr
		}
	}
	if err == nil {
		err = a.writeActive(ref.Identity, ref.Field, ref.Version, value)
	}
	if err == nil {
		cleanupErr := a.store.Delete(credentialService, candidateKey(ref))
		if cleanupErr == nil || errors.Is(cleanupErr, securestore.ErrNotFound) {
			cleanupErr = a.store.Delete(credentialService, candidateExpectationKey(ref))
		}
		if cleanupErr != nil && !errors.Is(cleanupErr, securestore.ErrNotFound) {
			return &CandidateCleanupPendingError{Version: ref.Version, Cause: classifyStoreError(cleanupErr)}
		}
	}
	return classifyStoreError(err)
}

func acquireCredentialStoreLock() (func(), error) {
	return artifactlock.AcquireContext(context.Background(), "credential-authority:store")
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
	release, err := acquireCredentialStoreLock()
	if err != nil {
		return fmt.Errorf("lock credential candidate rejection: %w", err)
	}
	defer release()
	a.mu.Lock()
	err = a.store.Delete(credentialService, candidateKey(ref))
	if err == nil || errors.Is(err, securestore.ErrNotFound) {
		err = a.store.Delete(credentialService, candidateExpectationKey(ref))
	}
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

func candidateExpectationKey(ref CandidateRef) string {
	return "candidate-expectation/" + storeKey(ref.Identity, ref.Field) + ":" + ref.Version
}

func newCredentialVersion() (string, error) {
	versionBytes := make([]byte, 16)
	if _, err := rand.Read(versionBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(versionBytes), nil
}
