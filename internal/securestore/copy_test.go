package securestore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyStoreRefusesRepositorySink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	_, err := CopyStore(source, filepath.Join(root, "repo", "copies"), filepath.Join(root, "receipt.json"), []string{filepath.Join(root, "repo")})
	var conflict *SinkConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("CopyStore error = %v, want SinkConflictError", err)
	}
}

func TestCopyStoreRefusesSymlinkedRepositorySink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	repository := filepath.Join(root, "repo")
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	sinkLink := filepath.Join(root, "sink-link")
	if err := os.Symlink(repository, sinkLink); err != nil {
		t.Fatal(err)
	}
	_, err := CopyStore(source, filepath.Join(sinkLink, "copies"), filepath.Join(root, "receipt.json"), []string{repository})
	var conflict *SinkConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("CopyStore error = %v, want SinkConflictError", err)
	}
}

func TestCopyStoreWritesEncryptedFileAndReceiptAtomically(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	sink := filepath.Join(root, "off-host")
	receipt := filepath.Join(root, "state", "copy.json")
	status, err := CopyStore(source, sink, receipt, nil)
	if err != nil {
		t.Fatalf("CopyStore: %v", err)
	}
	if status.Generation == "" || status.Path != filepath.Join(sink, filepath.Base(source)) {
		t.Fatalf("unexpected copy status: %+v", status)
	}
	if _, err := os.Stat(status.Path); err != nil {
		t.Fatalf("copied store: %v", err)
	}
	if _, err := os.Stat(receipt); err != nil {
		t.Fatalf("copy receipt: %v", err)
	}
	info, err := os.Stat(status.Path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("copy mode = %o, want owner-only", info.Mode().Perm())
	}
	if status.Checksum == "" || status.Verification != "readback" || status.VerifiedAt.IsZero() {
		t.Fatalf("copy lacks verification evidence: %+v", status)
	}
}

func TestCopyStoreWithPolicyRejectsSinkInsideCredentialSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "store", "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	_, err := CopyStoreWithPolicy(source, filepath.Join(root, "store", "copies"), filepath.Join(root, "receipt.json"), CopyPolicy{ProtectedRoots: []string{filepath.Dir(source)}})
	var conflict *SinkConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("CopyStoreWithPolicy error = %v, want source containment conflict", err)
	}
}

func TestCopyStoreWithPolicyRejectsSamePhysicalDevice(t *testing.T) {
	sourceRoot := t.TempDir()
	sinkRoot := t.TempDir()
	source := filepath.Join(sourceRoot, "secrets.enc.json")
	newDrillEncryptedStore(t, source, "copy-passphrase")
	_, err := CopyStoreWithPolicy(source, filepath.Join(sinkRoot, "sibling"), filepath.Join(sinkRoot, "receipt.json"), CopyPolicy{RequireIndependentDevice: true})
	if err == nil || !strings.Contains(err.Error(), "physical device") {
		t.Fatalf("same-device policy error = %v", err)
	}
}

func TestChangePassphraseAdvancesStoreGeneration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	store := newEncryptedStore(path, passphraseProvider{passphrase: "old-passphrase"})
	if _, err := store.initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	first, err := StoreGeneration(path)
	if err != nil {
		t.Fatalf("initial generation: %v", err)
	}
	if !strings.HasPrefix(first, "1:") {
		t.Fatalf("initial generation = %q, want generation 1", first)
	}
	if err := changePassphraseStore(newEncryptedStore(path, passphraseProvider{passphrase: "old-passphrase"}), "old-passphrase", "new-passphrase"); err != nil {
		t.Fatalf("ChangePassphraseStore: %v", err)
	}
	second, err := StoreGeneration(path)
	if err != nil {
		t.Fatalf("rotated generation: %v", err)
	}
	if !strings.HasPrefix(second, "2:") || second == first {
		t.Fatalf("rotated generation = %q, want a distinct generation 2", second)
	}
	if _, _, err := newEncryptedStore(path, passphraseProvider{passphrase: "new-passphrase"}).open(); err != nil {
		t.Fatalf("new passphrase does not open store: %v", err)
	}
}
