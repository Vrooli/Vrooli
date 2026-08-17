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
	"runtime"
	"strings"
	"testing"

	hostreq "github.com/vrooli/vrooli/packages/hostreq"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestStageBundledResourceArtifactsStagesVerifiedArtifactsAndPlan(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	mustMkdirAll(t, filepath.Join(scenarioPath, ".vrooli"))
	service := `{"dependencies":{"resources":{"openrouter":{"enabled":true,"required":true}}}}`
	mustWriteFile(t, filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(service), 0o644)
	resourceDir := filepath.Join(root, "resources", "openrouter")
	mustMkdirAll(t, resourceDir)
	resource := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-openrouter_${os}_${arch}"}},"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-client","architectures":["amd64","arm64"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"bundled-client","reason":"test"},"windows":{"support":"unsupported","mode":"bundled-client","reason":"test"}}}}}`
	mustWriteFile(t, filepath.Join(resourceDir, "resource.json"), []byte(resource), 0o644)
	artifactRoot := filepath.Join(root, "release")
	mustMkdirAll(t, artifactRoot)
	checksums := make([]string, 0, 6)
	for _, arch := range []string{"amd64", "arm64"} {
		name := "resource-openrouter_linux_" + arch
		for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
			file := name + suffix
			body := []byte("artifact-" + arch + suffix)
			mustWriteFile(t, filepath.Join(artifactRoot, file), body, 0o755)
			sum := sha256.Sum256(body)
			checksums = append(checksums, hex.EncodeToString(sum[:])+"  "+file)
		}
	}
	checksumData := []byte(strings.Join(checksums, "\n") + "\n")
	mustWriteFile(t, filepath.Join(artifactRoot, "SHA256SUMS"), checksumData, 0o644)
	writeTestReleaseSignature(t, root, artifactRoot, checksumData)

	bundleDir := filepath.Join(root, "bundle")
	plan, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, artifactRoot, "", []string{"linux-amd64"}, resourcedeployment.ArtifactTrustProduction)
	mustNoError(t, err, "resolve plan")
	copied, err := stageBundledResourceArtifacts(bundleDir, artifactRoot, plan)
	mustNoError(t, err, "stage artifacts")
	if len(copied) != 4 {
		t.Fatalf("copied = %v, want executable, contract, provenance, and plan", copied)
	}
	for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
		mustPathExist(t, filepath.Join(bundleDir, "resources", "openrouter", "resource-openrouter_linux_amd64"+suffix))
	}
	planData, err := os.ReadFile(filepath.Join(bundleDir, "resource-deployment-plan.json"))
	if err != nil || !strings.Contains(string(planData), "resource-openrouter_linux_amd64") {
		t.Fatalf("read plan: %v; content=%s", err, planData)
	}
}

func TestStageBundledServiceStagesSeparatelyPinnedServer(t *testing.T) {
	root := t.TempDir()
	scenarioPath := filepath.Join(root, "scenarios", "demo")
	mustMkdirAll(t, filepath.Join(scenarioPath, ".vrooli"))
	mustWriteFile(t, filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"dependencies":{"resources":{"vault":{"enabled":true,"required":true}}}}`), 0o644)
	artifactRoot := filepath.Join(root, "release")
	mustMkdirAll(t, artifactRoot)
	serverName, serverBody := "vault_linux_amd64", []byte("verified server")
	mustWriteFile(t, filepath.Join(artifactRoot, serverName), serverBody, 0o755)
	serverSum := sha256.Sum256(serverBody)
	resource := `{"cli":{"distribution":{"kind":"prebuilt_artifact","artifact_name":"resource-vault_${os}_${arch}"}},"managed_service":{"provider_policy":{"target_defaults":{"control-plane":"managed-shared","desktop-bundle":"managed-private"},"allowed_modes":["managed-private","managed-shared"],"shared_reuse_requires_consent":true},"artifact":{"path":"bin/vault","version":"1.17.6","bundle_artifact":"vault_${os}_${arch}","sha256":"` + hex.EncodeToString(serverSum[:]) + `"}},"ports":[{"name":"http","host":8200}],"health_checks":[{"type":"http","target":"http://127.0.0.1:${RESOURCE_PORT_HTTP}/v1/sys/health","expected_status":[200,501],"timeout_seconds":5}],"deployment":{"profiles":{"desktop":{"linux":{"support":"supported","mode":"bundled-service","architectures":["amd64"],"limitations":["test"],"evidence":["test"]},"macos":{"support":"unsupported","mode":"bundled-service","reason":"test"},"windows":{"support":"unsupported","mode":"bundled-service","reason":"test"}}}}}`
	resourceDir := filepath.Join(root, "resources", "vault")
	mustMkdirAll(t, resourceDir)
	mustWriteFile(t, filepath.Join(resourceDir, "resource.json"), []byte(resource), 0o644)
	checksums := []string{hex.EncodeToString(serverSum[:]) + "  " + serverName}
	controller := "resource-vault_linux_amd64"
	for _, suffix := range []string{"", ".manifest.json", ".build.json"} {
		file, body := controller+suffix, []byte("controller"+suffix)
		mustWriteFile(t, filepath.Join(artifactRoot, file), body, 0o755)
		sum := sha256.Sum256(body)
		checksums = append(checksums, hex.EncodeToString(sum[:])+"  "+file)
	}
	checksumData := []byte(strings.Join(checksums, "\n") + "\n")
	mustWriteFile(t, filepath.Join(artifactRoot, "SHA256SUMS"), checksumData, 0o644)
	writeTestReleaseSignature(t, root, artifactRoot, checksumData)

	plan, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, artifactRoot, "", []string{"linux-amd64"}, resourcedeployment.ArtifactTrustProduction)
	mustNoError(t, err, "resolve plan")
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
	mustNoError(t, err, "stage artifacts")
	if len(copied) != 5 {
		t.Fatalf("copied = %v, want controller, server, and plan", copied)
	}
	mustPathExist(t, filepath.Join(root, "bundle", "resources", "vault", serverName))
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	mustNoError(t, os.MkdirAll(path, 0o755), "create directory")
}

func mustWriteFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	mustNoError(t, os.WriteFile(path, data, mode), "write file")
}

func mustPathExist(t *testing.T, path string) {
	t.Helper()
	mustNoError(t, func() error { _, err := os.Stat(path); return err }(), "expected staged path")
}

func mustNoError(t *testing.T, err error, operation string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", operation, err)
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
	if _, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, artifactRoot, "", []string{"linux-amd64"}, resourcedeployment.ArtifactTrustProduction); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestProductionTrustExplainsMissingReleaseSignature(t *testing.T) {
	artifactRoot := t.TempDir()
	artifactName := "resource-vault_linux_amd64"
	if err := os.WriteFile(filepath.Join(artifactRoot, artifactName), []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("fixture"))
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: artifactName, SHA256: hex.EncodeToString(sum[:]), Role: "fixture", UpstreamProvenance: "fixture"}}}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "release-manifest.json"), canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = resourcedeployment.VerifyReleaseDirectory(artifactRoot, resourcedeployment.ArtifactTrustProduction, filepath.Join(artifactRoot, "vrooli-release.pub"))
	if err == nil || !strings.Contains(err.Error(), "production release signature missing") || !strings.Contains(err.Error(), "release-manifest.sig.json") {
		t.Fatalf("missing signature error = %v, want release-authority guidance", err)
	}
}

func TestDesktopToolEligibilityRequiresAStagedArtifactAndPreservesCrossPlatformUnknown(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "../../../.."))
	scenarioPath := filepath.Join(t.TempDir(), "fixture")
	mustMkdirAll(t, filepath.Join(scenarioPath, ".vrooli"))
	mustWriteFile(t, filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"service":{"name":"fixture"},"hostTools":[{"name":"yq","required":true,"reason":"fixture yq"}]}`), 0o644)

	toolRoot := t.TempDir()
	body := []byte("verified yq")
	toolName := "tool_yq_linux_amd64"
	mustWriteFile(t, filepath.Join(toolRoot, toolName), body, 0o755)
	sum := sha256.Sum256(body)
	manifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: toolName, SHA256: hex.EncodeToString(sum[:]), Role: "tool", UpstreamProvenance: "fixture"}}}
	canonical, err := manifest.CanonicalBytes()
	mustNoError(t, err, "canonicalize tool release")
	mustWriteFile(t, filepath.Join(toolRoot, "release-manifest.json"), canonical, 0o644)

	items, err := resolveDesktopHostRequirements(repoRoot, scenarioPath, []string{"none"}, resourcedeployment.Platform{OS: "linux", Arch: "amd64"}, toolRoot, resourcedeployment.ArtifactTrustDevelopmentLocal)
	mustNoError(t, err, "resolve staged tool")
	yq := findHostRequirement(t, items, "yq")
	if yq.Verdict != string(hostreq.EligibilityEligible) || yq.Artifact != toolName || yq.Architecture != "amd64" {
		t.Fatalf("staged yq requirement = %#v, want eligible staged amd64 artifact", yq)
	}

	missingRoot := t.TempDir()
	filler := []byte("unrelated")
	fillerSum := sha256.Sum256(filler)
	fillerName := "tool_fixture_linux_amd64"
	mustWriteFile(t, filepath.Join(missingRoot, fillerName), filler, 0o755)
	missingManifest := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{Name: fillerName, SHA256: hex.EncodeToString(fillerSum[:]), Role: "fixture", UpstreamProvenance: "fixture"}}}
	missingCanonical, err := missingManifest.CanonicalBytes()
	mustNoError(t, err, "canonicalize missing-tool release")
	mustWriteFile(t, filepath.Join(missingRoot, "release-manifest.json"), missingCanonical, 0o644)
	missingItems, err := resolveDesktopHostRequirements(repoRoot, scenarioPath, []string{"none"}, resourcedeployment.Platform{OS: "linux", Arch: "amd64"}, missingRoot, resourcedeployment.ArtifactTrustDevelopmentLocal)
	mustNoError(t, err, "resolve missing staged tool")
	missing := findHostRequirement(t, missingItems, "yq")
	if missing.Verdict != string(hostreq.EligibilityIneligible) || missing.Artifact != toolName {
		t.Fatalf("missing yq requirement = %#v, want ineligible with selected artifact name", missing)
	}

	mustWriteFile(t, filepath.Join(scenarioPath, ".vrooli", "service.json"), []byte(`{"service":{"name":"fixture"},"hostTools":[{"name":"ffmpeg","required":true,"reason":"fixture ffmpeg"}]}`), 0o644)
	crossPlatform, err := resolveDesktopHostRequirements(repoRoot, scenarioPath, []string{"none"}, resourcedeployment.Platform{OS: "windows", Arch: "amd64"}, "", resourcedeployment.ArtifactTrustDevelopmentLocal)
	mustNoError(t, err, "resolve cross-platform host requirement")
	ffmpeg := findHostRequirement(t, crossPlatform, "ffmpeg")
	if ffmpeg.Verdict != string(hostreq.EligibilityUnknown) || ffmpeg.Architecture != "amd64" || ffmpeg.OS != "windows" {
		t.Fatalf("cross-platform ffmpeg requirement = %#v, want unknown windows/amd64", ffmpeg)
	}
}

func findHostRequirement(t *testing.T, items []HostRequirementPlanItem, name string) HostRequirementPlanItem {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("host requirement %q not found in %#v", name, items)
	return HostRequirementPlanItem{}
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
	plan, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, "", "", []string{"linux-amd64"}, resourcedeployment.ArtifactTrustDevelopmentLocal)
	if err != nil {
		t.Fatalf("resolve fallback: %v", err)
	}
	if len(plan.Resources) != 1 || plan.Resources[0].Resource != "fallback" || plan.Resources[0].RequestedResource != "primary" {
		t.Fatalf("unexpected fallback plan: %#v", plan.Resources)
	}
	if plan.Resources[0].SelectedFallback == nil || plan.Resources[0].SelectedFallback.Resource != "fallback" || plan.Resources[0].SelectedFallback.Reason == "" {
		t.Fatalf("fallback selection must be explicit: %#v", plan.Resources[0])
	}
	windowsPlan, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, "", "", []string{"windows-amd64"}, resourcedeployment.ArtifactTrustDevelopmentLocal)
	if err != nil {
		t.Fatalf("unsupported target must remain a buildable, recorded limitation: %v", err)
	}
	if len(windowsPlan.Resources) != 1 {
		t.Fatalf("windows resources = %#v", windowsPlan.Resources)
	}
	windows := windowsPlan.Resources[0]
	if windows.Eligibility != "ineligible" || windows.Support != "unsupported" || !strings.Contains(windows.EligibilityReason, "not available") {
		t.Fatalf("windows limitation = %#v", windows)
	}
	if windowsPlan.Promotable {
		t.Fatal("an artifact with an ineligible required resource must be non-promotable")
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
	if _, err := resolveResourceDeploymentPlanWithTrust(scenarioPath, "", "", nil, resourcedeployment.ArtifactTrustDevelopmentLocal); err == nil || !strings.Contains(err.Error(), "target matrix") {
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
	var artifacts []resourcedeployment.ReleaseArtifact
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			artifacts = append(artifacts, resourcedeployment.ReleaseArtifact{Name: fields[1], SHA256: fields[0], Role: "fixture", UpstreamProvenance: "fixture"})
		}
	}
	canonical, err := (resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: artifacts}).CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactRoot, "release-manifest.json"), canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	envelope := []byte(`{"schema_version":"v1","key_id":"fixture","algorithm":"rsa-pkcs1v15-sha256","signature":"` + base64.StdEncoding.EncodeToString(signature) + `"}`)
	if err := os.WriteFile(filepath.Join(artifactRoot, "release-manifest.sig.json"), envelope, 0o644); err != nil {
		t.Fatal(err)
	}
}
