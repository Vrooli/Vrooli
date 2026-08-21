package securestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

// Default is the credential authority for this host. It is a chain: the native
// platform adapter first, and the encrypted file store only when the native
// adapter reports that this host has no adapter at all.
//
// The rule that this chain must NOT fall back on ErrUnavailable is the
// load-bearing decision of the whole design, and it is easy to get wrong while
// writing a chain because "try the next one on any error" looks correct. A
// native store that exists but is temporarily unreachable — a locked keychain,
// a stopped Secret Service — would then split credentials across two backends
// according to transient session health: a value provisioned while the keyring
// was up becomes invisible when it goes down, and a value written while it was
// down becomes invisible when it recovers. A degraded resource is an honest
// state an operator can see and fix. A silent second store is not.
// Every path out of here is wrapped by guardValues, including the override
// path. Guarding only the native adapter would be enough to prevent the defect
// that motivated it — GNOME Keyring is the backend that cannot hold a
// multi-line value — but it would make the stored form depend on which backend
// wrote it, and BackendOverrideEnv exists precisely so a host can change its
// mind about that. A uniform encoding is what lets a value written under one
// backend still be read under another.
func Default() Store {
	native := nativeDefault()
	if backend := strings.TrimSpace(os.Getenv(BackendOverrideEnv)); backend != "" {
		return guardValues(overrideStore(backend, native))
	}
	if backend, selected, err := SelectedBackend(); err != nil {
		return guardValues(Absent(err.Error()))
	} else if selected {
		return guardValues(overrideStore(backend, native))
	}
	return guardValues(&chainStore{native: native, fallback: defaultEncryptedStore()})
}

// BackendOverrideEnv lets an operator select the encrypted file store on a host
// that also has a working native one. It exists for the operator who has a
// desktop session today and will not have one tomorrow — a workstation being
// converted to a server, a laptop about to run headless — and who would
// otherwise have to break their keyring to move.
const BackendOverrideEnv = "VROOLI_CREDENTIAL_BACKEND"

// BackendNative and BackendEncryptedFile are the two accepted override values.
const (
	BackendNative        = "native"
	BackendEncryptedFile = adapterEncryptedFile
)

func overrideStore(backend string, native Store) Store {
	switch backend {
	case BackendEncryptedFile:
		return defaultEncryptedStore()
	case BackendNative:
		return native
	default:
		return Absent(fmt.Sprintf("%s=%q is not a credential backend; use %q or %q",
			BackendOverrideEnv, backend, BackendNative, BackendEncryptedFile))
	}
}

// credentialStorePath resolves the encrypted store file from the repo contract,
// so the location is declared in one place rather than assembled here. It is a
// variable so tests can point the whole chain at a temporary file without an
// environment variable that would also exist in production.
var credentialStorePath = func() (string, error) {
	return config.VrooliPath(repocontract.HomeKeySecretsEnc)
}

// passphraseSource supplies the operator passphrase for the encrypted store. It
// is a variable rather than a parameter because the CLI, the runtime injection
// path, and a scenario process all reach Default() without a common place to
// thread a prompt through. The default source never prompts: a credential read
// during a scenario start must not block on a terminal that may not exist.
var passphraseSource = func() string {
	passphraseMu.RLock()
	defer passphraseMu.RUnlock()
	return processPassphrase
}

// processPassphrase is the passphrase this process was given, if any. It is the
// operator surface's channel to the store: `credentials store init` and
// `credentials store unlock` read a passphrase from stdin and set it here,
// where the lazily-constructed provider picks it up.
var (
	passphraseMu      sync.RWMutex
	processPassphrase string
)

// SetPassphrase supplies the encrypted store's passphrase for this process. It
// never writes the passphrase anywhere, and a process that is never given one
// simply cannot open a passphrase-wrapped store — which is the correct outcome
// for a scenario start that must not block on a terminal.
func SetPassphrase(value string) {
	passphraseMu.Lock()
	defer passphraseMu.Unlock()
	processPassphrase = value
}

// defaultEncryptedStore builds the fallback adapter with both key providers,
// strongest first. A host with a TPM opens it with no human action after a
// reboot; a host without one falls through to what the operator remembers.
func defaultEncryptedStore() Store {
	path, err := credentialStorePath()
	if err != nil {
		return Absent(fmt.Sprintf("cannot resolve the credential store path: %v", err))
	}
	return newEncryptedStore(filepath.Clean(path), defaultKeyProviders()...)
}

// defaultKeyProviders are the wraps a store on this host is built with,
// strongest first. It is a variable so a test can pin the set: whether a real
// TPM happens to be reachable on the machine running the suite must not decide
// which wraps a store under test has, or the same test passes on one developer
// host and fails on another for reasons that have nothing to do with the code.
var defaultKeyProviders = func() []keyProvider {
	return []keyProvider{
		newNativeWrapProvider(),
		newHostBoundProvider(),
		passphraseProvider{source: passphraseSource},
	}
}

// chainStore delegates every operation to whichever backend is the authority on
// this host, deciding once per process.
type chainStore struct {
	native   Store
	fallback Store

	once     sync.Once
	selected Store
}

// active decides the authority exactly once. It probes the native adapter with
// a read, which is the same probe every read path already performs, so the cost
// is one extra native read per process and the decision cannot drift between
// operations within a process.
func (chain *chainStore) active() Store {
	chain.once.Do(func() {
		chain.selected = chain.native
		if errors.Is(Probe(chain.native), ErrAbsent) {
			chain.selected = chain.fallback
		}
	})
	return chain.selected
}

func (chain *chainStore) Put(service, key, value string) error {
	return chain.active().Put(service, key, value)
}

func (chain *chainStore) Get(service, key string) (string, error) {
	return chain.active().Get(service, key)
}

func (chain *chainStore) Delete(service, key string) error {
	return chain.active().Delete(service, key)
}

func (chain *chainStore) AdapterName() string { return AdapterName(chain.active()) }

// activeWrap reports the key wrap holding the store open, when the active
// backend is one that has wraps. A native store has none, and saying so is
// better than inventing a label.
func activeWrap(store Store) (string, string) {
	switch typed := store.(type) {
	case singleLineStore:
		return activeWrap(typed.inner)
	case *chainStore:
		return activeWrap(typed.active())
	case *encryptedStore:
		return typed.ActiveWrap()
	default:
		return "", ""
	}
}

// encryptedBackend returns the encrypted store behind a chain, whether or not
// it is the active authority. The operator surface needs it to initialize a
// store on a host whose native adapter is still working.
func encryptedBackend(store Store) (*encryptedStore, bool) {
	switch typed := store.(type) {
	case singleLineStore:
		return encryptedBackend(typed.inner)
	case *chainStore:
		return encryptedBackend(typed.fallback)
	case *encryptedStore:
		return typed, true
	default:
		return nil, false
	}
}
