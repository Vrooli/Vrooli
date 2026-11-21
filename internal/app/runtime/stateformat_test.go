package runtimeapp

import (
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func TestNativeStateFormatManifestCoversEveryOwner(t *testing.T) {
	withRepositoryCodecStamp(t)
	base, err := NativePersistentStateManifest()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := NativeStateFormatManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("generated manifest is invalid: %v", err)
	}
	if len(manifest.Stores) != len(base.Owners) {
		t.Fatalf("stores = %d, owners = %d; every owner must be represented exactly once", len(manifest.Stores), len(base.Owners))
	}
	owners := make(map[string]bool, len(base.Owners))
	for _, owner := range base.Owners {
		owners[owner.Owner] = true
	}
	for _, store := range manifest.Stores {
		if !owners[store.Name] {
			t.Fatalf("store %q has no persistence owner", store.Name)
		}
		delete(owners, store.Name)
		if _, versioned := versionedStateStores[store.Name]; versioned {
			if len(store.Schemas) != 1 || store.Schemas[store.Write] == "" {
				t.Fatalf("versioned store %q has no write-version digest: %+v", store.Name, store)
			}
			// A forced transition can only recover a store that declares the
			// files it owns. A store that owns no bytes must say so, so a
			// forgotten path list cannot look like a deliberate contract.
			if len(store.StatePaths) == 0 && !store.ContractOnly && !store.Discardable {
				t.Fatalf("versioned store %q declares no state paths and is neither contract-only nor discardable", store.Name)
			}
			continue
		}
		if store.Write != 1 || len(store.Schemas) != 1 || len(store.StatePaths) != 0 || store.ContractOnly || store.Discardable {
			t.Fatalf("unmigrated owner %q must be opaque until it has a declared format", store.Name)
		}
	}
	if len(owners) != 0 {
		t.Fatalf("owners missing from the manifest: %v", owners)
	}
}

func TestNativeStateFormatManifestIsDeterministic(t *testing.T) {
	withRepositoryCodecStamp(t)
	first, err := NativeStateFormatManifest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NativeStateFormatManifest()
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("manifest digest is not deterministic: %s != %s", first.Digest, second.Digest)
	}
}

// TestNativeStateFormatManifestSurvivesTheWire is the contract admission
// depends on: the JSON an image prints is exactly what the other image's probe
// accepts, including the self-digest.
func TestNativeStateFormatManifestSurvivesTheWire(t *testing.T) {
	withRepositoryCodecStamp(t)
	manifest, err := NativeStateFormatManifest()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded stateformat.Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("round-tripped manifest is invalid: %v", err)
	}
	if decoded.Digest != manifest.Digest {
		t.Fatalf("digest changed across the wire: %s != %s", decoded.Digest, manifest.Digest)
	}
}

// opaqueStoreCeiling is the number of owners that still lack a declared format.
// Converting a store lowers this number; nothing may raise it. A new owner
// therefore has to arrive with a descriptor, and a conversion is a one-line
// review event rather than a silent improvement.
const opaqueStoreCeiling = 7

func TestOpaqueStoreCountOnlyShrinks(t *testing.T) {
	withRepositoryCodecStamp(t)
	manifest, err := NativeStateFormatManifest()
	if err != nil {
		t.Fatal(err)
	}
	var opaque []string
	for _, store := range manifest.Stores {
		if _, converted := versionedStateStores[store.Name]; !converted {
			opaque = append(opaque, store.Name)
		}
	}
	if len(opaque) > opaqueStoreCeiling {
		t.Fatalf("opaque stores = %d, ceiling = %d; a new persistence owner must declare a format (still opaque: %v)", len(opaque), opaqueStoreCeiling, opaque)
	}
	if len(opaque) < opaqueStoreCeiling {
		t.Fatalf("opaque stores = %d, below the pinned ceiling %d; lower opaqueStoreCeiling to %d so the improvement is recorded", len(opaque), opaqueStoreCeiling, len(opaque))
	}
}
