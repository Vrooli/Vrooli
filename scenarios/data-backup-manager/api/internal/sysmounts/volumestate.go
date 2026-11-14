package sysmounts

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ReadOnlyCause explains *why* a volume is mounted read-only. A read-only
// mount is not self-describing: a write-protect switch, a driver that refused
// read/write because the filesystem needs a check, and an operator who asked
// for `ro` in fstab all look identical in the mount option list. Remediation
// differs completely between them, so the cause is evidence, never a guess —
// an unproven cause stays Unknown rather than becoming a plausible default.
type ReadOnlyCause string

const (
	// CauseNotReadOnly is the zero value for a read/write volume.
	CauseNotReadOnly ReadOnlyCause = ""
	// CauseUnknown means the volume is read-only but no evidence source on
	// this platform could attribute it. Never treat this as safe to remediate.
	CauseUnknown ReadOnlyCause = "unknown"
	// CauseDeviceWriteProtected means the block device itself refuses writes
	// (hardware switch, `blockdev --setro`, read-only loop device). No
	// filesystem repair can help; the device must be unprotected first.
	CauseDeviceWriteProtected ReadOnlyCause = "device_write_protected"
	// CauseFilesystemDirty means the filesystem carries a dirty / needs-check
	// flag and the driver refused a read/write mount because of it. This is
	// the repairable case.
	CauseFilesystemDirty ReadOnlyCause = "filesystem_dirty"
	// CauseMountOption means read-only was explicitly requested for this
	// mount (an `ro` entry in fstab). Repair is the wrong response: the
	// declared intent must be changed instead.
	CauseMountOption ReadOnlyCause = "mount_option"
)

// VolumeState is the unprivileged, read-only health probe result for one
// mounted volume. Every field is derived from evidence the owner account can
// read without elevation; nothing here executes a privileged tool.
type VolumeState struct {
	// FilesystemState is the dirty/clean/needs-check verdict.
	FilesystemState FilesystemState
	// ReadOnlyCause attributes a read-only mount. Empty when mounted rw.
	ReadOnlyCause ReadOnlyCause
	// DeviceWriteProtected reports the block layer's own read-only flag,
	// independent of how the filesystem was mounted.
	DeviceWriteProtected bool
	// EvidenceSource names what actually produced the verdict so a report can
	// state its provenance instead of implying uniform confidence.
	EvidenceSource string
}

// stateProber owns read-only volume health probing. Its injectable roots let
// the Linux evidence paths be exercised on any host against fixture trees,
// matching the classifier's approach.
type stateProber struct {
	goos          string
	sysClassBlock string
	procFS        string
	fstabPath     string
	readFile      func(string) ([]byte, error)
}

func newStateProber() *stateProber {
	return &stateProber{
		goos:          runtime.GOOS,
		sysClassBlock: "/sys/class/block",
		procFS:        "/proc/fs",
		fstabPath:     "/etc/fstab",
		readFile:      os.ReadFile,
	}
}

// probe collects filesystem state and read-only attribution for one mount.
// It never fails: an unreadable evidence source degrades to Unknown, because
// "we could not tell" and "the volume is healthy" must never collapse into the
// same verdict.
func (p *stateProber) probe(m mountInfo, readOnly bool) VolumeState {
	state := VolumeState{FilesystemState: filesystemState(m.Opts), EvidenceSource: "mount-options"}
	if p.goos != "linux" {
		// Native darwin/windows health adapters land with the control-plane
		// remediation package; until then the portable mount-option signal is
		// the only evidence, and its limits are stated rather than papered over.
		if readOnly {
			state.ReadOnlyCause = CauseUnknown
		}
		state.EvidenceSource = "mount-options (no native volume-state adapter on " + p.goos + ")"
		return state
	}

	state.DeviceWriteProtected = p.deviceWriteProtected(m.Device)

	if fsState, source, ok := p.linuxFilesystemState(m, readOnly); ok {
		state.FilesystemState = fsState
		state.EvidenceSource = source
	}

	if !readOnly {
		return state
	}
	switch {
	case state.DeviceWriteProtected:
		state.ReadOnlyCause = CauseDeviceWriteProtected
	case state.FilesystemState == FilesystemStateDirty || state.FilesystemState == FilesystemStateNeedsCheck:
		state.ReadOnlyCause = CauseFilesystemDirty
	case p.fstabRequestsReadOnly(m):
		state.ReadOnlyCause = CauseMountOption
	default:
		state.ReadOnlyCause = CauseUnknown
	}
	return state
}

// deviceWriteProtected reads the block layer's `ro` attribute for both the
// partition and its backing whole disk. Either being set means no filesystem
// action can restore writes.
func (p *stateProber) deviceWriteProtected(device string) bool {
	name := strings.TrimSpace(filepath.Base(strings.TrimSpace(device)))
	if name == "" || name == "." || name == "/" {
		return false
	}
	for _, candidate := range p.writeProtectCandidates(name) {
		data, err := p.readFile(filepath.Join(p.sysClassBlock, candidate, "ro"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == "1" {
			return true
		}
	}
	return false
}

// writeProtectCandidates returns the partition name plus its plausible
// whole-disk parent. The parent is derived by trimming the trailing partition
// suffix (sda1 -> sda, nvme0n1p2 -> nvme0n1); a candidate that does not exist
// simply yields an unreadable file and is skipped.
func (p *stateProber) writeProtectCandidates(name string) []string {
	candidates := []string{name}
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return candidates
	}
	if strings.HasSuffix(trimmed, "p") && len(trimmed) > 1 {
		trimmed = strings.TrimSuffix(trimmed, "p")
	}
	if trimmed != "" && trimmed != name {
		candidates = append(candidates, trimmed)
	}
	return candidates
}

// linuxFilesystemState reads a driver-published state file when the mounted
// filesystem exposes one. The kernel's own view beats any inference we could
// make from mount options.
func (p *stateProber) linuxFilesystemState(m mountInfo, readOnly bool) (FilesystemState, string, bool) {
	name := strings.TrimSpace(filepath.Base(strings.TrimSpace(m.Device)))
	if name == "" || name == "." || name == "/" {
		return FilesystemStateUnknown, "", false
	}
	switch strings.ToLower(strings.TrimSpace(m.Fstype)) {
	case "ntfs3":
		// ntfs3 publishes a world-readable per-volume summary whose field
		// order has changed across kernel releases, so the parser scans every
		// field for the state tokens instead of indexing a fixed position.
		path := filepath.Join(p.procFS, "ntfs3", name, "volinfo")
		data, err := p.readFile(path)
		if err != nil {
			return FilesystemStateUnknown, "", false
		}
		state := volinfoState(string(data))
		if state == FilesystemStateDirty && !readOnly {
			// NTFS sets its volume dirty flag for the duration of any
			// read/write mount — that is how an unclean shutdown is detected
			// later. So on a read/write mount the flag is ambiguous: it may be
			// this mount's in-use marker, or it may be damage that predates it.
			//
			// This used to resolve the ambiguity as Clean, which meant a
			// genuinely damaged volume reported healthy for as long as it
			// stayed mounted read/write — the exact state that preceded the
			// 2026-08-19 host panic. Unknown is the honest verdict: it does not
			// mark every healthy NTFS drive as needing repair, and it does not
			// assert a cleanliness this evidence cannot establish.
			return FilesystemStateUnknown, path + " (read/write mount; the NTFS dirty flag cannot be distinguished from this mount's in-use marker)", true
		}
		return state, path, true
	default:
		return FilesystemStateUnknown, "", false
	}
}

// volinfoState token-scans an ntfs3 volinfo body. Dirty wins over clean: a
// volume reporting both must be treated as needing a check.
func volinfoState(body string) FilesystemState {
	state := FilesystemStateUnknown
	for _, field := range strings.Fields(body) {
		switch strings.ToLower(strings.TrimSpace(field)) {
		case "dirty":
			return FilesystemStateDirty
		case "clean":
			state = FilesystemStateClean
		}
	}
	return state
}

// fstabRequestsReadOnly reports whether a declared fstab entry for this mount
// asks for `ro`. It matches on mountpoint or on the device spec in any of the
// forms fstab accepts, and treats an unreadable fstab as "no declaration".
func (p *stateProber) fstabRequestsReadOnly(m mountInfo) bool {
	data, err := p.readFile(p.fstabPath)
	if err != nil {
		return false
	}
	mountpoint := filepath.Clean(strings.TrimSpace(m.Mountpoint))
	device := filepath.Clean(strings.TrimSpace(m.Device))
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		spec := filepath.Clean(fields[0])
		if spec != device && filepath.Clean(fields[1]) != mountpoint {
			continue
		}
		for _, opt := range strings.Split(fields[3], ",") {
			if strings.EqualFold(strings.TrimSpace(opt), "ro") {
				return true
			}
		}
	}
	return false
}

// deviceSizeBytes reads the block layer's sector count for a device. It is the
// only size source available for a volume that is not mounted, where no
// statfs-backed usage probe can run.
func (p *stateProber) deviceSizeBytes(device string) int64 {
	if p.goos != "linux" {
		return 0
	}
	name := strings.TrimSpace(filepath.Base(strings.TrimSpace(device)))
	if name == "" || name == "." || name == "/" {
		return 0
	}
	data, err := p.readFile(filepath.Join(p.sysClassBlock, name, "size"))
	if err != nil {
		return 0
	}
	sectors, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || sectors <= 0 {
		return 0
	}
	// The `size` attribute is always in 512-byte units regardless of the
	// device's logical block size — a kernel ABI guarantee, not an assumption
	// about this particular disk.
	const sectorBytes = 512
	if sectors > (1<<62)/sectorBytes {
		return 0
	}
	return sectors * sectorBytes
}
