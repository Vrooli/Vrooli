// Package credentialauthority owns the credential authority used by the control plane.
// It deliberately exposes values only to the runtime injection boundary.
package credentialauthority

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/securestore"
)

const (
	authorityParameterA = 2
)

const credentialService = "vrooli.credentials.v1"

// The credential failure taxonomy. Three conditions exist on a real host and
// each one has a different operator action, so each one gets its own sentinel.
// Collapsing them is what let a broken keyring session read as "your API key is
// missing" and abort a scenario start.
var (
	// ErrProviderUnavailable: the host secure store exists but cannot be
	// reached now. Repair the session; `vrooli credentials doctor` names the
	// cause.
	ErrProviderUnavailable = errors.New("credential provider is unavailable")
	// ErrProviderAbsent: this host has no secure store implementation.
	// Install a backend, or accept degraded resources.
	ErrProviderAbsent = errors.New("credential provider is not implemented on this host")
	// ErrUnconfigured: the store works and holds no value for this
	// identity/field. Run `vrooli credentials provision`.
	ErrUnconfigured = errors.New("credential is not configured")
)

// ProviderState is the coarse credential-backend condition carried alongside
// every credential answer, so a caller can never read "not configured" without
// also learning whether the store was even reachable.
type ProviderState string

const (
	ProviderAvailable   ProviderState = "available"
	ProviderUnavailable ProviderState = "unavailable"
	ProviderAbsent      ProviderState = "absent"
)

// ProviderStateFor maps any credential error onto the backend condition it
// implies. A nil error and ErrUnconfigured both mean the backend answered.
func ProviderStateFor(err error) ProviderState {
	switch {
	case errors.Is(err, ErrProviderAbsent):
		return ProviderAbsent
	case errors.Is(err, ErrProviderUnavailable):
		return ProviderUnavailable
	default:
		return ProviderAvailable
	}
}

// providerError carries both the taxonomy sentinel and the untouched
// transport cause, so a consumer can render the host explanation without
// re-stating the sentinel that already prefixes it.
type providerError struct {
	kind  error
	cause error
}

func (e providerError) Error() string        { return e.kind.Error() + ": " + e.cause.Error() }
func (e providerError) Is(target error) bool { return target == e.kind }
func (e providerError) Unwrap() error        { return e.cause }

// Detail is the host explanation without the sentinel prefix.
func (e providerError) Detail() string { return e.cause.Error() }

// classifyStoreError translates a transport-level store failure into the
// credential taxonomy. Only a clean "no such value" becomes ErrUnconfigured;
// every other failure keeps its provider identity so no caller can mistake a
// host fault for an operator omission.
func classifyStoreError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, securestore.ErrNotFound):
		return ErrUnconfigured
	case errors.Is(err, securestore.ErrAbsent):
		return providerError{kind: ErrProviderAbsent, cause: err}
	default:
		return providerError{kind: ErrProviderUnavailable, cause: err}
	}
}

// ProviderDetail renders the host explanation behind a credential error
// without repeating the sentinel text.
func ProviderDetail(err error) string {
	if err == nil {
		return ""
	}
	var typed providerError
	if errors.As(err, &typed) {
		return typed.Detail()
	}
	return err.Error()
}

// Identity is backend-neutral and stable across desktop, local, and hosted
// deployment tiers. It must never contain a Vault path or an environment name.
type Identity string

func ParseIdentity(raw string) (Identity, error) {
	value := strings.Trim(strings.ToLower(strings.TrimSpace(raw)), "/")
	if value == "" || strings.Contains(value, "//") {
		return "", fmt.Errorf("credential logical identity is required and must be namespaced")
	}
	parts := strings.Split(value, "/")
	if len(parts) < authorityParameterA {
		return "", fmt.Errorf("credential logical identity must be namespaced")
	}
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("credential logical identity contains an empty segment")
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' && r != '.' {
				return "", fmt.Errorf("credential logical identity contains an invalid character")
			}
		}
	}
	return Identity(value), nil
}

type Status struct {
	Identity Identity `json:"identity"`
	Field    string   `json:"field"`
	// Version is the opaque authority version of the active value. It is
	// metadata only and never permits a caller to recover the value.
	Version string `json:"version,omitempty"`
	// Configured is only meaningful when ProviderState is available. A caller
	// that reads Configured alone while the store is down would conclude the
	// operator never set the value.
	Configured    bool          `json:"configured"`
	Provider      string        `json:"provider"`
	ProviderState ProviderState `json:"provider_state"`
	// ProviderDetail explains a non-available provider state. It never
	// contains a credential value.
	ProviderDetail string    `json:"provider_detail,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

// Authority is the only durable local credential writer. Its Store must be a
// securestore adapter — a native platform store, or the encrypted file store on
// a host that has none. Plaintext and process-environment fallbacks are
// intentionally absent from both.
type Authority struct {
	store securestore.Store

	mu sync.Mutex
	// availability caches the lazy read probe for the process lifetime, so
	// resolving a manifest with several credentialed resources probes once
	// rather than once per resource.
	availabilityProbed bool
	availabilityErr    error
}

// NewAuthority wraps a store. It performs no store I/O: whether the backend is
// reachable is a runtime property discovered when a secret is actually needed,
// not a precondition for constructing the authority.
func NewAuthority(store securestore.Store) (*Authority, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	return &Authority{store: store}, nil
}

var (
	defaultAuthorityOnce sync.Once
	defaultAuthority     *Authority
	defaultAuthorityErr  error
)

// DefaultAuthority is the single construction path over securestore.Default,
// memoized per process so the availability probe, the store handle, and the
// failure message are identical at every call site.
//
// It is a variable rather than a plain function so tests can inject a store
// representing a host condition that cannot be produced on a real machine — an
// unreachable keyring, a platform with no backend. Production code must never
// reassign it.
var DefaultAuthority = func() (*Authority, error) {
	defaultAuthorityOnce.Do(func() {
		defaultAuthority, defaultAuthorityErr = NewAuthority(securestore.Default())
	})
	return defaultAuthority, defaultAuthorityErr
}

// Availability reports whether the credential backend can be read, using a
// read-shaped probe whose result is cached for the process lifetime. It
// returns nil, ErrProviderUnavailable, or ErrProviderAbsent — never
// ErrUnconfigured, because a backend that cleanly reports "no such value" is a
// working backend.
func (a *Authority) Availability() error {
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.availabilityProbed {
		a.availabilityErr = classifyStoreError(securestore.Probe(a.store))
		a.availabilityProbed = true
	}
	return a.availabilityErr
}

// Recheck discards the cached availability verdict so the next Availability
// call probes the store again.
//
// Caching for the process lifetime is right for a short CLI invocation and
// wrong for a long-lived server. The onboarding API is started by the control
// plane and can outlive several store state changes, so without this it would
// keep reporting the store state it observed at start — a wizard opened hours
// later would call every credential unsupported because of a lock that has
// since been opened. Callers invoke this once per request, never once per
// credential.
func (a *Authority) Recheck() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.availabilityProbed = false
	a.availabilityErr = nil
}

// Provider names the backend for diagnostics. It never performs store I/O.
func (a *Authority) Provider() string {
	if a == nil {
		return "none"
	}
	return securestore.AdapterName(a.store)
}

func (a *Authority) Put(identity Identity, field, value string) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return err
	}
	field = strings.TrimSpace(field)
	if field == "" || strings.ContainsAny(field, "/\\") {
		return fmt.Errorf("credential field is required and cannot contain a path separator")
	}
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("credential value is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// Legacy and generated writes retain their existing single-write contract.
	// Operator input uses PutCandidate/ActivateCandidate, which commits the
	// opaque active version needed for exercised evidence without changing the
	// mint path's storage semantics.
	return classifyStoreError(a.store.Put(credentialService, storeKey(identity, field), value))
}

// Inject resolves values only into the supplied ephemeral environment map.
// The caller owns process creation; this method never exports globally.
//
// It returns ErrUnconfigured only when the backend answered and held no value.
// A provider failure keeps its own sentinel so the caller can tell an operator
// omission from a host fault.
func (a *Authority) Inject(identity Identity, field, env string, target map[string]string) error {
	_, err := a.InjectExpected(identity, field, env, target, "")
	return err
}

// InjectExpected resolves and injects one credential while holding the
// authority lock across the active-version check and value read. A non-empty
// expectedVersion therefore binds the delivered value to the exact version
// reviewed by the caller; rotation cannot occur between a separate Status and
// Inject call. The returned version is empty only for legacy active values
// that predate version metadata and were requested without a binding.
func (a *Authority) InjectExpected(identity Identity, field, env string, target map[string]string, expectedVersion string) (string, error) {
	if a == nil || a.store == nil {
		return "", fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return "", err
	}
	field = strings.TrimSpace(field)
	env = strings.TrimSpace(env)
	if field == "" || env == "" || target == nil {
		return "", fmt.Errorf("credential field, environment name, and target are required")
	}
	if err := credentialspec.ValidateEnvironmentName(env); err != nil {
		return "", err
	}
	if _, exists := target[env]; exists {
		return "", fmt.Errorf("credential environment target already contains %q", env)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	value, err := a.store.Get(credentialService, storeKey(identity, field))
	err = classifyStoreError(err)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", ErrUnconfigured
	}
	version, versionErr := a.activeVersion(identity, field)
	if expectedVersion != "" {
		if versionErr != nil {
			return "", fmt.Errorf("credential version for %s:%s is unavailable: %w", identity, field, versionErr)
		}
		if version != expectedVersion {
			return "", fmt.Errorf("credential version changed for %s:%s: expected %q, active %q", identity, field, expectedVersion, version)
		}
	}
	target[env] = value
	if versionErr != nil {
		return "", nil
	}
	return version, nil
}

// Resolve returns one credential value to a trusted runtime consumer. It is
// the value-bearing counterpart to Status; callers must keep the value in
// process memory and must not expose it in diagnostics or transport responses.
func (a *Authority) Resolve(identity Identity, field string) (string, error) {
	return a.get(identity, field)
}

func (a *Authority) Status(identity Identity, field string) Status {
	status := Status{
		Identity:      identity,
		Field:         strings.TrimSpace(field),
		Provider:      a.Provider(),
		ProviderState: ProviderAbsent,
		CheckedAt:     time.Now().UTC(),
	}
	if a == nil || a.store == nil {
		status.ProviderDetail = "no credential store on this host"
		return status
	}
	if status.Field == "" {
		status.ProviderState = ProviderAvailable
		return status
	}
	value, err := a.get(identity, status.Field)
	status.ProviderState = ProviderStateFor(err)
	if status.ProviderState != ProviderAvailable {
		status.ProviderDetail = ProviderDetail(err)
		return status
	}
	status.Configured = err == nil && strings.TrimSpace(value) != ""
	if status.Configured {
		version, versionErr := a.activeVersion(identity, status.Field)
		if versionErr != nil {
			if errors.Is(versionErr, ErrUnconfigured) {
				// Values written before versioned activation remain configured,
				// but cannot satisfy evidence that requires an exact authority
				// version until they are provisioned or rotated again.
				status.ProviderDetail = "active credential version metadata is absent; reprovision before claiming exercised evidence"
				return status
			}
			status.ProviderState = ProviderUnavailable
			status.ProviderDetail = "active credential version metadata is unavailable"
			return status
		}
		status.Version = version
	}
	return status
}

func (a *Authority) Delete(identity Identity, field string) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("%w: no credential store on this host", ErrProviderAbsent)
	}
	if _, err := ParseIdentity(string(identity)); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	err := a.store.Delete(credentialService, storeKey(identity, strings.TrimSpace(field)))
	if errors.Is(err, securestore.ErrNotFound) {
		return nil
	}
	return classifyStoreError(err)
}

func (a *Authority) get(identity Identity, field string) (string, error) {
	// The store is immutable after construction and its adapters own their
	// read-side synchronization. Do not serialize independent metadata probes
	// behind the authority availability mutex: onboarding may check many
	// declared credentials concurrently, and serialized secure-store reads make
	// readiness latency scale linearly with the number of descriptors.
	value, err := a.store.Get(credentialService, storeKey(identity, field))
	return value, classifyStoreError(err)
}

func storeKey(identity Identity, field string) string {
	return string(identity) + ":" + field
}

func versionKey(identity Identity, field string) string {
	return "version/" + storeKey(identity, field)
}

func (a *Authority) activeVersion(identity Identity, field string) (string, error) {
	version, err := a.store.Get(credentialService, versionKey(identity, field))
	if err != nil {
		return "", classifyStoreError(err)
	}
	if len(version) != 32 {
		return "", fmt.Errorf("credential active version metadata is invalid")
	}
	if _, err := hex.DecodeString(version); err != nil {
		return "", fmt.Errorf("credential active version metadata is invalid")
	}
	return version, nil
}
