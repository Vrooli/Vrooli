package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/envkit-go"
	_ "github.com/vrooli/vrooli/internal/acquisition" // register the caller-owned tar.zst archive decoder
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostinventory"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const (
	managedServiceAcquisitionDir = "dir"
)

const (
	managedServiceAcquisitionFile = "file"
)

// PruneManagedServiceArtifacts removes superseded versions from one resource's
// user-owned artifact store. The manifest's current version is always kept;
// unrecognized entries are removed only beneath that resource/version root.
func (c *Controller) PruneManagedServiceArtifacts(name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("resource name is required")
	}
	manifest, err := c.LoadManifest(filepath.Join(c.Root, "resources", name, "resource.json"))
	if err != nil {
		return 0, err
	}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return 0, nil
	}
	root, err := managedServiceArtifactVersionsRoot(c, manifest)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == manifest.ManagedService.Artifact.Version {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return removed, fmt.Errorf("prune %s artifact version %s: %w", name, entry.Name(), err)
		}
		removed++
	}
	return removed, nil
}

// managedServiceAcquisitionTarget resolves the same ordered, fact-predicated
// contract used by explain. There is no resource-specific source switch here:
// the manifest is the sole source of the selected target.
func managedServiceAcquisitionTarget(ctx context.Context, manifest ResourceManifest) (binaryfetch.AcquisitionTarget, error) {
	target, _, err := managedServiceAcquisitionTargetWithFacts(ctx, manifest)
	return target, err
}

// managedServiceAcquisitionTargetWithFacts resolves the target and returns the
// facts it was resolved from. The facts travel with the target because both the
// install record and the drift message need them, and re-collecting would risk
// comparing a target against a different observation than the one that produced
// it.
func managedServiceAcquisitionTargetWithFacts(ctx context.Context, manifest ResourceManifest) (binaryfetch.AcquisitionTarget, binaryfetch.Facts, error) {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return binaryfetch.AcquisitionTarget{}, nil, nil
	}
	// The accelerator-only collection path: a resource start must not wait on
	// desktop or credential-store probes to learn which artifact it needs.
	snapshot, err := hostinventory.CollectGPUFacts(ctx)
	if err != nil {
		return binaryfetch.AcquisitionTarget{}, nil, fmt.Errorf("collect host facts for %s acquisition: %w", manifest.Name, err)
	}
	facts := snapshot.AcceleratorFacts()
	target, err := manifest.ManagedService.Acquisition.Resolve(facts)
	if err != nil {
		return target, facts, fmt.Errorf("resolve acquisition target for %s: %w", manifest.Name, err)
	}
	return target, facts, nil
}

// ensureManagedServiceArtifact converges a declared managed service into the
// user-owned artifact store. Existing bytes are always verified under the
// shared artifact lock before a network operation is considered.
//
//nolint:gocyclo // acquisition preserves ordered archive, binary, and fallback artifact decisions.
func ensureManagedServiceArtifact(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if manifest.ManagedService == nil {
		return fmt.Errorf("managed_service is required")
	}
	if manifest.ManagedService.Acquisition == nil {
		if err := verifyManagedServiceArtifact(controller, manifest); err != nil {
			return err
		}
		return ensureManagedServiceDataArtifacts(ctx, controller, manifest)
	}
	release, err := artifactlock.Acquire("resource:" + manifest.Name)
	if err != nil {
		return err
	}
	defer release()

	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return err
	}
	target, err := managedServiceAcquisitionTarget(ctx, manifest)
	if err != nil {
		return err
	}
	if target.Unsupported != "" {
		return fmt.Errorf("resource %s is unsupported: %s", manifest.Name, target.Unsupported)
	}
	artifact, err := managedServiceArtifactForTarget(manifest, target)
	if err != nil {
		return err
	}
	if strings.TrimSpace(target.Executable) != "" {
		if strings.TrimSpace(target.Executable) == "" {
			return fmt.Errorf("resource %s host-tool acquisition did not resolve an executable", manifest.Name)
		}
		if filepath.Base(path) != filepath.Base(target.Executable) {
			return fmt.Errorf("resource %s host-tool executable %q does not match artifact %q", manifest.Name, target.Executable, manifest.ManagedService.Artifact.Path)
		}
		if err := verifyManagedServiceTargetArtifact(path, artifact, target); err != nil {
			return fmt.Errorf("verify adopted %s host tool: %w", manifest.Name, err)
		}
		return ensureManagedServiceDataArtifacts(ctx, controller, manifest)
	}
	if err := verifyManagedServiceTargetArtifact(path, artifact, target); err == nil {
		recordManagedServiceInstallFacts(ctx, manifest, path, target, artifact)
		return ensureManagedServiceDataArtifacts(ctx, controller, manifest)
	}
	if err := verifyManagedServiceProvenance(ctx, manifest.ManagedService.Acquisition.Provenance); err != nil {
		return fmt.Errorf("verify %s acquisition provenance: %w", manifest.Name, err)
	}

	layout := strings.ToLower(strings.TrimSpace(target.Layout))
	if layout == "" {
		layout = strings.ToLower(strings.TrimSpace(artifact.Layout))
	}
	if layout == "" {
		layout = managedServiceAcquisitionFile
	}
	mode := target.Mode
	if mode == "" {
		mode = "755"
	}
	// The kind is resolved per target, so one resource can stage an OCI
	// filesystem tree on one platform and a published archive on another.
	targetKind := manifest.ManagedService.Acquisition.EffectiveKind(target)
	if targetKind == "oci-image" {
		if layout == managedServiceAcquisitionDir {
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("clean %s artifact tree: %w", manifest.Name, err)
			}
			if _, err := binaryfetch.FetchOCI(ctx, target, path, nil); err != nil {
				return fmt.Errorf("acquire %s OCI artifact: %w", manifest.Name, err)
			}
		} else {
			if _, err := binaryfetch.FetchOCIFile(ctx, target, path, nil); err != nil {
				return fmt.Errorf("acquire %s OCI executable: %w", manifest.Name, err)
			}
		}
	} else if targetKind == "composed" {
		if err := composeManagedServiceArtifact(ctx, target, filepath.Join(controller.Root, "resources", manifest.Name), path); err != nil {
			return fmt.Errorf("compose %s artifact: %w", manifest.Name, err)
		}
	} else {
		name := filepath.Base(path)
		spec := binaryfetch.Target{
			Name: name, URL: target.URL, SHA256: target.SHA256,
			Archive: target.Archive, Layout: layout, BinPath: target.BinPath, Mode: mode,
		}
		if layout == managedServiceAcquisitionDir {
			if _, err := binaryfetch.FetchDir(ctx, spec, path, nil); err != nil {
				return fmt.Errorf("acquire %s artifact tree: %w", manifest.Name, err)
			}
		} else {
			if _, err := binaryfetch.Fetch(ctx, spec, filepath.Dir(path), nil); err != nil {
				return fmt.Errorf("acquire %s artifact: %w", manifest.Name, err)
			}
		}
	}
	if err := verifyManagedServiceTargetArtifact(path, artifact, target); err != nil {
		return fmt.Errorf("verify acquired %s artifact: %w", manifest.Name, err)
	}
	// The digest says these are the right bytes. The closure check says the
	// host can start them. An artifact that passes the first and fails the
	// second is worse than a missing one: it installs clean and then refuses to
	// run, with no signal until someone reads the service log.
	if verdict := verifyManagedServiceRuntimeClosure(manifest, path); verdict.State == ClosureUnresolved {
		if removeErr := os.RemoveAll(path); removeErr != nil {
			return fmt.Errorf("discard unusable %s artifact: %w", manifest.Name, removeErr)
		}
		return &RuntimeClosureError{Resource: manifest.Name, Artifact: path, Verdict: verdict}
	}
	recordManagedServiceInstallFacts(ctx, manifest, path, target, artifact)
	return ensureManagedServiceDataArtifacts(ctx, controller, manifest)
}

// recordManagedServiceInstallFacts writes the sidecar that makes a later
// mismatch explainable. It is best-effort: the artifact is already staged and
// verified, and a missing record only costs the better diagnosis later.
func recordManagedServiceInstallFacts(ctx context.Context, manifest ResourceManifest, path string, target binaryfetch.AcquisitionTarget, artifact resourcedeployment.ServiceArtifact) {
	snapshot, err := hostinventory.CollectGPUFacts(ctx)
	if err != nil {
		return
	}
	_ = writeInstallFacts(path, manifest.Name, snapshot.AcceleratorFacts(), target, artifact, time.Now())
}

//nolint:gocyclo // managed artifact composition preserves archive, binary, checksum, and filesystem outcomes.
func composeManagedServiceArtifact(ctx context.Context, target binaryfetch.AcquisitionTarget, resourceRoot, artifactRoot string) error {
	if err := os.RemoveAll(artifactRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(artifactRoot, tuning.PermDir); err != nil {
		return err
	}
	for index, step := range target.Compose {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("compose step %d (%s): %w", index, step.Role, err)
		}
		dest := filepath.Clean(filepath.Join(artifactRoot, filepath.FromSlash(step.Dest)))
		if dest != artifactRoot && !strings.HasPrefix(dest, artifactRoot+string(filepath.Separator)) {
			return fmt.Errorf("compose step %d destination escapes artifact root", index)
		}
		switch strings.ToLower(strings.TrimSpace(step.Kind)) {
		case "url":
			if err := binaryfetch.FetchTree(ctx, binaryfetch.Target{URL: step.URL, SHA256: step.SHA256, Archive: step.Archive}, dest, nil); err != nil {
				return err
			}
			if step.BinPath != "" {
				if err := flattenManagedComposePrefix(dest, step.BinPath); err != nil {
					return err
				}
			}
		case "python-wheels":
			lockfile := filepath.Join(resourceRoot, filepath.FromSlash(step.Lockfile))
			if err := os.MkdirAll(dest, tuning.PermDir); err != nil {
				return err
			}
			arch := runtime.GOARCH
			if arch == "amd64" {
				arch = "x86_64"
			}
			if arch == "arm64" {
				arch = "aarch64"
			}
			platform := arch + "-unknown-linux-gnu"
			if runtime.GOOS == "darwin" {
				platform = arch + "-apple-darwin"
			}
			if runtime.GOOS == "windows" {
				platform = arch + "-pc-windows-msvc"
			}
			command, err := pythonWheelsCommand(ctx, step, dest, lockfile, platform)
			if err != nil {
				return err
			}
			command.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkit.Env{"UV_NO_PROGRESS=1"})
			if output, err := command.CombinedOutput(); err != nil {
				return fmt.Errorf("uv wheel resolution failed: %w: %s", err, strings.TrimSpace(string(output)))
			}
		case "local":
			source := filepath.Join(resourceRoot, filepath.FromSlash(step.Source))
			if err := copyManagedComposeSource(source, dest); err != nil {
				return fmt.Errorf("copy local compose source %q: %w", step.Source, err)
			}
		default:
			return fmt.Errorf("unsupported compose step kind %q", step.Kind)
		}
	}
	platformOS := runtime.GOOS
	if platformOS == "darwin" {
		platformOS = "macos"
	}
	return writeManagedComposeManifest(target, artifactRoot, platformOS+"-"+runtime.GOARCH, resourceRoot)
}

func pythonWheelsCommand(ctx context.Context, step binaryfetch.ComposeStep, dest, lockfile, platform string) (*exec.Cmd, error) {
	// The lock is the complete resolved closure. Installing without dependency
	// re-resolution is important here: it prevents a package metadata edge from
	// reintroducing an unqualified source distribution after the lock has
	// selected a replacement wheel (Kokoro uses pyopenjtalk-plus for Linux).
	args := []string{"pip", "install", "--no-deps", "--target", dest, "--requirement", lockfile, "--python-version", "3.12", "--python-platform", platform}
	if !step.AllowSDists {
		args = append(args, "--only-binary", ":all:")
	}
	if index := strings.TrimSpace(step.IndexURL); index != "" {
		args = append(args, "--index-url", index)
	}
	for _, index := range step.ExtraIndexURLs {
		args = append(args, "--extra-index-url", strings.TrimSpace(index))
	}
	// A CUDA wheel index and PyPI both publish packages such as torch. The
	// default first-index strategy can reject a lock containing a local CUDA
	// version after it sees the PyPI project page. The lock carries hashes, so
	// considering all declared indexes is deterministic and still refuses any
	// byte not present in that lock.
	if strings.TrimSpace(step.IndexURL) != "" && len(step.ExtraIndexURLs) > 0 {
		args = append(args, "--index-strategy", "unsafe-best-match")
	}
	data, err := os.ReadFile(lockfile)
	if err != nil {
		return nil, fmt.Errorf("read python wheel lockfile: %w", err)
	}
	hashed := strings.Contains(string(data), "--hash=sha256:")
	if (strings.TrimSpace(step.IndexURL) != "" || len(step.ExtraIndexURLs) > 0) && !hashed {
		return nil, fmt.Errorf("python-wheels index requires a hash-pinned lockfile")
	}
	if hashed {
		args = append(args, "--require-hashes")
	}
	return shell.NewCommandContext(ctx, "uv", args...), nil
}

func copyManagedComposeSource(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("local compose source must be a file")
	}
	if err := os.MkdirAll(filepath.Dir(destination), tuning.PermDir); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, tuning.PermFile)
	if err != nil {
		return err
	}
	defer output.Close()
	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}

func flattenManagedComposePrefix(root, prefix string) error {
	prefixPath := filepath.Join(root, filepath.FromSlash(prefix))
	info, err := os.Stat(prefixPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("compose archive root %q is not a directory", prefix)
	}
	flat := root + ".flat"
	if err := os.Rename(root, flat); err != nil { //nolint:forbidigo // intentional directory flattening
		return err
	}
	if err := os.MkdirAll(root, tuning.PermDir); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(flat, filepath.FromSlash(prefix)))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.Rename(filepath.Join(flat, filepath.FromSlash(prefix), entry.Name()), filepath.Join(root, entry.Name())); err != nil { //nolint:forbidigo // intentional directory flattening
			return err
		}
	}
	return os.RemoveAll(flat)
}

func writeManagedComposeManifest(target binaryfetch.AcquisitionTarget, root, platform, resourceRoot string) error {
	type stepRecord struct {
		Role         string `json:"role"`
		Kind         string `json:"kind"`
		Dest         string `json:"dest"`
		SourceDigest string `json:"source_digest,omitempty"`
	}
	records := make([]stepRecord, 0, len(target.Compose))
	for _, step := range target.Compose {
		digest := step.SHA256
		if step.Kind == "python-wheels" {
			value, err := managedFileSHA256(filepath.Join(resourceRoot, filepath.FromSlash(step.Lockfile)))
			if err != nil {
				return err
			}
			digest = value
		}
		records = append(records, stepRecord{Role: step.Role, Kind: step.Kind, Dest: step.Dest, SourceDigest: digest})
	}
	data, err := json.MarshalIndent(struct {
		SchemaVersion string       `json:"schema_version"`
		Platform      string       `json:"platform"`
		Steps         []stepRecord `json:"steps"`
	}{"v1", platform, records}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ".vrooli-compose-manifest.json"), append(data, '\n'), tuning.PermFile)
}

func managedFileSHA256(path string) (string, error) {
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

// ensureManagedServiceDataArtifacts stages model and other non-executable
// inputs below RESOURCE_DATA_DIR. They use the same host-fact selection and
// digest verification as the launch artifact, but never become executable or
// part of the supervisor's launch path.
//
//nolint:gocyclo // managed-service data acquisition coordinates declaration, cache, fetch, and verification states.
func ensureManagedServiceDataArtifacts(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if manifest.ManagedService == nil || len(manifest.ManagedService.DataArtifacts) == 0 {
		return nil
	}
	env := managedServiceEnvValues(resourceEnvForResource(controller.Root, controller.Home, manifest.Name))
	dataRoot := strings.TrimSpace(env["RESOURCE_DATA_DIR"])
	if dataRoot == "" {
		return fmt.Errorf("managed-service data artifacts require RESOURCE_DATA_DIR")
	}
	release, err := artifactlock.Acquire("resource-data:" + manifest.Name)
	if err != nil {
		return err
	}
	defer release()

	snapshot, err := hostinventory.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect host facts for %s data artifacts: %w", manifest.Name, err)
	}
	for _, declaration := range manifest.ManagedService.DataArtifacts {
		target, err := declaration.Acquisition.Resolve(snapshot.AcceleratorFacts())
		if err != nil {
			return fmt.Errorf("resolve %s data artifact target: %w", declaration.Name, err)
		}
		if target.Unsupported != "" {
			return fmt.Errorf("data artifact %s is unsupported: %s", declaration.Name, target.Unsupported)
		}
		path := filepath.Join(dataRoot, filepath.FromSlash(declaration.Path))
		cleanRoot := filepath.Clean(dataRoot)
		cleanPath := filepath.Clean(path)
		if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator)) {
			return fmt.Errorf("data artifact %s path escapes RESOURCE_DATA_DIR", declaration.Name)
		}
		layout := strings.ToLower(strings.TrimSpace(target.Layout))
		if layout == "" {
			layout = managedServiceAcquisitionFile
		}
		if existingErr := binaryfetch.VerifyArtifact(path, layout, target.SHA256); existingErr == nil {
			if err := writeDataArtifactChecksum(path, layout, target.SHA256); err != nil {
				return fmt.Errorf("record %s data artifact checksum: %w", declaration.Name, err)
			}
			continue
		}
		// A source checkout may already contain the exact release artifact
		// produced by vrooli-dist. Treat it as a vendored input, not as a
		// source-build instruction, and still verify it against the manifest
		// digest before copying it into the managed data root.
		if layout == managedServiceAcquisitionFile {
			sourcePath := filepath.Join(controller.Root, "resources", manifest.Name, "artifacts", filepath.Base(cleanPath))
			if err := binaryfetch.VerifyArtifact(sourcePath, layout, target.SHA256); err == nil {
				if err := copyDataArtifact(sourcePath, cleanPath); err != nil {
					return fmt.Errorf("stage vendored %s data artifact: %w", declaration.Name, err)
				}
				if err := writeDataArtifactChecksum(cleanPath, layout, target.SHA256); err != nil {
					return fmt.Errorf("record %s data artifact checksum: %w", declaration.Name, err)
				}
				continue
			}
		}
		if strings.EqualFold(strings.TrimSpace(declaration.Acquisition.Kind), "oci-image") {
			return fmt.Errorf("data artifact %s uses unsupported OCI acquisition; model data must be a URL artifact", declaration.Name)
		}
		spec := binaryfetch.Target{
			Name: filepath.Base(cleanPath), URL: target.URL, SHA256: target.SHA256,
			Archive: target.Archive, Layout: layout, BinPath: target.BinPath, Mode: target.Mode,
		}
		if layout == managedServiceAcquisitionDir {
			if _, err := binaryfetch.FetchDir(ctx, spec, cleanPath, nil); err != nil {
				return fmt.Errorf("acquire %s data artifact tree: %w", declaration.Name, err)
			}
		} else {
			if _, err := binaryfetch.Fetch(ctx, spec, filepath.Dir(cleanPath), nil); err != nil {
				return fmt.Errorf("acquire %s data artifact: %w", declaration.Name, err)
			}
		}
		if err := binaryfetch.VerifyArtifact(cleanPath, layout, target.SHA256); err != nil {
			return fmt.Errorf("verify acquired %s data artifact: %w", declaration.Name, err)
		}
		if err := writeDataArtifactChecksum(cleanPath, layout, target.SHA256); err != nil {
			return fmt.Errorf("record %s data artifact checksum: %w", declaration.Name, err)
		}
	}
	return nil
}

func copyDataArtifact(sourcePath, destinationPath string) error {
	if err := os.MkdirAll(filepath.Dir(destinationPath), tuning.PermDir); err != nil {
		return err
	}
	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary, err := os.CreateTemp(filepath.Dir(destinationPath), ".data-artifact-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := io.Copy(temporary, input); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Chmod(tuning.PermFile); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	data, err := os.ReadFile(temporaryPath)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(destinationPath, data, tuning.PermFile)
}

func writeDataArtifactChecksum(path, layout, checksum string) error {
	if strings.ToLower(strings.TrimSpace(layout)) != managedServiceAcquisitionFile || strings.TrimSpace(checksum) == "" {
		return nil
	}
	return os.WriteFile(path+".sha256", []byte(strings.TrimSpace(checksum)+"\n"), tuning.PermFile)
}

func managedServiceArtifactForTarget(manifest ResourceManifest, target binaryfetch.AcquisitionTarget) (resourcedeployment.ServiceArtifact, error) {
	artifact := manifest.ManagedService.Artifact
	if !strings.EqualFold(strings.TrimSpace(artifact.Verification), "host-tool") {
		var err error
		artifact, err = artifact.ForCurrentPlatform()
		if err != nil {
			return resourcedeployment.ServiceArtifact{}, err
		}
	} else if err := artifact.Validate(); err != nil {
		return resourcedeployment.ServiceArtifact{}, err
	}
	if target.ArtifactSHA256 != "" {
		artifact.SHA256 = target.ArtifactSHA256
	}
	if strings.TrimSpace(target.Executable) != "" {
		artifact.Verification = "host-tool"
	}
	if target.Layout != "" {
		artifact.Layout = target.Layout
	}
	layout := strings.ToLower(strings.TrimSpace(target.Layout))
	if layout == "" {
		layout = strings.ToLower(strings.TrimSpace(artifact.Layout))
	}
	// A file-layout archive uses bin_path only during extraction. The staged
	// file is the launch artifact, so EntryPath is meaningful only for a tree.
	if layout == managedServiceAcquisitionDir && target.BinPath != "" {
		artifact.EntryPath = strings.TrimPrefix(filepath.ToSlash(target.BinPath), "/")
	}
	return artifact, nil
}

// managedServiceArtifactForLaunch returns the checksum/layout selected by the
// same host-fact predicate that acquisition uses. A manifest-level artifact
// checksum is only the platform default; a GPU or other fact-specific target
// may deliberately identify a different verified artifact. It also returns the artifact the resolver selects for
// this host right now. artifactPath is the staged artifact it will be compared
// against; an empty path skips the fact-drift check, which is correct for a
// caller that has no staged artifact to compare.
func managedServiceArtifactForLaunch(ctx context.Context, manifest ResourceManifest, artifactPath string) (resourcedeployment.ServiceArtifact, error) {
	if manifest.ManagedService.Acquisition == nil {
		return managedServiceArtifactForTarget(manifest, binaryfetch.AcquisitionTarget{})
	}
	target, facts, err := managedServiceAcquisitionTargetWithFacts(ctx, manifest)
	if err != nil {
		return resourcedeployment.ServiceArtifact{}, err
	}
	if target.Unsupported != "" {
		return resourcedeployment.ServiceArtifact{}, fmt.Errorf("resource %s is unsupported: %s", manifest.Name, target.Unsupported)
	}
	// Ask whether the host moved before letting a digest comparison speak for
	// it. "The bytes are corrupt" and "the host changed" have different cures,
	// and only one of them is a re-download of the same artifact.
	if strings.TrimSpace(artifactPath) != "" {
		if driftErr := checkFactDrift(artifactPath, manifest.Name, facts, target); driftErr != nil {
			return resourcedeployment.ServiceArtifact{}, driftErr
		}
	}
	return managedServiceArtifactForTarget(manifest, target)
}

func verifyManagedServiceTargetArtifact(path string, artifact resourcedeployment.ServiceArtifact, target binaryfetch.AcquisitionTarget) error {
	if target.ArtifactSHA256 == "" {
		return artifact.VerifyFile(path)
	}
	layout := target.Layout
	if layout == "" {
		layout = artifact.Layout
	}
	return binaryfetch.VerifyArtifact(path, layout, target.ArtifactSHA256)
}

func verifyManagedServiceProvenance(ctx context.Context, declaration *binaryfetch.Provenance) error {
	if declaration == nil || strings.EqualFold(strings.TrimSpace(declaration.Kind), "none") {
		return nil
	}
	dir, err := os.MkdirTemp("", "vrooli-acquisition-provenance-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	keyPath := filepath.Join(dir, "release-key.asc")
	manifestPath := filepath.Join(dir, "SHA256SUMS")
	signaturePath := filepath.Join(dir, "SHA256SUMS.sig")
	for _, item := range []struct{ url, path string }{
		{declaration.KeyURL, keyPath},
		{declaration.ChecksumManifest, manifestPath},
		{declaration.ChecksumSignature, signaturePath},
	} {
		if err := binaryfetch.Download(ctx, item.url, item.path, nil); err != nil {
			return err
		}
	}
	_, err = binaryfetch.VerifyProvenance(ctx, declaration, keyPath, manifestPath, signaturePath)
	return err
}

// verifyManagedServiceArtifact is the launch/status gate for both acquired
// and release-bundled artifacts.
func verifyManagedServiceArtifact(controller *Controller, manifest ResourceManifest) error {
	path, err := managedServiceArtifactPath(controller, manifest)
	if err != nil {
		return err
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return err
	}
	return artifact.VerifyFile(path)
}

func managedServiceArtifactVersionsRoot(controller *Controller, manifest ResourceManifest) (string, error) {
	root, err := managedServiceArtifactStoreRoot(controller.Home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, manifest.Name), nil
}

func acquisitionTargetRuntimeEnv(ctx context.Context, manifest ResourceManifest, artifactRoot string) (map[string]string, error) {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return nil, nil
	}
	target, err := managedServiceAcquisitionTarget(ctx, manifest)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(target.RuntimeEnv))
	for key, value := range target.RuntimeEnv {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("acquisition runtime_env contains an empty key")
		}
		value = strings.ReplaceAll(value, "${RESOURCE_ARTIFACT_DIR}", artifactRoot)
		value = strings.ReplaceAll(value, "$RESOURCE_ARTIFACT_DIR", artifactRoot)
		values[key] = value
	}
	return values, nil
}

// verifyManagedServiceRuntimeClosure checks the staged executable against the
// library paths its selected backend declares. A resource that declares no
// accelerator still gets the check: an unsatisfiable closure is a broken
// artifact whatever the reason for its selection.
func verifyManagedServiceRuntimeClosure(manifest ResourceManifest, path string) ClosureVerdict {
	var libraryPaths []string
	if declaration := manifest.EffectiveAcceleration(); declaration != nil {
		for _, backend := range declaration.Backends {
			if config, ok := declaration.Config(backend); ok {
				libraryPaths = append(libraryPaths, config.LibraryPaths...)
			}
		}
	}
	return VerifyRuntimeClosure(path, libraryPaths)
}
