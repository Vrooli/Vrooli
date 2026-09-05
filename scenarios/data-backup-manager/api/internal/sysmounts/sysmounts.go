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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

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
	// MountDriver is the kernel-side driver currently serving this mount, as
	// the OS reports it in its mount table (ext4, ntfs3, fuseblk, …). It is
	// empty for a volume that is not mounted.
	//
	// It is deliberately separate from Filesystem: one on-disk format can be
	// served by drivers with very different failure behaviour. NTFS is the
	// case that forced the distinction — the in-kernel `ntfs3` driver faults
	// in kernel context, while `ntfs-3g` (which the mount table reports as
	// `fuseblk`) fails as an ordinary userspace I/O error. Policy that cares
	// about blast radius must read this field, not Filesystem.
	MountDriver string
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
	// FilesystemState is the dirty/clean/needs-check verdict from the best
	// evidence the platform exposes without elevation; absence of a signal
	// remains unknown and never becomes an unsafe pass.
	FilesystemState FilesystemState
	// ReadOnlyCause attributes a read-only mount to write-protected hardware,
	// a dirty filesystem, or a declared `ro` mount option. Empty when the
	// volume is mounted read/write; "unknown" when read-only but unattributed.
	ReadOnlyCause ReadOnlyCause
	// DeviceWriteProtected is the block layer's own read-only flag, which is
	// independent of the mount and cannot be cleared by filesystem repair.
	DeviceWriteProtected bool
	// StateEvidence names the source behind FilesystemState/ReadOnlyCause so
	// a report can state provenance instead of implying uniform confidence.
	StateEvidence string
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
	state      *stateProber
	identity   func(context.Context, string) (VolumeIdentity, error)
}

// New constructs the production volume scanner backed by gopsutil/v3/disk.
func New() *Scanner {
	return &Scanner{
		partitions: disk.PartitionsWithContext,
		usage:      disk.UsageWithContext,
		classifier: newClassifier(),
		state:      newStateProber(),
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
		readOnly := isReadOnly(m.Opts)
		state := VolumeState{FilesystemState: filesystemState(m.Opts), EvidenceSource: "mount-options"}
		if s.state != nil {
			state = s.state.probe(m, readOnly)
		}
		vol := Volume{
			DevicePath:           m.Device,
			Mountpoint:           m.Mountpoint,
			Filesystem:           m.Fstype,
			MountDriver:          m.Fstype,
			Class:                class,
			Removable:            removable,
			ReadOnly:             readOnly,
			MountOptions:         append([]string(nil), m.Opts...),
			FilesystemState:      state.FilesystemState,
			ReadOnlyCause:        state.ReadOnlyCause,
			DeviceWriteProtected: state.DeviceWriteProtected,
			StateEvidence:        state.EvidenceSource,
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

// ErrDeviceNotFound reports that a block device could not be resolved at all —
// neither mounted nor visible to the block layer. It is distinct from "found
// but unmounted", which is a normal mid-remediation state.
var ErrDeviceNotFound = errors.New("block device not found")

// Device resolves one block device by path whether or not it is mounted.
//
// Scan only sees mounted filesystems, which is the wrong lens for remediation:
// unmounting the volume is a step *inside* a repair, and a flow that loses
// sight of its own device the moment it unmounts cannot re-verify identity
// before remounting. A returned Volume with an empty Mountpoint means the
// device exists and is not mounted.
func (s *Scanner) Device(ctx context.Context, devicePath string) (Volume, error) {
	devicePath = strings.TrimSpace(devicePath)
	if devicePath == "" {
		return Volume{}, fmt.Errorf("device path is required")
	}
	mounted, err := s.Scan(ctx)
	if err != nil {
		return Volume{}, err
	}
	for _, v := range mounted {
		if sameDevice(v.DevicePath, devicePath) {
			return v, nil
		}
	}

	state := s.state
	if state == nil {
		state = newStateProber()
	}
	size := state.deviceSizeBytes(devicePath)
	identity := VolumeIdentity{}
	if s.identity != nil {
		// A failed identity probe is not fatal: the caller re-verifies against
		// whatever fields the host could publish and treats an unprovable
		// identity as a refusal, never as a match.
		identity, _ = s.identity(ctx, devicePath)
	}
	if size == 0 && identity.UUID == "" && identity.Serial == "" && identity.Filesystem == "" {
		return Volume{}, fmt.Errorf("%w: %s", ErrDeviceNotFound, devicePath)
	}

	m := mountInfo{Device: devicePath, Fstype: identity.Filesystem}
	class, removable := s.classifier.classify(m)
	probed := state.probe(m, false)
	return Volume{
		DevicePath:           devicePath,
		Filesystem:           identity.Filesystem,
		Class:                class,
		Removable:            removable,
		TotalBytes:           size,
		DeviceWriteProtected: state.deviceWriteProtected(devicePath),
		FilesystemState:      probed.FilesystemState,
		StateEvidence:        probed.EvidenceSource,
		Label:                identity.Label,
		UUID:                 identity.UUID,
		Model:                identity.Model,
		Serial:               identity.Serial,
	}, nil
}

// sameDevice compares device specs tolerantly enough to survive the /dev
// symlink forms the OS hands back, without treating different devices as one.
func sameDevice(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	resolve := func(p string) string {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return real
		}
		return p
	}
	return resolve(a) == resolve(b)
}
