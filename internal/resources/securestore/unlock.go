package securestore

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A passphrase wrap that re-prompted on every CLI invocation would be
// unusable — provisioning three credentials would mean typing a passphrase
// three times, and a scenario start that reads six of them would be
// impossible. One unlock has to serve a session.
//
// What is cached is the unwrapped data key, not the passphrase: a stolen data
// key opens one store, while a stolen passphrase is a thing operators reuse.
// It is written only into the session runtime directory, which on Linux is a
// tmpfs that logind removes at logout and that never survives a reboot. This is
// the same place ssh-agent and gnome-keyring keep their session material.
//
// The host-bound wrap never touches any of this. It needs no unlock, so a
// server or a Pi with a working TPM never creates a cache entry at all — which
// is why this whole mechanism could be dropped without stranding the feature.

// unlockCacheDirName is the subdirectory Vrooli owns inside the session runtime
// directory.
const unlockCacheDirName = "vrooli"

// unlockCacheFileName holds the open data key for the current session.
const unlockCacheFileName = "credential-unlock"

// unlockCache is the session-scoped home for an open data key. A host with no
// usable session directory gets the no-op implementation, so an unlock lasts
// exactly one process there rather than silently landing on real disk.
type unlockCache interface {
	// Load returns the cached data key for a store fingerprint.
	Load(fingerprint string) ([]byte, bool)
	// Save stores the data key for later invocations in this session.
	Save(fingerprint string, dataKey []byte) error
	// Clear discards it immediately.
	Clear() error
	// Location describes where the cache lives, for the operator diagnosis. It
	// is empty when there is no cache on this host.
	Location() string
}

// storeFingerprint identifies which store a cached key belongs to, so
// re-initializing a store invalidates every cached key for the old one rather
// than handing a stale key to a new file. It reveals nothing: everything it
// hashes is already in the file.
//
// It is derived from the passphrase wrap alone, and that is the whole point.
// Only the passphrase path ever writes this cache, so the passphrase wrap is
// the exact thing whose change should invalidate it: a re-initialized store and
// a changed passphrase both produce a new one, and both must invalidate. An
// earlier version hashed every wrap, which meant *adding* a wrap invalidated
// the cache too — so the moment setup converged a host on unattended access, it
// knocked the operator's own live session out of its unlock and made them type
// the passphrase again to finish the run that had just removed the need for it.
// Adding a wrap does not change the data key, so the cached key stays correct.
//
// An empty result means there is nothing to cache against: a store with no
// passphrase wrap is opened by a provider that needs no unlock.
func storeFingerprint(file *sealedFile) string {
	wrap, found := file.wrapFor(providerPassphrase)
	if !found {
		return ""
	}
	// Only the ciphertext is hashed. The KDF parameters would add nothing —
	// every rewrap draws a fresh salt and nonce, so the ciphertext already
	// changes whenever they do — and they are not byte-stable across a
	// write/read cycle, because the sealed file is written indented and a
	// json.RawMessage is re-indented with it.
	digest := sha256.New()
	digest.Write([]byte(wrap.Provider))
	digest.Write([]byte{0})
	digest.Write([]byte(wrap.Ciphertext))
	return hex.EncodeToString(digest.Sum(nil))
}

// legacyStoreFingerprint is the identity written before it narrowed to the
// passphrase wrap. It is still accepted on read, and only on read.
//
// Changing how a cache is keyed silently invalidates every entry written under
// the old scheme, and here that would mean logging out every live session at
// the moment the upgrade landed — asking for the passphrase in order to install
// the fix that stops asking for the passphrase. Accepting the old form once and
// rewriting the entry under the new one costs a few lines and makes the change
// invisible to the operator, which is what an upgrade should be.
func legacyStoreFingerprint(file *sealedFile) string {
	digest := sha256.New()
	for _, wrap := range file.Wraps {
		digest.Write([]byte(wrap.Provider))
		digest.Write([]byte{0})
		digest.Write([]byte(wrap.Ciphertext))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// noUnlockCache is the honest answer on a host with no session-scoped memory:
// an unlock serves this process and no other. Writing the key somewhere durable
// instead would trade a re-prompt for a key sitting on disk after logout.
type noUnlockCache struct{}

func (noUnlockCache) Load(string) ([]byte, bool) { return nil, false }
func (noUnlockCache) Save(string, []byte) error  { return nil }
func (noUnlockCache) Clear() error               { return nil }
func (noUnlockCache) Location() string           { return "" }

// fileUnlockCache keeps the data key in the session runtime directory.
type fileUnlockCache struct{ path string }

func (cache fileUnlockCache) Location() string { return cache.path }

func (cache fileUnlockCache) Load(fingerprint string) ([]byte, bool) {
	data, err := os.ReadFile(cache.path)
	if err != nil {
		return nil, false
	}
	storedFingerprint, encodedKey, found := strings.Cut(strings.TrimSpace(string(data)), " ")
	if !found || storedFingerprint != fingerprint {
		// A cache for a different store is not a failure; it is a store that
		// was re-initialized. Silently ignoring it is correct, and using it
		// would hand the wrong key to the new file.
		return nil, false
	}
	dataKey, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil || len(dataKey) != dataKeyLen {
		return nil, false
	}
	return dataKey, true
}

func (cache fileUnlockCache) Save(fingerprint string, dataKey []byte) error {
	dir := filepath.Dir(cache.path)
	if err := os.MkdirAll(dir, sealedDirPerm); err != nil {
		return fmt.Errorf("create credential unlock directory: %w", err)
	}
	payload := fingerprint + " " + base64.StdEncoding.EncodeToString(dataKey)
	// Written with O_EXCL-free truncation but owner-only mode, in a directory
	// only this user can enter. The tmpfs it sits on is the protection that
	// matters: it is gone at logout.
	if err := os.WriteFile(cache.path, []byte(payload), sealedFilePerm); err != nil {
		return fmt.Errorf("write credential unlock: %w", err)
	}
	return nil
}

func (cache fileUnlockCache) Clear() error {
	if err := os.Remove(cache.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("discard credential unlock: %w", err)
	}
	return nil
}

// sessionUnlockCache is the cache for this host, or the no-op one when there is
// no session directory that is provably ours. It is a variable so tests can
// point it at a temporary directory.
var sessionUnlockCache = func() unlockCache {
	dir, ok := sessionRuntimeDir()
	if !ok {
		return noUnlockCache{}
	}
	return fileUnlockCache{path: filepath.Join(dir, unlockCacheDirName, unlockCacheFileName)}
}
