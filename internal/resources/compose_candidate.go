package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

type ComposeCandidateReceipt struct {
	Status         string                        `json:"status"`
	Resource       string                        `json:"resource"`
	Platform       string                        `json:"platform"`
	ArtifactSHA256 string                        `json:"artifact_sha256"`
	Inputs         map[string]string             `json:"input_sha256"`
	Target         binaryfetch.AcquisitionTarget `json:"target"`
	ElapsedMS      int64                         `json:"elapsed_ms"`
}
type composeArtifactFunc func(context.Context, binaryfetch.AcquisitionTarget, string, string) error

const maxComposeCandidateManifestBytes = 1 << 20

var composeCandidateResolveTarget = managedServiceAcquisitionTargetWithFacts

type candidateManifestBuffer struct{ bytes.Buffer }

func (b *candidateManifestBuffer) Write(p []byte) (int, error) {
	if len(p) > maxComposeCandidateManifestBytes-b.Len() {
		return 0, errors.New("candidate manifest exceeds 1 MiB limit")
	}
	return b.Buffer.Write(p)
}

// ComposeCandidate stages reviewed compose inputs without installing, executing
// the target, acquiring model data or publishing any runtime verification facts.
func (c *Controller) ComposeCandidate(ctx context.Context, name, output string) (ComposeCandidateReceipt, error) {
	var empty ComposeCandidateReceipt
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if !composeCandidatePlatformSupported() {
		return empty, errors.New("compose-candidate currently supports Linux only")
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return empty, errors.New("compose-candidate requires one resource name")
	}
	resourceRoot := filepath.Join(c.Root, "resources", name)
	var buffer candidateManifestBuffer
	if err := copyCandidateInput(ctx, resourceRoot, "resource.json", &buffer); err != nil {
		return empty, fmt.Errorf("read candidate manifest: %w", err)
	}
	raw := buffer.Bytes()
	manifest, err := manifestpkg.ParseForComposeCandidate(filepath.Join(resourceRoot, "resource.json"), raw)
	if err != nil {
		return empty, err
	}
	if manifest.Name != name {
		return empty, errors.New("compose-candidate resource identity does not match manifest")
	}
	target, _, err := composeCandidateResolveTarget(ctx, manifest)
	if err != nil {
		return empty, err
	}
	return composeCandidate(ctx, manifest, raw, target, resourceRoot, output, func(ctx context.Context, target binaryfetch.AcquisitionTarget, resourceRoot, artifactRoot string) error {
		return composeManagedServiceArtifactWithCache(ctx, name, target, resourceRoot, artifactRoot)
	})
}

func composeCandidate(ctx context.Context, manifest ResourceManifest, rawManifest []byte, target binaryfetch.AcquisitionTarget, resourceRoot, output string, compose composeArtifactFunc) (ComposeCandidateReceipt, error) {
	var empty ComposeCandidateReceipt
	started := time.Now()
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if !composeCandidatePlatformSupported() {
		return empty, errors.New("compose-candidate currently supports Linux only")
	}
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil || manifest.ManagedService.Acquisition.EffectiveKind(target) != "composed" || target.Unsupported != "" {
		return empty, errors.New("compose-candidate requires a supported composed target")
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return empty, err
	}
	if _, err := os.Lstat(output); err == nil {
		return empty, errors.New("candidate output already exists")
	} else if !os.IsNotExist(err) {
		return empty, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(output), ".vrooli-compose-candidate-")
	if err != nil {
		return empty, err
	}
	defer os.RemoveAll(stage)
	snapshot := filepath.Join(stage, "inputs")
	if err := os.Mkdir(snapshot, 0o700); err != nil {
		return empty, err
	}
	hashes := map[string]string{"resource.json": fmt.Sprintf("%x", sha256.Sum256(rawManifest))}
	if err := os.WriteFile(filepath.Join(snapshot, "resource.json"), rawManifest, 0o600); err != nil {
		return empty, err
	}
	type destination struct{ path, kind string }
	destinations := []destination{}
	for _, step := range target.Compose {
		if err := step.Validate(); err != nil {
			return empty, err
		}
		dest := filepath.Clean(filepath.FromSlash(step.Dest))
		if !filepath.IsLocal(dest) || dest == "." {
			return empty, errors.New("candidate destination must be a nonempty relative path")
		}
		for _, prior := range destinations {
			// Ordered local file overlays into an earlier archive tree are the
			// canonical wrapper pattern. Other overlaps can remove prior inputs.
			childOverlay := prior.kind == "url" && step.Kind == "local" && strings.HasPrefix(dest, prior.path+string(filepath.Separator))
			if !childOverlay && (prior.path == dest || strings.HasPrefix(prior.path, dest+string(filepath.Separator)) || strings.HasPrefix(dest, prior.path+string(filepath.Separator))) {
				return empty, fmt.Errorf("candidate compose destinations overlap: %s and %s", prior.path, dest)
			}
		}
		destinations = append(destinations, destination{dest, step.Kind})
		switch step.Kind {
		case "url":
			if step.BinPath != "" && (!filepath.IsLocal(filepath.FromSlash(step.BinPath)) || filepath.Clean(step.BinPath) == ".") {
				return empty, errors.New("candidate archive prefix must remain within extracted tree")
			}
			if err := candidatePublicURL(step.URL); err != nil {
				return empty, err
			}
		case "local":
			if err := snapshotComposeInput(ctx, resourceRoot, snapshot, step.Source, hashes); err != nil {
				return empty, err
			}
		case "python-wheels":
			if step.AllowSDists {
				return empty, errors.New("compose-candidate forbids source distributions")
			}
			if step.IndexURL == "" {
				return empty, errors.New("compose-candidate requires an explicit wheel index")
			}
			for _, index := range append([]string{step.IndexURL}, step.ExtraIndexURLs...) {
				if err := candidatePublicURL(index); err != nil {
					return empty, err
				}
			}
			if err := snapshotComposeInput(ctx, resourceRoot, snapshot, step.Lockfile, hashes); err != nil {
				return empty, err
			}
			lock, err := os.ReadFile(filepath.Join(snapshot, filepath.FromSlash(step.Lockfile)))
			if err != nil {
				return empty, err
			}
			if len(lock) == 0 || !strings.Contains(string(lock), "--hash=sha256:") {
				return empty, errors.New("compose-candidate requires a nonempty hashed wheel lock")
			}
		default:
			return empty, fmt.Errorf("unsupported candidate step %q", step.Kind)
		}
	}
	// Check before any archive download/wheel staging as well as before publication.
	if err := checkComposeInputs(ctx, resourceRoot, hashes); err != nil {
		return empty, err
	}
	artifactRoot := filepath.Join(stage, "artifact")
	if err := compose(ctx, target, snapshot, artifactRoot); err != nil {
		return empty, err
	}
	digest, err := binaryfetch.TreeDigestContext(ctx, artifactRoot)
	if err != nil {
		return empty, err
	}
	if err := checkComposeInputs(ctx, resourceRoot, hashes); err != nil {
		return empty, err
	}
	receipt := ComposeCandidateReceipt{Status: "candidate-only", Resource: manifest.Name, Platform: runtime.GOOS + "-" + runtime.GOARCH, ArtifactSHA256: digest, Inputs: hashes, Target: target, ElapsedMS: time.Since(started).Milliseconds()}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return empty, err
	}
	if err := os.WriteFile(filepath.Join(stage, "receipt.json"), append(encoded, '\n'), 0o600); err != nil {
		return empty, err
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if err := publishComposeCandidate(stage, output); err != nil {
		return empty, fmt.Errorf("publish candidate without replacement: %w", err)
	}
	return receipt, nil
}

func candidatePublicURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("candidate sources/indexes require HTTPS without credentials, queries or fragments")
	}
	return nil
}

func candidateInputRelative(relative string) (string, error) {
	path := filepath.Clean(filepath.FromSlash(relative))
	if !filepath.IsLocal(path) || path == "." {
		return "", errors.New("candidate input must remain under resource root")
	}
	return filepath.ToSlash(path), nil
}

func candidateInputPath(root, relative string) (string, error) {
	relative, err := candidateInputRelative(relative)
	if err != nil {
		return "", err
	}
	path := filepath.FromSlash(relative)
	current := root
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("candidate input symlink refused: %s", relative)
		}
	}
	info, err := os.Lstat(current)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("candidate input must be regular: %s", relative)
	}
	return current, nil
}

func snapshotComposeInput(ctx context.Context, resourceRoot, snapshot, relative string, hashes map[string]string) error {
	var err error
	relative, err = candidateInputRelative(relative)
	if err != nil {
		return err
	}
	if _, exists := hashes[relative]; exists {
		return nil
	}
	destination := filepath.Join(snapshot, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	err = copyCandidateInput(ctx, resourceRoot, relative, io.MultiWriter(out, hasher))
	closeErr := out.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	hashes[relative] = fmt.Sprintf("%x", hasher.Sum(nil))
	return nil
}

func checkComposeInputs(ctx context.Context, resourceRoot string, hashes map[string]string) error {
	for relative, expected := range hashes {
		hasher := sha256.New()
		if err := copyCandidateInput(ctx, resourceRoot, relative, hasher); err != nil {
			return fmt.Errorf("candidate input %s changed: %w", relative, err)
		}
		actual := fmt.Sprintf("%x", hasher.Sum(nil))
		if actual != expected {
			return fmt.Errorf("candidate input %s changed during preparation", relative)
		}
	}
	return nil
}

// Every read uses the same rooted nonblocking regular-file contract, including
// rechecks. Refuse replacement or in-place mutation observed during a copy.
func copyCandidateInput(ctx context.Context, root, relative string, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := candidateInputPath(root, relative)
	if err != nil {
		return err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return err
	}
	input, err := openCandidateInput(root, relative)
	if err != nil {
		return err
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return errors.New("candidate input replaced before open")
	}
	buffer := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := input.Read(buffer)
		if err := ctx.Err(); err != nil {
			return err
		}
		if n > 0 {
			if _, err := output.Write(buffer[:n]); err != nil {
				return err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}
	after, err := input.Stat()
	if err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !os.SameFile(opened, current) || !os.SameFile(opened, after) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) || after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) {
		return errors.New("candidate input changed while reading")
	}
	return nil
}
