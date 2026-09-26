package portability

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
)

func TestRecordQualificationObservationReplacesOnlyWithNewerProof(t *testing.T) {
	root := t.TempDir()
	observed := QualificationObservation{Resource: "whisper", HostOS: "linux", Architecture: "amd64", Node: "node-a", RunID: "run-1", ObservedAt: time.Unix(2, 0).UTC(), HealthPassed: true}
	if err := RecordQualificationObservation(root, observed); err != nil {
		t.Fatal(err)
	}
	older := observed
	older.Node, older.RunID, older.ObservedAt = "node-old", "run-old", time.Unix(1, 0).UTC()
	if err := RecordQualificationObservation(root, older); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, qualificationObservationPath))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || string(data) == "null" {
		t.Fatal("observation ledger is empty")
	}
	if got := readQualificationObservations(filepath.Join(root, qualificationObservationPath)); len(got) != 1 || got[0].Node != "node-a" {
		t.Fatalf("observations = %+v", got)
	}
}

func TestResourceAcquisitionConformanceUsesDeclaredPlatformTargets(t *testing.T) {
	resource := ResourceInput{
		Platforms:   map[string]string{"linux": "supported"},
		Deployment:  ResourceDeploymentInput{Profiles: map[string]map[string]ResourceProfileInput{"desktop": {"linux": {Architectures: []string{"amd64"}}}}},
		Acquisition: &ResourceAcquisitionInput{Kind: "url", Targets: []binaryfetch.AcquisitionTarget{{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/tool", SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}},
	}
	resolved, reason := resourceAcquisitionResolves(resource, "linux", "amd64")
	if !resolved || reason == "" {
		t.Fatalf("resolved=%v reason=%q", resolved, reason)
	}
	resource.Acquisition.Targets[0].When["arch"] = "arm64"
	resolved, reason = resourceAcquisitionResolves(resource, "linux", "amd64")
	if resolved || reason == "" {
		t.Fatalf("mismatched target resolved=%v reason=%q", resolved, reason)
	}
}
