package bundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

// PackageComponent exports a real, previously packaged dependency without its
// Electron shell, runtime or control-plane shim. The release producer supplies
// metadata and its signing authority after the exact bytes are known.
func PackageComponent(source, archivePath string, entry component.Entry) (component.Entry, error) {
	raw, err := os.ReadFile(filepath.Join(source, "bundle.json"))
	if err != nil {
		return entry, err
	}
	var bundle manifest.Manifest
	if err = json.Unmarshal(raw, &bundle); err != nil {
		return entry, err
	}
	if entry.Owner != bundle.App.Scenario || entry.Version != bundle.App.Version {
		return entry, fmt.Errorf("component identity must match the packaged owner release")
	}
	bundle.Services, err = manifest.ProjectComponent(entry.Owner, bundle.Services)
	if err != nil {
		return entry, err
	}
	bundle.Components = nil
	// Installed features are declared by the signed entry. A component must
	// never inherit the owner's full-application peer advertisements.
	bundle.Features = nil
	tree, err := os.MkdirTemp(filepath.Dir(archivePath), ".component-export-")
	if err != nil {
		return entry, err
	}
	defer os.RemoveAll(tree)
	peerRoot := filepath.Join(tree, "peers", entry.Owner)
	copyPath := func(relative string) error {
		if !filepath.IsLocal(filepath.FromSlash(relative)) {
			return fmt.Errorf("component input escapes its bundle")
		}
		src := filepath.Join(source, filepath.FromSlash(relative))
		target := filepath.Join(peerRoot, filepath.FromSlash(relative))
		return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			dest := filepath.Join(target, rel)
			info, err := d.Info()
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("component input symlink is not allowed")
			}
			if d.IsDir() {
				return os.MkdirAll(dest, 0o755)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("unsupported component input")
			}
			if err = os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
			if err != nil {
				return err
			}
			_, err = io.Copy(out, in)
			closeErr := out.Close()
			if err != nil {
				return err
			}
			return closeErr
		})
	}
	var original manifest.Manifest
	_ = json.Unmarshal(raw, &original)
	copied := map[string]bool{}
	copyOnce := func(path string) error {
		if copied[path] {
			return nil
		}
		copied[path] = true
		return copyPath(path)
	}
	for _, svc := range original.Services {
		for _, binary := range svc.Binaries {
			if _, statErr := os.Stat(filepath.Join(source, filepath.FromSlash(binary.Path))); statErr == nil {
				if err = copyOnce(binary.Path); err != nil {
					return entry, err
				}
			}
		}
		for _, asset := range svc.Assets {
			if err = copyOnce(asset.Path); err != nil {
				return entry, err
			}
		}
		for _, dir := range svc.AssetDirs {
			if err = copyOnce(dir); err != nil {
				return entry, err
			}
		}
		if svc.DistRoot != "" {
			if err = copyOnce(svc.DistRoot); err != nil {
				return entry, err
			}
		}
	}
	// The owner catalog is read-only metadata, never a second runtime. Preserve
	// its existing contract root while executable/service paths use peers/<owner>.
	if _, statErr := os.Stat(filepath.Join(source, "catalog")); statErr == nil {
		if err = copyOnce("catalog"); err != nil {
			return entry, err
		}
		if err = os.Rename(filepath.Join(peerRoot, "catalog"), filepath.Join(tree, "catalog")); err != nil {
			return entry, err
		}
	}
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return entry, err
	}
	if err = os.WriteFile(filepath.Join(tree, "bundle.json"), encoded, 0o644); err != nil {
		return entry, err
	}
	entry.Manifest = encoded
	entry.TreeSHA256, err = binaryfetch.TreeDigest(tree)
	if err != nil {
		return entry, err
	}
	archive, err := os.OpenFile(archivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return entry, err
	}
	hash := sha256.New()
	gzipWriter := gzip.NewWriter(io.MultiWriter(archive, hash))
	tarWriter := tar.NewWriter(gzipWriter)
	err = filepath.WalkDir(tree, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == tree {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(tree, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(tarWriter, in)
		return err
	})
	if closeErr := tarWriter.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gzipWriter.Close(); err == nil {
		err = closeErr
	}
	if closeErr := archive.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return entry, err
	}
	stat, err := os.Stat(archivePath)
	if err != nil {
		return entry, err
	}
	entry.Size = stat.Size()
	entry.SHA256 = hex.EncodeToString(hash.Sum(nil))
	for i := range entry.Provides {
		if !strings.HasPrefix(entry.Provides[i].Service, entry.Owner+"--") {
			entry.Provides[i].Service = entry.Owner + "--" + entry.Provides[i].Service
		}
	}
	if _, err = entry.CanonicalBytes(); err != nil {
		return entry, err
	}
	return entry, nil
}
