package instance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func absentTools(string) (string, error) { return "", errors.New("not installed") }

func writeManifest(t *testing.T, images []ImageRecord) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "qemu-images.json")
	data, err := json.Marshal(ImageManifest{SchemaVersion: 1, Images: images})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestReadinessNamesSetupWhenToolsAreAbsent [REQ:STC-P0-041] proves the lane
// reports a typed unavailable readiness naming vrooli setup, never a silent
// failure, when the declared host tools are missing (EXT-06).
func TestReadinessNamesSetupWhenToolsAreAbsent(t *testing.T) {
	report := Readiness(context.Background(), ReadinessOptions{
		LookPath:          absentTools,
		KVMDevice:         filepath.Join(t.TempDir(), "kvm-absent"),
		HostArchitecture:  ArchAMD64,
		ImageManifestPath: filepath.Join(t.TempDir(), "missing.json"),
	})
	if report.Ready || report.State != ReadinessUnavailable {
		t.Fatalf("report = %+v, want unavailable", report)
	}
	if report.NextAction.Kind != "host_setup" || !strings.Contains(report.NextAction.Reference, "vrooli setup") {
		t.Fatalf("next action = %+v, want vrooli setup", report.NextAction)
	}
	err := report.Err()
	if !errors.Is(err, ErrProviderUnavailable) || !strings.Contains(err.Error(), "vrooli setup") {
		t.Fatalf("Err() = %v, want typed provider unavailable naming vrooli setup", err)
	}
	owners := map[string]string{}
	for _, tool := range report.Tools {
		if tool.Present {
			t.Fatalf("tool %s reported present with an absent lookup", tool.Name)
		}
		owners[tool.Name] = tool.DeclaredOwner
	}
	if owners["qemu-system-x86_64"] != "qemu" || owners["qemu-img"] != "qemu" || owners["cloud-localds"] != "cloud-localds" {
		t.Fatalf("declared owners = %v, want service.json hostTools names", owners)
	}
	if report.Architectures[ArchAMD64] != ArchStateUnavailable || report.Architectures[ArchARM64] != ArchStateUnavailable {
		t.Fatalf("architectures = %v, want both unavailable", report.Architectures)
	}
	if !hasLimitation(report, "EXT-06") || !hasLimitation(report, "EXT-05") {
		t.Fatalf("limitations = %v, want EXT-05 and EXT-06 named", report.Limitations)
	}
	if report.ImageManifest.Present || len(report.Images) != 0 {
		t.Fatalf("image manifest = %+v images = %v, want absent and empty", report.ImageManifest, report.Images)
	}
}

// TestReadinessArm64IsTCGOnAmd64Host [REQ:STC-P0-041] P20-A06: an arm64 lane
// on an amd64 host is emulation, never reported as accelerated.
func TestReadinessArm64IsTCGOnAmd64Host(t *testing.T) {
	lookPath := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	report := Readiness(context.Background(), ReadinessOptions{
		LookPath: lookPath, KVMDevice: filepath.Join(t.TempDir(), "no-kvm"), HostArchitecture: ArchAMD64,
		ImageManifestPath: filepath.Join(t.TempDir(), "missing.json"),
	})
	if report.Architectures[ArchARM64] != ArchStateTCG || report.Architectures[ArchAMD64] != ArchStateTCG {
		t.Fatalf("architectures = %v, want tcg for both without KVM", report.Architectures)
	}
	if report.Ready {
		t.Fatalf("tools present but no image recorded must not be ready")
	}
	if report.NextAction.Kind != "record_image" {
		t.Fatalf("next action = %+v, want record_image", report.NextAction)
	}
}

// TestReadinessReadyWhenToolsAndVerifiedImagePresent [REQ:STC-P0-041]
func TestReadinessReadyWhenToolsAndVerifiedImagePresent(t *testing.T) {
	image := filepath.Join(t.TempDir(), "base.qcow2")
	content := []byte("pristine-image")
	if err := os.WriteFile(image, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	manifest := writeManifest(t, []ImageRecord{
		{Path: image, SHA256: "sha256:" + hex.EncodeToString(sum[:]), OS: "ubuntu-24.04", Arch: ArchAMD64},
		{Path: filepath.Join(t.TempDir(), "absent.qcow2"), SHA256: strings.Repeat("0", 64), OS: "ubuntu-24.04", Arch: ArchARM64},
	})
	lookPath := func(name string) (string, error) { return "/usr/bin/" + name, nil }
	report := Readiness(context.Background(), ReadinessOptions{
		LookPath: lookPath, KVMDevice: filepath.Join(t.TempDir(), "no-kvm"), HostArchitecture: ArchAMD64,
		ImageManifestPath: manifest, VerifyImages: true,
	})
	if !report.Ready || report.State != ReadinessReady || report.Err() != nil {
		t.Fatalf("report = %+v, want ready", report)
	}
	if report.Images[0].DigestState != "verified" || report.Images[1].DigestState != "missing" {
		t.Fatalf("images = %+v, want verified and missing", report.Images)
	}
	if report.NextAction.Kind != "run_qualification" {
		t.Fatalf("next action = %+v, want run_qualification", report.NextAction)
	}
	// A mutated source image is a mismatch, never silently accepted (P20-A05).
	if err := os.WriteFile(image, []byte("mutated"), 0o600); err != nil {
		t.Fatal(err)
	}
	report = Readiness(context.Background(), ReadinessOptions{LookPath: lookPath, KVMDevice: "/nonexistent", HostArchitecture: ArchAMD64, ImageManifestPath: manifest, VerifyImages: true})
	if report.Ready || report.Images[0].DigestState != "mismatch" || report.NextAction.Kind != "repair_image" {
		t.Fatalf("report after mutation = %+v, want mismatch and repair_image", report)
	}
}

func TestVerifyImageDigestRejectsMalformedRecord(t *testing.T) {
	err := VerifyImageDigest(context.Background(), "/nonexistent", "sha256:short")
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("err = %v, want invalid request", err)
	}
}

func TestReadinessReportsUnparsableManifestInsteadOfIgnoringIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qemu-images.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := Readiness(context.Background(), ReadinessOptions{LookPath: absentTools, KVMDevice: "/nonexistent", ImageManifestPath: path})
	if !report.ImageManifest.Present || report.ImageManifest.Error == "" {
		t.Fatalf("manifest status = %+v, want present with parse error", report.ImageManifest)
	}
}

func TestProviderReadinessUsesItsSeams(t *testing.T) {
	provider := LocalQEMUProvider{LookPath: absentTools, ImageManifest: filepath.Join(t.TempDir(), "none.json")}
	report := provider.Readiness(context.Background())
	if report.Ready || report.Provider != "local-qemu" {
		t.Fatalf("report = %+v", report)
	}
}

func hasLimitation(report LaneReadiness, needle string) bool {
	for _, limitation := range report.Limitations {
		if strings.Contains(limitation, needle) {
			return true
		}
	}
	return false
}
