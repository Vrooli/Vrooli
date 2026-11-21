package coreactivation

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
)

func workerManifest(t *testing.T, name string, write stateformat.Version, reads []stateformat.Version) stateformat.Manifest {
	t.Helper()
	digest, err := stateformat.DigestSchema(name)
	if err != nil {
		t.Fatal(err)
	}
	schemas := make(map[stateformat.Version]string, len(reads))
	for _, version := range reads {
		schemas[version] = digest
	}
	stores := []stateformat.Store{{Name: name, Write: write, Reads: reads, Schemas: schemas}}
	manifest := stateformat.Manifest{Format: stateformat.ManifestFormat, Stores: stores}
	if err := manifest.Seal(); err != nil {
		t.Fatal(err)
	}
	return manifest
}

// A watchdog published before the last compatible install can carry a different
// legacy whole-source identity while speaking the same formats. The versioned
// manifests decide, so the worker is accepted.
func TestWorkerFormatAgreementAcceptsACompatibleStaleWatchdog(t *testing.T) {
	agrees, err := workerFormatAgreement(workerManifest(t, "registry", 1, []stateformat.Version{1}), workerManifest(t, "registry", 1, []stateformat.Version{1}), true)
	if err != nil || !agrees {
		t.Fatalf("agreement = %v, %v; a compatible manifest pair must be accepted", agrees, err)
	}
}

func TestWorkerFormatAgreementRefusesAnIncompatibleWatchdog(t *testing.T) {
	prior := workerManifest(t, "registry", 1, []stateformat.Version{1})
	candidate := prior
	candidate.Stores = []stateformat.Store{{Name: "registry", Write: 2, Reads: []stateformat.Version{2}, Schemas: map[stateformat.Version]string{2: mustDigest(t, "registry-v2")}}}
	if err := candidate.Seal(); err != nil {
		t.Fatal(err)
	}
	agrees, err := workerFormatAgreement(prior, candidate, true)
	if err == nil || agrees {
		t.Fatalf("agreement = %v, %v; a watchdog that writes an unreadable format must be refused", agrees, err)
	}
	if !strings.Contains(err.Error(), "state format") {
		t.Fatalf("refusal does not explain itself: %v", err)
	}
}

func TestWorkerFormatAgreementFallsBackWhenAManifestIsMissing(t *testing.T) {
	agrees, err := workerFormatAgreement(stateformat.Manifest{}, stateformat.Manifest{}, false)
	if err != nil || agrees {
		t.Fatalf("agreement = %v, %v; a missing manifest must fall back to the identity rule", agrees, err)
	}
}

func mustDigest(t *testing.T, label string) string {
	t.Helper()
	digest, err := stateformat.DigestSchema(label)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
