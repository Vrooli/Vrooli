package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// resolveScenarioRoot returns the absolute scenario directory.
func resolveScenarioRoot(scenario string) string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return ""
	}
	root := resolveRepoRoot()
	if root == "" {
		return ""
	}
	resolved, err := repocontract.ResolveScenarioPath(root, scenario)
	if err != nil {
		return ""
	}
	return resolved
}

func resolveRepoRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

// populateAssetMetadata fills in missing/pending asset hashes and sizes using files on disk.
func populateAssetMetadata(manifest *Manifest, scenarioRoot string) error {
	if manifest == nil {
		return nil
	}
	_ = expandUIAssets(manifest, scenarioRoot)

	for svcIdx, svc := range manifest.Services {
		for assetIdx, asset := range svc.Assets {
			if asset.SHA256 != "" && asset.SHA256 != "pending" && asset.SizeBytes > 0 {
				continue
			}
			abs := filepath.Join(scenarioRoot, filepath.FromSlash(asset.Path))
			info, err := os.Stat(abs)
			if err != nil || info.IsDir() {
				continue
			}
			hash, size, hErr := hashFile(abs)
			if hErr != nil {
				continue
			}
			manifest.Services[svcIdx].Assets[assetIdx].SHA256 = hash
			manifest.Services[svcIdx].Assets[assetIdx].SizeBytes = size
		}
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func expandUIAssets(manifest *Manifest, scenarioRoot string) error {
	if manifest == nil {
		return nil
	}
	var firstErr error
	for svcIdx, svc := range manifest.Services {
		if !strings.EqualFold(svc.Type, "ui-bundle") {
			continue
		}
		uiRoot := filepath.Join(scenarioRoot, "ui", "dist")
		entries, err := os.ReadDir(uiRoot)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read ui dist: %w", err)
			}
			continue
		}
		var assets []Asset
		for _, entry := range entries {
			err := filepath.WalkDir(filepath.Join(uiRoot, entry.Name()), func(path string, d os.DirEntry, werr error) error {
				if werr != nil {
					return werr
				}
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(scenarioRoot, path)
				hash, size, herr := hashFile(path)
				if herr != nil {
					if firstErr == nil {
						firstErr = herr
					}
					return nil
				}
				assets = append(assets, Asset{
					Path:      filepath.ToSlash(rel),
					SHA256:    hash,
					SizeBytes: size,
				})
				return nil
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if len(assets) > 0 {
			manifest.Services[svcIdx].Assets = assets
		}
	}
	return firstErr
}

func stageBundleArtifacts(manifest *Manifest, scenarioRoot, outputDir string) (stageBundleResult, error) {
	if manifest == nil {
		return stageBundleResult{}, fmt.Errorf("manifest is nil")
	}
	if strings.TrimSpace(outputDir) == "" {
		return stageBundleResult{}, fmt.Errorf("output_dir is required for bundle staging")
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return stageBundleResult{}, fmt.Errorf("resolve output_dir: %w", err)
	}
	if err := os.MkdirAll(absOutput, 0o755); err != nil {
		return stageBundleResult{}, fmt.Errorf("create output_dir: %w", err)
	}

	var missing []string
	seen := make(map[string]bool)

	copyEntry := func(relPath string, isBinary bool) error {
		cleanRel, err := sanitizeRelPath(relPath)
		if err != nil {
			return err
		}
		if seen[cleanRel] {
			return nil
		}
		seen[cleanRel] = true
		src := filepath.Join(scenarioRoot, cleanRel)
		info, err := os.Stat(src)
		if err != nil {
			missing = append(missing, cleanRel)
			return nil
		}
		dest := filepath.Join(absOutput, cleanRel)
		if info.IsDir() {
			if isBinary {
				return fmt.Errorf("binary path is a directory: %s", cleanRel)
			}
			return copyDir(src, dest)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return copyFilePreserveMode(src, dest)
	}

	for _, svc := range manifest.Services {
		for _, bin := range svc.Binaries {
			if bin.Path == "" {
				continue
			}
			if err := copyEntry(bin.Path, true); err != nil {
				return stageBundleResult{}, err
			}
		}
		for _, asset := range svc.Assets {
			if asset.Path == "" {
				continue
			}
			if err := copyEntry(asset.Path, false); err != nil {
				return stageBundleResult{}, err
			}
		}
	}

	manifestPath := filepath.Join(absOutput, "bundle.json")
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return stageBundleResult{}, fmt.Errorf("serialize manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
		return stageBundleResult{}, fmt.Errorf("write bundle.json: %w", err)
	}

	return stageBundleResult{
		ManifestPath: manifestPath,
		Missing:      missing,
	}, nil
}

func sanitizeRelPath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("path must be relative: %s", relPath)
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid path: %s", relPath)
	}
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("path escapes root: %s", relPath)
	}
	return clean, nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFilePreserveMode(path, target)
	})
}

func copyFilePreserveMode(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(src); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}
