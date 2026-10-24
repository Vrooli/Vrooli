package resourcedeployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/binaryfetch"
)

// ArtifactCacheKey identifies one resource build result. SourceIdentity must
// change whenever any local build input changes; Version and Platform keep
// artifacts from distinct upstream releases and targets isolated.
type ArtifactCacheKey struct {
	Resource       string   `json:"resource"`
	Version        string   `json:"version"`
	Platform       Platform `json:"platform"`
	SourceIdentity string   `json:"source_identity"`
}

// CachedArtifact describes one independently verified file in a cached step.
type CachedArtifact struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type artifactCacheManifest struct {
	SchemaVersion string           `json:"schema_version"`
	Key           ArtifactCacheKey `json:"key"`
	Artifacts     []CachedArtifact `json:"artifacts"`
}

const artifactCacheManifestName = "artifact-cache-v1.json"

var artifactCacheLocks sync.Map

// StageCachedArtifacts reuses a verified per-resource/platform staging step,
// or builds it into an isolated directory and atomically publishes it. The
// caller's output is a union directory: only the files in this key's verified
// manifest are copied there, so unrelated resource failures cannot block it.
func StageCachedArtifacts(ctx context.Context, cacheRoot, outputRoot string, key ArtifactCacheKey, build func(context.Context, string) error) ([]CachedArtifact, error) {
	if strings.TrimSpace(key.Resource) == "" || strings.TrimSpace(key.Version) == "" || strings.TrimSpace(key.SourceIdentity) == "" || key.Platform.OS == "" || key.Platform.Arch == "" {
		return nil, fmt.Errorf("resource artifact cache key requires resource, version, platform, and source identity")
	}
	if build == nil {
		return nil, fmt.Errorf("resource artifact cache build function is required")
	}
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("encode resource artifact cache key: %w", err)
	}
	keyDigest := sha256.Sum256(keyJSON)
	cacheDir := filepath.Join(cacheRoot, hex.EncodeToString(keyDigest[:]))
	lockValue, _ := artifactCacheLocks.LoadOrStore(cacheDir, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	if artifacts, ok := verifyArtifactCache(cacheDir, key); ok {
		if err := copyCachedArtifacts(cacheDir, outputRoot, artifacts); err != nil {
			return nil, err
		}
		return artifacts, nil
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create resource artifact cache: %w", err)
	}
	tmp, err := os.MkdirTemp(cacheRoot, ".resource-artifact-*")
	if err != nil {
		return nil, fmt.Errorf("create resource artifact cache staging directory: %w", err)
	}
	defer os.RemoveAll(tmp)
	if err := build(ctx, tmp); err != nil {
		return nil, err
	}
	artifacts, err := inventoryCachedArtifacts(tmp)
	if err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("resource artifact cache build produced no artifacts")
	}
	manifestData, err := json.MarshalIndent(artifactCacheManifest{SchemaVersion: "v1", Key: key, Artifacts: artifacts}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode resource artifact cache manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, artifactCacheManifestName), append(manifestData, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write resource artifact cache manifest: %w", err)
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return nil, fmt.Errorf("replace invalid resource artifact cache: %w", err)
	}
	if err := os.Rename(tmp, cacheDir); err != nil {
		return nil, fmt.Errorf("publish resource artifact cache: %w", err)
	}
	if err := copyCachedArtifacts(cacheDir, outputRoot, artifacts); err != nil {
		return nil, err
	}
	return artifacts, nil
}

func verifyArtifactCache(cacheDir string, key ArtifactCacheKey) ([]CachedArtifact, bool) {
	data, err := os.ReadFile(filepath.Join(cacheDir, artifactCacheManifestName))
	if err != nil {
		return nil, false
	}
	var manifest artifactCacheManifest
	if json.Unmarshal(data, &manifest) != nil || manifest.SchemaVersion != "v1" || manifest.Key != key || len(manifest.Artifacts) == 0 {
		return nil, false
	}
	for _, artifact := range manifest.Artifacts {
		if !safeCacheArtifactName(artifact.Name) {
			return nil, false
		}
		digest, err := cacheArtifactDigest(cacheDir, filepath.Join(cacheDir, filepath.FromSlash(artifact.Name)))
		if err != nil || digest != artifact.SHA256 {
			return nil, false
		}
	}
	return manifest.Artifacts, true
}

func inventoryCachedArtifacts(root string) ([]CachedArtifact, error) {
	var artifacts []CachedArtifact
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Name() == artifactCacheManifestName {
			return fmt.Errorf("resource artifact cache contains unsupported entry %q", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if entry.IsDir() && strings.Contains(name, "/") {
			return nil
		}
		digest, err := cacheArtifactDigest(root, path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, CachedArtifact{Name: name, SHA256: digest})
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inventory resource artifact cache: %w", err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Name < artifacts[j].Name })
	return artifacts, nil
}

func cacheArtifactDigest(root, path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(link)))
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return "", fmt.Errorf("cache symlink %q escapes its resource artifact", path)
		}
		if _, err := os.Stat(resolved); err != nil {
			return "", fmt.Errorf("cache symlink %q is dangling: %w", path, err)
		}
		sum := sha256.Sum256(append([]byte("symlink\x00"), []byte(link)...))
		return hex.EncodeToString(sum[:]), nil
	}
	if info.IsDir() {
		return binaryfetchTreeDigest(path)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("resource artifact cache contains unsupported entry %q", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyCachedArtifacts(cacheDir, outputRoot string, artifacts []CachedArtifact) error {
	for _, artifact := range artifacts {
		if !safeCacheArtifactName(artifact.Name) {
			return fmt.Errorf("unsafe cached artifact name %q", artifact.Name)
		}
		source := filepath.Join(cacheDir, filepath.FromSlash(artifact.Name))
		dest := filepath.Join(outputRoot, filepath.FromSlash(artifact.Name))
		if info, err := os.Lstat(source); err != nil {
			return err
		} else if info.IsDir() {
			// The output root is a union of independently staged artifacts. A
			// changed directory artifact must replace its prior tree so entries
			// removed by normalization cannot survive as stale bundle inputs.
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("replace staged resource artifact %s: %w", artifact.Name, err)
			}
		}
		if err := copyCacheEntry(cacheDir, source, dest); err != nil {
			return fmt.Errorf("copy cached resource artifact %s: %w", artifact.Name, err)
		}
	}
	return nil
}

func copyCacheEntry(root, source, dest string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		if _, err := cacheArtifactDigest(root, source); err != nil {
			return err
		}
		link, err := os.Readlink(source)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if existing, err := os.Lstat(dest); err == nil {
			if existing.Mode()&os.ModeSymlink != 0 {
				if existingTarget, readErr := os.Readlink(dest); readErr == nil && existingTarget == link {
					return nil
				}
			}
			if existing.IsDir() {
				return fmt.Errorf("cannot replace directory with cached symlink %q", dest)
			}
			if err := os.Remove(dest); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(link, dest)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dest, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyCacheEntry(root, filepath.Join(source, entry.Name()), filepath.Join(dest, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func safeCacheArtifactName(name string) bool {
	clean := filepath.Clean(filepath.FromSlash(name))
	return name != "" && filepath.ToSlash(clean) == name && clean != "." && !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

// binaryfetchTreeDigest is kept behind this package boundary so resource
// consumers get the same deterministic directory hashing as acquisition code.
func binaryfetchTreeDigest(path string) (string, error) {
	return binaryfetch.TreeDigest(path)
}
