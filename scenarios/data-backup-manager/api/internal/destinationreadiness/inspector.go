package destinationreadiness

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"data-backup-manager/internal/sysmounts"
)

const maxTopLevelEntries = 20

// VolumeScanner is the mounted-volume seam shared with discovery/sysmounts.
type VolumeScanner interface {
	Scan(ctx context.Context) ([]sysmounts.Volume, error)
}

// ReadOnlyInspector matches a path to its mounted volume and collects bounded,
// read-only content clues from the mount root.
type ReadOnlyInspector struct {
	Volumes VolumeScanner
	ReadDir func(name string) ([]os.DirEntry, error)
}

// NewReadOnlyInspector constructs the production read-only inspector.
func NewReadOnlyInspector(volumes VolumeScanner) *ReadOnlyInspector {
	return &ReadOnlyInspector{Volumes: volumes, ReadDir: os.ReadDir}
}

// Inspect returns a read-only snapshot of the candidate's mount and content
// state. It never creates, deletes, formats, relabels, or probes with writes.
func (i *ReadOnlyInspector) Inspect(ctx context.Context, location string) (Inspection, error) {
	location = cleanPath(location)
	if location == "" {
		return Inspection{}, ErrInvalidReadiness{Field: "location", Reason: "required"}
	}
	if i.Volumes == nil {
		return Inspection{}, ErrInvalidReadiness{Field: "volumes", Reason: "required"}
	}
	readDir := i.ReadDir
	if readDir == nil {
		readDir = os.ReadDir
	}
	volumes, err := i.Volumes.Scan(ctx)
	if err != nil {
		return Inspection{}, err
	}
	vol, ok := bestVolumeForLocation(location, volumes)
	if !ok {
		return Inspection{}, ErrInvalidReadiness{Field: "location", Reason: "not under a mounted volume"}
	}
	entries, _ := boundedTopLevel(readDir, vol.Mountpoint)
	return Inspection{
		Identity: DeviceIdentity{
			DevicePath: vol.DevicePath,
			Mountpoint: cleanPath(vol.Mountpoint),
			Filesystem: vol.Filesystem,
			TotalBytes: vol.TotalBytes,
		},
		FreeBytes:       vol.FreeBytes,
		ReadOnly:        vol.ReadOnly,
		Removable:       vol.Removable,
		DriveClass:      string(vol.Class),
		MountOptions:    append([]string(nil), vol.MountOptions...),
		TopLevelEntries: entries,
		NonEmptyRoot:    len(entries) > 0,
		InstallerMedia:  looksLikeInstallerMedia(entries),
		Platform:        runtime.GOOS,
	}, nil
}

func bestVolumeForLocation(location string, volumes []sysmounts.Volume) (sysmounts.Volume, bool) {
	location = cleanPath(location)
	var best sysmounts.Volume
	bestLen := -1
	for _, v := range volumes {
		mount := cleanPath(v.Mountpoint)
		if mount == "" || !within(location, mount) {
			continue
		}
		if len(mount) > bestLen {
			best = v
			bestLen = len(mount)
		}
	}
	return best, bestLen >= 0
}

func boundedTopLevel(readDir func(string) ([]os.DirEntry, error), root string) ([]string, error) {
	entries, err := readDir(root)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= maxTopLevelEntries {
			break
		}
	}
	sort.Strings(names)
	return names, nil
}

func looksLikeInstallerMedia(entries []string) bool {
	if len(entries) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		set[strings.ToLower(filepath.ToSlash(e))] = struct{}{}
	}
	_, hasDisk := set[".disk"]
	_, hasCasper := set["casper"]
	_, hasBootCatalog := set["boot.catalog"]
	_, hasBoot := set["boot"]
	_, hasSources := set["sources"]
	_, hasEFI := set["efi"]
	_, hasBootDir := set["[boot]"]
	return (hasDisk && hasCasper) || (hasBootCatalog && (hasBoot || hasEFI || hasBootDir)) || (hasSources && hasEFI)
}
