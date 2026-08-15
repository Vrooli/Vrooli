package securestore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type migrationTestStore struct {
	values map[string]string
	failOn string
}

func (s *migrationTestStore) Put(service, key, value string) error {
	if service+"/"+key == s.failOn {
		return errors.New("injected destination failure")
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *migrationTestStore) Get(service, key string) (string, error) {
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (s *migrationTestStore) Delete(service, key string) error {
	delete(s.values, service+"/"+key)
	return nil
}

func TestBackendSelectionRoundTripsWithoutCredentialMaterial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "credential-backend.json")
	previous := backendSelectionPath
	backendSelectionPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { backendSelectionPath = previous })

	if _, found, err := SelectedBackend(); err != nil || found {
		t.Fatalf("missing selection = found %t, err %v; want absent without error", found, err)
	}
	if err := SelectBackend(BackendEncryptedFile, "native backend was not writable during setup"); err != nil {
		t.Fatalf("SelectBackend: %v", err)
	}
	backend, found, err := SelectedBackend()
	if err != nil || !found || backend != BackendEncryptedFile {
		t.Fatalf("SelectedBackend = %q, found %t, err %v; want encrypted-file", backend, found, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read selection: %v", err)
	}
	if string(data) == "" || filepath.Base(path) != "credential-backend.json" {
		t.Fatalf("selection file was not written")
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("selection permissions = %v, err %v; want 0600", info.Mode().Perm(), err)
	}
}

func TestBackendSelectionRejectsUnknownBackend(t *testing.T) {
	if err := SelectBackend("split-brain", "test"); err == nil {
		t.Fatal("SelectBackend accepted an unknown backend")
	}
}

func TestReselectBackendMigratesOnlyAfterReadback(t *testing.T) {
	replaceSetupBackendSeams(t)
	path := filepath.Join(t.TempDir(), "state", "credential-backend.json")
	backendSelectionPath = func() (string, error) { return path, nil }
	source := &migrationTestStore{values: map[string]string{"vrooli.credentials.v1/vrooli/demo:token": "secret"}}
	destination := &migrationTestStore{}
	nativeStoreForSelectionFn = func() Store { return destination }
	encryptedStoreForSelectionFn = func() Store { return source }
	diagnoseNativeForSetupFn = func() Diagnosis { return Diagnosis{Available: true, Writable: true} }
	if err := SelectBackend(BackendEncryptedFile, "initial fallback"); err != nil {
		t.Fatal(err)
	}
	receipt, err := ReselectBackend([]MigrationEntry{{Service: "vrooli.credentials.v1", Key: "vrooli/demo:token"}})
	if err != nil {
		t.Fatalf("ReselectBackend: %v", err)
	}
	if !receipt.Committed || len(receipt.Verified) != 1 {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
	backend, found, err := SelectedBackend()
	if err != nil || !found || backend != BackendNative {
		t.Fatalf("selected backend = %q, found=%t, err=%v", backend, found, err)
	}
	if got := destination.values["vrooli.credentials.v1/vrooli/demo:token"]; got != "secret" {
		t.Fatalf("migrated value = %q, want secret", got)
	}
}

func TestReselectBackendPartialFailureLeavesSelectionAndSourceIntact(t *testing.T) {
	replaceSetupBackendSeams(t)
	path := filepath.Join(t.TempDir(), "state", "credential-backend.json")
	backendSelectionPath = func() (string, error) { return path, nil }
	source := &migrationTestStore{values: map[string]string{
		"vrooli.credentials.v1/vrooli/demo:first":  "one",
		"vrooli.credentials.v1/vrooli/demo:second": "two",
	}}
	destination := &migrationTestStore{failOn: "vrooli.credentials.v1/vrooli/demo:second"}
	nativeStoreForSelectionFn = func() Store { return destination }
	encryptedStoreForSelectionFn = func() Store { return source }
	diagnoseNativeForSetupFn = func() Diagnosis { return Diagnosis{Available: true, Writable: true} }
	if err := SelectBackend(BackendEncryptedFile, "initial fallback"); err != nil {
		t.Fatal(err)
	}
	_, err := ReselectBackend([]MigrationEntry{
		{Service: "vrooli.credentials.v1", Key: "vrooli/demo:first"},
		{Service: "vrooli.credentials.v1", Key: "vrooli/demo:second"},
	})
	if err == nil {
		t.Fatal("partial migration unexpectedly succeeded")
	}
	backend, found, readErr := SelectedBackend()
	if readErr != nil || !found || backend != BackendEncryptedFile {
		t.Fatalf("selection after failed migration = %q, found=%t, err=%v", backend, found, readErr)
	}
	if len(destination.values) != 0 {
		t.Fatalf("failed migration left destination values: %#v", destination.values)
	}
	if source.values["vrooli.credentials.v1/vrooli/demo:first"] != "one" || source.values["vrooli.credentials.v1/vrooli/demo:second"] != "two" {
		t.Fatalf("failed migration changed source: %#v", source.values)
	}
}
