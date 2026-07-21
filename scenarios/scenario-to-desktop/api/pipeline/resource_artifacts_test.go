package pipeline

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageBundledResourceArtifactsStagesVerifiedArtifactsAndPlan(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := `{"dependencies":{"resources":{"openrouter":{"enabled":true,"required":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(service), 0o644); err != nil {
		t.Fatal(err)
	}
	resourceDir := filepath.Join(root, "resources", "openrouter")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resource := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-openrouter_${os}_${arch}"}},"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-client","architectures":["amd64","arm64"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"bundled-client","reason":"test"},"windows":{"support":"unsupported","mode":"bundled-client","reason":"test"}}}}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(resource), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(root, "release")
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	checksums := make([]string, 0, 6)
	for _, arch := range []string{"amd64", "arm64"} {
		name := "resource-openrouter_linux_" + arch
		for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
			file := name + suffix
			body := []byte("artifact-" + arch + suffix)
			if err := os.WriteFile(filepath.Join(artifactRoot, file), body, 0o755); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(body)
			checksums = append(checksums, hex.EncodeToString(sum[:])+"  "+file)
		}
	}
	checksumData := []byte(strings.Join(checksums, "\n") + "\n")
	if err := os.WriteFile(filepath.Join(artifactRoot, "SHA256SUMS"), checksumData, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestReleaseSignature(t, root, artifactRoot, checksumData)

	bundleDir := filepath.Join(root, "bundle")
	plan, err := resolveResourceDeploymentPlan(scenarioPath, artifactRoot, []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("resolve plan: %v", err)
	}
	copied, err := stageBundledResourceArtifacts(bundleDir, artifactRoot, plan)
	if err != nil {
		t.Fatalf("stage artifacts: %v", err)
	}
	if len(copied) != 4 {
		t.Fatalf("copied = %v, want executable, contract, provenance, and plan", copied)
	}
	for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
		if _, err := os.Stat(filepath.Join(bundleDir, "resources", "openrouter", "resource-openrouter_linux_amd64"+suffix)); err != nil {
			t.Fatalf("staged artifact %s: %v", suffix, err)
		}
	}
	planData, err := os.ReadFile(filepath.Join(bundleDir, "resource-deployment-plan.json"))
	if err != nil || !strings.Contains(string(planData), "resource-openrouter_linux_amd64") {
		t.Fatalf("read plan: %v; content=%s", err, planData)
	}
}

func TestStageBundledResourceArtifactsRejectsChecksumMismatch(t *testing.T) {
	// Signature validation is covered above; this focused regression case ensures
	// a signed manifest cannot authorize altered bytes.
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"dependencies":{"resources":{"openrouter":{"enabled":true,"required":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "resources", "openrouter"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-openrouter_${os}_${arch}"}},"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-client","architectures":["amd64"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"bundled-client","reason":"test"},"windows":{"support":"unsupported","mode":"bundled-client","reason":"test"}}}}}`
	if err := os.WriteFile(filepath.Join(root, "resources", "openrouter", "resource.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(root, "release")
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "resource-openrouter_linux_amd64"), []byte("altered"), 0o755); err != nil {
		t.Fatal(err)
	}
	checksumData := []byte(strings.Repeat("0", 64) + "  resource-openrouter_linux_amd64\n")
	if err := os.WriteFile(filepath.Join(artifactRoot, "SHA256SUMS"), checksumData, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestReleaseSignature(t, root, artifactRoot, checksumData)
	if _, err := resolveResourceDeploymentPlan(scenarioPath, artifactRoot, []string{"linux-amd64"}); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestResolveResourceDeploymentPlanSelectsOnlyValidatedFallback(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := `{"dependencies":{"resources":{"primary":{"enabled":true,"required":true}}},"deployment":{"dependencies":{"resources":{"primary":{"platform_support":{"tier-2-desktop":{"alternatives":["fallback"]}}}}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(service), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, manifest := range map[string]string{
		"primary":  `{"deployment":{"profiles":{"desktop":{"linux":{"support":"unsupported","mode":"manual","reason":"not available"},"macos":{"support":"unsupported","mode":"manual","reason":"not available"},"windows":{"support":"unsupported","mode":"manual","reason":"not available"}}}}}`,
		"fallback": `{"deployment":{"profiles":{"desktop":{"linux":{"support":"conditional","mode":"native-host-tool","architectures":["amd64"],"limitations":["host tool required"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"manual","reason":"not available"},"windows":{"support":"unsupported","mode":"manual","reason":"not available"}}}}}`,
	} {
		dir := filepath.Join(root, "resources", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "resource.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := resolveResourceDeploymentPlan(scenarioPath, "", []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("resolve fallback: %v", err)
	}
	if len(plan.Resources) != 1 || plan.Resources[0].Resource != "fallback" || plan.Resources[0].RequestedResource != "primary" {
		t.Fatalf("unexpected fallback plan: %#v", plan.Resources)
	}
	if _, err := resolveResourceDeploymentPlan(scenarioPath, "", []string{"windows-amd64"}); err == nil {
		t.Fatal("expected unsupported fallback to be rejected")
	}
}

func writeTestReleaseSignature(t *testing.T, root, artifactRoot string, checksums []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "install"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "install", "vrooli-release.pub"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub}), 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(checksums)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "SHA256SUMS.sig"), []byte(base64.StdEncoding.EncodeToString(signature)), 0o644); err != nil {
		t.Fatal(err)
	}
}
