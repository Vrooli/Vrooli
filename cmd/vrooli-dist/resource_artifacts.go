package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/envkit-go"
	_ "github.com/vrooli/vrooli/internal/artifactcodecs" // register the caller-owned tar.zst archive decoder
	"github.com/vrooli/vrooli/internal/hostreqkit"
	resourceacquisition "github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tools"
	rendererprofile "github.com/vrooli/vrooli/internal/tools/renderer-profile"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	desktopmanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	desktopprofiles "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/toolprofiles"
)

const (
	literalResourceArtifactsDir            = "dir"
	literalResourceArtifactsWindows        = "windows"
	literalResourceArtifactsPrebuilt       = "prebuilt_artifact"
	literalResourceArtifactsManagedService = "managed-service"
)

const (
	mndResourceArtifactsNumberOctal644 = 0o644
	mndResourceArtifactsNumberOctal755 = 0o755
	resourceCoordinatePartCount        = 3
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
	Version  string
	File     string
	Deferred []deferredResourceTargetRecord
}

type deferredResourceTargetRecord struct {
	Resource     string            `json:"resource"`
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	TargetIndex  int               `json:"target_index"`
	When         map[string]string `json:"when"`
	AbsentFacts  []string          `json:"absent_facts"`
}

type releaseArtifactMetadata struct {
	Role       string `json:"role"`
	Provenance string `json:"provenance"`
}

const releaseArtifactMetadataFile = "release-artifact-metadata-v1.json"

const ociTreeNormalizationCacheRevision = "oci-empty-directory-links-v1"

// stageToolArtifacts uses the same acquisition contract as managed services.
// A release stager has only the target platform facts, so any target requiring
// a runtime fact is rejected instead of producing a bundle that cannot be
// reproduced on a clean machine.
func stageToolArtifacts(ctx context.Context, root, outDir string) error {
	if err := os.MkdirAll(outDir, mndResourceArtifactsNumberOctal755); err != nil {
		return fmt.Errorf("create tool artifact output: %w", err)
	}
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
		for _, platform := range []resourcedeployment.Platform{{OS: "linux", Arch: "amd64"}, {OS: "linux", Arch: "arm64"}, {OS: "macos", Arch: "amd64"}, {OS: "macos", Arch: "arm64"}, {OS: literalResourceArtifactsWindows, Arch: "amd64"}} {
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
	return os.WriteFile(filepath.Join(outDir, "tool-artifacts-v1.txt"), []byte(strings.Join(index, "\n")+"\n"), mndResourceArtifactsNumberOctal644)
}

const rendererProfilesFile = "renderer_profiles_linux_amd64.json"

func stageManagedRendererProfiles(ctx context.Context, selectors []string, outDir string) error {
	if len(selectors) == 0 {
		return errors.New("renderer profile selection is empty")
	}
	if err := os.MkdirAll(outDir, mndResourceArtifactsNumberOctal755); err != nil {
		return fmt.Errorf("create renderer release directory: %w", err)
	}
	declarations := make([]desktopmanifest.ToolProfilePackage, 0, len(selectors))
	declarationBySelector := make(map[string]desktopmanifest.ToolProfilePackage, len(selectors))
	seen := make(map[string]bool, len(selectors))
	for _, selector := range selectors {
		declaration, _, ok := desktopprofiles.PinnedDeclaration(selector)
		if !ok {
			return fmt.Errorf("unsupported renderer profile %q", selector)
		}
		if seen[selector] {
			return fmt.Errorf("duplicate renderer profile %q", selector)
		}
		seen[selector] = true
		declarations = append(declarations, declaration)
		declarationBySelector[selector] = declaration
	}
	if _, err := os.Lstat(filepath.Join(outDir, rendererProfilesFile)); err == nil {
		return fmt.Errorf("renderer profile declaration already exists: %s", rendererProfilesFile)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, selector := range selectors {
		if _, err := os.Lstat(filepath.Join(outDir, rendererArtifactName(selector))); err == nil {
			return fmt.Errorf("renderer package already exists for %q", selector)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := desktopmanifest.ValidateToolProfiles(declarations); err != nil {
		return fmt.Errorf("validate renderer release declarations: %w", err)
	}
	sort.Slice(declarations, func(i, j int) bool { return declarations[i].ToolID < declarations[j].ToolID })
	for _, selector := range selectors {
		declaration := declarationBySelector[selector]
		err := rendererprofile.WithManagedTree(ctx, selector, func(source rendererprofile.ManagedTree) error {
			if source.ProfileID != declaration.ProfileID || source.OS != declaration.OS || source.Architecture != declaration.Architecture || source.TreeSHA256 != declaration.TreeSHA256 {
				return fmt.Errorf("renderer owner identity differs from pinned release declaration for %q", selector)
			}
			return stageVerifiedRendererTree(ctx, source, outDir)
		})
		if err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(declarations, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(outDir, rendererProfilesFile)
	if err := writeFileExclusive(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write renderer profile declaration: %w", err)
	}
	if err := updateReleaseArtifactMetadata(outDir, rendererProfilesFile, releaseArtifactMetadata{Role: "renderer-profile-index", Provenance: "verified-managed-tree-declarations; development-local; non-promotable"}); err != nil {
		return err
	}
	return nil
}

func writeFileExclusive(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func rendererArtifactName(selector string) string {
	_, toolName, _ := desktopprofiles.PinnedDeclaration(selector)
	return "tool_" + toolName + "_linux_amd64"
}

func stageVerifiedRendererTree(ctx context.Context, source rendererprofile.ManagedTree, outDir string) error {
	temp, err := os.MkdirTemp(outDir, ".renderer-profile-stage-")
	if err != nil {
		return fmt.Errorf("create renderer staging directory: %w", err)
	}
	defer os.RemoveAll(temp)
	payload := filepath.Join(temp, "payload")
	if err := copyPinnedTree(ctx, source.SourceRoot, payload); err != nil {
		return err
	}
	digest, err := binaryfetch.TreeDigestContext(ctx, payload)
	if err != nil {
		return err
	}
	if digest != source.TreeSHA256 {
		return fmt.Errorf("staged renderer tree %q failed whole-tree verification", source.Selector)
	}
	destination := filepath.Join(outDir, source.ArtifactName)
	if err := publishRendererDirectory(payload, destination); err != nil {
		return fmt.Errorf("publish renderer package %q without overwrite: %w", source.Selector, err)
	}
	return updateReleaseArtifactMetadata(outDir, source.ArtifactName, releaseArtifactMetadata{Role: source.Role, Provenance: source.Provenance})
}

func copyPinnedTree(ctx context.Context, source, destination string) error {
	if err := os.Mkdir(destination, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.Mkdir(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("renderer tree contains unsupported entry %q", rel)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		inErr := in.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if inErr != nil {
			return inErr
		}
		return closeErr
	})
}

func stageAcquisitionTarget(ctx context.Context, declaredKind string, target binaryfetch.AcquisitionTarget, platform resourcedeployment.Platform, name, path, outDir string) error {
	kind := strings.ToLower(strings.TrimSpace(declaredKind))
	if kind == "" {
		kind = "url"
	}
	layout := strings.ToLower(strings.TrimSpace(target.Layout))
	if kind == "oci-image" {
		if layout == literalResourceArtifactsDir {
			_, err := binaryfetch.FetchOCIForPlatform(ctx, target, path, artifactOS(platform.OS), platform.Arch, nil)
			return err
		}
		_, err := binaryfetch.FetchOCIFileForPlatform(ctx, target, path, artifactOS(platform.OS), platform.Arch, nil)
		return err
	}
	spec := binaryfetch.Target{Name: name, URL: target.URL, SHA256: target.SHA256, Archive: target.Archive, Layout: layout, BinPath: target.BinPath, Mode: target.Mode}
	if layout == literalResourceArtifactsDir {
		_, err := binaryfetch.FetchDir(ctx, spec, path, nil)
		return err
	}
	_, err := binaryfetch.Fetch(ctx, spec, outDir, nil)
	return err
}

// stageResourceArtifactsSelected composes a release from independently cached
// resource/platform steps. Empty filters retain full-release behavior; desktop
// callers pass only the resources and platforms in their deployment plan.
func stageResourceArtifactsSelected(ctx context.Context, root, outDir string, selectedResources, selectedPlatforms []string) error {
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
	resourceSeen := make(map[string]bool)
	resourceTargetSeen := make(map[string]bool)
	platforms := make(map[string]bool, len(selectedPlatforms))
	for _, platform := range selectedPlatforms {
		parsed, err := resourcedeployment.ParsePlatform(platform)
		if err != nil {
			return err
		}
		platforms[parsed.String()] = true
	}
	resources := make(map[string]bool, len(selectedResources))
	for _, resource := range selectedResources {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			return fmt.Errorf("resource filter contains an empty name")
		}
		resources[resource] = true
	}
	for _, path := range manifests {
		manifest, err := readResourceArtifactManifest(path)
		if err != nil {
			return err
		}
		resourceTargets, err := resourceArtifactBuildTargets(manifest)
		if err != nil {
			return fmt.Errorf("resource %s: %w", manifest.Name, err)
		}
		if len(resources) > 0 && !resources[manifest.Name] {
			continue
		}
		resourceSeen[manifest.Name] = true
		for _, target := range resourceTargets {
			if len(platforms) > 0 && !platforms[target.Platform.String()] {
				continue
			}
			resourceTargetSeen[manifest.Name] = true
			targets = append(targets, target)
		}
	}
	for resource := range resources {
		if !resourceSeen[resource] {
			return fmt.Errorf("selected resource %q has no manifest", resource)
		}
		if !resourceTargetSeen[resource] && resource != "doc-parse" {
			return fmt.Errorf("selected resource %q has no bundled artifact for requested platform set", resource)
		}
	}
	if err := os.MkdirAll(outDir, mndResourceArtifactsNumberOctal755); err != nil {
		return fmt.Errorf("create resource artifact output: %w", err)
	}
	index, err := readExistingArtifactIndex(outDir)
	if err != nil {
		return err
	}
	deferredTargets, err := readExistingDeferredTargets(outDir)
	if err != nil {
		return err
	}
	var stageErrors []error
	cacheRoot, err := resourceArtifactCacheRoot()
	if err != nil {
		return err
	}
	for _, target := range targets {
		resourceDir := filepath.Join(root, "resources", target.Resource)
		staged, err := stageResourceTargetCached(ctx, resourceDir, outDir, cacheRoot, target)
		if err != nil {
			stageErrors = append(stageErrors, err)
			continue
		}
		if target.Driver != literalResourceArtifactsManagedService {
			continue
		}
		index = append(index, strings.Join([]string{target.Resource, staged.Version, artifactOS(target.Platform.OS), target.Platform.Arch, staged.File}, "\t"))
		deferredTargets = append(deferredTargets, staged.Deferred...)
	}
	docParseSelected := len(resources) == 0 || resources["doc-parse"]
	docParseStaged := false
	if docParseSelected {
		if _, err := os.Stat(filepath.Join(root, "resources", "doc-parse", "resource.json")); err == nil {
			if err := stageDocParseWASI(root, outDir); err != nil {
				stageErrors = append(stageErrors, fmt.Errorf("stage doc-parse WASI artifact: %w", err))
			} else {
				docParseStaged = true
			}
		}
	}
	sort.Strings(index)
	index = uniqueStrings(index)
	if docParseStaged {
		index = append(index, "doc-parse\tdeclared\tall\tall\tdoc-parse.wasm")
		sort.Strings(index)
	}
	if data, err := json.MarshalIndent(deferredTargets, "", "  "); err != nil {
		stageErrors = append(stageErrors, fmt.Errorf("encode deferred resource targets: %w", err))
	} else if err := os.WriteFile(filepath.Join(outDir, "deferred-resource-targets-v1.json"), append(data, '\n'), mndResourceArtifactsNumberOctal644); err != nil {
		stageErrors = append(stageErrors, fmt.Errorf("write deferred resource targets: %w", err))
	}
	if err := os.WriteFile(filepath.Join(outDir, "resource-artifacts-v1.txt"), []byte(strings.Join(index, "\n")+"\n"), mndResourceArtifactsNumberOctal644); err != nil {
		stageErrors = append(stageErrors, fmt.Errorf("write resource artifact index: %w", err))
	}
	return errors.Join(stageErrors...)
}

func readExistingArtifactIndex(outDir string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(outDir, "resource-artifacts-v1.txt"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read prior resource artifact index: %w", err)
	}
	var entries []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			entries = append(entries, line)
		}
	}
	return entries, nil
}

func readExistingDeferredTargets(outDir string) ([]deferredResourceTargetRecord, error) {
	data, err := os.ReadFile(filepath.Join(outDir, "deferred-resource-targets-v1.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read prior deferred resource targets: %w", err)
	}
	var entries []deferredResourceTargetRecord
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse prior deferred resource targets: %w", err)
	}
	return entries, nil
}

func uniqueStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func resourceArtifactCacheRoot() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}
	return filepath.Join(cacheDir, "vrooli", "resource-artifacts-v1"), nil
}

func stageResourceTargetCached(ctx context.Context, resourceDir, outDir, cacheRoot string, target resourceBuildTarget) (stagedManagedServiceArtifact, error) {
	return stageResourceTargetCachedWithBuilder(ctx, resourceDir, outDir, cacheRoot, target, func(ctx context.Context, resourceRoot, staging string, target resourceBuildTarget) error {
		if err := buildResourceController(ctx, resourceRoot, target, staging); err != nil {
			return err
		}
		if target.Driver != literalResourceArtifactsManagedService {
			return nil
		}
		_, err := stageManagedServiceArtifact(ctx, target.Manifest, resourceRoot, staging, target.Platform)
		return err
	})
}

func stageResourceTargetCachedWithBuilder(ctx context.Context, resourceDir, outDir, cacheRoot string, target resourceBuildTarget, builder func(context.Context, string, string, resourceBuildTarget) error) (stagedManagedServiceArtifact, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(resourceDir, "resource.json"))
	if err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	moduleDir := filepath.Join(resourceDir, filepath.FromSlash(target.Module))
	moduleDigest, err := binaryfetch.TreeDigest(moduleDir)
	if err != nil {
		return stagedManagedServiceArtifact{}, fmt.Errorf("hash %s build inputs: %w", target.Resource, err)
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	sourceIdentityParts := []string{fmt.Sprintf("manifest=%x", manifestDigest), "module=" + moduleDigest}
	if target.Manifest.ManagedService != nil && target.Manifest.ManagedService.Acquisition != nil {
		if strings.EqualFold(strings.TrimSpace(target.Manifest.ManagedService.Acquisition.Kind), "oci-image") {
			// Invalidate OCI directory artifacts staged before empty-directory
			// aliases were pruned from filesystem trees.
			sourceIdentityParts = append(sourceIdentityParts, "tree-normalization="+ociTreeNormalizationCacheRevision)
		}
		facts := binaryfetch.Facts{"os": target.Platform.OS, "arch": target.Platform.Arch}
		acquisitionTarget, resolveErr := target.Manifest.ManagedService.Acquisition.Resolve(facts)
		if resolveErr != nil {
			return stagedManagedServiceArtifact{}, fmt.Errorf("resolve cache inputs for %s/%s: %w", target.Resource, target.Platform, resolveErr)
		}
		for _, step := range acquisitionTarget.Compose {
			if strings.TrimSpace(step.Lockfile) == "" {
				continue
			}
			lockDigest, hashErr := fileSHA256(filepath.Join(resourceDir, filepath.FromSlash(step.Lockfile)))
			if hashErr != nil {
				return stagedManagedServiceArtifact{}, fmt.Errorf("hash %s lockfile %s: %w", target.Resource, step.Lockfile, hashErr)
			}
			sourceIdentityParts = append(sourceIdentityParts, "lockfile="+step.Lockfile+":"+lockDigest)
		}
	}
	sourceDigest := strings.Join(sourceIdentityParts, "|")
	version := "source-" + moduleDigest[:16]
	if target.Manifest.ManagedService != nil && strings.TrimSpace(target.Manifest.ManagedService.Artifact.Version) != "" {
		version = target.Manifest.ManagedService.Artifact.Version
	}
	key := resourcedeployment.ArtifactCacheKey{Resource: target.Resource, Version: version, Platform: target.Platform, SourceIdentity: sourceDigest}
	_, err = resourcedeployment.StageCachedArtifacts(ctx, cacheRoot, outDir, key, func(ctx context.Context, staging string) error {
		return builder(ctx, resourceDir, staging, target)
	})
	if err != nil {
		return stagedManagedServiceArtifact{}, fmt.Errorf("stage resource %s for %s: %w", target.Resource, target.Platform, err)
	}
	var staged stagedManagedServiceArtifact
	if target.Driver == literalResourceArtifactsManagedService {
		artifactName, artifactErr := target.Manifest.ManagedService.Artifact.BundleArtifactForPlatform(target.Platform.OS, target.Platform.Arch)
		if artifactErr != nil {
			return stagedManagedServiceArtifact{}, artifactErr
		}
		staged = stagedManagedServiceArtifact{Version: target.Manifest.ManagedService.Artifact.Version, File: artifactName, Deferred: deferredResourceTargets(target.Manifest, target.Platform)}
		if err := updateReleaseArtifactMetadata(outDir, staged.File, releaseArtifactMetadata{Role: target.Manifest.ManagedService.ArtifactRole, Provenance: target.Manifest.ManagedService.ProvenanceClass}); err != nil {
			return stagedManagedServiceArtifact{}, err
		}
	}
	return staged, nil
}

func stageDocParseWASI(root, outDir string) error {
	resourceRoot := filepath.Join(root, "resources", "doc-parse")
	source := filepath.Join(resourceRoot, "artifacts", "doc-parse.wasm")
	sidecar := source + ".sha256"
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("doc-parse WASI artifact is missing; run resources/doc-parse/build/build.sh first: %w", err)
	}
	if _, err := os.Stat(sidecar); err != nil {
		return fmt.Errorf("doc-parse WASI artifact checksum is missing: %w", err)
	}
	if err := copyFile(source, filepath.Join(outDir, "doc-parse.wasm"), mndResourceArtifactsNumberOctal755); err != nil {
		return fmt.Errorf("stage doc-parse WASI artifact: %w", err)
	}
	if err := copyFile(sidecar, filepath.Join(outDir, "doc-parse.wasm.sha256"), mndResourceArtifactsNumberOctal644); err != nil {
		return fmt.Errorf("stage doc-parse WASI checksum: %w", err)
	}
	if err := updateReleaseArtifactMetadata(outDir, "doc-parse.wasm", releaseArtifactMetadata{Role: "resource-data", Provenance: "vrooli-rust-build"}); err != nil {
		return err
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destination, data, mode); err != nil {
		return err
	}
	return os.Chmod(destination, mode)
}

// writeReleaseChecksumManifest records the immutable bytes that a release
// signer must authorize. It intentionally does not create a signature: signing
// authority is kept outside source builds, and consumers reject this directory
// until SHA256SUMS.sig is supplied by that authority.
//
//nolint:gocyclo // release checksum emission handles deterministic traversal and independent file failures.
func writeReleaseChecksumManifest(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return fmt.Errorf("read release artifact directory: %w", err)
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Name() == "SHA256SUMS" || entry.Name() == "SHA256SUMS.sig" || entry.Name() == "release-manifest.json" || entry.Name() == "release-manifest.sig.json" || entry.Name() == releaseArtifactMetadataFile || entry.Name() == "resource-artifacts-v1.txt" || entry.Name() == "tool-artifacts-v1.txt" || entry.Name() == "deferred-resource-targets-v1.json" {
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
	if err := os.WriteFile(filepath.Join(outDir, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), mndResourceArtifactsNumberOctal644); err != nil {
		return err
	}
	artifacts := make([]resourcedeployment.ReleaseArtifact, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		name := fields[1]
		role := "vendored-tool"
		provenance := "verified-by-vrooli-stager"
		if metadata, ok := readReleaseArtifactMetadata(outDir, name); ok {
			if metadata.Role != "" {
				role = metadata.Role
			}
			if metadata.Provenance != "" {
				provenance = metadata.Provenance
			}
		} else if strings.HasPrefix(name, "resource-") {
			// Controller artifacts are a generic, manifest-independent release
			// class; service artifacts must always provide metadata above.
			role = "resource-controller"
		}
		osName, arch := releaseArtifactPlatform(name)
		artifacts = append(artifacts, resourcedeployment.ReleaseArtifact{Name: name, SHA256: fields[0], Role: role, OS: osName, Arch: arch, UpstreamProvenance: provenance})
	}
	canonical, err := (resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: artifacts}).CanonicalBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "release-manifest.json"), append(canonical, '\n'), mndResourceArtifactsNumberOctal644)
}

func releaseArtifactPlatform(name string) (string, string) {
	stem := name
	for _, suffix := range []string{".manifest.json", ".build.json", ".sha256", ".exe", ".json"} {
		stem = strings.TrimSuffix(stem, suffix)
	}
	parts := strings.Split(stem, "_")
	if len(parts) < resourceCoordinatePartCount {
		return "", ""
	}
	osName, arch := parts[len(parts)-2], parts[len(parts)-1]
	if osName != "linux" && osName != "darwin" && osName != literalResourceArtifactsWindows {
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
	if manifest.CLI.Distribution.Kind != literalResourceArtifactsPrebuilt || strings.TrimSpace(manifest.CLI.Distribution.ArtifactName) == "" || strings.TrimSpace(manifest.CLI.Adapter.ModuleDir) == "" {
		return nil, fmt.Errorf("bundled resource artifacts require prebuilt distribution and Go module adapter")
	}
	var targets []resourceBuildTarget
	for _, platform := range []string{"linux", "macos", literalResourceArtifactsWindows} {
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
	command := shell.Command(shell.Spec{
		Context: ctx,
		Name:    "go",
		Args:    []string{"build", "-trimpath", "-buildvcs=false", "-o", filepath.Join(outDir, target.Artifact), "."},
		Dir:     filepath.Join(resourceDir, filepath.FromSlash(target.Module)),
		Env:     envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"CGO_ENABLED=0", "GOOS=" + goos, "GOARCH=" + target.Platform.Arch}),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})
	if err := command.Run(); err != nil {
		return fmt.Errorf("build resource %s for %s: %w", target.Resource, target.Platform, err)
	}
	manifestData, err := os.ReadFile(filepath.Join(resourceDir, "resource.json"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, target.Artifact+".manifest.json"), manifestData, mndResourceArtifactsNumberOctal644); err != nil {
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
	return os.WriteFile(filepath.Join(outDir, target.Artifact+".build.json"), append(metadata, '\n'), mndResourceArtifactsNumberOctal644)
}

//nolint:gocyclo // artifact staging preserves the archive, binary, checksum, and platform fallback matrix.
func stageManagedServiceArtifact(ctx context.Context, manifest resourceArtifactManifest, resourceRoot, outDir string, platform resourcedeployment.Platform) (stagedManagedServiceArtifact, error) {
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
	if layout == literalResourceArtifactsDir && target.BinPath != "" {
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
	if strings.EqualFold(strings.TrimSpace(manifest.ManagedService.Acquisition.Kind), "composed") {
		if err := composeAcquisitionTarget(ctx, target, resourceRoot, path, platform); err != nil {
			return stagedManagedServiceArtifact{}, fmt.Errorf("compose managed service %s: %w", manifest.Name, err)
		}
	} else if err := stageAcquisitionTarget(ctx, manifest.ManagedService.Acquisition.Kind, target, platform, name, path, outDir); err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	if err := artifact.VerifyFile(path); err != nil {
		return stagedManagedServiceArtifact{}, fmt.Errorf("verify staged %s: %w", manifest.Name, err)
	}
	return stagedManagedServiceArtifact{Version: artifact.Version, File: name}, nil
}

func deferredResourceTargets(manifest resourceArtifactManifest, platform resourcedeployment.Platform) []deferredResourceTargetRecord {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return nil
	}
	factsOS := artifactOS(platform.OS)
	result := make([]deferredResourceTargetRecord, 0)
	for index, target := range manifest.ManagedService.Acquisition.Targets {
		if binaryfetch.UsesOnlyBuildTimeFacts(target) {
			continue
		}
		if value := strings.TrimSpace(target.When["os"]); value != "" && value != factsOS {
			continue
		}
		if value := strings.TrimSpace(target.When["arch"]); value != "" && value != platform.Arch {
			continue
		}
		absent := make([]string, 0)
		for name := range target.When {
			if name != "os" && name != "arch" {
				absent = append(absent, name)
			}
		}
		sort.Strings(absent)
		result = append(result, deferredResourceTargetRecord{Resource: manifest.Name, OS: platform.OS, Architecture: platform.Arch, TargetIndex: index, When: cloneAcquisitionFacts(target.When), AbsentFacts: absent})
	}
	return result
}

func cloneAcquisitionFacts(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

type ComposeStepError struct {
	Index int
	Role  string
	Kind  string
	Err   error
}

func (e *ComposeStepError) Error() string {
	return fmt.Sprintf("compose step %d (%s/%s): %v", e.Index, e.Role, e.Kind, e.Err)
}
func (e *ComposeStepError) Unwrap() error { return e.Err }

func composeAcquisitionTarget(ctx context.Context, target binaryfetch.AcquisitionTarget, resourceRoot, artifactRoot string, platform resourcedeployment.Platform) error {
	type provenanceStep struct {
		Role         string `json:"role"`
		Kind         string `json:"kind"`
		Dest         string `json:"dest"`
		SourceDigest string `json:"source_digest,omitempty"`
	}
	records := make([]provenanceStep, 0, len(target.Compose))
	for index, step := range target.Compose {
		if err := step.Validate(); err != nil {
			return &ComposeStepError{Index: index, Role: step.Role, Kind: step.Kind, Err: err}
		}
		if err := runComposeStep(ctx, step, resourceRoot, artifactRoot, platform); err != nil {
			return &ComposeStepError{Index: index, Role: step.Role, Kind: step.Kind, Err: err}
		}
		record := provenanceStep{Role: step.Role, Kind: step.Kind, Dest: step.Dest, SourceDigest: step.SHA256}
		if strings.EqualFold(strings.TrimSpace(step.Kind), "python-wheels") {
			lockfile := filepath.Join(resourceRoot, filepath.FromSlash(step.Lockfile))
			digest, err := fileSHA256(lockfile)
			if err != nil {
				return &ComposeStepError{Index: index, Role: step.Role, Kind: step.Kind, Err: err}
			}
			record.SourceDigest = digest
		}
		records = append(records, record)
	}
	manifest := struct {
		SchemaVersion string           `json:"schema_version"`
		Platform      string           `json:"platform"`
		Steps         []provenanceStep `json:"steps"`
	}{SchemaVersion: "v1", Platform: platform.String(), Steps: records}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(artifactRoot, ".vrooli-compose-manifest.json"), append(data, '\n'), mndResourceArtifactsNumberOctal644)
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func runComposeStep(ctx context.Context, step binaryfetch.ComposeStep, resourceRoot, artifactRoot string, platform resourcedeployment.Platform) error {
	dest := filepath.Clean(filepath.Join(artifactRoot, filepath.FromSlash(step.Dest)))
	root := filepath.Clean(artifactRoot)
	if dest != root && !strings.HasPrefix(dest, root+string(filepath.Separator)) {
		return fmt.Errorf("destination %q escapes artifact root", step.Dest)
	}
	switch strings.ToLower(strings.TrimSpace(step.Kind)) {
	case "url":
		if err := binaryfetch.FetchTree(ctx, binaryfetch.Target{URL: step.URL, SHA256: step.SHA256, Archive: step.Archive}, dest, nil); err != nil {
			return err
		}
		if strings.TrimSpace(step.BinPath) != "" {
			return flattenComposePrefix(dest, step.BinPath)
		}
		return nil
	case "python-wheels":
		lockfile := filepath.Join(resourceRoot, filepath.FromSlash(step.Lockfile))
		if info, err := os.Stat(lockfile); err != nil || info.IsDir() {
			return fmt.Errorf("lockfile %q is unavailable for target %s: %w", step.Lockfile, platform, err)
		}
		if err := os.MkdirAll(dest, mndResourceArtifactsNumberOctal755); err != nil {
			return err
		}
		pythonPlatform := uvPlatform(platform)
		command, err := resourceacquisition.PythonWheelsCommand(ctx, step, dest, lockfile, pythonPlatform)
		if err != nil {
			return err
		}
		command.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"UV_NO_PROGRESS=1"})
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("uv wheel resolution failed: %w: %s (command: %s)", err, strings.TrimSpace(string(output)), strings.Join(command.Args, " "))
		}
		return nil
	default:
		return fmt.Errorf("unsupported compose step kind %q", step.Kind)
	}
}

// flattenComposePrefix removes the conventional single directory emitted by
// source/runtime archives. Compose steps use bin_path for this archive-root
// hint; the resulting artifact has stable paths such as runtime/bin/python and
// source/searx rather than release-specific tarball directory names.
func flattenComposePrefix(root, prefix string) error {
	prefixPath := filepath.Join(root, filepath.FromSlash(prefix))
	info, err := os.Stat(prefixPath)
	if err != nil {
		return fmt.Errorf("compose archive root %q is unavailable: %w", prefix, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("compose archive root %q is not a directory", prefix)
	}
	flat := root + ".flat"
	if err := os.Rename(root, flat); err != nil {
		return err
	}
	if err := os.MkdirAll(root, mndResourceArtifactsNumberOctal755); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(flat, filepath.FromSlash(prefix)))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.Rename(filepath.Join(flat, filepath.FromSlash(prefix), entry.Name()), filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return os.RemoveAll(flat)
}

func uvPlatform(platform resourcedeployment.Platform) string {
	switch platform.OS {
	case "linux":
		if platform.Arch == "arm64" {
			return "aarch64-unknown-linux-gnu"
		}
		return "x86_64-unknown-linux-gnu"
	case "macos":
		if platform.Arch == "arm64" {
			return "aarch64-apple-darwin"
		}
		return "x86_64-apple-darwin"
	case literalResourceArtifactsWindows:
		return "x86_64-pc-windows-msvc"
	default:
		return platform.OS + "-" + platform.Arch
	}
}

func updateReleaseArtifactMetadata(outDir, name string, metadata releaseArtifactMetadata) error {
	all := map[string]releaseArtifactMetadata{}
	path := filepath.Join(outDir, releaseArtifactMetadataFile)
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &all); err != nil {
			return err
		}
	}
	if metadata.Role == "" {
		metadata.Role = literalResourceArtifactsManagedService
	}
	if metadata.Provenance == "" {
		metadata.Provenance = "verified-by-vrooli-stager"
	}
	all[name] = metadata
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), mndResourceArtifactsNumberOctal644)
}

func readReleaseArtifactMetadata(outDir, name string) (releaseArtifactMetadata, bool) {
	data, err := os.ReadFile(filepath.Join(outDir, releaseArtifactMetadataFile))
	if err != nil {
		return releaseArtifactMetadata{}, false
	}
	all := map[string]releaseArtifactMetadata{}
	if json.Unmarshal(data, &all) != nil {
		return releaseArtifactMetadata{}, false
	}
	value, ok := all[name]
	return value, ok
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
	if platform.OS == literalResourceArtifactsWindows {
		suffix = ".exe"
	}
	return fmt.Sprintf("tool_%s_%s_%s%s", name, artifactOS(platform.OS), platform.Arch, suffix)
}
