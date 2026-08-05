// Package sysmounts is the OS boundary for enumerating mounted volumes. It is
// the single place in the scenario allowed to import gopsutil; the rest of the
// code depends on the discovery.VolumeScanner seam and substitutes a fake.
//
// Enumeration and free/total space come from gopsutil/v3/disk (cross-platform);
// removable/network classification — which gopsutil does NOT provide — is a
// thin per-OS helper in classify.go, kept gopsutil-free so it is unit-testable
// against a fixture /sys/block tree.
//
// seam: discovery.VolumeScanner. Production wires *sysmounts.Scanner; discovery
// unit tests wire mocks.FakeVolumeScanner.
package sysmounts

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/disk"
)

// DriveClass is the canonical domain vocabulary for a volume's class. The
// discovery handler translates it to the proto DriveClass enum at the boundary
// so domain code never imports the generated proto types.
type DriveClass string

const (
	// ClassUnspecified means the class could not be determined.
	ClassUnspecified DriveClass = ""
	// ClassRemovable is external/removable media (USB stick, external disk, SD card).
	ClassRemovable DriveClass = "removable"
	// ClassFixed is a fixed internal volume.
	ClassFixed DriveClass = "fixed"
	// ClassNetwork is a network/remote mount (NFS, SMB, etc.).
	ClassNetwork DriveClass = "network"
)

// FilesystemState is the best-effort state exposed by mounted filesystem
// metadata. Unknown is intentional: most platforms do not publish a dirty
// bit through the portable mount enumeration API.
type FilesystemState string

const (
	FilesystemStateUnknown    FilesystemState = "unknown"
	FilesystemStateClean      FilesystemState = "clean"
	FilesystemStateDirty      FilesystemState = "dirty"
	FilesystemStateNeedsCheck FilesystemState = "needs-check"
)

// Volume is a single mounted, real (non-pseudo) filesystem.
type Volume struct {
	// DevicePath is the source device/path backing this mount when the OS
	// reports one (for example /dev/sdb1 on Linux).
	DevicePath string
	// Mountpoint is the absolute path the filesystem is mounted at.
	Mountpoint string
	// Filesystem is the filesystem type (ext4, apfs, ntfs, nfs, …).
	Filesystem string
	// Class is the removable/fixed/network classification.
	Class DriveClass
	// Removable mirrors Class == ClassRemovable for convenience.
	Removable bool
	// FreeBytes / TotalBytes come from gopsutil disk.Usage; 0 when unknown.
	FreeBytes  int64
	TotalBytes int64
	// ReadOnly is true when the mount carries the `ro` option.
	ReadOnly bool
	// MountOptions is the OS-reported mount option list.
	MountOptions []string
	// FilesystemState is derived only from explicit OS-reported mount options;
	// absence of a signal remains unknown and never becomes an unsafe pass.
	FilesystemState FilesystemState
	// Stable identity metadata. Fields are empty only when the host cannot
	// expose them; callers must treat an unavailable identity as uncertain.
	Label  string
	UUID   string
	Model  string
	Serial string
}

// Scanner enumerates mounted volumes via gopsutil and classifies them. The
// gopsutil calls are stored as fields so they can be swapped in tests, but the
// production path uses the real package functions.
type Scanner struct {
	partitions func(ctx context.Context, all bool) ([]disk.PartitionStat, error)
	usage      func(ctx context.Context, path string) (*disk.UsageStat, error)
	classifier *classifier
	identity   func(context.Context, string) (VolumeIdentity, error)
}

// New constructs the production volume scanner backed by gopsutil/v3/disk.
func New() *Scanner {
	return &Scanner{
		partitions: disk.PartitionsWithContext,
		usage:      disk.UsageWithContext,
		classifier: newClassifier(),
		identity:   platformVolumeIdentity,
	}
}

// Scan enumerates mounted volumes, skipping pseudo filesystems, classifying
// each, and attaching best-effort free/total space. A usage probe that fails
// for one mountpoint never fails the whole scan — that volume is reported with
// zero free/total.
func (s *Scanner) Scan(ctx context.Context) ([]Volume, error) {
	parts, err := s.partitions(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("enumerate partitions: %w", err)
	}
	out := make([]Volume, 0, len(parts))
	for _, p := range parts {
		m := mountInfo{Device: p.Device, Mountpoint: p.Mountpoint, Fstype: p.Fstype, Opts: p.Opts}
		if s.classifier.isPseudoFS(m.Fstype) {
			continue
		}
		class, removable := s.classifier.classify(m)
		vol := Volume{
			DevicePath:      m.Device,
			Mountpoint:      m.Mountpoint,
			Filesystem:      m.Fstype,
			Class:           class,
			Removable:       removable,
			ReadOnly:        isReadOnly(m.Opts),
			MountOptions:    append([]string(nil), m.Opts...),
			FilesystemState: filesystemState(m.Opts),
		}
		if s.identity != nil {
			if identity, ierr := s.identity(ctx, m.Device); ierr == nil {
				vol.Label, vol.UUID, vol.Model, vol.Serial = identity.Label, identity.UUID, identity.Model, identity.Serial
			}
		}
		if u, uerr := s.usage(ctx, m.Mountpoint); uerr == nil && u != nil {
			vol.FreeBytes = clampUint64(u.Free)
			vol.TotalBytes = clampUint64(u.Total)
		}
		out = append(out, vol)
	}
	return out, nil
}

// clampUint64 converts a gopsutil uint64 byte count to int64 for the wire,
// saturating rather than overflowing on the (practically impossible) >8 EiB case.
func clampUint64(v uint64) int64 {
	const maxInt64 = int64(^uint64(0) >> 1)
	if v > uint64(maxInt64) {
		return maxInt64
	}
	return int64(v)
}
