package assets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestApplyStopsBeforeWorkWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := apply(ctx, t.TempDir(), Plan{}, "", "", Inspection{}, Inspection{}, 0, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("apply with canceled context = %v, want context.Canceled", err)
	}
}

func TestInspectCarriesVerifiedChecksumEvidenceIntoApply(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("asset filesystem inspection is Linux-only")
	}
	root, destination := t.TempDir(), t.TempDir()
	imageName := "pinned.img"
	imageBytes := []byte("pinned image fixture")
	digest := sha256.Sum256(imageBytes)
	imageSHA := hex.EncodeToString(digest[:])
	imagePath := filepath.Join(destination, imageName)
	if err := os.WriteFile(imagePath, imageBytes, 0o444); err != nil {
		t.Fatal(err)
	}
	plan := Plan{
		OutputDirectory: destination, ImageManifest: "images.json", ReceiptPath: filepath.Join(root, "receipt.json"),
		ChecksumURL:  "https://cloud-images.ubuntu.com/fixture/SHA256SUMS",
		SignatureURL: "https://cloud-images.ubuntu.com/fixture/SHA256SUMS.gpg", KeyringPath: filepath.Join(root, "keyring.gpg"),
		MinimumFreeBytes: 1, MaximumGrowthBytes: 1 << 30, MaximumImageBytes: 100,
		Images: []Image{{FileName: imageName, SHA256: imageSHA, SourceURL: "https://cloud-images.ubuntu.com/fixture/" + imageName}},
	}
	fetchCalls := 0
	inspection, err := inspect(context.Background(), root, plan, "setup-assets.json", func(context.Context, Plan) (verifiedChecksumMetadata, error) {
		fetchCalls++
		return verifiedChecksumMetadata{
			checksums: map[string]string{imageName: imageSHA}, fingerprint: "ABCDEF",
			checksumSHA256: "checksum-digest", signatureSHA256: "signature-digest",
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if fetchCalls != 1 || inspection.verifiedChecksumFingerprint != "ABCDEF" || inspection.verifiedChecksumSHA256 != "checksum-digest" || inspection.verifiedSignatureSHA256 != "signature-digest" {
		t.Fatalf("inspection checksum evidence = calls %d, fingerprint %q, digests %q/%q", fetchCalls, inspection.verifiedChecksumFingerprint, inspection.verifiedChecksumSHA256, inspection.verifiedSignatureSHA256)
	}
	data, err := json.Marshal(inspection)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "verifiedChecksum") || strings.Contains(string(data), "signature-digest") {
		t.Fatalf("internal apply evidence leaked into preview JSON: %s", data)
	}
	invalid, err := inspect(context.Background(), root, plan, "setup-assets.json", func(context.Context, Plan) (verifiedChecksumMetadata, error) {
		return verifiedChecksumMetadata{checksums: map[string]string{imageName: "wrong-digest"}, fingerprint: "UNTRUSTED"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(invalid.Stops) == 0 || invalid.verifiedChecksumFingerprint != "" {
		t.Fatalf("mismatched signed checksum retained apply evidence: stops=%v fingerprint=%q", invalid.Stops, invalid.verifiedChecksumFingerprint)
	}
}

func TestScenarioToCloudA2PlanPinsExactScope(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, found, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil || !found {
		t.Fatalf("LoadScenarioPlan = found %v, err %v", found, err)
	}
	if plan.MinimumFreeBytes != 120_000_000_000 || plan.MaximumGrowthBytes != 25_000_000_000 {
		t.Fatalf("capacity bounds = reserve %d growth %d", plan.MinimumFreeBytes, plan.MaximumGrowthBytes)
	}
	if len(plan.Images) != 2 || len(plan.HostTools) != 3 {
		t.Fatalf("plan images/tools = %d/%d; want exactly 2/3", len(plan.Images), len(plan.HostTools))
	}
}

func TestPlanRejectsWidenedA2Scope(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*Plan)
	}{
		{"reserve", func(p *Plan) { p.MinimumFreeBytes = 119_000_000_000 }},
		{"growth", func(p *Plan) { p.MaximumGrowthBytes++ }},
		{"digest", func(p *Plan) { p.Images[0].SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }},
		{"package", func(p *Plan) { p.HostTools[0].Package = "qemu" }},
		{"destination", func(p *Plan) { p.OutputDirectory = "/tmp/vm" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			candidate := plan
			candidate.Images = append([]Image(nil), plan.Images...)
			candidate.HostTools = append([]ToolPlan(nil), plan.HostTools...)
			mutation.edit(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("widened or changed plan unexpectedly validated")
			}
		})
	}
}

func TestValidateToolPreviewRequiresExactThreeOwnersAndAptPackages(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil {
		t.Fatal(err)
	}
	report := hostreqkit.Report{Host: hostreqkit.Host{OS: "linux", PackageManager: "apt"}, Tools: []hostreqkit.ToolStatus{
		{Name: "qemu", PackageName: "qemu-system-x86"},
		{Name: "qemu-arm", PackageName: "qemu-system-arm"},
		{Name: "cloud-localds", PackageName: "cloud-image-utils"},
	}}
	if err := ValidateToolPreview(plan, report); err != nil {
		t.Fatalf("exact preview rejected: %v", err)
	}
	report.Tools[0].PackageName = "qemu"
	if err := ValidateToolPreview(plan, report); err == nil {
		t.Fatal("changed package plan unexpectedly passed")
	}
}

func TestLoadScenarioPlanDoesNotMutateManifest(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := filepath.Join(root, "scenarios", "scenario-to-cloud", ".vrooli", planName)
	before, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("loading setup plan mutated its source manifest")
	}
	var plan Plan
	if err := json.Unmarshal(after, &plan); err != nil {
		t.Fatal(err)
	}
}
