package retirement

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"react-component-library/internal/components"
)

// Result always returns completed preservation paths, including on failure.
// Retired reports a committed withdrawal; an accompanying error means durable
// completion bookkeeping still requires recovery.
type Result struct {
	ComponentID        string
	LibraryID          string
	Preflight          PreflightResult
	Archive            components.RetirementArchive
	CatalogArchivePath string
	Retired            bool
}

// Retire coordinates withdrawal of an indexed asset after consumer preflight.
// Registry removal is last; filesystem moves are restored if that step fails.
// Snapshot evidence is never removed during rollback.
func Retire(ctx context.Context, root string, repo components.Repository, libraryID string) (Result, error) {
	out := Result{LibraryID: libraryID}
	ctx, releaseMutation, err := components.AcquireLibraryRetirement(ctx, libraryRoot(root))
	if err != nil {
		return out, err
	}
	defer releaseMutation()
	recovered, hadPending, err := recoverPending(ctx, root, repo)
	if err != nil {
		return recovered, err
	}
	if hadPending && recovered.Retired && recovered.LibraryID == libraryID {
		return recovered, nil
	}
	registry, ok := repo.(components.RetirementRepository)
	if !ok {
		return out, fmt.Errorf("repository does not support targeted retirement")
	}
	c, err := repo.GetByLibraryID(ctx, libraryID)
	if err != nil {
		return out, err
	}
	out.ComponentID = c.ID
	out.Preflight, err = Preflight(ctx, root, c)
	if err != nil {
		return out, err
	}
	if !out.Preflight.Ready() {
		return out, fmt.Errorf("asset retirement has catalog or source consumers")
	}
	catalogPath := out.Preflight.CatalogPath
	catalogRoot := filepath.Join(root, "scenarios/react-component-library/catalog/assets")
	if err := verifyCatalogPath(catalogRoot, catalogPath); err != nil {
		return out, err
	}
	catalogRelative, err := filepath.Rel(catalogRoot, catalogPath)
	if err != nil {
		return out, err
	}
	if err := persistJSON(pendingPath(root), pendingRetirement{Component: c, CatalogRelative: catalogRelative}); err != nil {
		return out, err
	}
	failure := func(cause error) (Result, error) {
		_, _, recoveryErr := recoverPending(context.WithoutCancel(ctx), root, repo)
		return out, errors.Join(cause, recoveryErr)
	}
	store := components.NewFSContentStore(libraryRoot(root))
	out.Archive, err = components.ArchiveRetirementHistory(ctx, repo, store, c)
	if err != nil {
		return failure(err)
	}
	activeSource := filepath.Dir(filepath.Join(libraryRoot(root), c.ManifestPath))
	catalogArchive := out.Archive.SourceArchivePath + ".catalog.json"
	if err = ctx.Err(); err == nil {
		err = moveExclusive(catalogPath, catalogArchive)
	}
	if err != nil {
		return failure(err)
	}
	out.CatalogArchivePath = catalogArchive
	for _, dir := range []string{filepath.Dir(activeSource), filepath.Dir(out.Archive.SourceArchivePath), filepath.Dir(catalogPath)} {
		if err := syncDirectory(dir); err != nil {
			return failure(err)
		}
	}
	if err = registry.DeleteRetiredComponent(ctx, c.ID, c.LibraryID); err != nil {
		return failure(err)
	}
	out.Retired = true
	if err := persistJSON(out.Archive.SourceArchivePath+".receipt.json", out); err != nil {
		return out, err
	}
	if err := clearPending(root); err != nil {
		return out, err
	}
	return out, nil
}

// The catalog loader supplies the path, but withdrawal additionally requires
// containment and real directories/files, rather than followed symlinks.
func verifyCatalogPath(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || !filepath.IsLocal(rel) {
		return fmt.Errorf("retirement catalog path escapes assets: %s", path)
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("retirement refuses symlink catalog path: %s", current)
		}
		if current == path && !info.Mode().IsRegular() {
			return fmt.Errorf("retirement catalog declaration is not a regular file")
		}
		if current == root {
			break
		}
	}
	return nil
}

func moveExclusive(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("retirement destination already exists: %s", to)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(from, to)
}

func restoreMove(from, to string) error {
	if err := moveExclusive(from, to); err != nil {
		return fmt.Errorf("restore %s from %s: %w", to, from, err)
	}
	return nil
}
