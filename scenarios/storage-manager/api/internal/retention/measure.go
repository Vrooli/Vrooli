package retention

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	coreStorage "github.com/vrooli/api-core/storage"
)

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
