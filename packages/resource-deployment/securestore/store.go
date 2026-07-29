// Package securestore exposes the platform credential-store contract to
// deployment consumers such as the desktop runtime. The implementation stays
// in the control plane so every target uses the same fail-closed adapters.
package securestore

import internalstore "github.com/vrooli/vrooli/internal/resources/securestore"

type Store = internalstore.Store

var ErrUnavailable = internalstore.ErrUnavailable

func Default() Store                  { return internalstore.Default() }
func Unavailable(reason string) Store { return internalstore.Unavailable(reason) }
func Probe(store Store) error         { return internalstore.Probe(store) }
