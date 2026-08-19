package credentialescrow

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

func TestDiscoverReturnsTypedMissingInputsAndMetadataOnlyCandidates(t *testing.T) {
	home := t.TempDir()
	provider := NewProvider(t.TempDir(), home)
	provider.now = func() time.Time { return time.Date(2026, 8, 19, 16, 0, 0, 0, time.UTC) }
	provider.describeStore = func() (securestore.StoreStatus, error) {
		return securestore.StoreStatus{Initialized: true, Path: filepath.Join(home, "secrets.enc.json")}, nil
	}
	provider.loadRepositories = func() ([]kopiaregistry.Entry, error) { return nil, nil }
	provider.discoverStorage = func(policy hostinventory.StoragePolicy) ([]hostinventory.StorageCandidate, error) {
		if len(policy.ProtectedRoots) == 0 || !policy.RequirePhysicalSeparation {
			t.Fatalf("provider did not pass the separation policy: %+v", policy)
		}
		return []hostinventory.StorageCandidate{{
			ID: "candidate-1", Kind: "removable-or-local", Location: filepath.Join(home, "external"),
			StableIdentity: "stable-1", DeviceIdentity: "dev-2", Filesystem: "ntfs3",
			Writable: true, PhysicalIndependence: "observed", Status: "ready",
		}}, nil
	}
	provider.readCopyConfig = func(string) (securestore.CopyConfig, error) { return securestore.CopyConfig{}, nil }
	provider.readReceipt = func(string) (credentialauthority.RecoveryReceipt, bool, error) {
		return credentialauthority.RecoveryReceipt{}, false, nil
	}

	status, err := provider.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "needs_operator_input" {
		t.Fatalf("state = %s, want needs_operator_input", status.State)
	}
	if strings.Join(status.MissingInputs, ",") != "sink,recovery_passphrase" {
		t.Fatalf("missing inputs = %v", status.MissingInputs)
	}
	if len(status.Descriptor.Inputs) == 0 || len(status.Descriptor.Inputs[0].Candidates) != 1 {
		t.Fatalf("descriptor candidates = %+v", status.Descriptor.Inputs)
	}
	payload, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	// The word passphrase is contract metadata; no answer value can appear
	// because discovery has no value-bearing input.
	if strings.Contains(string(payload), "secret-answer") {
		t.Fatalf("discovery response leaked a secret answer: %s", payload)
	}
}
