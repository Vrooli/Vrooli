package resources

import (
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

// ComposeCandidate stages reviewed compose inputs without installing, executing
// the target, acquiring model data or publishing any runtime verification facts.
func (c *Controller) ComposeCandidate(ctx context.Context, name, output string) (ComposeCandidateReceipt, error) {
	var empty ComposeCandidateReceipt
	if !composeCandidatePlatformSupported() {
		return empty, errors.New("compose-candidate currently supports Linux only")
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\") {
		return empty, errors.New("compose-candidate requires one resource name")
	}
	resourceRoot := filepath.Join(c.Root, "resources", name)
	manifest, raw, err := manifestpkg.LoadForComposeCandidate(filepath.Join(resourceRoot, "resource.json"))
	if err != nil {
		return empty, err
	}
	if manifest.Name != name {
		return empty, errors.New("compose-candidate resource identity does not match manifest")
	}
	target, _, err := managedServiceAcquisitionTargetWithFacts(ctx, manifest)
	if err != nil {
		return empty, err
	}
	return composeCandidate(ctx, manifest, raw, target, resourceRoot, output, composeManagedServiceArtifact)
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
	if err := os.Mkdir(snapshot, 0700); err != nil {
		return empty, err
	}
	hashes := map[string]string{"resource.json": fmt.Sprintf("%x", sha256.Sum256(rawManifest))}
	if err := os.WriteFile(filepath.Join(snapshot, "resource.json"), rawManifest, 0600); err != nil {
		return empty, err
	}
	destinations := []string{}
	for _, step := range target.Compose {
		if err := step.Validate(); err != nil {
			return empty, err
		}
		dest := filepath.Clean(filepath.FromSlash(step.Dest))
		if !filepath.IsLocal(dest) || dest == "." {
			return empty, errors.New("candidate destination must be a nonempty relative path")
		}
		for _, prior := range destinations {
			if prior == dest || strings.HasPrefix(prior, dest+string(filepath.Separator)) || strings.HasPrefix(dest, prior+string(filepath.Separator)) {
				return empty, fmt.Errorf("candidate compose destinations overlap: %s and %s", prior, dest)
			}
		}
		destinations = append(destinations, dest)
		switch step.Kind {
		case "url":
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
	if err := os.WriteFile(filepath.Join(stage, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
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

func candidateInputPath(root, relative string) (string, error) {
	path := filepath.FromSlash(relative)
	if !filepath.IsLocal(path) || path == "." {
		return "", errors.New("candidate input must remain under resource root")
	}
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
	relative = filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
	if _, exists := hashes[relative]; exists {
		return nil
	}
	path, err := candidateInputPath(resourceRoot, relative)
	if err != nil {
		return err
	}
	destination := filepath.Join(snapshot, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	buffer := make([]byte, 32*1024)
	for {
		if err = ctx.Err(); err != nil {
			break
		}
		n, readErr := in.Read(buffer)
		if n > 0 {
			if _, err = io.MultiWriter(out, hasher).Write(buffer[:n]); err != nil {
				break
			}
		}
		if readErr == io.EOF {
			err = nil
			break
		}
		if readErr != nil {
			err = readErr
			break
		}
	}
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
		path, err := candidateInputPath(resourceRoot, relative)
		if err != nil {
			return fmt.Errorf("candidate input %s changed: %w", relative, err)
		}
		actual, err := managedRegularFileSHA256(ctx, path)
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("candidate input %s changed during preparation", relative)
		}
	}
	return nil
}
