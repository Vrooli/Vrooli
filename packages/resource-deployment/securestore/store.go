// Package securestore exposes the platform credential-store contract to
// deployment consumers such as the desktop runtime. The implementation stays
// in the control plane so every target uses the same fail-closed adapters and
// the same three-way failure taxonomy.
package securestore

import internalstore "github.com/vrooli/vrooli/internal/resources/securestore"

type (
	Store   = internalstore.Store
	Adapter = internalstore.Adapter
)

var (
	// ErrUnavailable: an adapter exists for this host but cannot be reached.
	ErrUnavailable = internalstore.ErrUnavailable
	// ErrAbsent: this host has no usable adapter at all.
	ErrAbsent = internalstore.ErrAbsent
	// ErrNotFound: the backend answered and holds no value for the key.
	ErrNotFound = internalstore.ErrNotFound
)

func Default() Store                  { return internalstore.Default() }
func Unavailable(reason string) Store { return internalstore.Unavailable(reason) }
func Absent(reason string) Store      { return internalstore.Absent(reason) }

// Probe is the read-shaped availability check used by read paths.
func Probe(store Store) error { return internalstore.Probe(store) }

// ProbeWritable is the stronger check used only before writing durable
// recovery material.
func ProbeWritable(store Store) error { return internalstore.ProbeWritable(store) }

func AdapterName(store Store) string { return internalstore.AdapterName(store) }
