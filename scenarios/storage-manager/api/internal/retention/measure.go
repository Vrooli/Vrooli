package retention

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	coreStorage "github.com/vrooli/api-core/storage"
)

// measureEntryBounded uses streaming directory reads so neither a large tree
// nor a single large directory can monopolize the recovery lock. Partial
// counts are discarded on every error or limit; they are not usage totals.
func measureEntryBounded(ctx context.Context, repoRoot string, owner coreStorage.OwnerManifest, entry coreStorage.StorageEntry, platform coreStorage.Platform, limit int) (EntryMeasurement, error) {
	locations, err := coreStorage.ResolveOwnerStorageLocations(repoRoot, owner, entry, platform, coreStorage.PlatformSeams{})
	if err != nil {
		return EntryMeasurement{}, err
	}
	exclude := nestedSiblingLocations(repoRoot, owner, entry, platform, locations)
	out := EntryMeasurement{Locations: make([]LocationMeasurement, 0, len(locations))}
	visited := 0
	for _, root := range locations {
		if err := ctx.Err(); err != nil {
			return EntryMeasurement{}, err
		}
		info, err := os.Lstat(root)
		if errors.Is(err, fs.ErrNotExist) {
			out.Locations = append(out.Locations, LocationMeasurement{Path: root})
			continue
		}
		if err != nil {
			return EntryMeasurement{}, err
		}
		visited++
		if visited > limit {
			return EntryMeasurement{}, fmt.Errorf("measurement entry limit %d exceeded", limit)
		}
		var bytes int64
		if info.Mode().IsRegular() {
			bytes = info.Size()
		}
		pending := []string{}
		if info.IsDir() {
			pending = append(pending, root)
		}
		for len(pending) > 0 {
			if err := ctx.Err(); err != nil {
				return EntryMeasurement{}, err
			}
			dir := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			f, err := os.Open(dir)
			if err != nil {
				return EntryMeasurement{}, err
			}
			for {
				if err = ctx.Err(); err != nil {
					break
				}
				entries, readErr := f.ReadDir(128)
				for _, child := range entries {
					if err = ctx.Err(); err != nil {
						break
					}
					visited++
					if visited > limit {
						err = fmt.Errorf("measurement entry limit %d exceeded", limit)
						break
					}
					path := filepath.Join(dir, child.Name())
					if _, skip := exclude[filepath.Clean(path)]; skip {
						continue
					}
					var childInfo fs.FileInfo
					childInfo, err = child.Info()
					if err != nil {
						break
					}
					if childInfo.IsDir() {
						pending = append(pending, path)
					} else if childInfo.Mode().IsRegular() {
						bytes += childInfo.Size()
					}
				}
				if err != nil {
					break
				}
				if readErr != nil {
					err = readErr
					break
				}
			}
			closeErr := f.Close()
			if err != nil && !errors.Is(err, io.EOF) {
				return EntryMeasurement{}, err
			}
			if closeErr != nil {
				return EntryMeasurement{}, closeErr
			}
		}
		out.Bytes += bytes
		out.Locations = append(out.Locations, LocationMeasurement{Path: root, Bytes: bytes, Exists: true})
	}
	if err := ctx.Err(); err != nil {
		return EntryMeasurement{}, err
	}
	return out, nil
}

// LocationMeasurement is one physical location of an entry and its bytes.
type LocationMeasurement struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Exists bool   `json:"exists"`
}

// EntryMeasurement is an entry's bytes summed across every location that can
// hold them.
type EntryMeasurement struct {
	Bytes     int64                 `json:"bytes"`
	Locations []LocationMeasurement `json:"locations"`
}

// MeasureEntry measures an owner entry at every location
// ResolveOwnerStorageLocations names. It is the accounting authority for
// non-regenerable budgets and the inspect surface, so both report the bytes an
// owner actually holds rather than whichever single root one resolver picked.
//
// Bytes that belong to a more specific sibling entry of the same owner are
// excluded, so a whole-directory entry and a subpath entry inside it never
// count the same file twice. Measurement never deletes anything.
func MeasureEntry(repoRoot string, owner coreStorage.OwnerManifest, entry coreStorage.StorageEntry, platform coreStorage.Platform) (EntryMeasurement, error) {
	locations, err := coreStorage.ResolveOwnerStorageLocations(repoRoot, owner, entry, platform, coreStorage.PlatformSeams{})
	if err != nil {
		return EntryMeasurement{}, err
	}
	exclude := nestedSiblingLocations(repoRoot, owner, entry, platform, locations)
	out := EntryMeasurement{Locations: make([]LocationMeasurement, 0, len(locations))}
	for _, location := range locations {
		bytes, exists, err := sizeExcluding(location, exclude)
		if err != nil {
			return EntryMeasurement{}, fmt.Errorf("measure %s: %w", location, err)
		}
		out.Bytes += bytes
		out.Locations = append(out.Locations, LocationMeasurement{Path: location, Bytes: bytes, Exists: exists})
	}
	return out, nil
}

func nestedSiblingLocations(repoRoot string, owner coreStorage.OwnerManifest, entry coreStorage.StorageEntry, platform coreStorage.Platform, parents []string) map[string]struct{} {
	exclude := map[string]struct{}{}
	for _, sibling := range owner.StorageEntries {
		if sibling.Name == entry.Name {
			continue
		}
		paths, err := coreStorage.ResolveOwnerStorageLocations(repoRoot, owner, sibling, platform, coreStorage.PlatformSeams{})
		if err != nil {
			continue
		}
		for _, path := range paths {
			for _, parent := range parents {
				if strictlyWithin(path, parent) {
					exclude[filepath.Clean(path)] = struct{}{}
				}
			}
		}
	}
	return exclude
}

func strictlyWithin(path, parent string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(path))
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// sizeExcluding sums regular-file sizes under root without following
// symlinks. A missing location is zero bytes, not an error: an owner may hold
// data in only one of its locations.
func sizeExcluding(root string, exclude map[string]struct{}) (int64, bool, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return info.Size(), true, nil
		}
		return 0, true, nil
	}
	var total int64
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if _, skip := exclude[filepath.Clean(path)]; skip {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		fileInfo, infoErr := d.Info()
		if infoErr != nil {
			if errors.Is(infoErr, fs.ErrNotExist) {
				return nil
			}
			return infoErr
		}
		total += fileInfo.Size()
		return nil
	})
	return total, true, err
}
