package securestore

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/credentialpolicy"
)

const (
	sealedfileParameterA = 24
)

// The sealed file is the storage layer of the credential backend for hosts with
// no native credential store. Every native adapter — libsecret, Keychain,
// Credential Manager — is a desktop-session facility, so a headless server, a CI
// runner, or a Raspberry Pi has none of them and would otherwise have no path
// from a fresh install to a working credential.
//
// The contract this satisfies is not "no credential material on disk" but "no
// credential value that is recoverable from the file alone": each value is
// sealed with AES-256-GCM under a data key that is never written unwrapped, and
// the key-encryption keys live in the TPM or in the operator's head. Reading the
// file yields ciphertext. Owner-only permissions stay required and stop being
// what the protection rests on.

const (
	// sealedFormatVersion gates format evolution. A file written by a newer
	// Vrooli must be refused outright rather than read as an empty store,
	// because "empty" would invite an overwrite that destroys every entry.
	sealedFormatVersion = 1

	sealedFilePerm = tuning.PermSecret
	sealedDirPerm  = tuning.PermPrivateDir

	// dataKeyLen is AES-256. One random data key seals every entry and is never
	// written unwrapped; each key-encryption provider stores its own wrap of it.
	dataKeyLen = 32

	// aeadContext keeps a sealed entry from opening under any other AEAD in
	// this codebase even if a key were ever shared.
	aeadContext = "vrooli.securestore.sealedfile"
)

// Failure modes of the sealed file. Both wrap ErrUnavailable rather than
// ErrNotFound: a store we cannot parse or decrypt is a broken backend, and
// reporting it as "this key holds no value" is precisely the collapse that once
// let a host fault read as an unset credential.
var (
	// errSealedVersion means the file is a sealed store of a format this build
	// does not know. It is deliberately distinct from a corrupt file: the
	// operator action is to upgrade Vrooli, not to re-provision.
	errSealedVersion = errors.New("credential store file has an unsupported format version")
	// errSealedCorrupt means the file exists and did not survive verification —
	// truncated, malformed, tampered with, or written under a different key.
	errSealedCorrupt = errors.New("credential store file failed integrity verification")
)

// sealedFile is the on-disk envelope. Service and key names stay in cleartext
// on purpose: they are already public in resource manifests, so sealing them
// would buy nothing and would make listing entries require an unlock.
type sealedFile struct {
	Version int           `json:"version"`
	Wraps   []wrappedKey  `json:"wrapped_keys"`
	Entries []sealedEntry `json:"entries"`
}

// wrappedKey is one key-encryption key's copy of the data key. Several sit side
// by side so any single one can open the store, which is what lets a host gain a
// TPM later without re-encrypting a single stored value.
type wrappedKey struct {
	// Provider names the key-encryption provider that wrote this wrap.
	Provider string `json:"provider"`
	// KeyStore names what actually protects the wrap, for hosts where the
	// provider's own strength varies — a TPM versus a host key on the same
	// disk. It is reported to the operator rather than averaged away.
	KeyStore string `json:"key_store,omitempty"`
	// Params is provider-specific public material, such as a KDF salt and
	// iteration count. It never contains key material.
	Params json.RawMessage `json:"params,omitempty"`
	// Ciphertext is the wrapped data key, base64-encoded.
	Ciphertext string `json:"ciphertext"`
}

// sealedEntry is one credential value sealed under its own random nonce, so a
// partial write or a single tampered entry cannot corrupt unrelated entries.
type sealedEntry struct {
	Service    string `json:"service"`
	Key        string `json:"key"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// entryAAD binds the format version, service, and key into the AEAD additional
// data. Without it an attacker with write access to the file could relocate a
// sealed value under a different service and key — moving, say, a low-value
// token into the slot a privileged resource reads — without ever holding the
// data key. Every component is length-prefixed so no two different (service,
// key) pairs can produce the same bytes.
func entryAAD(version int, service, key string) []byte {
	aad := make([]byte, 0, len(aeadContext)+len(service)+len(key)+sealedfileParameterA)
	aad = appendLengthPrefixed(aad, []byte(aeadContext))
	aad = binary.BigEndian.AppendUint32(aad, uint32(version))
	aad = appendLengthPrefixed(aad, []byte(service))
	aad = appendLengthPrefixed(aad, []byte(key))
	return aad
}

func appendLengthPrefixed(dst, value []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(value)))
	return append(dst, value...)
}

// sealEntry encrypts one value under a fresh random nonce. The plaintext never
// reaches an error string, so a failure here is safe to log.
func sealEntry(dataKey []byte, service, key, value string) (sealedEntry, error) {
	sealed, err := credentialpolicy.Seal(dataKey, []byte(value), "store-entry/"+service+"/"+key, sealedFormatVersion)
	if err != nil {
		return sealedEntry{}, err
	}
	return sealedEntry{
		Service:    service,
		Key:        key,
		Nonce:      base64.StdEncoding.EncodeToString(sealed.Nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(sealed.Ciphertext),
	}, nil
}

// openEntry decrypts one entry. It verifies against the service and key the
// caller asked for rather than the ones recorded in the entry, so an entry moved
// under another name fails to open instead of answering for its new slot.
func openEntry(dataKey []byte, service, key string, entry sealedEntry) (string, error) {
	nonce, err := base64.StdEncoding.DecodeString(entry.Nonce)
	if err != nil {
		return "", fmt.Errorf("%w: entry nonce is not valid base64", errSealedCorrupt)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(entry.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: entry ciphertext is not valid base64", errSealedCorrupt)
	}
	plain, err := credentialpolicy.Open(dataKey, credentialpolicy.Envelope{
		Version: sealedFormatVersion, Purpose: "store-entry/" + service + "/" + key, Nonce: nonce, Ciphertext: ciphertext,
	})
	if err == nil {
		return string(plain), nil
	}
	// Existing stores used the local entryAAD framing. Keep this compatibility
	// reader while every new write uses credentialpolicy's common envelope.
	block, cipherErr := aes.NewCipher(dataKey)
	if cipherErr != nil {
		return "", cipherErr
	}
	gcm, cipherErr := cipher.NewGCM(block)
	if cipherErr != nil {
		return "", cipherErr
	}
	if len(nonce) != gcm.NonceSize() {
		return "", fmt.Errorf("%w: entry nonce is %d bytes, want %d", errSealedCorrupt, len(nonce), gcm.NonceSize())
	}
	plain, legacyErr := gcm.Open(nil, nonce, ciphertext, entryAAD(sealedFormatVersion, service, key))
	if legacyErr != nil {
		return "", fmt.Errorf("%w: %s/%s did not authenticate", errSealedCorrupt, service, key)
	}
	return string(plain), nil
}

// findEntry locates an entry by service and key. Entries are kept sorted, so
// this is the one place ordering is assumed.
func (file *sealedFile) findEntry(service, key string) (int, bool) {
	index := sort.Search(len(file.Entries), func(i int) bool {
		if file.Entries[i].Service != service {
			return file.Entries[i].Service > service
		}
		return file.Entries[i].Key >= key
	})
	if index < len(file.Entries) && file.Entries[index].Service == service && file.Entries[index].Key == key {
		return index, true
	}
	return index, false
}

// putEntry replaces an existing entry or inserts a new one in sorted position.
func (file *sealedFile) putEntry(entry sealedEntry) {
	index, found := file.findEntry(entry.Service, entry.Key)
	if found {
		file.Entries[index] = entry
		return
	}
	file.Entries = append(file.Entries, sealedEntry{})
	copy(file.Entries[index+1:], file.Entries[index:])
	file.Entries[index] = entry
}

// deleteEntry removes an entry and reports whether one was there. Reporting the
// difference is what lets the CLI tell an operator whether a delete actually
// revoked something.
func (file *sealedFile) deleteEntry(service, key string) bool {
	index, found := file.findEntry(service, key)
	if !found {
		return false
	}
	file.Entries = append(file.Entries[:index], file.Entries[index+1:]...)
	return true
}

// wrapFor returns this file's wrap for a named provider.
func (file *sealedFile) wrapFor(provider string) (wrappedKey, bool) {
	for _, wrap := range file.Wraps {
		if wrap.Provider == provider {
			return wrap, true
		}
	}
	return wrappedKey{}, false
}

// putWrap replaces a provider's wrap or appends a new one. Adding a wrap never
// touches the data key or any sealed entry, which is the whole reason a host
// that gains a TPM does not become a re-encryption event.
func (file *sealedFile) putWrap(wrap wrappedKey) {
	for index := range file.Wraps {
		if file.Wraps[index].Provider == wrap.Provider {
			file.Wraps[index] = wrap
			return
		}
	}
	file.Wraps = append(file.Wraps, wrap)
}

// readSealedFile loads and structurally validates the store. A missing file is
// reported as os.ErrNotExist so the caller can distinguish "not initialized on
// this host" from "initialized and broken"; the two have different operator
// actions and must not collapse.
func readSealedFile(path string) (*sealedFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file sealedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%w: %s is not a readable credential store", errSealedCorrupt, filepath.Base(path))
	}
	if file.Version != sealedFormatVersion {
		return nil, fmt.Errorf("%w: file declares version %d, this build writes version %d",
			errSealedVersion, file.Version, sealedFormatVersion)
	}
	if len(file.Wraps) == 0 {
		return nil, fmt.Errorf("%w: the store holds no wrapped data key, so nothing in it can ever be opened", errSealedCorrupt)
	}
	for _, entry := range file.Entries {
		if entry.Service == "" || entry.Key == "" || entry.Ciphertext == "" {
			return nil, fmt.Errorf("%w: an entry is missing its service, key, or ciphertext", errSealedCorrupt)
		}
	}
	return &file, nil
}

// writeSealedFile writes the store atomically: a temporary file in the same
// directory, fsync'd, then renamed over the target. An interrupted write leaves
// the previous file intact rather than a half-written one — which for a
// credential store is the difference between a failed command and an operator
// who has lost every value they hold.
func writeSealedFile(path string, file *sealedFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, sealedDirPerm); err != nil {
		return fmt.Errorf("create credential store directory: %w", err)
	}
	// A pre-existing directory keeps whatever mode it had, so strip any group
	// or other access it came with. Permissions are no longer the protection,
	// but a world-readable credential directory is still a defect. Only the
	// group and other bits are touched: narrowing the owner's own access is not
	// this function's business, and quietly re-granting owner write would
	// defeat an operator who deliberately froze the directory.
	if info, err := os.Stat(dir); err == nil && info.Mode().Perm()&tuning.PermGroupAndOtherMask != 0 {
		if err := os.Chmod(dir, info.Mode().Perm()&^tuning.PermGroupAndOtherMask); err != nil {
			return fmt.Errorf("restrict credential store directory: %w", err)
		}
	}
	file.Version = sealedFormatVersion
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}

	if err := config.WriteOwnedFileAtomic(path, data, sealedFilePerm); err != nil {
		return fmt.Errorf("replace credential store: %w", err)
	}
	return nil
}
