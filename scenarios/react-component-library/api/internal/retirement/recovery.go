package retirement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
	"react-component-library/internal/components"
)

type pendingRetirement struct {
	Component       components.Component
	CatalogRelative string
}

func libraryRoot(root string) string {
	return filepath.Join(root, "scenarios/react-component-library/library")
}
func pendingPath(root string) string {
	return filepath.Join(libraryRoot(root), components.RetirementPendingFile)
}
func archivePaths(root string, c components.Component) components.RetirementArchive {
	key := sha256.Sum256([]byte("asset:" + c.LibraryID))
	base := filepath.Join(libraryRoot(root), ".retired", "asset-"+hex.EncodeToString(key[:]))
	return components.RetirementArchive{SourceArchivePath: base, SnapshotPath: base + ".snapshot.json"}
}

func persistJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(path, append(data, '\n'), 0600); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func clearPending(root string) error {
	if err := os.Remove(pendingPath(root)); err != nil {
		return err
	}
	return syncDirectory(libraryRoot(root))
}

// recoverPending runs under the retirement lock. A registry row means the
// transaction has not committed, so restore source and catalog. An absent row
// means it committed; require all archives before recording completion.
func recoverPending(ctx context.Context, root string, repo components.Repository) (Result, bool, error) {
	var out Result
	info, err := os.Lstat(pendingPath(root))
	if os.IsNotExist(err) {
		return out, false, nil
	}
	if err != nil {
		return out, true, err
	}
	if !info.Mode().IsRegular() {
		return out, true, fmt.Errorf("retirement journal is not a regular file")
	}
	data, err := os.ReadFile(pendingPath(root))
	if err != nil {
		return out, true, err
	}
	var pending pendingRetirement
	if err := json.Unmarshal(data, &pending); err != nil {
		return out, true, err
	}
	c := pending.Component
	parts := strings.Split(c.ManifestPath, "/")
	if c.ID == "" || c.LibraryID == "" || c.CatalogID == "" || len(parts) != 3 || parts[1] != c.Slug || c.Slug == "" || strings.HasPrefix(parts[0], ".") || parts[2] != "component.json" || !filepath.IsLocal(c.ManifestPath) || filepath.ToSlash(filepath.Clean(c.ManifestPath)) != c.ManifestPath || strings.Contains(c.ManifestPath, "\\") || !filepath.IsLocal(pending.CatalogRelative) {
		return out, true, fmt.Errorf("invalid retirement journal identity or paths")
	}
	catalogRoot := filepath.Join(root, "scenarios/react-component-library/catalog/assets")
	catalog := filepath.Join(catalogRoot, pending.CatalogRelative)
	active := filepath.Dir(filepath.Join(libraryRoot(root), c.ManifestPath))
	out = Result{ComponentID: c.ID, LibraryID: c.LibraryID, Archive: archivePaths(root, c)}
	out.CatalogArchivePath = out.Archive.SourceArchivePath + ".catalog.json"
	for _, path := range []string{active, catalog, out.Archive.SourceArchivePath} {
		for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
			info, err := os.Lstat(parent)
			// The archive directory need not exist if interrupted before snapshotting.
			if os.IsNotExist(err) && parent == filepath.Join(libraryRoot(root), ".retired") {
				continue
			}
			if err != nil {
				return out, true, err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return out, true, fmt.Errorf("unsafe recovery parent: %s", parent)
			}
			if parent == filepath.Clean(root) {
				break
			}
			if parent == filepath.Dir(parent) {
				return out, true, fmt.Errorf("recovery path escapes repository")
			}
		}
	}
	current, lookupErr := repo.GetByLibraryID(ctx, c.LibraryID)
	var notFound components.ErrComponentNotFound
	if lookupErr != nil && !errors.As(lookupErr, &notFound) {
		return out, true, lookupErr
	}
	if lookupErr == nil {
		if current.ID != c.ID {
			return out, true, fmt.Errorf("retirement recovery registry identity changed")
		}
		if err := recoverMove(out.Archive.SourceArchivePath, active); err != nil {
			return out, true, err
		}
		if err := recoverMove(out.CatalogArchivePath, catalog); err != nil {
			return out, true, err
		}
		if _, err := verifiedAssetRoot(root, c); err != nil {
			return out, true, err
		}
		if err := verifyCatalogPath(catalogRoot, catalog); err != nil {
			return out, true, err
		}
	} else {
		for _, path := range []string{active, catalog} {
			if _, err := os.Lstat(path); !os.IsNotExist(err) {
				return out, true, fmt.Errorf("committed retirement still has active path %s: %v", path, err)
			}
		}
		for _, path := range []string{out.Archive.SourceArchivePath, out.Archive.SnapshotPath, out.CatalogArchivePath} {
			info, err := os.Lstat(path)
			if err != nil {
				return out, true, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return out, true, fmt.Errorf("retirement archive is a symlink: %s", path)
			}
		}
		out.Retired = true
		var snapshot struct{ Component components.Component }
		data, err := os.ReadFile(out.Archive.SnapshotPath)
		if err != nil {
			return out, true, err
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return out, true, err
		}
		if snapshot.Component.ID != c.ID || snapshot.Component.LibraryID != c.LibraryID {
			return out, true, fmt.Errorf("retirement snapshot identity mismatch")
		}
		var manifest struct {
			LibraryID string `json:"libraryId"`
			CatalogID string `json:"catalogId"`
		}
		data, err = os.ReadFile(filepath.Join(out.Archive.SourceArchivePath, "component.json"))
		if err != nil {
			return out, true, err
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return out, true, err
		}
		if manifest.LibraryID != c.LibraryID || manifest.CatalogID != c.CatalogID {
			return out, true, fmt.Errorf("retirement source archive identity mismatch")
		}
		var catalogDocument struct {
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
		}
		data, err = os.ReadFile(out.CatalogArchivePath)
		if err != nil {
			return out, true, err
		}
		if err := json.Unmarshal(data, &catalogDocument); err != nil {
			return out, true, err
		}
		if catalogDocument.Asset.ID != c.CatalogID {
			return out, true, fmt.Errorf("retirement catalog archive identity mismatch")
		}
		if err := persistJSON(out.Archive.SourceArchivePath+".receipt.json", out); err != nil {
			return out, true, err
		}
	}
	return out, true, clearPending(root)
}

func recoverMove(archive, active string) error {
	archiveInfo, archiveErr := os.Lstat(archive)
	activeInfo, activeErr := os.Lstat(active)
	if (archiveErr == nil && archiveInfo.Mode()&os.ModeSymlink != 0) || (activeErr == nil && activeInfo.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("retirement recovery refuses symlink source or destination")
	}
	if os.IsNotExist(archiveErr) && activeErr == nil {
		return nil
	}
	if archiveErr == nil && os.IsNotExist(activeErr) {
		if err := restoreMove(archive, active); err != nil {
			return err
		}
		return errors.Join(syncDirectory(filepath.Dir(archive)), syncDirectory(filepath.Dir(active)))
	}
	return fmt.Errorf("ambiguous retirement recovery: archive %s (%v), active %s (%v)", archive, archiveErr, active, activeErr)
}
