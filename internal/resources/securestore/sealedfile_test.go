package securestore

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testDataKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, dataKeyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("generate data key: %v", err)
	}
	return key
}

func TestSealedEntryRoundTrips(t *testing.T) {
	key := testDataKey(t)
	for _, testCase := range []struct {
		name  string
		value string
	}{
		{"ascii", "plain-value"},
		{"unicode", "pässwörd-🔐-日本語"},
		{"4 KiB", strings.Repeat("k", 4096)},
		{"empty", ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			entry, err := sealEntry(key, "svc", "k", testCase.value)
			if err != nil {
				t.Fatalf("sealEntry: %v", err)
			}
			if strings.Contains(entry.Ciphertext, testCase.value) && testCase.value != "" {
				t.Fatalf("sealed ciphertext contains the plaintext")
			}
			got, err := openEntry(key, "svc", "k", entry)
			if err != nil {
				t.Fatalf("openEntry: %v", err)
			}
			if got != testCase.value {
				t.Fatalf("openEntry returned %d bytes, want %d", len(got), len(testCase.value))
			}
		})
	}
}

// TestSealedEntryNonceIsFreshPerSeal is why the same value sealed twice must not
// produce the same ciphertext: a fixed nonce under one key both leaks equality
// between entries and breaks GCM outright.
func TestSealedEntryNonceIsFreshPerSeal(t *testing.T) {
	key := testDataKey(t)
	first, err := sealEntry(key, "svc", "k", "same-value")
	if err != nil {
		t.Fatalf("sealEntry: %v", err)
	}
	second, err := sealEntry(key, "svc", "k", "same-value")
	if err != nil {
		t.Fatalf("sealEntry: %v", err)
	}
	if first.Nonce == second.Nonce {
		t.Fatalf("two seals reused the nonce %q", first.Nonce)
	}
	if first.Ciphertext == second.Ciphertext {
		t.Fatalf("two seals of the same value produced identical ciphertext")
	}
}

// TestSealedEntryRejectsRelocation is the AEAD additional-data requirement. An
// attacker with write access to the file must not be able to move a sealed
// value into another resource's slot without holding the data key.
func TestSealedEntryRejectsRelocation(t *testing.T) {
	key := testDataKey(t)
	entry, err := sealEntry(key, "low-value-service", "token", "s3cret")
	if err != nil {
		t.Fatalf("sealEntry: %v", err)
	}

	relocated := entry
	relocated.Service = "privileged-service"
	relocated.Key = "root-token"
	if _, err := openEntry(key, "privileged-service", "root-token", relocated); !errors.Is(err, errSealedCorrupt) {
		t.Fatalf("relocated entry opened under a new name: err = %v, want errSealedCorrupt", err)
	}
	// The same rewritten entry still opens under its ORIGINAL names, and that is
	// the point rather than a leak: the names are authenticated from the
	// caller's arguments, never from the file. An attacker who rewrites the
	// recorded names therefore cannot relocate a value into a privileged slot —
	// the assertion above proves the relocation fails — they can only corrupt
	// the lookup, which fails closed.
	//
	// Do not "fix" this to expect an error. Expecting one would mean the file's
	// own fields decide what gets verified, which is exactly the property that
	// would make relocation possible.
	if _, err := openEntry(key, "low-value-service", "token", relocated); err != nil {
		t.Fatalf("the caller's names are not what is verified; open under the original names failed: %v", err)
	}
}

func TestSealedEntryFailsClosedOnCorruption(t *testing.T) {
	key := testDataKey(t)
	seal := func(t *testing.T) sealedEntry {
		t.Helper()
		entry, err := sealEntry(key, "svc", "k", "value-under-test")
		if err != nil {
			t.Fatalf("sealEntry: %v", err)
		}
		return entry
	}

	t.Run("flipped ciphertext byte", func(t *testing.T) {
		entry := seal(t)
		raw, err := base64.StdEncoding.DecodeString(entry.Ciphertext)
		if err != nil {
			t.Fatalf("decode ciphertext: %v", err)
		}
		raw[0] ^= 0x01
		entry.Ciphertext = base64.StdEncoding.EncodeToString(raw)
		assertSealedFailure(t, entry, key)
	})

	t.Run("flipped nonce byte", func(t *testing.T) {
		entry := seal(t)
		raw, err := base64.StdEncoding.DecodeString(entry.Nonce)
		if err != nil {
			t.Fatalf("decode nonce: %v", err)
		}
		raw[0] ^= 0x01
		entry.Nonce = base64.StdEncoding.EncodeToString(raw)
		assertSealedFailure(t, entry, key)
	})

	t.Run("truncated ciphertext", func(t *testing.T) {
		entry := seal(t)
		raw, err := base64.StdEncoding.DecodeString(entry.Ciphertext)
		if err != nil {
			t.Fatalf("decode ciphertext: %v", err)
		}
		entry.Ciphertext = base64.StdEncoding.EncodeToString(raw[:len(raw)-4])
		assertSealedFailure(t, entry, key)
	})

	t.Run("non-base64 ciphertext", func(t *testing.T) {
		entry := seal(t)
		entry.Ciphertext = "this is not base64!!"
		assertSealedFailure(t, entry, key)
	})

	t.Run("short nonce", func(t *testing.T) {
		entry := seal(t)
		entry.Nonce = base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
		assertSealedFailure(t, entry, key)
	})

	t.Run("wrong data key", func(t *testing.T) {
		entry := seal(t)
		if _, err := openEntry(testDataKey(t), "svc", "k", entry); !errors.Is(err, errSealedCorrupt) {
			t.Fatalf("open under a foreign key: err = %v, want errSealedCorrupt", err)
		}
	})
}

// assertSealedFailure holds the property every corruption case shares: the read
// fails, the failure is distinguishable from a missing entry, and no plaintext
// leaks into the error string.
func assertSealedFailure(t *testing.T, entry sealedEntry, key []byte) {
	t.Helper()
	value, err := openEntry(key, "svc", "k", entry)
	if err == nil {
		t.Fatalf("corrupt entry opened cleanly and returned %d bytes", len(value))
	}
	if errors.Is(err, ErrNotFound) {
		t.Fatalf("corruption reported as a missing entry: %v", err)
	}
	if !errors.Is(err, errSealedCorrupt) {
		t.Fatalf("corruption error = %v, want errSealedCorrupt", err)
	}
	if strings.Contains(err.Error(), "value-under-test") {
		t.Fatalf("error message leaked the credential value: %v", err)
	}
}

func writeTestStore(t *testing.T, dir string, file *sealedFile) string {
	t.Helper()
	path := filepath.Join(dir, "credentials.enc.json")
	if err := writeSealedFile(path, file); err != nil {
		t.Fatalf("writeSealedFile: %v", err)
	}
	return path
}

func testStoreWithOneWrap() *sealedFile {
	return &sealedFile{
		Version: sealedFormatVersion,
		Wraps:   []wrappedKey{{Provider: "passphrase", Ciphertext: base64.StdEncoding.EncodeToString([]byte("wrapped"))}},
	}
}

func TestSealedFileWritesOwnerOnlyAndAtomically(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "store")
	file := testStoreWithOneWrap()
	path := writeTestStore(t, dir, file)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if got := info.Mode().Perm(); got != fs.FileMode(sealedFilePerm) {
		t.Fatalf("store file mode = %o, want %o", got, sealedFilePerm)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat store directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != fs.FileMode(sealedDirPerm) {
		t.Fatalf("store directory mode = %o, want %o", got, sealedDirPerm)
	}

	// The write is temp-then-rename, so nothing must be left behind.
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read store directory: %v", err)
	}
	if len(names) != 1 {
		leftovers := make([]string, 0, len(names))
		for _, name := range names {
			leftovers = append(leftovers, name.Name())
		}
		t.Fatalf("write left %d files behind: %v", len(names), leftovers)
	}
}

// TestSealedFileTightensALooseExistingDirectory covers the host where the state
// directory predates this store and was created world-readable. Permissions are
// not what the sealed file relies on, but leaving them loose is still a defect.
func TestSealedFileTightensALooseExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod store directory: %v", err)
	}
	writeTestStore(t, dir, testStoreWithOneWrap())

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat store directory: %v", err)
	}
	if got := info.Mode().Perm() & 0o077; got != 0 {
		t.Fatalf("store directory still grants %o to group and other", got)
	}
}

// TestSealedFileWriteLeavesPreviousFileIntactWhenInterrupted is the interrupted
// write case: an operator whose disk fills mid-write must still hold every value
// they had, not a truncated file.
func TestSealedFileWriteLeavesPreviousFileIntactWhenInterrupted(t *testing.T) {
	dir := t.TempDir()
	key := testDataKey(t)
	file := testStoreWithOneWrap()
	entry, err := sealEntry(key, "svc", "kept", "still-here")
	if err != nil {
		t.Fatalf("sealEntry: %v", err)
	}
	file.putEntry(entry)
	path := writeTestStore(t, dir, file)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}

	// Make the temporary file impossible to create by removing write access to
	// the directory: the same observable state as any interrupted write, and
	// the assertion is about what survives.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod store directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, sealedDirPerm) })

	file.putEntry(sealedEntry{Service: "svc", Key: "new", Nonce: "n", Ciphertext: "c"})
	if err := writeSealedFile(path, file); err == nil {
		t.Fatalf("write into a read-only directory succeeded")
	}

	if err := os.Chmod(dir, sealedDirPerm); err != nil {
		t.Fatalf("restore store directory: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read store: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("a failed write modified the previous store file")
	}
	reloaded, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("readSealedFile after failed write: %v", err)
	}
	index, found := reloaded.findEntry("svc", "kept")
	if !found {
		t.Fatalf("the surviving store lost its entry")
	}
	value, err := openEntry(key, "svc", "kept", reloaded.Entries[index])
	if err != nil || value != "still-here" {
		t.Fatalf("surviving entry = %q, %v; want %q, nil", value, err, "still-here")
	}
}

func TestReadSealedFileRejectsUnknownVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeTestStore(t, dir, testStoreWithOneWrap())

	var raw map[string]any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode store: %v", err)
	}
	raw["version"] = sealedFormatVersion + 1
	rewritten, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("encode store: %v", err)
	}
	if err := os.WriteFile(path, rewritten, sealedFilePerm); err != nil {
		t.Fatalf("write store: %v", err)
	}

	file, err := readSealedFile(path)
	if !errors.Is(err, errSealedVersion) {
		t.Fatalf("readSealedFile error = %v, want errSealedVersion", err)
	}
	if errors.Is(err, errSealedCorrupt) {
		t.Fatalf("an unknown version must not be reported as corruption: %v", err)
	}
	if file != nil {
		t.Fatalf("readSealedFile returned a store for an unknown version: %+v", file)
	}
}

func TestReadSealedFileRejectsStructuralDamage(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		content string
	}{
		{"truncated json", `{"version":1,"wrapped_keys":[{"provider":"passphrase",`},
		{"not json at all", "this file used to be a credential store"},
		{"empty file", ""},
		{"no wrapped key", `{"version":1,"wrapped_keys":[],"entries":[]}`},
		{"entry without a key", `{"version":1,"wrapped_keys":[{"provider":"p","ciphertext":"aGk="}],"entries":[{"service":"svc","nonce":"n","ciphertext":"c"}]}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "credentials.enc.json")
			if err := os.WriteFile(path, []byte(testCase.content), sealedFilePerm); err != nil {
				t.Fatalf("write store: %v", err)
			}
			file, err := readSealedFile(path)
			if !errors.Is(err, errSealedCorrupt) {
				t.Fatalf("readSealedFile error = %v, want errSealedCorrupt", err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("damage reported as a missing entry: %v", err)
			}
			if file != nil {
				t.Fatalf("readSealedFile returned a store for a damaged file")
			}
		})
	}
}

// TestReadSealedFileDistinguishesMissingFromDamaged is the distinction that
// decides whether the operator initializes a store or stops touching it: a host
// with no store yet is not a host with a broken one.
func TestReadSealedFileDistinguishesMissingFromDamaged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.enc.json")
	_, err := readSealedFile(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readSealedFile on a missing file = %v, want os.ErrNotExist", err)
	}
	if errors.Is(err, errSealedCorrupt) || errors.Is(err, errSealedVersion) {
		t.Fatalf("an uninitialized host was reported as a broken store: %v", err)
	}
}

func TestSealedFileEntryBookkeeping(t *testing.T) {
	file := testStoreWithOneWrap()
	for _, name := range []string{"delta", "alpha", "charlie", "bravo"} {
		file.putEntry(sealedEntry{Service: "svc", Key: name, Nonce: "n", Ciphertext: "c"})
	}
	if len(file.Entries) != 4 {
		t.Fatalf("entries = %d, want 4", len(file.Entries))
	}
	for index := 1; index < len(file.Entries); index++ {
		if file.Entries[index-1].Key >= file.Entries[index].Key {
			t.Fatalf("entries are not sorted: %v", file.Entries)
		}
	}

	// Put replaces rather than duplicating.
	file.putEntry(sealedEntry{Service: "svc", Key: "alpha", Nonce: "n2", Ciphertext: "c2"})
	if len(file.Entries) != 4 {
		t.Fatalf("overwrite duplicated an entry: %d entries", len(file.Entries))
	}
	index, found := file.findEntry("svc", "alpha")
	if !found || file.Entries[index].Ciphertext != "c2" {
		t.Fatalf("overwrite did not replace the entry: %+v", file.Entries)
	}

	if !file.deleteEntry("svc", "alpha") {
		t.Fatalf("deleteEntry reported nothing removed for a present entry")
	}
	if file.deleteEntry("svc", "alpha") {
		t.Fatalf("deleteEntry reported a removal for an absent entry")
	}
	if _, found := file.findEntry("svc", "alpha"); found {
		t.Fatalf("deleted entry is still findable")
	}
	// A different service with the same key name must stay separate.
	file.putEntry(sealedEntry{Service: "other", Key: "bravo", Nonce: "n", Ciphertext: "other-c"})
	index, found = file.findEntry("svc", "bravo")
	if !found || file.Entries[index].Ciphertext != "c" {
		t.Fatalf("service names are not distinguishing entries: %+v", file.Entries)
	}
}

func TestSealedFileWrapBookkeeping(t *testing.T) {
	file := testStoreWithOneWrap()
	file.putWrap(wrappedKey{Provider: "host-bound", KeyStore: "tpm2", Ciphertext: "wrapped-by-tpm"})
	if len(file.Wraps) != 2 {
		t.Fatalf("wraps = %d, want 2", len(file.Wraps))
	}
	file.putWrap(wrappedKey{Provider: "host-bound", KeyStore: "host-key", Ciphertext: "rewrapped"})
	if len(file.Wraps) != 2 {
		t.Fatalf("replacing a wrap appended instead: %d wraps", len(file.Wraps))
	}
	wrap, found := file.wrapFor("host-bound")
	if !found || wrap.KeyStore != "host-key" || wrap.Ciphertext != "rewrapped" {
		t.Fatalf("wrapFor returned %+v", wrap)
	}
	if _, found := file.wrapFor("nonexistent"); found {
		t.Fatalf("wrapFor invented a wrap that was never stored")
	}
}

// TestSealedFileAddingAWrapLeavesEntriesReadable is the property that keeps a
// host which gains a TPM from becoming a re-encryption event: only the wrap
// list changes, and the data key that sealed every entry does not.
func TestSealedFileAddingAWrapLeavesEntriesReadable(t *testing.T) {
	dir := t.TempDir()
	key := testDataKey(t)
	file := testStoreWithOneWrap()
	entry, err := sealEntry(key, "svc", "k", "unchanged-by-rewrap")
	if err != nil {
		t.Fatalf("sealEntry: %v", err)
	}
	file.putEntry(entry)
	path := writeTestStore(t, dir, file)

	reloaded, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("readSealedFile: %v", err)
	}
	reloaded.putWrap(wrappedKey{Provider: "host-bound", KeyStore: "tpm2", Ciphertext: "second-wrap"})
	if err := writeSealedFile(path, reloaded); err != nil {
		t.Fatalf("writeSealedFile: %v", err)
	}

	final, err := readSealedFile(path)
	if err != nil {
		t.Fatalf("readSealedFile after rewrap: %v", err)
	}
	index, found := final.findEntry("svc", "k")
	if !found {
		t.Fatalf("adding a wrap lost the entry")
	}
	value, err := openEntry(key, "svc", "k", final.Entries[index])
	if err != nil || value != "unchanged-by-rewrap" {
		t.Fatalf("entry after rewrap = %q, %v; want %q, nil", value, err, "unchanged-by-rewrap")
	}
}
