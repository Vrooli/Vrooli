package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// resourceArtifactManifest contains only the release-facing parts of a
// resource contract. Resource deployment devices receive these already-built
// and release-signed artifacts; this command is the sole source-build path.
type resourceArtifactManifest struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	CLI    struct {
		Adapter struct {
			ModuleDir string `json:"module_dir"`
		} `json:"adapter"`
		Distribution struct {
			Kind         string `json:"kind"`
			ArtifactName string `json:"artifact_name"`
		} `json:"distribution"`
	} `json:"cli"`
	Deployment resourcedeployment.Deployment `json:"deployment"`
}

type resourceBuildTarget struct {
	Resource string
	Platform resourcedeployment.Platform
	Artifact string
	Module   string
	Driver   string
}

type stagedManagedServiceArtifact struct {
	Version string
	File    string
}

type managedServiceArtifactStager func(context.Context, string, string, resourcedeployment.Platform) (stagedManagedServiceArtifact, error)

var managedServiceArtifactStagers = map[string]managedServiceArtifactStager{}

func registerManagedServiceArtifactStager(resource string, stager managedServiceArtifactStager) {
	if strings.TrimSpace(resource) == "" || stager == nil {
		panic("managed-service artifact stager requires a resource and implementation")
	}
	if _, exists := managedServiceArtifactStagers[resource]; exists {
		panic("duplicate managed-service artifact stager for " + resource)
	}
	managedServiceArtifactStagers[resource] = stager
}

// stageResourceArtifacts builds each controller that a non-unsupported bundled
// profile declares, writes its immutable companion metadata, and stages the
// separately pinned server artifact for managed services. It deliberately uses
// direct exec argv rather than a resource-local shell script.
func stageResourceArtifacts(ctx context.Context, root, outDir string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	manifests, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		return err
	}
	sort.Strings(manifests)
	var targets []resourceBuildTarget
	for _, path := range manifests {
		manifest, err := readResourceArtifactManifest(path)
		if err != nil {
			return err
		}
		resourceTargets, err := resourceArtifactBuildTargets(manifest)
		if err != nil {
			return fmt.Errorf("resource %s: %w", manifest.Name, err)
		}
		targets = append(targets, resourceTargets...)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create resource artifact output: %w", err)
	}
	index := make([]string, 0)
	for _, target := range targets {
		resourceDir := filepath.Join(root, "resources", target.Resource)
		if err := buildResourceController(ctx, resourceDir, target, outDir); err != nil {
			return err
		}
		if target.Driver != "managed-service" {
			continue
		}
		stager, ok := managedServiceArtifactStagers[target.Resource]
		if !ok {
			return fmt.Errorf("managed-service %s has no registered release server stager", target.Resource)
		}
		staged, err := stager(ctx, root, outDir, target.Platform)
		if err != nil {
			return fmt.Errorf("stage managed-service %s for %s: %w", target.Resource, target.Platform, err)
		}
		index = append(index, strings.Join([]string{target.Resource, staged.Version, artifactOS(target.Platform.OS), target.Platform.Arch, staged.File}, "\t"))
	}
	sort.Strings(index)
	return os.WriteFile(filepath.Join(outDir, "resource-artifacts-v1.txt"), []byte(strings.Join(index, "\n")+"\n"), 0o644)
}

// writeReleaseChecksumManifest records the immutable bytes that a release
// signer must authorize. It intentionally does not create a signature: signing
// authority is kept outside source builds, and consumers reject this directory
// until SHA256SUMS.sig is supplied by that authority.
func writeReleaseChecksumManifest(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("read release artifact directory: %w", err)
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "SHA256SUMS" || entry.Name() == "SHA256SUMS.sig" || entry.Name() == "release-manifest.json" || entry.Name() == "release-manifest.sig.json" {
			continue
		}
		if !entry.Type().IsRegular() || !resourcedeployment.IsSafeArtifactName(entry.Name()) {
			return fmt.Errorf("release artifact directory contains unsafe entry %q", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(outDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read release artifact %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(data)
		lines = append(lines, fmt.Sprintf("%x  %s", sum, entry.Name()))
	}
	if len(lines) == 0 {
		return fmt.Errorf("release artifact directory contains no artifacts")
	}
	sort.Strings(lines)
	if err := os.WriteFile(filepath.Join(outDir, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	artifacts := make([]resourcedeployment.ReleaseArtifact, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		name := fields[1]
		role := "vendored-tool"
		if strings.HasPrefix(name, "resource-") {
			role = "resource-controller"
		}
		if strings.HasPrefix(name, "vault_") {
			role = "managed-service"
		}
		osName, arch := releaseArtifactPlatform(name)
		provenance := "verified-by-vrooli-stager"
		if strings.HasPrefix(name, "vault_") {
			provenance = "hashicorp-checksum-signature-verified"
		}
		artifacts = append(artifacts, resourcedeployment.ReleaseArtifact{Name: name, SHA256: fields[0], Role: role, OS: osName, Arch: arch, UpstreamProvenance: provenance})
	}
	canonical, err := (resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: artifacts}).CanonicalBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "release-manifest.json"), append(canonical, '\n'), 0o644)
}

func releaseArtifactPlatform(name string) (string, string) {
	stem := strings.TrimSuffix(name, ".exe")
	parts := strings.Split(stem, "_")
	if len(parts) < 3 {
		return "", ""
	}
	osName, arch := parts[len(parts)-2], parts[len(parts)-1]
	if osName != "linux" && osName != "darwin" && osName != "windows" {
		return "", ""
	}
	return osName, arch
}

func readResourceArtifactManifest(path string) (resourceArtifactManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceArtifactManifest{}, err
	}
	var manifest resourceArtifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return resourceArtifactManifest{}, fmt.Errorf("parse resource manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return resourceArtifactManifest{}, fmt.Errorf("resource manifest has no name")
	}
	return manifest, nil
}

func resourceArtifactBuildTargets(manifest resourceArtifactManifest) ([]resourceBuildTarget, error) {
	if manifest.CLI.Distribution.Kind == "" {
		return nil, nil
	}
	if manifest.CLI.Distribution.Kind != "prebuilt_artifact" || strings.TrimSpace(manifest.CLI.Distribution.ArtifactName) == "" || strings.TrimSpace(manifest.CLI.Adapter.ModuleDir) == "" {
		return nil, fmt.Errorf("bundled resource artifacts require prebuilt distribution and Go module adapter")
	}
	var targets []resourceBuildTarget
	for _, platform := range []string{"linux", "macos", "windows"} {
		target, found := manifest.Deployment.Target("desktop", platform, "")
		if !found || target.Support == "unsupported" || !strings.HasPrefix(target.Mode, "bundled-") {
			continue
		}
		for _, arch := range target.Architectures {
			concrete, err := resourcedeployment.ParsePlatform(platform + "-" + arch)
			if err != nil {
				return nil, err
			}
			artifact, err := resourcedeployment.ArtifactName(manifest.CLI.Distribution.ArtifactName, concrete.OS, concrete.Arch)
			if err != nil {
				return nil, err
			}
			targets = append(targets, resourceBuildTarget{Resource: manifest.Name, Platform: concrete, Artifact: artifact, Module: manifest.CLI.Adapter.ModuleDir, Driver: manifest.Driver})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Platform.String() < targets[j].Platform.String() })
	return targets, nil
}

func buildResourceController(ctx context.Context, resourceDir string, target resourceBuildTarget, outDir string) error {
	goos := artifactOS(target.Platform.OS)
	command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", filepath.Join(outDir, target.Artifact), ".")
	command.Dir = filepath.Join(resourceDir, filepath.FromSlash(target.Module))
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+target.Platform.Arch)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build resource %s for %s: %w", target.Resource, target.Platform, err)
	}
	manifestData, err := os.ReadFile(filepath.Join(resourceDir, "resource.json"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, target.Artifact+".manifest.json"), manifestData, 0o644); err != nil {
		return err
	}
	metadata, err := json.Marshal(struct {
		SchemaVersion string `json:"schema_version"`
		Resource      string `json:"resource"`
		Artifact      string `json:"artifact"`
		OS            string `json:"os"`
		Arch          string `json:"arch"`
		Build         struct {
			Tool       string `json:"tool"`
			CGOEnabled bool   `json:"cgo_enabled"`
			TrimPath   bool   `json:"trimpath"`
			BuildVCS   bool   `json:"buildvcs"`
		} `json:"build"`
	}{SchemaVersion: "v1", Resource: target.Resource, Artifact: target.Artifact, OS: goos, Arch: target.Platform.Arch})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, target.Artifact+".build.json"), append(metadata, '\n'), 0o644)
}

func artifactOS(platform string) string {
	if platform == "macos" {
		return "darwin"
	}
	return platform
}
