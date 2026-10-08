package securestore

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// The operator surface for the encrypted store. Every native adapter is managed
// by the operating system, so this is the only backend Vrooli has to create,
// unlock, and describe itself.

// WrapInfo is one key-encryption wrap as an operator sees it. It carries no key
// material — only which provider wrote the wrap and what protects it.
type WrapInfo struct {
	Provider string `json:"provider"`
	// KeyStore distinguishes a TPM-protected wrap from one protected by a host
	// key on the same disk. They are not equally strong and are never merged
	// into a single "encrypted" claim.
	KeyStore string `json:"key_store"`
}

// StoreStatus is what `vrooli credentials store status` prints. Listing the
// wraps does not require opening the store, which is exactly why service and
// key names are not sealed.
type StoreStatus struct {
	Path        string `json:"path"`
	Initialized bool   `json:"initialized"`
	// Unlocked is true when a wrap has opened the data key in this process.
	Unlocked       bool       `json:"unlocked"`
	ActiveWrap     string     `json:"active_wrap,omitempty"`
	ActiveKeyStore string     `json:"active_key_store,omitempty"`
	Wraps          []WrapInfo `json:"wraps"`
	// Entries is how many credentials the store holds. It is readable without
	// the key and never includes a value.
	Entries int `json:"entries"`
	// Active is true when this store is the backend the host is actually using,
	// rather than a store sitting beside a working native adapter.
	Active bool `json:"active"`
	// UnlockCache is where a passphrase unlock is remembered for the rest of
	// this login session, or empty on a host with no session-scoped memory —
	// where an unlock lasts exactly one command. It never holds a credential
	// value, only the key that opens them, and it is never on durable storage.
	UnlockCache string `json:"unlock_cache,omitempty"`
	// HostBoundBlocked names what stops the unattended host-bound wrap from
	// working here, or is empty when nothing does. It is reported because the
	// difference between "this host reboots unattended" and "this host needs a
	// passphrase typed at every boot" is invisible otherwise, and an operator
	// who assumes the wrong one finds out during an outage.
	HostBoundBlocked string `json:"host_bound_blocked,omitempty"`
	// Unattended is the verified answer to the only question that decides
	// whether a reboot needs a human: does a wrap open this store with no
	// operator action? It is proved by opening one, never inferred from a wrap
	// being listed, because a wrap that has stopped working still appears in
	// the file and would otherwise report this host as unattended right up
	// until the reboot that strands it.
	Unattended UnattendedStatus `json:"unattended"`
	Copy       *CopyStatus      `json:"copy,omitempty"`
}

// EntryRef is metadata from the encrypted store's cleartext index. It is safe
// for reconciliation because the sealed file intentionally keeps service and
// key names visible while protecting only values.
type EntryRef struct {
	Service string `json:"service"`
	Key     string `json:"key"`
}

// CopyStatus is non-secret freshness metadata for the encrypted root copy.
type CopyStatus struct {
	Path          string    `json:"path"`
	Sink          string    `json:"sink"`
	SinkIdentity  string    `json:"sink_identity,omitempty"`
	CopiedAt      time.Time `json:"copied_at"`
	Generation    string    `json:"generation"`
	Checksum      string    `json:"checksum,omitempty"`
	VerifiedAt    time.Time `json:"verified_at,omitempty"`
	Verification  string    `json:"verification,omitempty"`
	ScheduleState string    `json:"schedule_state,omitempty"`
	Remediation   string    `json:"remediation,omitempty"`
}

type CopyPolicy struct {
	RepositoryPaths          []string
	ProtectedRoots           []string
	RequireIndependentDevice bool
}

// SinkConflictError is returned when a root copy would be placed inside a
// repository that it is needed to unlock. That would make bare-host recovery
// circular.
type SinkConflictError struct {
	Sink       string
	Repository string
}

func (e *SinkConflictError) Error() string {
	return fmt.Sprintf("credential-store copy sink %q is inside kopia repository %q", e.Sink, e.Repository)
}

// errNoEncryptedBackend means the process selected a backend that has no
// encrypted store behind it, which only happens under an invalid override.
var errNoEncryptedBackend = errors.New("this host has no encrypted credential store backend")

func encryptedStoreForAdmin() (*encryptedStore, Store, error) {
	store := Default()
	encrypted, ok := encryptedBackend(store)
	if !ok {
		return nil, store, fmt.Errorf("%w: %s", errNoEncryptedBackend, AdapterName(store))
	}
	return encrypted, store, nil
}

// DescribeStore reports the encrypted store on this host without unlocking it.
func DescribeStore() (StoreStatus, error) {
	encrypted, chain, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	status := StoreStatus{
		Path:             encrypted.path,
		Active:           backendName(chain) == adapterEncryptedFile,
		UnlockCache:      encrypted.cache.Location(),
		HostBoundBlocked: hostBoundFix(),
	}
	provider, keyStore := encrypted.ActiveWrap()
	status.ActiveWrap, status.ActiveKeyStore = provider, keyStore
	status.Unlocked = provider != ""

	file, err := readSealedFile(encrypted.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return status, nil
		}
		return status, encrypted.classifyFileError(err)
	}
	status.Initialized = true
	status.Entries = len(file.Entries)
	for _, wrap := range file.Wraps {
		status.Wraps = append(status.Wraps, WrapInfo{Provider: wrap.Provider, KeyStore: wrap.KeyStore})
	}

	// "Unlocked" has to mean "a wrap opens this store without asking the
	// operator for anything new", not "this particular handle happens to have
	// opened it". Every command is a fresh process, so the handle-local answer
	// would report a working host-bound store as locked forever.
	if !status.Unlocked {
		if _, _, err := encrypted.open(); err == nil {
			provider, keyStore := encrypted.ActiveWrap()
			status.Unlocked, status.ActiveWrap, status.ActiveKeyStore = true, provider, keyStore
		}
	}
	// A store the host-bound wrap opens needs no unlock and therefore never
	// writes one. Naming a location it does not use would tell an operator to
	// go looking for a file that is not there.
	if status.ActiveWrap == providerHostBound || status.ActiveWrap == providerNativeWrap {
		status.UnlockCache = ""
	}
	status.Unattended = inspectUnattendedWrap(file)
	return status, nil
}

// ListEntryRefs returns the store's metadata-only index. It never opens a
// ciphertext and never returns credential values.
func ListEntryRefs() ([]EntryRef, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return nil, err
	}
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return nil, encrypted.classifyFileError(err)
	}
	refs := make([]EntryRef, 0, len(file.Entries))
	for _, entry := range file.Entries {
		refs = append(refs, EntryRef{Service: entry.Service, Key: entry.Key})
	}
	return refs, nil
}

// DeleteEntryRef removes one metadata-addressed store entry. The caller must
// supply an explicit confirmation at the CLI boundary; this function never
// resolves or returns the removed value.
func DeleteEntryRef(service, key string) (bool, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return false, err
	}
	encrypted.mu.Lock()
	defer encrypted.mu.Unlock()
	var deleted bool
	err = encrypted.mutate(func(file *sealedFile, _ []byte) (bool, error) {
		deleted = file.deleteEntry(service, key)
		return deleted, nil
	})
	return deleted, err
}
