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
			// files it owns.
			if len(store.StatePaths) == 0 {
				t.Fatalf("versioned store %q declares no state paths", store.Name)
			}
			continue
		}
		if store.Write != 1 || len(store.Schemas) != 1 || len(store.StatePaths) != 0 {
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
