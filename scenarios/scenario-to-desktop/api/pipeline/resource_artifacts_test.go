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

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
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

func TestStageBundledServiceStagesSeparatelyPinnedServer(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"dependencies":{"resources":{"vault":{"enabled":true,"required":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(root, "release")
	if err := os.MkdirAll(artifactRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	serverName, serverBody := "vault_linux_amd64", []byte("verified server")
	if err := os.WriteFile(filepath.Join(artifactRoot, serverName), serverBody, 0o755); err != nil {
		t.Fatal(err)
	}
	serverSum := sha256.Sum256(serverBody)
	resource := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-vault_${os}_${arch}"}},"managed_service":{"provider_policy":{"target_defaults":{"control-plane":"managed-shared","desktop-bundle":"managed-private"},"allowed_modes":["managed-private","managed-shared"],"shared_reuse_requires_consent":true},"artifact":{"path":"bin/vault","version":"1.17.6","bundle_artifact":"vault_${os}_${arch}","sha256":"` + hex.EncodeToString(serverSum[:]) + `"}},"ports":[{"name":"http","host":8200}],"health_checks":[{"type":"http","target":"http://127.0.0.1:${RESOURCE_PORT_HTTP}/v1/sys/health","expected_status":[200,501],"timeout_seconds":5}],"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-service","architectures":["amd64"],"limitations":["test"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"bundled-service","reason":"test"},"windows":{"support":"unsupported","mode":"bundled-service","reason":"test"}}}}}`
	resourceDir := filepath.Join(root, "resources", "vault")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(resource), 0o644); err != nil {
		t.Fatal(err)
	}
	checksums := []string{hex.EncodeToString(serverSum[:]) + "  " + serverName}
	controller := "resource-vault_linux_amd64"
	for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
		file, body := controller+suffix, []byte("controller"+suffix)
		if err := os.WriteFile(filepath.Join(artifactRoot, file), body, 0o755); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		checksums = append(checksums, hex.EncodeToString(sum[:])+"  "+file)
	}
	checksumData := []byte(strings.Join(checksums, "\n") + "\n")
	if err := os.WriteFile(filepath.Join(artifactRoot, "SHA256SUMS"), checksumData, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestReleaseSignature(t, root, artifactRoot, checksumData)

	plan, err := resolveResourceDeploymentPlan(scenarioPath, artifactRoot, []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("resolve plan: %v", err)
	}
	item := plan.Resources[0]
	if item.Service == nil || item.Service.Artifact != serverName || len(item.Service.Files) != 1 {
		t.Fatalf("bundled service plan = %#v", item.Service)
	}
	if item.Service.ProviderPolicy.TargetDefaults[resourcedeployment.ProviderTargetControlPlane] != resourcedeployment.ProviderManagedShared || item.Service.ProviderPolicy.TargetDefaults[resourcedeployment.ProviderTargetDesktopBundle] != resourcedeployment.ProviderManagedPrivate || !item.Service.ProviderPolicy.SharedReuseRequiresConsent {
		t.Fatalf("bundled service provider policy = %#v", item.Service.ProviderPolicy)
	}
	if len(item.Service.Ports) != 1 || item.Service.Ports[0].Name != "http" || item.Service.Ports[0].Host != 8200 {
		t.Fatalf("bundled service ports = %#v", item.Service.Ports)
	}
	if len(item.Service.HealthChecks) != 1 || item.Service.HealthChecks[0].Target != "http://127.0.0.1:${RESOURCE_PORT_HTTP}/v1/sys/health" {
		t.Fatalf("bundled service health contract = %#v", item.Service.HealthChecks)
	}
	copied, err := stageBundledResourceArtifacts(filepath.Join(root, "bundle"), artifactRoot, plan)
	if err != nil {
		t.Fatalf("stage artifacts: %v", err)
	}
	if len(copied) != 5 {
		t.Fatalf("copied = %v, want controller, server, and plan", copied)
	}
	if _, err := os.Stat(filepath.Join(root, "bundle", "resources", "vault", serverName)); err != nil {
		t.Fatalf("staged server: %v", err)
	}
}

func TestResolveResourceForTargetRejectsStaticManagedServicePolicy(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "vault")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-vault_${os}_${arch}"}},"managed_service":{"provider_policy":{"default_mode":"managed-private","allowed_modes":["managed-private"]}},"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-service","architectures":["amd64"]}}}}}`
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveResourceForTarget(root, "vault", "vault", nil, resourcedeployment.Platform{OS: "linux", Arch: "amd64"}, map[string]bool{})
	if err == nil || !strings.Contains(err.Error(), "invalid managed-service provider policy") {
		t.Fatalf("expected static managed-service policy rejection, got %v", err)
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

func TestLoadReleaseChecksumsExplainsMissingReleaseSignature(t *testing.T) {
	artifactRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(artifactRoot, "SHA256SUMS"), []byte(strings.Repeat("0", 64)+"  resource-vault_linux_amd64\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadReleaseChecksums(artifactRoot, filepath.Join(artifactRoot, "vrooli-release.pub"))
	if err == nil || !strings.Contains(err.Error(), "Vrooli release signing authority") || !strings.Contains(err.Error(), "SHA256SUMS.sig") {
		t.Fatalf("missing signature error = %v, want release-authority guidance", err)
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
	if plan.Resources[0].SelectedFallback == nil || plan.Resources[0].SelectedFallback.Resource != "fallback" || plan.Resources[0].SelectedFallback.Reason == "" {
		t.Fatalf("fallback selection must be explicit: %#v", plan.Resources[0])
	}
	if _, err := resolveResourceDeploymentPlan(scenarioPath, "", []string{"windows-amd64"}); err == nil {
		t.Fatal("expected unsupported fallback to be rejected")
	}
}

func TestResolveResourceDeploymentPlanRejectsEmptyOrDuplicateTargetMatrix(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"dependencies":{"resources":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveResourceDeploymentPlan(scenarioPath, "", nil); err == nil || !strings.Contains(err.Error(), "target matrix") {
		t.Fatalf("expected empty matrix to fail, got %v", err)
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
