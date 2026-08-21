package securestore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// StoreGeneration returns a stable, non-secret identifier for the current
// passphrase wrap. It deliberately hashes only public envelope material and
// never derives an identifier from the passphrase itself.
func StoreGeneration(path string) (string, error) {
	file, err := readSealedFile(path)
	if err != nil {
		return "", err
	}
	for _, wrap := range file.Wraps {
		if wrap.Provider != providerPassphrase {
			continue
		}
		generation, err := passphraseWrapGeneration(wrap)
		if err != nil {
			return "", err
		}
		// Keep a digest of the public wrap material alongside the counter. The
		// counter is what makes rotation observable; the digest distinguishes
		// legacy or manually-rewritten envelopes that happen to reuse it.
		h := sha256.New()
		_, _ = h.Write(wrap.Params)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(wrap.Ciphertext))
		return fmt.Sprintf("%d:%s", generation, hex.EncodeToString(h.Sum(nil))), nil
	}
	return "", fmt.Errorf("credential store has no passphrase wrap")
}

func passphraseWrapGeneration(wrap wrappedKey) (uint64, error) {
	var params passphraseParams
	if len(wrap.Params) != 0 {
		if err := json.Unmarshal(wrap.Params, &params); err != nil {
			return 0, fmt.Errorf("%w: passphrase wrap has unreadable parameters", errSealedCorrupt)
		}
	}
	return normalizedPassphraseGeneration(params.Generation), nil
}

// CopyStore atomically copies an already encrypted store into sink. The sink
// is a directory and receives secrets.enc.json plus non-secret receipt
// metadata. repositoryPaths must contain every local repository root known to
// the control plane; a copy inside one is refused.
func CopyStore(source, sink, receiptPath string, repositoryPaths []string) (CopyStatus, error) {
	return CopyStoreWithPolicy(source, sink, receiptPath, CopyPolicy{RepositoryPaths: repositoryPaths})
}

// CopyStoreWithPolicy validates the complete sink policy before mutation and
// verifies the copied encrypted envelope before recording evidence. The
// compatibility CopyStore wrapper above preserves the older repository-only
// API for low-level callers; production onboarding supplies the stronger
// policy explicitly.
func CopyStoreWithPolicy(source, sink, receiptPath string, policy CopyPolicy) (CopyStatus, error) {
	source, err := filepath.Abs(filepath.Clean(source))
	if err != nil {
		return CopyStatus{}, fmt.Errorf("resolve credential store path: %w", err)
	}
	sink, err = filepath.Abs(filepath.Clean(sink))
	if err != nil {
		return CopyStatus{}, fmt.Errorf("resolve credential copy sink: %w", err)
	}
	for _, repository := range policy.RepositoryPaths {
		resolved, resolveErr := resolveExistingOrParent(repository)
		if resolveErr != nil || resolved == "" {
			continue
		}
		resolvedSink, sinkResolveErr := resolveExistingOrParent(sink)
		if sinkResolveErr != nil {
			return CopyStatus{}, fmt.Errorf("resolve credential copy sink for safety check: %w", sinkResolveErr)
		}
		rel, relErr := filepath.Rel(resolved, resolvedSink)
		if relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return CopyStatus{}, &SinkConflictError{Sink: resolvedSink, Repository: resolved}
		}
	}
	if err := validateCopySink(source, sink, policy); err != nil {
		return CopyStatus{}, err
	}
	if _, err := readSealedFile(source); err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	generation, err := StoreGeneration(source)
	if err != nil {
		return CopyStatus{}, err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return CopyStatus{}, fmt.Errorf("read encrypted credential store: %w", err)
	}
	if err := os.MkdirAll(sink, sealedDirPerm); err != nil {
		return CopyStatus{}, fmt.Errorf("create credential copy sink: %w", err)
	}
	if err := restrictCredentialDirectory(sink); err != nil {
		return CopyStatus{}, fmt.Errorf("restrict credential copy sink: %w", err)
	}
	destination := filepath.Join(sink, filepath.Base(source))
	if err := atomicCopy(destination, data); err != nil {
		return CopyStatus{}, err
	}
	verified, err := verifyCopiedStore(destination, data, generation)
	if err != nil {
		return CopyStatus{}, err
	}
	status := CopyStatus{Path: destination, Sink: sink, SinkIdentity: stableSinkIdentity(sink), CopiedAt: time.Now().UTC(), Generation: generation, Checksum: verified, VerifiedAt: time.Now().UTC(), Verification: "readback"}
	if err := writeCopyReceipt(receiptPath, status); err != nil {
		return CopyStatus{}, err
	}
	return status, nil
}

func validateCopySink(source, sink string, policy CopyPolicy) error {
	resolvedSource, err := resolveExistingOrParent(source)
	if err != nil {
		return fmt.Errorf("resolve encrypted credential store for sink policy: %w", err)
	}
	resolvedSink, err := resolveExistingOrParent(sink)
	if err != nil {
		return fmt.Errorf("resolve credential copy sink for policy: %w", err)
	}
	if (len(policy.ProtectedRoots) > 0 || policy.RequireIndependentDevice) && pathContainedBy(resolvedSink, filepath.Dir(resolvedSource)) {
		return &SinkConflictError{Sink: resolvedSink, Repository: filepath.Dir(resolvedSource)}
	}
	for _, root := range policy.ProtectedRoots {
		resolvedRoot, rootErr := resolveExistingOrParent(root)
		if rootErr != nil {
			return fmt.Errorf("resolve protected root %q: %w", root, rootErr)
		}
		if pathContainedBy(resolvedSink, resolvedRoot) {
			return &SinkConflictError{Sink: resolvedSink, Repository: resolvedRoot}
		}
	}
	if policy.RequireIndependentDevice {
		sourceDevice, sourceKnown := pathDeviceIdentity(filepath.Dir(resolvedSource))
		sinkDevice, sinkKnown := pathDeviceIdentity(resolvedSink)
		if !sinkKnown {
			sinkDevice, sinkKnown = pathDeviceIdentity(filepath.Dir(resolvedSink))
		}
		if !sourceKnown || !sinkKnown {
			return fmt.Errorf("credential copy sink physical independence is unknown")
		}
		if sourceDevice == sinkDevice {
			return fmt.Errorf("credential copy sink is on the same physical device as the encrypted store")
		}
	}
	return nil
}

func pathContainedBy(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func verifyCopiedStore(destination string, expected []byte, generationData string) (string, error) {
	data, err := os.ReadFile(destination)
	if err != nil {
		return "", fmt.Errorf("read back encrypted credential-store copy: %w", err)
	}
	if _, err := readSealedFile(destination); err != nil {
		return "", fmt.Errorf("verify encrypted credential-store copy: %w", err)
	}
	actual := sha256.Sum256(data)
	expectedSum := sha256.Sum256(expected)
	if actual != expectedSum {
		return "", fmt.Errorf("verify encrypted credential-store copy: checksum mismatch")
	}
	actualGeneration, err := StoreGeneration(destination)
	if err != nil {
		return "", fmt.Errorf("verify encrypted credential-store generation: %w", err)
	}
	if actualGeneration != generationData {
		return "", fmt.Errorf("verify encrypted credential-store generation: expected %s, got %s", generationData, actualGeneration)
	}
	return hex.EncodeToString(actual[:]), nil
}

func stableSinkIdentity(path string) string {
	resolved, err := resolveExistingOrParent(path)
	if err != nil || resolved == "" {
		resolved = filepath.Clean(path)
	}
	hash := sha256.Sum256([]byte(resolved))
	return hex.EncodeToString(hash[:])
}

// resolveExistingOrParent resolves symlinks for a path that may not exist yet.
// Checking only lexical absolute paths would allow a symlinked sink to land
// inside a repository after the containment check.
func resolveExistingOrParent(path string) (string, error) {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved), nil
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path, nil
	}
	resolvedParent, err := resolveExistingOrParent(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(path)), nil
}

func atomicCopy(destination string, data []byte) error {
	dir := filepath.Dir(destination)
	temp, err := os.CreateTemp(dir, ".secrets-*.tmp")
	if err != nil {
		return fmt.Errorf("create credential copy temporary file: %w", err)
	}
	tempName := temp.Name()
	defer func() { _ = temp.Close(); _ = os.Remove(tempName) }()
	if err := temp.Chmod(sealedFilePerm); err != nil {
		return fmt.Errorf("restrict credential copy temporary file: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("write credential copy: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("flush credential copy: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close credential copy: %w", err)
	}
	if err := os.Rename(tempName, destination); err != nil {
		return fmt.Errorf("replace credential copy: %w", err)
	}
	if err := RestrictCredentialFile(destination); err != nil {
		return fmt.Errorf("restrict credential copy: %w", err)
	}
	return nil
}

func writeCopyReceipt(path string, status CopyStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credential copy receipt: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), sealedDirPerm); err != nil {
		return fmt.Errorf("create credential copy receipt directory: %w", err)
	}
	if err := atomicCopy(path, data); err != nil {
		return fmt.Errorf("write credential copy receipt: %w", err)
	}
	return nil
}

// WriteCopyReceipt republishes an already verified copy status after a caller
// adds non-secret schedule metadata to the same receipt.
func WriteCopyReceipt(path string, status CopyStatus) error { return writeCopyReceipt(path, status) }

// InitializeStore creates the encrypted store on this host. The passphrase may
// be empty on a host whose host-bound wrap works, which is the case that lets a
// server reboot into a working state with no human at all.
func InitializeStore(passphrase string) (StoreStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	SetPassphrase(passphrase)
	if _, err := encrypted.initialize(); err != nil {
		return StoreStatus{}, err
	}
	return DescribeStore()
}

// UnlockStore opens the encrypted store with an operator passphrase and keeps
// the result available to later commands. It proves the passphrase before
// reporting success, so an operator never walks away believing a typo unlocked
// anything.
func UnlockStore(passphrase string) (StoreStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return StoreStatus{}, err
	}
	if !encrypted.initialized() {
		return StoreStatus{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	SetPassphrase(passphrase)
	encrypted.lock()
	if _, _, err := encrypted.open(); err != nil {
		return StoreStatus{}, err
	}
	return DescribeStore()
}

// LockStore discards the open data key immediately, both in this process and
// for later ones.
func LockStore() error {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return err
	}
	encrypted.lock()
	SetPassphrase("")
	return nil
}

// UnattendedStatus answers whether this host opens its credential store with no
// human action, and what had to change to make that true. It is the shape both
// `credentials store status` and setup report from, so the operator is told the
// same thing by both.
type UnattendedStatus struct {
	// Enabled is proved, not declared: a wrap was opened to produce it.
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider,omitempty"`
	KeyStore string `json:"key_store,omitempty"`
	// Added is true when this call created the wrap; Repaired when it replaced
	// one that had stopped opening. They are separate because they mean
	// different things to an operator: the first is a host reaching its
	// intended state, the second is a host that had silently left it.
	Added    bool `json:"added,omitempty"`
	Repaired bool `json:"repaired,omitempty"`
	// Blocked says why this host still needs a human at boot, in terms of
	// something that can be changed. It is empty when Enabled is true.
	Blocked string `json:"blocked,omitempty"`
}

// unattendedProviders are the wraps that need no human at boot, strongest
// first. The passphrase provider is deliberately absent: it is the recovery
// path that makes the store portable to a new host, not a way to boot without
// an operator, and counting it here would let a host report itself unattended
// because it can still be opened by hand.
// It is a variable so a test can substitute providers whose availability it
// controls: neither a TPM that has been cleared nor a Keychain item that has
// been deleted can be produced on demand on a real machine, and those are the
// states this logic exists to handle.
var unattendedProviders = func() []keyProvider {
	providers := defaultKeyProviders()
	unattended := make([]keyProvider, 0, len(providers))
	for _, provider := range providers {
		if provider.Name() == providerPassphrase {
			continue
		}
		unattended = append(unattended, provider)
	}
	return unattended
}

// inspectUnattendedWrap reports the unattended state and changes nothing. It
// opens each candidate wrap rather than trusting its presence, which is what
// makes a TPM that was cleared, a Keychain item that was deleted, or a wrap
// invalidated by a firmware update show up as the passphrase prompt it will
// actually become.
func inspectUnattendedWrap(file *sealedFile) UnattendedStatus {
	var reasons []string
	for _, provider := range unattendedProviders() {
		wrap, found := file.wrapFor(provider.Name())
		if !found {
			if _, err := provider.Available(); err != nil {
				reasons = append(reasons, provider.Name()+": "+conciseReason(err))
				continue
			}
			reasons = append(reasons, provider.Name()+": this host supports it but the store has no such wrap yet")
			continue
		}
		if _, err := provider.Unwrap(wrap); err != nil {
			reasons = append(reasons, provider.Name()+": the store has this wrap and it no longer opens: "+conciseReason(err))
			continue
		}
		return UnattendedStatus{Enabled: true, Provider: wrap.Provider, KeyStore: wrap.KeyStore}
	}
	return UnattendedStatus{Blocked: strings.Join(reasons, "; ")}
}

// EnsureUnattendedWrap converges this host on opening its credential store
// without a human, and reports what it found or changed.
//
// It is the single place that decision is made. Every path that has the store
// open — setup, onboarding, an explicit unlock, an explicit rewrap — calls it,
// because the alternative shipped once already: the wrap was added on exactly
// one code path, the two paths an operator actually reaches supplied a
// passphrase and returned without it, and the host typed a passphrase at every
// boot for as long as it existed.
//
// The data key is never regenerated, so no stored value is re-encrypted and a
// failure part-way through cannot lose a credential.
func EnsureUnattendedWrap(passphrase string) (UnattendedStatus, error) {
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return UnattendedStatus{}, err
	}
	if !encrypted.initialized() {
		return UnattendedStatus{}, fmt.Errorf("%w: no credential store on this host; run `vrooli credentials store init`", ErrAbsent)
	}
	if passphrase != "" {
		SetPassphrase(passphrase)
	}
	return ensureUnattendedWrap(encrypted)
}

func ensureUnattendedWrap(encrypted *encryptedStore) (UnattendedStatus, error) {
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return UnattendedStatus{}, encrypted.classifyFileError(err)
	}
	if status := inspectUnattendedWrap(file); status.Enabled {
		return status, nil
	}

	var reasons []string
	for _, provider := range unattendedProviders() {
		if _, err := provider.Available(); err != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(err))
			continue
		}
		_, replacing := file.wrapFor(provider.Name())
		// addWrap opens the store first, so this is where a missing passphrase
		// surfaces — and it surfaces as a reason rather than a hard failure,
		// because a host that cannot add an unattended wrap is degraded, not
		// broken.
		wrap, addErr := encrypted.addWrap(provider)
		if addErr != nil {
			reasons = append(reasons, provider.Name()+": "+conciseReason(addErr))
			continue
		}
		return UnattendedStatus{
			Enabled:  true,
			Provider: wrap.Provider,
			KeyStore: wrap.KeyStore,
			Added:    !replacing,
			Repaired: replacing,
		}, nil
	}
	return UnattendedStatus{Blocked: strings.Join(reasons, "; ")}, nil
}

// RewrapStore adds or refreshes the unattended wrap of an existing store. It is
// how a host that gains a TPM starts using it: the data key does not change, so
// no stored value is re-encrypted and nothing can be lost.
func RewrapStore(passphrase string) (WrapInfo, error) {
	status, err := EnsureUnattendedWrap(passphrase)
	if err != nil {
		return WrapInfo{}, err
	}
	if !status.Enabled {
		return WrapInfo{}, fmt.Errorf("%w: no unattended key wrap can protect this store (%s)",
			errKeyProviderUnavailable, status.Blocked)
	}
	return WrapInfo{Provider: status.Provider, KeyStore: status.KeyStore}, nil
}

// ChangePassphraseStore replaces the passphrase wrap around the existing data
// key. It first opens the store with the current passphrase, so a wrong
// current value leaves the file untouched. Stored credential entries are not
// read or re-encrypted.
func ChangePassphraseStore(current, next string) error {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == "" || next == "" {
		return fmt.Errorf("current and new credential store passphrases are required")
	}
	encrypted, _, err := encryptedStoreForAdmin()
	if err != nil {
		return err
	}
	return changePassphraseStore(encrypted, current, next)
}

func changePassphraseStore(encrypted *encryptedStore, current, next string) error {
	// Validate the supplied current passphrase against its own wrap. The
	// normal store chain may also have a host-bound wrap; using it here would
	// let a typo pass and would violate the command's promise that the current
	// passphrase is required before rotation.
	currentStore := newEncryptedStore(encrypted.path, passphraseProvider{passphrase: current})
	currentStore.cache = noUnlockCache{}
	if _, _, err := currentStore.open(); err != nil {
		currentStore.lock()
		return err
	}
	currentStore.lock()
	file, err := readSealedFile(encrypted.path)
	if err != nil {
		return err
	}
	currentWrap, found := file.wrapFor(providerPassphrase)
	if !found {
		return fmt.Errorf("credential store has no passphrase wrap")
	}
	currentGeneration, err := passphraseWrapGeneration(currentWrap)
	if err != nil {
		return err
	}
	SetPassphrase(current)
	if _, err := encrypted.addWrap(passphraseProvider{passphrase: next, generation: currentGeneration + 1}); err != nil {
		encrypted.lock()
		SetPassphrase("")
		return err
	}
	// The old cache fingerprint must not survive the wrap replacement. Locking
	// also zeroes the in-process data key before the command returns.
	encrypted.lock()
	SetPassphrase("")
	return nil
}
