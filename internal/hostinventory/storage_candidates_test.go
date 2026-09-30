package hostinventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateStorageCandidateRejectsProtectedAndUnknownPhysicalRoots(t *testing.T) {
	root := t.TempDir()
	candidate := StorageCandidate{Location: filepath.Join(root, "recovery"), Writable: true, PhysicalIndependence: "unknown"}
	if err := ValidateStorageCandidate(candidate, StoragePolicy{ProtectedRoots: []string{root}, RequirePhysicalSeparation: true}); err == nil {
		t.Fatal("candidate inside protected root was accepted")
	}
	candidate.Location = filepath.Join(t.TempDir(), "recovery")
	if err := ValidateStorageCandidate(candidate, StoragePolicy{RequirePhysicalSeparation: true}); err == nil || !strings.Contains(err.Error(), "physical independence") {
		t.Fatalf("unknown physical identity error = %v", err)
	}
}

func TestValidateStorageCandidateAllowsObservedIndependentCandidate(t *testing.T) {
	candidate := StorageCandidate{Location: filepath.Join(t.TempDir(), "recovery"), Writable: true, PhysicalIndependence: "observed"}
	if err := ValidateStorageCandidate(candidate, StoragePolicy{RequirePhysicalSeparation: true}); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverStorageCandidatesDoesNotUseWorkingDirectoryAsFallback(t *testing.T) {
	candidates, err := DiscoverStorageCandidates(StoragePolicy{ProtectedRoots: []string{filepath.Dir(os.TempDir())}, RequirePhysicalSeparation: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if candidate.Location == "." || candidate.Location == "" {
			t.Fatalf("invalid fallback candidate = %+v", candidate)
		}
	}
}

func TestInspectStorageMountKeepsRepositoryDeviceEligible(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	if err := os.MkdirAll(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	candidate := inspectStorageMount(storageMount{Location: root, Kind: "removable-or-local", Filesystem: "ntfs3"}, StoragePolicy{
		RepositoryRoots:           []string{repository},
		RequirePhysicalSeparation: true,
	})
	if candidate.Status != "ready" || candidate.PhysicalIndependence != "observed" {
		t.Fatalf("repository-containing device should remain eligible relative to the credential source: %+v", candidate)
	}
	if candidate.ID == "" || candidate.ID != candidate.StableIdentity {
		t.Fatalf("candidate identity = %+v", candidate)
	}
}

func TestInspectStorageMountRejectsSameProtectedDevice(t *testing.T) {
	root := t.TempDir()
	protected := t.TempDir()
	candidate := inspectStorageMount(storageMount{Location: root, Kind: "local", Filesystem: "ext4"}, StoragePolicy{
		ProtectedRoots:            []string{protected},
		RequirePhysicalSeparation: true,
	})
	if candidate.Status != "rejected" || candidate.PhysicalIndependence != "same-device" {
		t.Fatalf("same protected device should be rejected: %+v", candidate)
	}
}
