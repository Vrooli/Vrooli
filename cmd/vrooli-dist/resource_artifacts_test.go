package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func TestFinalizeReleaseArtifactStagingReportsStageCauseBeforeChecksumError(t *testing.T) {
	out := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	stageFailure := fmt.Errorf("stage Kokoro Linux artifact: wheel-only resolver failed")
	err := finalizeReleaseArtifactStaging(out, []error{stageFailure})
	if !errors.Is(err, stageFailure) || !strings.Contains(err.Error(), "wheel-only resolver failed") {
		t.Fatalf("stage cause was hidden: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(out, "SHA256SUMS")); !os.IsNotExist(statErr) {
		t.Fatalf("failed staging wrote a checksum manifest: %v", statErr)
	}
}

func TestComposedAcquisitionIsDeterministicAndNamesUnavailableStep(t *testing.T) {
	// Use a real tar.gz fixture so this test covers the same extraction path as
	// runtime/source composition without downloading a platform artifact.
	buildArchive := func(name, contents string) []byte {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		tarWriter := tar.NewWriter(gz)
		_ = tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(contents))})
		_, _ = tarWriter.Write([]byte(contents))
		_ = tarWriter.Close()
		_ = gz.Close()
		return b.Bytes()
	}
	archiveBytesA := buildArchive("bin/a", "alpha")
	archiveBytesB := buildArchive("lib/b", "beta")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := archiveBytesA
		if r.URL.Path == "/b" {
			value = archiveBytesB
		}
		_, _ = w.Write(value)
	}))
	defer server.Close()
	digest := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	target := binaryfetch.AcquisitionTarget{Layout: "dir", ArtifactSHA256: strings.Repeat("0", 64), Compose: []binaryfetch.ComposeStep{
		{Role: "first", Kind: "url", Dest: "one", URL: server.URL + "/a", SHA256: digest(archiveBytesA), Archive: "tar.gz"},
		{Role: "second", Kind: "url", Dest: "two", URL: server.URL + "/b", SHA256: digest(archiveBytesB), Archive: "tar.gz"},
	}}
	first, second := t.TempDir(), t.TempDir()
	requireCompose := func(root string) string {
		t.Helper()
		if err := composeAcquisitionTarget(context.Background(), target, t.TempDir(), root, resourcedeployment.Platform{OS: "linux", Arch: "amd64"}); err != nil {
			t.Fatal(err)
		}
		digest, err := binaryfetch.TreeDigest(root)
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}
	firstDigest, secondDigest := requireCompose(first), requireCompose(second)
	if firstDigest != secondDigest {
		t.Fatalf("composed tree digest changed across runs: %s != %s", firstDigest, secondDigest)
	}
	bad := target
	bad.Compose = []binaryfetch.ComposeStep{{Role: "missing", Kind: "url", Dest: ".", URL: server.URL + "/missing", SHA256: strings.Repeat("f", 64), Archive: "tar.gz"}}
	err := composeAcquisitionTarget(context.Background(), bad, t.TempDir(), t.TempDir(), resourcedeployment.Platform{OS: "windows", Arch: "amd64"})
	var stepErr *ComposeStepError
	if !errors.As(err, &stepErr) || stepErr.Role != "missing" {
		t.Fatalf("compose error = %v", err)
	}
}

func TestDeferredResourceTargetsRecordsRuntimeFacts(t *testing.T) {
	manifest := resourceArtifactManifest{Name: "whisper"}
	manifest.ManagedService = &resourcedeployment.ManagedService{Acquisition: &binaryfetch.Acquisition{Kind: "oci-image", Targets: []binaryfetch.AcquisitionTarget{
		{When: map[string]string{"os": "linux", "arch": "amd64", "accel.backends": "has:vulkan"}, Kind: "oci-image", Image: "ghcr.io/example/whisper@sha256:" + strings.Repeat("a", 64)},
		{When: map[string]string{"os": "linux", "arch": "amd64"}, Kind: "oci-image", Image: "ghcr.io/example/whisper@sha256:" + strings.Repeat("b", 64)},
	}}}
	got := deferredResourceTargets(manifest, resourcedeployment.Platform{OS: "linux", Arch: "amd64"})
	if len(got) != 1 || got[0].TargetIndex != 0 || len(got[0].AbsentFacts) != 1 || got[0].AbsentFacts[0] != "accel.backends" {
		t.Fatalf("deferred = %#v", got)
	}
}

func TestWriteReleaseChecksumManifestIsDeterministicAndUnsigned(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "controller_linux_amd64"), []byte("controller"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service_linux_amd64"), []byte("server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "deferred-resource-targets-v1.json"), []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := updateReleaseArtifactMetadata(dir, "controller_linux_amd64", releaseArtifactMetadata{Role: "resource-controller", Provenance: "controller-build"}); err != nil {
		t.Fatal(err)
	}
	if err := updateReleaseArtifactMetadata(dir, "service_linux_amd64", releaseArtifactMetadata{Role: "managed-service", Provenance: "upstream-digest"}); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatalf("write checksum manifest: %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) == 0 || !strings.Contains(string(manifest), "controller_linux_amd64") || !strings.Contains(string(manifest), "service_linux_amd64") {
		t.Fatalf("unexpected checksum manifest: %s", manifest)
	}
	if strings.Contains(string(manifest), "deferred-resource-targets-v1.json") {
		t.Fatalf("metadata sidecar must not be signed as an artifact: %s", manifest)
	}
	if _, err := os.Stat(filepath.Join(dir, "SHA256SUMS.sig")); !os.IsNotExist(err) {
		t.Fatalf("source staging must not create a release signature, stat error: %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := resourcedeployment.LoadReleaseManifest(dir)
	if err != nil || len(parsed.Artifacts) != 2 {
		t.Fatalf("release manifest = %#v, err=%v", parsed, err)
	}
	if parsed.Artifacts[0].Role != "resource-controller" || parsed.Artifacts[1].Role != "managed-service" {
		t.Fatalf("release roles = %#v", parsed.Artifacts)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil || string(first) != string(second) {
		t.Fatalf("release manifest must be deterministic: %v", err)
	}
}

func TestReleaseArtifactPlatformRecognizesControllerSidecars(t *testing.T) {
	for _, name := range []string{"resource-postgres_linux_amd64", "resource-postgres_linux_amd64.build.json", "resource-postgres_linux_amd64.manifest.json"} {
		osName, arch := releaseArtifactPlatform(name)
		if osName != "linux" || arch != "amd64" {
			t.Errorf("releaseArtifactPlatform(%q) = (%q, %q), want linux/amd64", name, osName, arch)
		}
	}
}

func TestWriteReleaseChecksumManifestAuthenticatesDirectoryServiceTree(t *testing.T) {
	dir := t.TempDir()
	service := filepath.Join(dir, "sherpa-onnx-server_linux_amd64")
	if err := os.MkdirAll(filepath.Join(service, "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(service, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(service, "server", "sherpa-onnx-server"):   "adapter",
		filepath.Join(service, "lib", "libsherpa-onnx-c-api.so"): "sherpa",
		filepath.Join(service, "lib", "libonnxruntime.so"):       "onnx",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := updateReleaseArtifactMetadata(dir, "sherpa-onnx-server_linux_amd64", releaseArtifactMetadata{
		Role:       "managed-service",
		Provenance: "release-signed-vrooli-adapter",
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatalf("write checksum manifest: %v", err)
	}

	digest, err := binaryfetch.TreeDigest(service)
	if err != nil {
		t.Fatal(err)
	}
	checksums, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(checksums), digest+"  sherpa-onnx-server_linux_amd64") {
		t.Fatalf("directory tree digest missing from checksums: %s", checksums)
	}
	release, err := resourcedeployment.LoadReleaseManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(release.Artifacts) != 1 || release.Artifacts[0].SHA256 != digest || release.Artifacts[0].Role != "managed-service" {
		t.Fatalf("directory release artifact = %#v", release.Artifacts)
	}
}

func TestStageManagedServiceArtifactAcquiresAndVerifiesDirectoryTree(t *testing.T) {
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gz)
	entries := map[string]string{
		"server/sherpa-onnx-server":   strings.Repeat("adapter", 256),
		"lib/libsherpa-onnx-c-api.so": strings.Repeat("sherpa", 256),
		"lib/libonnxruntime.so":       strings.Repeat("onnx", 256),
	}
	for name, contents := range entries {
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	archiveSum := sha256.Sum256(archive.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()

	tree := t.TempDir()
	for name, contents := range entries {
		path := filepath.Join(tree, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	treeDigest, err := binaryfetch.TreeDigest(tree)
	if err != nil {
		t.Fatal(err)
	}

	manifest := resourceArtifactManifest{
		Name:   "sherpa-onnx",
		Driver: "managed-service",
		ManagedService: &resourcedeployment.ManagedService{
			ArtifactRole:    "managed-service",
			ProvenanceClass: "release-signed-vrooli-adapter",
			Artifact: resourcedeployment.ServiceArtifact{
				Path: "server/sherpa-onnx-server", Version: "1.13.2-vrooli.1", Layout: "dir",
				EntryPath: "server/sherpa-onnx-server", BundleArtifact: "sherpa-onnx-server_${os}_${arch}",
				SHA256ByPlatform: map[string]string{"linux-amd64": treeDigest},
			},
			Acquisition: &binaryfetch.Acquisition{
				Kind: "url",
				Targets: []binaryfetch.AcquisitionTarget{{
					When: map[string]string{"os": "linux", "arch": "amd64"}, URL: server.URL,
					SHA256: hex.EncodeToString(archiveSum[:]), ArtifactSHA256: treeDigest,
					Archive: "tar.gz", Layout: "dir", BinPath: "server/sherpa-onnx-server",
				}},
			},
		},
	}
	out := t.TempDir()
	staged, err := stageManagedServiceArtifact(context.Background(), manifest, t.TempDir(), out, resourcedeployment.Platform{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatalf("stage managed-service directory: %v", err)
	}
	if staged.File != "sherpa-onnx-server_linux_amd64" || staged.Version != "1.13.2-vrooli.1" {
		t.Fatalf("staged artifact = %#v", staged)
	}
	if _, err := os.Stat(filepath.Join(out, staged.File, "server", "sherpa-onnx-server")); err != nil {
		t.Fatalf("staged server entry missing: %v", err)
	}
	if got, err := binaryfetch.TreeDigest(filepath.Join(out, staged.File)); err != nil || got != treeDigest {
		t.Fatalf("staged tree digest = %q, %v; want %q", got, err, treeDigest)
	}
}

func TestRunVerifiesGenericReleaseManifestInDevelopmentLocalMode(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "controller_linux_amd64"), []byte("controller"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := updateReleaseArtifactMetadata(dir, "controller_linux_amd64", releaseArtifactMetadata{Role: "resource-controller", Provenance: "controller-build"}); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(dir); err != nil {
		t.Fatal(err)
	}
	if got := run([]string{"--verify-release-manifest", "--release-artifact-root", dir, "--trust-mode", "development-local"}); got != 0 {
		t.Fatalf("development-local verification exit code = %d, want 0", got)
	}
	if got := run([]string{"--verify-release-manifest", "--root", dir, "--release-artifact-root", dir, "--trust-mode", "production"}); got != 1 {
		t.Fatalf("unsigned production verification exit code = %d, want 1", got)
	}
}

func TestResourceArtifactBuildTargetsIncludeOnlyDeclaredBundledPlatforms(t *testing.T) {
	manifest := resourceArtifactManifest{Name: "fixture", Driver: "managed-service"}
	manifest.CLI.Adapter.ModuleDir = "cli"
	manifest.CLI.Distribution.Kind = "prebuilt_artifact"
	manifest.CLI.Distribution.ArtifactName = "resource-fixture_${os}_${arch}"
	manifest.Deployment.Profiles = map[string]resourcedeployment.Profile{
		"desktop": {
			Linux:   &resourcedeployment.Target{Support: "conditional", Mode: "bundled-service", Architectures: []string{"amd64", "arm64"}},
			MacOS:   &resourcedeployment.Target{Support: "unsupported", Mode: "bundled-service", Architectures: []string{"amd64"}},
			Windows: &resourcedeployment.Target{Support: "conditional", Mode: "remote-service", Architectures: []string{"amd64"}},
		},
	}
	targets, err := resourceArtifactBuildTargets(manifest)
	if err != nil {
		t.Fatalf("resourceArtifactBuildTargets: %v", err)
	}
	if len(targets) != 2 || targets[0].Platform.String() != "linux-amd64" || targets[1].Platform.String() != "linux-arm64" {
		t.Fatalf("targets = %#v", targets)
	}
	if targets[0].Artifact != "resource-fixture_linux_amd64" {
		t.Fatalf("artifact = %q", targets[0].Artifact)
	}
}

func TestResourceArtifactBuildTargetsRejectIncompleteBundledContract(t *testing.T) {
	manifest := resourceArtifactManifest{Name: "fixture"}
	manifest.CLI.Distribution.Kind = "prebuilt_artifact"
	manifest.Deployment.Profiles = map[string]resourcedeployment.Profile{
		"desktop": {Linux: &resourcedeployment.Target{Support: "conditional", Mode: "bundled-client", Architectures: []string{"amd64"}}},
	}
	if _, err := resourceArtifactBuildTargets(manifest); err == nil {
		t.Fatal("expected incomplete bundled contract to be rejected")
	}
}

func TestStageResourceTargetCachedReusesVerifiedSourcePlatformStep(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "fixture")
	moduleDir := filepath.Join(resourceDir, "cli")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), []byte(`{"name":"fixture"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	moduleSource := filepath.Join(moduleDir, "main.go")
	if err := os.WriteFile(moduleSource, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module fixture\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := resourceBuildTarget{Resource: "fixture", Platform: resourcedeployment.Platform{OS: "linux", Arch: "amd64"}, Artifact: "resource-fixture_linux_amd64", Module: "cli", Manifest: resourceArtifactManifest{Name: "fixture"}}
	cacheRoot, outputRoot := filepath.Join(root, "cache"), filepath.Join(root, "release")
	builds := 0
	builder := func(_ context.Context, _, staging string, target resourceBuildTarget) error {
		builds++
		for name, contents := range map[string]string{
			target.Artifact:                    "controller",
			target.Artifact + ".manifest.json": "manifest",
			target.Artifact + ".build.json":    "build metadata",
		} {
			if err := os.WriteFile(filepath.Join(staging, name), []byte(contents), 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	stage := func() error {
		_, err := stageResourceTargetCachedWithBuilder(context.Background(), resourceDir, outputRoot, cacheRoot, target, builder)
		return err
	}
	if err := stage(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(outputRoot, target.Artifact)); err != nil {
		t.Fatal(err)
	}
	if err := stage(); err != nil {
		t.Fatal(err)
	}
	if builds != 1 {
		t.Fatalf("build count after cache hit = %d, want one", builds)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, target.Artifact)); err != nil {
		t.Fatalf("cache hit did not restage artifact: %v", err)
	}
	if err := os.WriteFile(moduleSource, []byte("package main\n// changed source identity\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stage(); err != nil {
		t.Fatal(err)
	}
	if builds != 2 {
		t.Fatalf("build count after source change = %d, want a new source-keyed build", builds)
	}
}

func TestSplitArtifactSelectionRejectsEmptyAndDuplicateEntries(t *testing.T) {
	got, err := splitArtifactSelection("postgres, vault")
	if err != nil || len(got) != 2 || got[0] != "postgres" || got[1] != "vault" {
		t.Fatalf("selection = %#v, err=%v", got, err)
	}
	for _, input := range []string{"postgres,,vault", "postgres,postgres"} {
		if _, err := splitArtifactSelection(input); err == nil {
			t.Errorf("splitArtifactSelection(%q) unexpectedly succeeded", input)
		}
	}
}

func TestStageDocParseWASIStagesAndChecksumsArtifact(t *testing.T) {
	root := t.TempDir()
	artifactDir := filepath.Join(root, "resources", "doc-parse", "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "doc-parse.wasm"), []byte("wasi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, "doc-parse.wasm.sha256"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "release")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageDocParseWASI(root, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "doc-parse.wasm")); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseChecksumManifest(out); err != nil {
		t.Fatal(err)
	}
	checksums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	if err != nil || !strings.Contains(string(checksums), "doc-parse.wasm") {
		t.Fatalf("checksums = %s, err = %v", checksums, err)
	}
}
