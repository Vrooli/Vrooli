package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	_ "github.com/vrooli/vrooli/internal/acquisition"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/tools"
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
	Deployment     resourcedeployment.Deployment      `json:"deployment"`
	ManagedService *resourcedeployment.ManagedService `json:"managed_service,omitempty"`
}

type resourceBuildTarget struct {
	Resource string
	Platform resourcedeployment.Platform
	Artifact string
	Module   string
	Driver   string
	Manifest resourceArtifactManifest
}

type stagedManagedServiceArtifact struct {
	Version string
	File    string
}

// stageToolArtifacts uses the same acquisition contract as managed services.
// A release stager has only the target platform facts, so any target requiring
// a runtime fact is rejected instead of producing a bundle that cannot be
// reproduced on a clean machine.
func stageToolArtifacts(ctx context.Context, root, outDir string) error {
	var manifests []string
	if err := fs.WalkDir(tools.Manifests, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "tool.json" {
			manifests = append(manifests, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("walk tool manifests: %w", err)
	}
	sort.Strings(manifests)
	var index []string
	for _, manifestPath := range manifests {
		data, err := fs.ReadFile(tools.Manifests, manifestPath)
		if err != nil {
			return err
		}
		var manifest hostreqkit.ToolManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("parse tool manifest %s: %w", manifestPath, err)
		}
		if manifest.Bundling != "vendorable" || manifest.Acquisition == nil {
			continue
		}
		for _, platform := range []resourcedeployment.Platform{{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64"}, {OS: "macos", Arch: "amd64"}, {OS: "macos", Arch: "arm64"}, {OS: "windows", Arch: "amd64"}} {
			factsOS := artifactOS(platform.OS)
			target, err := manifest.Acquisition.Resolve(binaryfetch.Facts{"os": factsOS, "arch": platform.Arch})
			if err != nil {
				var noMatch *binaryfetch.NoMatchingTargetError
				var unsupported *binaryfetch.UnsupportedError
				if errors.As(err, &noMatch) || errors.As(err, &unsupported) {
					continue
				}
				return fmt.Errorf("tool %s %s: %w", manifest.Name, platform, err)
			}
			if target.Unsupported != "" {
				continue
			}
			if !binaryfetch.UsesOnlyBuildTimeFacts(target) {
				return fmt.Errorf("tool %s/%s acquisition uses runtime facts and cannot be vendored", manifest.Name, platform)
			}
			name := toolArtifactName(manifest.Name, platform)
			path := filepath.Join(outDir, name)
			if err := stageAcquisitionTarget(ctx, manifest.Acquisition.Kind, target, platform, name, path, outDir); err != nil {
				return fmt.Errorf("stage tool %s for %s: %w", manifest.Name, platform, err)
			}
			version := manifest.Version
			if version == "" {
				version = "declared"
			}
			index = append(index, strings.Join([]string{manifest.Name, version, artifactOS(platform.OS), platform.Arch, name}, "\t"))
		}
	}
	sort.Strings(index)
	return os.WriteFile(filepath.Join(outDir, "tool-artifacts-v1.txt"), []byte(strings.Join(index, "\n")+"\n"), 0o644)
}

func stageAcquisitionTarget(ctx context.Context, declaredKind string, target binaryfetch.AcquisitionTarget, platform resourcedeployment.Platform, name, path, outDir string) error {
	kind := strings.ToLower(strings.TrimSpace(declaredKind))
	if kind == "" {
		kind = "url"
	}
	layout := strings.ToLower(strings.TrimSpace(target.Layout))
	if kind == "oci-image" {
		if layout == "dir" {
			_, err := binaryfetch.FetchOCI(ctx, target, path, nil)
			return err
		}
		_, err := binaryfetch.FetchOCIFileForPlatform(ctx, target, path, artifactOS(platform.OS), platform.Arch, nil)
		return err
	}
	spec := binaryfetch.Target{Name: name, URL: target.URL, SHA256: target.SHA256, Archive: target.Archive, Layout: layout, BinPath: target.BinPath, Mode: target.Mode}
	if layout == "dir" {
		_, err := binaryfetch.FetchDir(ctx, spec, path, nil)
		return err
	}
	_, err := binaryfetch.Fetch(ctx, spec, outDir, nil)
	return err
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
		staged, err := stageManagedServiceArtifact(ctx, target.Manifest, outDir, target.Platform)
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
		if entry.Name() == "SHA256SUMS" || entry.Name() == "SHA256SUMS.sig" || entry.Name() == "release-manifest.json" || entry.Name() == "release-manifest.sig.json" {
			continue
		}
		if (!entry.Type().IsRegular() && !entry.IsDir()) || !resourcedeployment.IsSafeArtifactName(entry.Name()) {
			return fmt.Errorf("release artifact directory contains unsafe entry %q", entry.Name())
		}
		artifactPath := filepath.Join(outDir, entry.Name())
		var digest string
		if entry.IsDir() {
			digest, err = binaryfetch.TreeDigest(artifactPath)
		} else {
			data, readErr := os.ReadFile(artifactPath)
			err = readErr
			if err == nil {
				sum := sha256.Sum256(data)
				digest = fmt.Sprintf("%x", sum)
			}
		}
		if err != nil {
			return fmt.Errorf("hash release artifact %s: %w", entry.Name(), err)
		}
		lines = append(lines, fmt.Sprintf("%s  %s", digest, entry.Name()))
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
			targets = append(targets, resourceBuildTarget{Resource: manifest.Name, Platform: concrete, Artifact: artifact, Module: manifest.CLI.Adapter.ModuleDir, Driver: manifest.Driver, Manifest: manifest})
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

func stageManagedServiceArtifact(ctx context.Context, manifest resourceArtifactManifest, outDir string, platform resourcedeployment.Platform) (stagedManagedServiceArtifact, error) {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return stagedManagedServiceArtifact{}, fmt.Errorf("managed-service %s must declare acquisition", manifest.Name)
	}
	facts := binaryfetch.Facts{"os": platform.OS, "arch": platform.Arch}
	target, err := manifest.ManagedService.Acquisition.Resolve(facts)
	if err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	if !binaryfetch.UsesOnlyBuildTimeFacts(target) {
		return stagedManagedServiceArtifact{}, fmt.Errorf("acquisition target for %s/%s uses runtime facts and cannot be vendored", manifest.Name, platform)
	}
	if target.Unsupported != "" {
		return stagedManagedServiceArtifact{}, fmt.Errorf("acquisition target for %s/%s is unsupported: %s", manifest.Name, platform, target.Unsupported)
	}
	artifact, err := manifest.ManagedService.Artifact.ForPlatform(platform.OS, platform.Arch)
	if err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	if target.ArtifactSHA256 != "" {
		artifact.SHA256 = target.ArtifactSHA256
	}
	if target.Layout != "" {
		artifact.Layout = target.Layout
	}
	layout := strings.ToLower(strings.TrimSpace(target.Layout))
	if layout == "" {
		layout = strings.ToLower(strings.TrimSpace(artifact.Layout))
	}
	// For file-layout archives, bin_path locates the executable inside the
	// downloaded archive and Fetch extracts it to the staged file. It is not a
	// launch-time entry path; only a directory artifact retains EntryPath.
	if layout == "dir" && target.BinPath != "" {
		artifact.EntryPath = strings.TrimPrefix(filepath.ToSlash(target.BinPath), "/")
	}
	name, err := artifact.BundleArtifactForPlatform(platform.OS, platform.Arch)
	if err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	if err := verifyStagerProvenance(ctx, manifest.ManagedService.Acquisition.Provenance); err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	path := filepath.Join(outDir, name)
	if strings.EqualFold(strings.TrimSpace(manifest.ManagedService.Acquisition.Kind), "oci-image") {
		if layout == "dir" {
			if _, err := binaryfetch.FetchOCIForPlatform(ctx, target, path, artifactOS(platform.OS), platform.Arch, nil); err != nil {
				return stagedManagedServiceArtifact{}, err
			}
		} else {
			if _, err := binaryfetch.FetchOCIFile(ctx, target, path, nil); err != nil {
				return stagedManagedServiceArtifact{}, err
			}
		}
	} else {
		spec := binaryfetch.Target{Name: name, URL: target.URL, SHA256: target.SHA256, Archive: target.Archive, Layout: layout, BinPath: target.BinPath, Mode: target.Mode}
		if layout == "dir" {
			if _, err := binaryfetch.FetchDir(ctx, spec, path, nil); err != nil {
				return stagedManagedServiceArtifact{}, err
			}
		} else if _, err := binaryfetch.Fetch(ctx, spec, outDir, nil); err != nil {
			return stagedManagedServiceArtifact{}, err
		}
	}
	if err := artifact.VerifyFile(path); err != nil {
		return stagedManagedServiceArtifact{}, fmt.Errorf("verify staged %s: %w", manifest.Name, err)
	}
	return stagedManagedServiceArtifact{Version: artifact.Version, File: name}, nil
}

func verifyStagerProvenance(ctx context.Context, declaration *binaryfetch.Provenance) error {
	if declaration == nil || strings.EqualFold(strings.TrimSpace(declaration.Kind), "none") {
		return nil
	}
	dir, err := os.MkdirTemp("", "vrooli-dist-provenance-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	keyPath := filepath.Join(dir, "release-key.asc")
	manifestPath := filepath.Join(dir, "SHA256SUMS")
	signaturePath := filepath.Join(dir, "SHA256SUMS.sig")
	for _, item := range []struct{ url, path string }{{declaration.KeyURL, keyPath}, {declaration.ChecksumManifest, manifestPath}, {declaration.ChecksumSignature, signaturePath}} {
		if err := binaryfetch.Download(ctx, item.url, item.path, nil); err != nil {
			return err
		}
	}
	_, err = binaryfetch.VerifyProvenance(ctx, declaration, keyPath, manifestPath, signaturePath)
	return err
}

func artifactOS(platform string) string {
	if platform == "macos" {
		return "darwin"
	}
	return platform
}

func toolArtifactName(name string, platform resourcedeployment.Platform) string {
	suffix := ""
	if platform.OS == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("tool_%s_%s_%s%s", name, artifactOS(platform.OS), platform.Arch, suffix)
}
