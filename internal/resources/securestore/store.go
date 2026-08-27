// Package securestore is the resource-host-private seam for operating-system
// credential storage. It intentionally has no dependency on api-core or
// cli-core, and broker state must never implement this interface.
package securestore

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	storeParameterA = 16
)

// The three transport-level conditions a credential backend can be in. They
// stay distinct all the way to the operator because each one has a different
// fix: repair the session, install a backend, or provision the value.
var (
	// ErrUnavailable means an adapter exists for this host but cannot be
	// reached right now — a broken keyring session, a locked keychain, a
	// stopped credential service.
	ErrUnavailable = errors.New("operating-system secure storage is unavailable")
	// ErrAbsent means this host has no usable adapter at all, so no amount of
	// session repair will help.
	ErrAbsent = errors.New("operating-system secure storage is not implemented on this host")
	// ErrNotFound means the backend answered cleanly and holds no value for
	// the requested key. It is a normal answer, never a host fault.
	ErrNotFound = errors.New("operating-system secure storage holds no value for this key")
)

// Store persists bootstrap material in the platform credential facility. The
// value is never written to resource state, diagnostics, or application
// environment variables.
//
// Get must return an error satisfying errors.Is(err, ErrNotFound) when the
// backend is reachable and holds no value. Collapsing "absent value" into
// "backend broken" is what turned a missing API key into a failed scenario
// start, so adapters are required to keep the two apart.
type Store interface {
	Put(service, key, value string) error
	Get(service, key string) (string, error)
	Delete(service, key string) error
}

// Adapter names the backend behind a Store for diagnostics. It is optional;
// Diagnose falls back to a generic label when a Store does not implement it.
type Adapter interface {
	AdapterName() string
}

type unavailableStore struct {
	reason string
	kind   error
}

func (s unavailableStore) err() error { return fmt.Errorf("%w: %s", s.kind, s.reason) }

func (s unavailableStore) Put(string, string, string) error { return s.err() }
func (s unavailableStore) Get(string, string) (string, error) {
	return "", s.err()
}
func (s unavailableStore) Delete(string, string) error { return s.err() }

func (s unavailableStore) AdapterName() string {
	if errors.Is(s.kind, ErrAbsent) {
		return "none"
	}
	return "unreachable"
}

// Unavailable returns a fail-closed store for a host whose credential facility
// exists but cannot be verified right now. Callers must surface this as a
// conditional target limitation; they must not replace it with a tuning.PermSecret
// plaintext file.
func Unavailable(reason string) Store { return unavailableStore{reason: reason, kind: ErrUnavailable} }

// Absent returns a fail-closed store for a platform with no adapter at all.
// The distinction from Unavailable matters to the operator: an absent backend
// is installed, an unavailable one is repaired.
func Absent(reason string) Store { return unavailableStore{reason: reason, kind: ErrAbsent} }

const (
	probeService = "vrooli.securestore.probe"
	probeKey     = "availability"
)

// Probe reports whether a credential backend can be read. It deliberately
// performs a read and not a store-then-delete cycle: resolving environment for
// a scenario must not require write access to the operator keyring, and a
// clean "no such value" is proof that the backend is reachable.
//
// It returns nil when the backend answers, and otherwise an error satisfying
// errors.Is against ErrAbsent or ErrUnavailable.
func Probe(store Store) error {
	if store == nil {
		return fmt.Errorf("%w: credential store is not configured", ErrAbsent)
	}
	_, err := store.Get(probeService, probeKey)
	switch {
	case err == nil, errors.Is(err, ErrNotFound):
		return nil
	case errors.Is(err, ErrAbsent), errors.Is(err, ErrUnavailable):
		return err
	default:
		return fmt.Errorf("%w: read probe: %v", ErrUnavailable, err)
	}
}

// ProbeWritable proves that a credential backend can store, retrieve, and
// remove a throwaway value. Only callers that are about to write durable
// recovery material need this stronger evidence; read paths use Probe.
// ProbeWritable never reports the value it writes and always attempts cleanup
// before returning.
func ProbeWritable(store Store) error {
	if store == nil {
		return fmt.Errorf("%w: credential store is not configured", ErrAbsent)
	}
	bytes := make([]byte, storeParameterA)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Errorf("generate secure-store probe key: %w", err)
	}
	key := "probe-" + hex.EncodeToString(bytes)
	const value = "ready"
	if err := store.Put(probeService, key, value); err != nil {
		return classifyProbeError("write probe", err)
	}
	defer func() { _ = store.Delete(probeService, key) }()
	got, err := store.Get(probeService, key)
	if err != nil {
		return classifyProbeError("read probe", err)
	}
	if got != value {
		return fmt.Errorf("%w: probe readback did not match", ErrUnavailable)
	}
	if err := store.Delete(probeService, key); err != nil {
		return classifyProbeError("delete probe", err)
	}
	return nil
}

func classifyProbeError(stage string, err error) error {
	if errors.Is(err, ErrAbsent) {
		return fmt.Errorf("%w: %s: %v", ErrAbsent, stage, err)
	}
	return fmt.Errorf("%w: %s: %v", ErrUnavailable, stage, err)
}

// AdapterName reports the backend label behind a Store for diagnostics.
func AdapterName(store Store) string {
	if store == nil {
		return "none"
	}
	if named, ok := store.(Adapter); ok {
		return named.AdapterName()
	}
	return "unknown"
}
