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

// DeviceResolver resolves a block device whether or not it is mounted. It is a
// separate seam from VolumeScanner because most callers only ever need mounted
// volumes; only remediation needs to keep observing a device it unmounted.
type DeviceResolver interface {
	Device(ctx context.Context, devicePath string) (sysmounts.Volume, error)
}

// DeviceInspector inspects a volume by device identity rather than by path.
// A path-scoped inspection cannot describe an unmounted volume — the path
// simply stops existing — so remediation, which passes through exactly that
// state, needs this lens to re-verify identity before remounting.
type DeviceInspector interface {
	InspectDevice(ctx context.Context, expected DeviceIdentity) (Inspection, error)
}

// ReadOnlyInspector matches a path to its mounted volume and collects bounded,
// read-only content clues from the mount root.
type ReadOnlyInspector struct {
	Volumes VolumeScanner
	Devices DeviceResolver
	ReadDir func(name string) ([]os.DirEntry, error)
	Stat    func(name string) (os.FileInfo, error)
}

var _ DeviceInspector = (*ReadOnlyInspector)(nil)

// NewReadOnlyInspector constructs the production read-only inspector. Device
// resolution is wired only when the scanner actually provides it, so a fake
// scanner in a unit test stays a valid VolumeScanner.
func NewReadOnlyInspector(volumes VolumeScanner) *ReadOnlyInspector {
	inspector := &ReadOnlyInspector{Volumes: volumes, ReadDir: os.ReadDir, Stat: os.Stat}
	if devices, ok := volumes.(DeviceResolver); ok {
		inspector.Devices = devices
	}
	return inspector
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
	stat := i.Stat
	if stat == nil {
		stat = os.Stat
	}
	locationInfo, locationErr := stat(location)
	volumes, err := i.Volumes.Scan(ctx)
	if err != nil {
		return Inspection{}, err
	}
	vol, ok := bestVolumeForLocation(location, volumes)
	if !ok {
		return Inspection{}, ErrInvalidReadiness{Field: "location", Reason: "not under a mounted volume"}
	}
	inspection := inspectionFromVolume(vol)
	entries, readErr := boundedTopLevel(readDir, vol.Mountpoint)
	inspection.LocationExists = locationErr == nil
	inspection.LocationIsDirectory = locationErr == nil && locationInfo.IsDir()
	if locationErr != nil {
		inspection.LocationError = locationErr.Error()
	}
	inspection.TopLevelEntries = entries
	inspection.NonEmptyRoot = len(entries) > 0
	inspection.InstallerMedia = looksLikeInstallerMedia(entries)
	if readErr != nil {
		inspection.ReadDirError = readErr.Error()
	}
	return inspection, nil
}

// InspectDevice observes a volume by device path, whether or not it is
// currently mounted. When the volume is mounted it also collects the same
// bounded content evidence Inspect gathers; when it is not, the location
// fields stay false and Mounted reports the reason.
func (i *ReadOnlyInspector) InspectDevice(ctx context.Context, expected DeviceIdentity) (Inspection, error) {
	device := strings.TrimSpace(expected.DevicePath)
	if device == "" {
		return Inspection{}, ErrInvalidReadiness{Field: "device_path", Reason: "required"}
	}
	if i.Devices == nil {
		return Inspection{}, ErrInvalidReadiness{Field: "devices", Reason: "device-scoped inspection is not available on this host"}
	}
	vol, err := i.Devices.Device(ctx, device)
	if err != nil {
		return Inspection{}, err
	}
	inspection := inspectionFromVolume(vol)
	if !inspection.Mounted {
		return inspection, nil
	}
	readDir := i.ReadDir
	if readDir == nil {
		readDir = os.ReadDir
	}
	stat := i.Stat
	if stat == nil {
		stat = os.Stat
	}
	mountpoint := cleanPath(vol.Mountpoint)
	info, statErr := stat(mountpoint)
	inspection.LocationExists = statErr == nil
	inspection.LocationIsDirectory = statErr == nil && info.IsDir()
	if statErr != nil {
		inspection.LocationError = statErr.Error()
	}
	entries, readErr := boundedTopLevel(readDir, mountpoint)
	inspection.TopLevelEntries = entries
	inspection.NonEmptyRoot = len(entries) > 0
	inspection.InstallerMedia = looksLikeInstallerMedia(entries)
	if readErr != nil {
		inspection.ReadDirError = readErr.Error()
	}
	return inspection, nil
}

// inspectionFromVolume maps the volume-level facts both inspection lenses
// share. Location and directory evidence is layered on by the caller, because
// only one of the two lenses has a path to evaluate.
func inspectionFromVolume(vol sysmounts.Volume) Inspection {
	return Inspection{
		Identity: DeviceIdentity{
			DevicePath: vol.DevicePath,
			Mountpoint: cleanPath(vol.Mountpoint),
			Filesystem: vol.Filesystem,
			TotalBytes: vol.TotalBytes,
			Label:      vol.Label,
			UUID:       vol.UUID,
			Model:      vol.Model,
			Serial:     vol.Serial,
		},
		FreeBytes:            vol.FreeBytes,
		ReadOnly:             vol.ReadOnly,
		Removable:            vol.Removable,
		DriveClass:           string(vol.Class),
		MountOptions:         append([]string(nil), vol.MountOptions...),
		MountDriver:          vol.MountDriver,
		FilesystemState:      vol.FilesystemState,
		ReadOnlyCause:        vol.ReadOnlyCause,
		DeviceWriteProtected: vol.DeviceWriteProtected,
		StateEvidence:        vol.StateEvidence,
		Mounted:              strings.TrimSpace(vol.Mountpoint) != "",
		Platform:             runtime.GOOS,
	}
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
