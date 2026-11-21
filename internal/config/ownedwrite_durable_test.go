package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDurableOwnedIntentSyncsNewAncestryAfterVisibleCommit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "new", "nested", "intent.json")
	var synced []string
	err := writeOwnedFileDurableWith(path, []byte("intent"), 0o600, WriteOwnedFileAtomic, func(dir string) error {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "intent" {
			t.Fatalf("directory synced before committed intent: %q %v", data, err)
		}
		synced = append(synced, dir)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	physical, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for dir := physical; ; dir = filepath.Dir(dir) {
		position := -1
		for i, observed := range synced {
			if observed == dir {
				position = i
				break
			}
		}
		if position <= last {
			t.Fatalf("physical ancestor %s missing or confirmed before its child: %v", dir, synced)
		}
		last = position
		if filepath.Dir(dir) == dir {
			break
		}
	}
}

func TestDurableOwnedIntentReportsUnconfirmedRename(t *testing.T) {
	path := filepath.Join(t.TempDir(), "intent.json")
	failure := errors.New("injected directory sync failure")
	err := writeOwnedFileDurableWith(path, []byte("intent"), 0o600, WriteOwnedFileAtomic, func(string) error { return failure })
	if !errors.Is(err, failure) {
		t.Fatalf("durability failure hidden: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "intent" {
		t.Fatalf("visible commit lost: %q %v", data, err)
	}
	// A retry must be able to make the same visible intent durable.
	if err := WriteOwnedFileDurable(path, []byte("intent"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDurableOwnedIntentDoesNotAcknowledgeFailedWrite(t *testing.T) {
	failure := errors.New("injected file sync failure")
	synced := false
	err := writeOwnedFileDurableWith(filepath.Join(t.TempDir(), "intent.json"), nil, 0o600, func(string, []byte, os.FileMode) error { return failure }, func(string) error { synced = true; return nil })
	if !errors.Is(err, failure) || synced {
		t.Fatalf("failed file commit advanced to acknowledgement: err=%v synced=%v", err, synced)
	}
}

func TestDurableOwnedRetryResyncsVisibleAncestryAfterLostAcknowledgement(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "new", "nested", "intent.json")
	lost := errors.New("injected parent-directory acknowledgement loss")
	if err := writeOwnedFileDurableWith(path, []byte("intent"), 0o600, WriteOwnedFileAtomic, func(dir string) error {
		if dir == root {
			return lost
		}
		return nil
	}); !errors.Is(err, lost) {
		t.Fatalf("initial durability failure not injected: %v", err)
	}
	// All directories are now visible. Their existence cannot establish that
	// the prior write persisted the entry for new/ in its existing parent.
	resyncedParent := false
	if err := writeOwnedFileDurableWith(path, []byte("intent"), 0o600, WriteOwnedFileAtomic, func(dir string) error {
		resyncedParent = resyncedParent || dir == root
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !resyncedParent {
		t.Fatal("retry acknowledged a visible file without confirming its ancestor entries")
	}
}

func TestDurableOwnedStreamRejectsInvalidPayloadBeforeReplacingExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retained-image")
	if err := os.WriteFile(path, []byte("verified-old-image"), 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := errors.New("injected streamed image checksum mismatch")
	err := WriteOwnedFileDurableStream(path, 0o700, func(dst io.Writer) error {
		if _, err := io.WriteString(dst, "invalid-candidate-image"); err != nil {
			return err
		}
		return invalid
	})
	if !errors.Is(err, invalid) {
		t.Fatalf("invalid streamed payload acknowledged: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "verified-old-image" {
		t.Fatal("stream validation failure replaced the retained image")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed stream leaked staging files: %v", err)
	}
}

func TestDurableOwnedStreamCommitsValidatedExecutableWithOwnedMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "nested", "image")
	if err := WriteOwnedFileDurableStream(path, 0o700, func(dst io.Writer) error {
		_, err := io.WriteString(dst, "verified-image")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("executable mode not committed: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "verified-image" {
		t.Fatal("validated streamed image not retained")
	}
}
