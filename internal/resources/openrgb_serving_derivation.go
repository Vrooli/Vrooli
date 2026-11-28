package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
)

const openRGBSealedServingProfile = "serving-profile.json"
const openRGBMacOrphanFramework = "Contents/Frameworks/QtCore5Compat.framework"

func (p openRGBReleaseProfile) verifyServingStageLayout(ctx context.Context, root string) error {
	if p.OS != "darwin" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	wanted := map[string]bool{p.bundleDirectory(): true, openRGBSealedEnvelope: true, openRGBSealedServingProfile: true}
	if len(entries) != len(wanted) {
		return errors.New("Mac serving stage contains undeclared adjacent entries")
	}
	for _, entry := range entries {
		if !wanted[entry.Name()] || entry.Type()&os.ModeSymlink != 0 {
			return errors.New("Mac serving stage contains foreign or aliased adjacent content")
		}
		if entry.Name() == p.bundleDirectory() && !entry.IsDir() || entry.Name() != p.bundleDirectory() && !entry.Type().IsRegular() {
			return errors.New("Mac serving stage layout has invalid entry types")
		}
	}
	return nil
}

func (p openRGBReleaseProfile) macOmissionTreeSHA256() string {
	if p.Arch == "amd64" {
		return "1cae3e0b289dd425c30a3c019d3923823f24dc138ee19f0f30e4a696430dd623"
	}
	if p.Arch == "arm64" {
		return "c1f484d77a4b732990e16838c1634aa97d30ab9407e5ccba0afb9a8bbd323695"
	}
	return ""
}

func (p openRGBReleaseProfile) servingTreeSHA256() string {
	if p.ServingTreeSHA256 != "" {
		return p.ServingTreeSHA256
	}
	return p.TreeSHA256
}

func (p openRGBReleaseProfile) servingRecipeBytes() ([]byte, error) {
	if p.ServingRecipeID == "" || p.OS != "windows" && p.OS != "darwin" {
		return nil, errors.New("profile has no serving derivation")
	}
	var names []string
	modules := openRGBPawnIOSourceModules()
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	var omitted []map[string]any
	if p.OS == "windows" {
		for _, name := range names {
			module := modules[name]
			omitted = append(omitted, map[string]any{"path": name, "sha256": module.SHA256, "bytes": module.Bytes})
		}
	} else {
		omitted = []map[string]any{{"kind": "directory", "path": openRGBMacOrphanFramework, "tree_sha256": p.macOmissionTreeSHA256()}}
	}
	// encoding/json sorts every object's keys. Values are ASCII strings and
	// exact bounded integers, so the recipe has one reproducible representation.
	return json.Marshal(map[string]any{"schema_version": 1, "id": p.ServingRecipeID, "resource": openRGBResourceName, "version": openRGBAppImageVersion, "os": p.OS, "arch": p.Arch, "source_envelope_sha256": p.EnvelopeSHA256, "source_tree_sha256": p.TreeSHA256, "payload_sha256": p.PayloadSHA256, "serving_tree_sha256": p.ServingTreeSHA256, "omit": omitted})
}

func (p openRGBReleaseProfile) servingProfileBytes(source ManagedServiceArtifact) ([]byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	if source.Resource != openRGBResourceName || source.Version != openRGBAppImageVersion || !p.acceptsArtifactSHA256(source.SHA256) {
		return nil, errOpenRGBReleasePayloadUnqualified
	}
	recipe, err := p.servingRecipeBytes()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"schema_version": 1, "recipe_sha256": p.ServingRecipeSHA256, "recipe": json.RawMessage(recipe), "selected_source_artifact_sha256": source.SHA256})
}

// The complete official source is proven before the fixed transformation.
// This is a closed derived serving tree, never an incomplete upstream bundle.
func (p openRGBReleaseProfile) deriveServingTree(ctx context.Context, root string) error {
	if p.ServingRecipeID == "" {
		return nil
	}
	if err := p.validate(); err != nil {
		return err
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, root, "dir", p.TreeSHA256); err != nil {
		return fmt.Errorf("verify complete original OpenRGB source before HID derivation: %w", err)
	}
	if err := p.verifyBundleABI(ctx, root); err != nil {
		return err
	}
	if p.OS == "darwin" {
		framework := filepath.Join(root, filepath.FromSlash(openRGBMacOrphanFramework))
		if err := binaryfetch.VerifyArtifactContext(ctx, framework, "dir", p.macOmissionTreeSHA256()); err != nil {
			return err
		}
		if err := os.RemoveAll(framework); err != nil {
			return err
		}
		if _, err := p.darwinLoaderSystemImages(ctx, root); err != nil {
			return err
		}
	} else {
		for name, module := range openRGBPawnIOSourceModules() {
			if err := ctx.Err(); err != nil {
				return err
			}
			path := filepath.Join(root, name)
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() || info.Size() != module.Bytes {
				return errors.Join(errors.New("serving omission is not the fixed source module"), err)
			}
			if err := binaryfetch.VerifyArtifactContext(ctx, path, "file", module.SHA256); err != nil {
				return err
			}
			// Windows' immutable writer initially sets the read-only attribute.
			// Only the privileged publisher controls these pending source objects.
			if err := os.Chmod(path, 0o600); err != nil {
				return err
			}
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, root, "dir", p.servingTreeSHA256()); err != nil {
		return fmt.Errorf("verify complete derived OpenRGB HID serving tree: %w", err)
	}
	return p.verifyServingModulesAbsent(ctx, root)
}

func (p openRGBReleaseProfile) verifyServingModulesAbsent(ctx context.Context, root string) error {
	if p.ServingRecipeID == "" {
		return nil
	}
	if p.OS == "darwin" {
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(openRGBMacOrphanFramework)))
		if !os.IsNotExist(err) {
			return errors.Join(errors.New("derived Mac serving tree contains the omitted orphan framework"), err)
		}
		return nil
	}
	for name := range openRGBPawnIOSourceModules() {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := os.Lstat(filepath.Join(root, name))
		if !os.IsNotExist(err) {
			return errors.Join(fmt.Errorf("derived HID serving tree must omit %s", name), err)
		}
	}
	return nil
}

func (p openRGBReleaseProfile) writeServingProfile(ctx context.Context, root string, source ManagedServiceArtifact) error {
	if p.ServingRecipeID == "" {
		return nil
	}
	data, err := p.servingProfileBytes(source)
	if err != nil {
		return err
	}
	return openRGBWriteStageFile(ctx, filepath.Join(root, openRGBSealedServingProfile), 0o444, func(w io.Writer) error { _, err := w.Write(data); return err })
}

func (p openRGBReleaseProfile) verifyServingProfile(ctx context.Context, root string, source ManagedServiceArtifact) error {
	if p.ServingRecipeID == "" {
		return nil
	}
	expected, err := p.servingProfileBytes(source)
	if err != nil {
		return err
	}
	path := filepath.Join(root, openRGBSealedServingProfile)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(expected)) || info.Mode().Perm()&0o222 != 0 {
		return errors.Join(errors.New("derived serving profile metadata is not exact and read-only"), err)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if err := platform.VerifyImmutableNativePath(ctx, path, false); err != nil {
			return err
		}
	}
	return binaryfetch.VerifyArtifactContext(ctx, path, "file", fmt.Sprintf("%x", sha256.Sum256(expected)))
}
