package credentialauthority

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/securestore"
)

type candidateActivationStore struct {
	values              map[string]string
	failActive          bool
	failPointer         bool
	failReads           bool
	failCandidateDelete bool
}

func (s *candidateActivationStore) Put(service, key, value string) error {
	if s.failPointer && strings.HasPrefix(key, "version/") {
		return securestore.ErrUnavailable
	}
	if s.failActive && !strings.HasPrefix(key, candidateKeyPrefix) {
		return securestore.ErrUnavailable
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func TestPointerWriteFailureLeavesOldVersionActive(t *testing.T) {
	store := &candidateActivationStore{}
	identity, _ := ParseIdentity("vrooli/pointer-failure")
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.Put(identity, "token", "initial"); err != nil {
		t.Fatal(err)
	}
	before := authority.Status(identity, "token")
	candidate, err := authority.PutCandidate(identity, "token", "next")
	if err != nil {
		t.Fatal(err)
	}
	store.failPointer = true
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("pointer write failure = %v", err)
	}
	after := authority.Status(identity, "token")
	value, err := authority.Resolve(identity, "token")
	if err != nil || value != "initial" || after.Version != before.Version {
		t.Fatalf("failed pointer commit changed active credential: value=%q status=%+v err=%v", value, after, err)
	}
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
	if s.failCandidateDelete && strings.HasPrefix(key, candidateKeyPrefix) {
		return securestore.ErrUnavailable
	}
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

func TestCandidateActivationCommitsVersionedReferenceAfterValueWrite(t *testing.T) {
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/versioned-active-reference")
	candidate, err := authority.PutCandidate(identity, "token", "synthetic-secret-value")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatal(err)
	}
	ref := store.values[credentialService+"/"+versionKey(identity, "token")]
	version, isRef, err := decodeActiveCredentialReference(ref)
	if err != nil || !isRef || version != candidate.Version {
		t.Fatalf("active reference mismatch: marked=%t version-match=%t err=%v", isRef, version == candidate.Version, err)
	}
	value := store.values[credentialService+"/"+activeValueKey(identity, "token", candidate.Version)]
	if value != "synthetic-secret-value" {
		t.Fatal("versioned immutable value was not persisted before pointer activation")
	}
	if raw, exists := store.values[credentialService+"/"+storeKey(identity, "token")]; exists {
		t.Fatalf("new versioned activation wrote an inline value marker at the legacy key: %q", raw)
	}
	status := authority.Status(identity, "token")
	if !status.Configured || status.Version != candidate.Version {
		t.Fatalf("status does not read versioned active reference: %+v", status)
	}
}

func TestLegacyValueCollidingWithRetiredMarkerFailsClosedAndIsPreserved(t *testing.T) {
	store := &candidateActivationStore{values: map[string]string{}}
	identity, _ := ParseIdentity("vrooli/legacy-marker-collision")
	legacy := legacyActiveCredentialRecordPrefix + "synthetic-opaque-value"
	store.values[credentialService+"/"+storeKey(identity, "token")] = legacy
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authority.Resolve(identity, "token"); !errors.Is(err, ErrAmbiguousLegacyActiveRecord) {
		t.Fatalf("ambiguous legacy value was not refused: %v", err)
	}
	status := authority.Status(identity, "token")
	if status.Configured || status.ProviderState != ProviderAvailable || !strings.Contains(status.ProviderDetail, "collides") {
		t.Fatalf("ambiguous status must be non-secret and explicit: %+v", status)
	}
	if got := store.values[credentialService+"/"+storeKey(identity, "token")]; got != legacy {
		t.Fatal("refusal mutated the legacy source value")
	}
}

func TestExplicitMigrationCanPreserveRawMarkerPrefixedValue(t *testing.T) {
	store := &candidateActivationStore{values: map[string]string{}}
	identity, _ := ParseIdentity("vrooli/legacy-raw-migration")
	legacyKey := credentialService + "/" + storeKey(identity, "token")
	legacy := legacyActiveCredentialRecordPrefix + "not-a-v1-envelope-but-valid-raw-secret"
	store.values[legacyKey] = legacy
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := authority.MigrateAmbiguousLegacyActive(identity, "token", ""); err == nil {
		t.Fatal("missing owner interpretation was accepted")
	}
	receipt, err := authority.MigrateAmbiguousLegacyActive(identity, "token", LegacyActiveAsRawValue)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.SourceRetained || receipt.Version == "" || receipt.Interpretation != LegacyActiveAsRawValue {
		t.Fatalf("invalid metadata receipt: %+v", receipt)
	}
	receiptJSON, err := json.Marshal(receipt)
	if err != nil || strings.Contains(string(receiptJSON), legacy) {
		t.Fatalf("receipt leaked legacy value: %s err=%v", receiptJSON, err)
	}
	if got, err := authority.Resolve(identity, "token"); err != nil || got != legacy {
		t.Fatalf("migrated raw value = %q, %v", got, err)
	}
	if store.values[legacyKey] != legacy {
		t.Fatal("migration mutated retained source item")
	}
}

func TestExplicitMigrationUnwrapsSyntheticV1RecordOnlyByOwnerChoice(t *testing.T) {
	store := &candidateActivationStore{values: map[string]string{}}
	identity, _ := ParseIdentity("vrooli/legacy-envelope-migration")
	legacyKey := credentialService + "/" + storeKey(identity, "token")
	version := "0123456789abcdef0123456789abcdef"
	encoded, err := json.Marshal(legacyActiveCredentialRecord{Format: legacyActiveCredentialRecordFormat, Version: version, Value: "synthetic-inner-secret"})
	if err != nil {
		t.Fatal(err)
	}
	legacy := legacyActiveCredentialRecordPrefix + base64.RawStdEncoding.EncodeToString(encoded)
	store.values[legacyKey] = legacy
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := authority.MigrateAmbiguousLegacyActive(identity, "token", LegacyActiveAsV1Record)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Version != version || !receipt.SourceRetained {
		t.Fatalf("migration receipt = %+v", receipt)
	}
	if got, err := authority.Resolve(identity, "token"); err != nil || got != "synthetic-inner-secret" {
		t.Fatalf("unwrapped v1 value = %q, %v", got, err)
	}
	if store.values[legacyKey] != legacy {
		t.Fatal("v1 source was not retained")
	}
}

func TestCandidateCanReplaceAmbiguousLegacyValueUsingSourceSnapshotCAS(t *testing.T) {
	store := &candidateActivationStore{values: map[string]string{}}
	identity, _ := ParseIdentity("vrooli/legacy-marker-replacement")
	legacyKey := credentialService + "/" + storeKey(identity, "token")
	legacy := legacyActiveCredentialRecordPrefix + "synthetic-collision"
	store.values[legacyKey] = legacy
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "replacement-secret")
	if err != nil {
		t.Fatalf("stage explicit replacement: %v", err)
	}
	if !candidate.ExpectedLegacy || candidate.ExpectedVersion != "" {
		t.Fatalf("candidate lacks legacy CAS witness: %+v", candidate)
	}
	refJSON, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(refJSON), legacy) || strings.Contains(string(refJSON), "replacement-secret") {
		t.Fatalf("candidate metadata contains a credential value: %s", refJSON)
	}
	var snapshot candidateLegacySnapshot
	if err := json.Unmarshal([]byte(store.values[credentialService+"/"+candidateExpectationKey(candidate)]), &snapshot); err != nil || snapshot.Value != legacy {
		t.Fatal("expected source snapshot was not isolated in the credential store")
	}
	if err := authority.ActivateCandidate(candidate); err != nil {
		t.Fatalf("activate replacement: %v", err)
	}
	value, err := authority.Resolve(identity, "token")
	if err != nil || value != "replacement-secret" {
		t.Fatalf("resolved replacement = %q, %v", value, err)
	}
	if store.values[legacyKey] != legacy {
		t.Fatal("activation rewrote the old ambiguous source item")
	}
}

func TestAmbiguousLegacySnapshotCASRefusesChangedSource(t *testing.T) {
	store := &candidateActivationStore{values: map[string]string{}}
	identity, _ := ParseIdentity("vrooli/legacy-marker-conflict")
	legacyKey := credentialService + "/" + storeKey(identity, "token")
	store.values[legacyKey] = legacyActiveCredentialRecordPrefix + "original"
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "replacement")
	if err != nil {
		t.Fatal(err)
	}
	store.values[legacyKey] = legacyActiveCredentialRecordPrefix + "changed"
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("changed ambiguous source was not refused: %v", err)
	}
	if got := store.values[legacyKey]; got != legacyActiveCredentialRecordPrefix+"changed" {
		t.Fatal("CAS refusal changed source")
	}
}

func TestCandidateActivationReportsCommittedValueWhenCleanupFails(t *testing.T) {
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/cleanup-pending")
	initial, err := authority.PutCandidate(identity, "token", "initial")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(initial); err != nil {
		t.Fatal(err)
	}
	candidate, err := authority.PutCandidate(identity, "token", "new-active")
	if err != nil {
		t.Fatal(err)
	}
	store.failCandidateDelete = true
	err = authority.ActivateCandidateIfVersion(candidate, initial.Version)
	if !errors.Is(err, ErrCandidateCleanupPending) {
		t.Fatalf("activation with candidate cleanup outage = %v, want committed cleanup-pending error", err)
	}
	status := authority.Status(identity, "token")
	active, resolveErr := authority.Resolve(identity, "token")
	if !status.Configured || status.Version != candidate.Version || resolveErr != nil || active != "new-active" {
		t.Fatalf("cleanup error obscured commit: configured=%t version-match=%t value-match=%t resolveErr=%v", status.Configured, status.Version == candidate.Version, active == "new-active", resolveErr)
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

func TestCandidateActivationKeepsAtomicActiveRecordOnStoreFailure(t *testing.T) {
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
	store.failActive = true
	if err := authority.ActivateCandidate(candidate); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("failed active-record commit = %v, want provider unavailable", err)
	}
	store.failActive = false
	active, err := authority.Resolve(identity, "token")
	if err != nil || active != "active-value" {
		t.Fatalf("active value after failed atomic commit = %q, %v", active, err)
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

func TestCandidateActivationUsesExpectedVersionAcrossAuthorityInstances(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	store := &candidateActivationStore{}
	firstAuthority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	secondAuthority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/candidate-cas")
	initial, err := firstAuthority.PutCandidate(identity, "token", "initial")
	if err != nil {
		t.Fatal(err)
	}
	if err := firstAuthority.ActivateCandidate(initial); err != nil {
		t.Fatal(err)
	}
	candidateA, err := firstAuthority.PutCandidate(identity, "token", "candidate-a")
	if err != nil {
		t.Fatal(err)
	}
	candidateB, err := secondAuthority.PutCandidate(identity, "token", "candidate-b")
	if err != nil {
		t.Fatal(err)
	}
	if candidateA.ExpectedVersion != initial.Version || candidateB.ExpectedVersion != initial.Version {
		t.Fatal("candidates did not capture the active version at staging")
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, attempt := range []struct {
		authority *Authority
		candidate CandidateRef
	}{{firstAuthority, candidateA}, {secondAuthority, candidateB}} {
		wg.Add(1)
		go func(authority *Authority, candidate CandidateRef) {
			defer wg.Done()
			<-start
			results <- authority.ActivateCandidateIfVersion(candidate, initial.Version)
		}(attempt.authority, attempt.candidate)
	}
	close(start)
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrVersionConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected concurrent activation error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent activations: successes=%d conflicts=%d", successes, conflicts)
	}
	active, err := secondAuthority.Resolve(identity, "token")
	if err != nil || (active != "candidate-a" && active != "candidate-b") {
		t.Fatalf("active value after candidate race = %q err=%v", active, err)
	}
}

func TestCandidateActivationDefaultUsesCapturedVersion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	store := &candidateActivationStore{}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/candidate-default-cas")
	first, err := authority.PutCandidate(identity, "token", "first")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(first); err != nil {
		t.Fatal(err)
	}
	stale, err := authority.PutCandidate(identity, "token", "stale")
	if err != nil {
		t.Fatal(err)
	}
	current, err := authority.PutCandidate(identity, "token", "current")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(current); err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("default candidate activation = %v, want version conflict", err)
	}
}

func TestCandidateActivationSerializesProcesses(t *testing.T) {
	if os.Getenv("VROOLI_TEST_CREDENTIAL_CAS_CHILD") == "1" {
		path := os.Getenv("VROOLI_TEST_CREDENTIAL_CAS_PATH")
		readyPath := os.Getenv("VROOLI_TEST_CREDENTIAL_CAS_READY")
		gatePath := os.Getenv("VROOLI_TEST_CREDENTIAL_CAS_GATE")
		if err := os.WriteFile(readyPath, []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		for {
			if _, err := os.Stat(gatePath); err == nil {
				break
			}
			time.Sleep(time.Millisecond)
		}
		store, err := securestore.OpenEncryptedFile(path, "synthetic-test-passphrase")
		if err != nil {
			t.Fatal(err)
		}
		authority, err := NewAuthority(store)
		if err != nil {
			t.Fatal(err)
		}
		var candidate CandidateRef
		if err := json.Unmarshal([]byte(os.Getenv("VROOLI_TEST_CREDENTIAL_CAS_CANDIDATE")), &candidate); err != nil {
			t.Fatal(err)
		}
		err = authority.ActivateCandidateIfVersion(candidate, candidate.ExpectedVersion)
		if errors.Is(err, ErrVersionConflict) {
			os.Exit(42)
		}
		if err != nil {
			t.Fatalf("child activation failed: %v", err)
		}
		return
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	root := t.TempDir()
	store, err := securestore.OpenEncryptedFile(filepath.Join(root, "authority.sealed"), "synthetic-test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, _ := ParseIdentity("vrooli/candidate-process-cas")
	initial, err := authority.PutCandidate(identity, "token", "initial")
	if err != nil {
		t.Fatal(err)
	}
	if err := authority.ActivateCandidate(initial); err != nil {
		t.Fatal(err)
	}
	parentCandidate, err := authority.PutCandidate(identity, "token", "parent-wins-or-loses")
	if err != nil {
		t.Fatal(err)
	}
	childCandidate, err := authority.PutCandidate(identity, "token", "child-wins-or-loses")
	if err != nil {
		t.Fatal(err)
	}
	candidateJSON, err := json.Marshal(childCandidate)
	if err != nil {
		t.Fatal(err)
	}
	readyPath := filepath.Join(root, "child-ready")
	gatePath := filepath.Join(root, "race-start")
	cmd := exec.Command(os.Args[0], "-test.run=^TestCandidateActivationSerializesProcesses$")
	cmd.Env = append(os.Environ(),
		"VROOLI_TEST_CREDENTIAL_CAS_CHILD=1",
		"VROOLI_TEST_CREDENTIAL_CAS_PATH="+filepath.Join(root, "authority.sealed"),
		"VROOLI_TEST_CREDENTIAL_CAS_READY="+readyPath,
		"VROOLI_TEST_CREDENTIAL_CAS_GATE="+gatePath,
		"VROOLI_TEST_CREDENTIAL_CAS_CANDIDATE="+string(candidateJSON),
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	parentResult := make(chan error, 1)
	go func() { parentResult <- authority.ActivateCandidateIfVersion(parentCandidate, initial.Version) }()
	if err := os.WriteFile(gatePath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	childErr := cmd.Wait()
	var childExit *exec.ExitError
	childConflict := errors.As(childErr, &childExit) && childExit.ExitCode() == 42
	if childErr != nil && !childConflict {
		t.Fatalf("child process failed: %v", childErr)
	}
	parentErr := <-parentResult
	parentConflict := errors.Is(parentErr, ErrVersionConflict)
	if parentErr != nil && !parentConflict {
		t.Fatalf("parent activation failed: %v", parentErr)
	}
	if (parentErr == nil) == (childErr == nil) {
		t.Fatalf("expected exactly one process to activate: parent=%v child=%v", parentErr, childErr)
	}
	active, err := authority.Resolve(identity, "token")
	if err != nil || (active != "parent-wins-or-loses" && active != "child-wins-or-loses") {
		t.Fatalf("active value after process race = %q, err=%v", active, err)
	}
}

func TestCredentialStoreWriteLockPreservesConcurrentDifferentAddresses(t *testing.T) {
	if os.Getenv("VROOLI_TEST_CREDENTIAL_STORE_WRITE_CHILD") == "1" {
		path := os.Getenv("VROOLI_TEST_CREDENTIAL_STORE_WRITE_PATH")
		readyPath := os.Getenv("VROOLI_TEST_CREDENTIAL_STORE_WRITE_READY")
		gatePath := os.Getenv("VROOLI_TEST_CREDENTIAL_STORE_WRITE_GATE")
		if err := os.WriteFile(readyPath, []byte("ready"), 0o600); err != nil {
			t.Fatal(err)
		}
		for {
			if _, err := os.Stat(gatePath); err == nil {
				break
			}
			time.Sleep(time.Millisecond)
		}
		store, err := securestore.OpenEncryptedFile(path, "synthetic-test-passphrase")
		if err != nil {
			t.Fatal(err)
		}
		authority, err := NewAuthority(store)
		if err != nil {
			t.Fatal(err)
		}
		identity := Identity(os.Getenv("VROOLI_TEST_CREDENTIAL_STORE_WRITE_IDENTITY"))
		if err := authority.Put(identity, "token", "child-synthetic-value"); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	root := t.TempDir()
	path := filepath.Join(root, "authority.sealed")
	store, err := securestore.OpenEncryptedFile(path, "synthetic-test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	authority, err := NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	parentIdentity, _ := ParseIdentity("vrooli/parallel-parent")
	childIdentity, _ := ParseIdentity("vrooli/parallel-child")
	readyPath := filepath.Join(root, "child-ready")
	gatePath := filepath.Join(root, "race-start")
	cmd := exec.Command(os.Args[0], "-test.run=^TestCredentialStoreWriteLockPreservesConcurrentDifferentAddresses$")
	cmd.Env = append(os.Environ(),
		"VROOLI_TEST_CREDENTIAL_STORE_WRITE_CHILD=1",
		"VROOLI_TEST_CREDENTIAL_STORE_WRITE_PATH="+path,
		"VROOLI_TEST_CREDENTIAL_STORE_WRITE_READY="+readyPath,
		"VROOLI_TEST_CREDENTIAL_STORE_WRITE_GATE="+gatePath,
		"VROOLI_TEST_CREDENTIAL_STORE_WRITE_IDENTITY="+string(childIdentity),
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(readyPath); err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	parentDone := make(chan error, 1)
	go func() { parentDone <- authority.Put(parentIdentity, "token", "parent-synthetic-value") }()
	if err := os.WriteFile(gatePath, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child store write failed: %v", err)
	}
	if err := <-parentDone; err != nil {
		t.Fatalf("parent store write failed: %v", err)
	}

	readerStore, err := securestore.OpenEncryptedFile(path, "synthetic-test-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewAuthority(readerStore)
	if err != nil {
		t.Fatal(err)
	}
	for identity, want := range map[Identity]string{
		parentIdentity: "parent-synthetic-value",
		childIdentity:  "child-synthetic-value",
	} {
		got, err := reader.Resolve(identity, "token")
		if err != nil || got != want {
			t.Fatalf("concurrent cross-address write %s missing: value-match=%t err=%v", identity, got == want, err)
		}
	}
}
