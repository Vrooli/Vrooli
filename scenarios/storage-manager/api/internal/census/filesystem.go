package census

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem is the read-only filesystem seam used by census walks. The
// production implementation delegates to the host filesystem; tests use an
// in-memory tree and therefore cannot accidentally inspect the operator host.
type FileSystem interface {
	WalkDir(root string, fn fs.WalkDirFunc) error
	Stat(path string) (os.FileInfo, error)
	Lstat(path string) (os.FileInfo, error)
}

type hostFileSystem struct{}

func (hostFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}
func (hostFileSystem) Stat(path string) (os.FileInfo, error)  { return os.Stat(path) }
func (hostFileSystem) Lstat(path string) (os.FileInfo, error) { return os.Lstat(path) }

type metadataFileSystem interface {
	inspectMetadata(path string) (fileMetadata, error)
}

func inspectPathWith(fs FileSystem, path string) (fileMetadata, error) {
	if provider, ok := fs.(metadataFileSystem); ok {
		return provider.inspectMetadata(path)
	}
	info, err := fs.Lstat(path)
	if err != nil {
		return fileMetadata{}, err
	}
	return fileMetadata{bytes: info.Size(), allocated: info.Size()}, nil
}

type DeviceInfo struct {
	TotalBytes     int64
	AvailableBytes int64
	Privilege      string
}

// DeviceProbe supplies the device denominator and the process privilege
// classification. A missing answer is a named degraded result, never a
// silent false measured-by-device flag.
type DeviceProbe interface {
	Probe(root string) (DeviceInfo, error)
}

func deviceCoverageWith(probe DeviceProbe, root string, measured int64, complete bool) ScanCoverage {
	coverage := ScanCoverage{MeasuredBytes: measured, Complete: complete}
	if probe == nil {
		coverage.Complete = false
		coverage.DegradedReason = "device_probe_unavailable"
		return coverage
	}
	device, err := probe.Probe(root)
	if err != nil {
		coverage.Complete = false
		coverage.DegradedReason = "device_probe_unavailable: " + err.Error()
		return coverage
	}
	coverage.DeviceTotalBytes = device.TotalBytes
	coverage.DeviceUsedBytes = device.TotalBytes - device.AvailableBytes
	coverage.PrivilegeLevel = device.Privilege
	coverage.MeasuredByDevice = device.TotalBytes > 0 && device.AvailableBytes >= 0 && coverage.DeviceUsedBytes >= 0
	if !coverage.MeasuredByDevice {
		coverage.Complete = false
		coverage.DegradedReason = "device_probe_returned_invalid_totals"
	}
	return coverage
}
