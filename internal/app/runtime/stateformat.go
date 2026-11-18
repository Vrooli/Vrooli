package runtimeapp

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/stateformat"
)

// versionedStateStores is the migration registry for owners whose stored format
// is declared as a version rather than inferred from their source. An owner
// that is not listed here is still emitted as an opaque identity store, which
// keeps the old "any change is breaking" envelope until it is converted.
//
// Converting an owner means: declare its write version and read versions, derive
// its schema digest from its compiled types, declare the state paths a forced
// transition must snapshot, and prove any additive claim with a downlevel
// fixture. The opaque entry then disappears without a second code path.
var versionedStateStores = map[string]func() (stateformat.Store, error){
	"runtime-registry": func() (stateformat.Store, error) { return scenarioruntime.StateFormat(), nil },
	"core-activation":  coreactivation.StateFormat,
}

// NativeStateFormatManifest is the versioned compatibility manifest for this
// executable. Stores that have a declared format are emitted from it; every
// other owner is emitted as an opaque identity store until it is migrated.
// Opaque stores keep the old safety envelope: any change is breaking.
func NativeStateFormatManifest() (stateformat.Manifest, error) {
	base, err := NativePersistentStateManifest()
	if err != nil {
		return stateformat.Manifest{}, err
	}
	stores := make([]stateformat.Store, 0, len(base.Owners))
	converted := map[string]bool{}
	for _, owner := range base.Owners {
		if provider, versioned := versionedStateStores[owner.Owner]; versioned {
			store, err := provider()
			if err != nil {
				return stateformat.Manifest{}, fmt.Errorf("state format for owner %s: %w", owner.Owner, err)
			}
			stores = append(stores, store)
			converted[owner.Owner] = true
			continue
		}
		store, err := stateformat.Opaque(owner.Owner, owner.Digest)
		if err != nil {
			return stateformat.Manifest{}, fmt.Errorf("state format for owner %s: %w", owner.Owner, err)
		}
		stores = append(stores, store)
	}
	for name := range versionedStateStores {
		if !converted[name] {
			return stateformat.Manifest{}, fmt.Errorf("versioned state store %s has no persistence owner", name)
		}
	}
	if err := stateformat.SortStores(stores); err != nil {
		return stateformat.Manifest{}, err
	}
	manifest := stateformat.Manifest{Format: stateformat.ManifestFormat, Stores: stores}
	if err := manifest.Seal(); err != nil {
		return stateformat.Manifest{}, err
	}
	return manifest, nil
}
