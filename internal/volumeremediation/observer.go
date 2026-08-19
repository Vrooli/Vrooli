package volumeremediation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// HostObserver reads volume state from the host without elevation and without
// writing. Its roots are injectable so the Linux evidence paths can be tested
// on any platform against fixture trees.
type HostObserver struct {
	goos          string
	procMounts    string
	sysClassBlock string
	procFS        string
	devDiskByUUID string
	readFile      func(string) ([]byte, error)
	readDir       func(string) ([]os.DirEntry, error)
	evalSymlinks  func(string) (string, error)
	run           func(ctx context.Context, argv []string) ([]byte, error)
}

// NewHostObserver constructs the production observer for a platform.
func NewHostObserver(goos string) *HostObserver {
	if goos == "" {
		goos = runtime.GOOS
	}
	return &HostObserver{
		goos:          goos,
		procMounts:    "/proc/mounts",
		sysClassBlock: "/sys/class/block",
		procFS:        "/proc/fs",
		devDiskByUUID: "/dev/disk/by-uuid",
		readFile:      os.ReadFile,
		readDir:       os.ReadDir,
		evalSymlinks:  filepath.EvalSymlinks,
		run: func(ctx context.Context, argv []string) ([]byte, error) {
			return exec.CommandContext(ctx, argv[0], argv[1:]...).Output()
		},
	}
}

// Observe reports the current state of one device.
func (o *HostObserver) Observe(ctx context.Context, devicePath string) (State, error) {
	if err := validateDevicePath(devicePath); err != nil {
		return State{}, err
	}
	switch o.goos {
	case "linux":
		return o.observeLinux(ctx, devicePath)
	case "darwin":
		return o.observeDarwin(ctx, devicePath)
	default:
		return State{}, ErrUnsupported{
			Reason:          "no volume observation adapter for " + o.goos,
			OperatorCommand: "Get-Volume | Format-List",
		}
	}
}

func (o *HostObserver) observeLinux(ctx context.Context, devicePath string) (State, error) {
	state := State{
		Device: Device{Path: devicePath},
		Dirty:  TristateUnknown,
	}
	var observations []string

	mountpoint, fstype, opts, mounted := o.linuxMountEntry(devicePath)
	state.Mounted = mounted
	if mounted {
		state.Device.Mountpoint = mountpoint
		state.Device.Filesystem = fstype
		state.ReadOnly = hasOption(opts, "ro")
		observations = append(observations, "mount: "+mountpoint+" ("+fstype+")")
	} else {
		observations = append(observations, "not mounted")
	}

	name := filepath.Base(devicePath)
	if size := o.linuxDeviceSize(name); size > 0 {
		state.Device.TotalBytes = size
	}
	if o.linuxDeviceReadOnly(name) {
		// Block-layer write protection makes the volume read-only regardless of
		// how it is mounted, so it must be reported even when unmounted.
		state.ReadOnly = true
		observations = append(observations, "block device is write-protected")
	}

	if uuid := o.linuxUUID(devicePath); uuid != "" {
		state.Device.UUID = uuid
	}
	o.enrichWithLsblk(ctx, devicePath, &state.Device)
	if state.Device.Serial == "" {
		// lsblk reports SERIAL for the whole disk, not for a partition, so a
		// partition-only query leaves the second identity anchor empty. Ask the
		// backing disk for it rather than settling for a single anchor.
		if disk := wholeDiskName(name); disk != "" {
			o.enrichWithLsblk(ctx, filepath.Join(filepath.Dir(devicePath), disk), &state.Device)
		}
	}

	dirty, evidence := o.linuxDirty(name, state.Device.Filesystem, state.Mounted && !state.ReadOnly)
	state.Dirty = dirty
	state.Evidence = evidence
	if evidence != "" {
		observations = append(observations, "state evidence: "+evidence)
	}
	state.Observations = observations
	return state, nil
}

// linuxMountEntry finds the mount entry for a device, tolerating the /dev
// symlink forms /proc/mounts may report.
func (o *HostObserver) linuxMountEntry(devicePath string) (mountpoint, fstype string, opts []string, mounted bool) {
	data, err := o.readFile(o.procMounts)
	if err != nil {
		return "", "", nil, false
	}
	want := o.resolve(devicePath)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if o.resolve(fields[0]) != want {
			continue
		}
		// /proc/mounts octal-escapes spaces and tabs in paths.
		return unescapeMountPath(fields[1]), fields[2], strings.Split(fields[3], ","), true
	}
	return "", "", nil, false
}

func (o *HostObserver) resolve(path string) string {
	path = strings.TrimSpace(path)
	if o.evalSymlinks == nil {
		return path
	}
	if real, err := o.evalSymlinks(path); err == nil {
		return real
	}
	return path
}

func (o *HostObserver) linuxDeviceSize(name string) int64 {
	data, err := o.readFile(filepath.Join(o.sysClassBlock, name, "size"))
	if err != nil {
		return 0
	}
	sectors, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || sectors <= 0 || sectors > (1<<62)/512 {
		return 0
	}
	return sectors * 512
}

func (o *HostObserver) linuxDeviceReadOnly(name string) bool {
	for _, candidate := range []string{name, wholeDiskName(name)} {
		if candidate == "" {
			continue
		}
		data, err := o.readFile(filepath.Join(o.sysClassBlock, candidate, "ro"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == "1" {
			return true
		}
	}
	return false
}

// wholeDiskName derives the backing disk for a partition name.
func wholeDiskName(name string) string {
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "p") && len(trimmed) > 1 {
		trimmed = strings.TrimSuffix(trimmed, "p")
	}
	if trimmed == name {
		return ""
	}
	return trimmed
}

// linuxUUID reverse-resolves /dev/disk/by-uuid, which needs no elevation and no
// external tool.
func (o *HostObserver) linuxUUID(devicePath string) string {
	entries, err := o.readDir(o.devDiskByUUID)
	if err != nil {
		return ""
	}
	want := o.resolve(devicePath)
	for _, entry := range entries {
		link := filepath.Join(o.devDiskByUUID, entry.Name())
		if o.resolve(link) == want {
			return entry.Name()
		}
	}
	return ""
}

var lsblkProperty = regexp.MustCompile(`([A-Z]+)="([^"]*)"`)

// enrichWithLsblk fills identity fields the filesystem evidence cannot supply —
// notably the serial, and the filesystem type of an unmounted volume. A failure
// leaves the fields unset, which the safety gates treat as unprovable identity.
func (o *HostObserver) enrichWithLsblk(ctx context.Context, devicePath string, device *Device) {
	if o.run == nil {
		return
	}
	out, err := o.run(ctx, []string{"lsblk", "-P", "-n", "-o", "UUID,SERIAL,FSTYPE,LABEL", devicePath})
	if err != nil {
		return
	}
	for _, match := range lsblkProperty.FindAllStringSubmatch(string(out), -1) {
		value := strings.TrimSpace(match[2])
		if value == "" {
			continue
		}
		switch match[1] {
		case "UUID":
			if device.UUID == "" {
				device.UUID = value
			}
		case "SERIAL":
			if device.Serial == "" {
				device.Serial = value
			}
		case "FSTYPE":
			if device.Filesystem == "" {
				device.Filesystem = value
			}
		}
	}
}

// linuxDirty reads the driver-published state file when the filesystem exposes
// one. Absence of a signal stays unknown rather than becoming a clean verdict.
//
// mountedReadWrite matters: NTFS sets its volume dirty flag for the duration of
// any read/write mount, which is how an unclean shutdown is detected later.
// Once the driver has accepted a read/write mount the flag is an in-use marker,
// not a fault — treating it as one would report every healthy NTFS volume as
// needing repair.
func (o *HostObserver) linuxDirty(name, filesystem string, mountedReadWrite bool) (Tristate, string) {
	family, _ := filesystemFamily(filesystem)
	if family != "ntfs" {
		return TristateUnknown, ""
	}
	path := filepath.Join(o.procFS, "ntfs3", name, "volinfo")
	data, err := o.readFile(path)
	if err != nil {
		return TristateUnknown, ""
	}
	fields := strings.Fields(string(data))
	for _, field := range fields {
		if strings.EqualFold(field, "dirty") {
			if mountedReadWrite {
				return TristateNo, path + " (read/write mount; the NTFS dirty flag is the driver's in-use marker, not a fault)"
			}
			return TristateYes, path
		}
	}
	for _, field := range fields {
		if strings.EqualFold(field, "clean") {
			return TristateNo, path
		}
	}
	return TristateUnknown, path
}

// observeDarwin reads attachment state from `mount` output. macOS publishes no
// portable dirty flag, so health stays unknown and the gates treat it as such.
func (o *HostObserver) observeDarwin(ctx context.Context, devicePath string) (State, error) {
	state := State{Device: Device{Path: devicePath}, Dirty: TristateUnknown}
	if o.run == nil {
		return state, nil
	}
	out, err := o.run(ctx, []string{"mount"})
	if err != nil {
		return state, nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		// Format: /dev/disk2s1 on /Volumes/USB (exfat, local, read-only)
		if !strings.HasPrefix(line, devicePath+" on ") {
			continue
		}
		rest := strings.TrimPrefix(line, devicePath+" on ")
		mountpoint, attrs, found := strings.Cut(rest, " (")
		state.Mounted = true
		state.Device.Mountpoint = strings.TrimSpace(mountpoint)
		if found {
			attrs = strings.TrimSuffix(strings.TrimSpace(attrs), ")")
			parts := strings.Split(attrs, ",")
			if len(parts) > 0 {
				state.Device.Filesystem = strings.TrimSpace(parts[0])
			}
			for _, part := range parts {
				if strings.EqualFold(strings.TrimSpace(part), "read-only") {
					state.ReadOnly = true
				}
			}
		}
		state.Observations = append(state.Observations, "mount: "+state.Device.Mountpoint)
		return state, nil
	}
	state.Observations = append(state.Observations, "not mounted")
	return state, nil
}

func hasOption(opts []string, want string) bool {
	for _, opt := range opts {
		if strings.EqualFold(strings.TrimSpace(opt), want) {
			return true
		}
	}
	return false
}

// unescapeMountPath decodes the octal escapes /proc/mounts uses for spaces,
// tabs, newlines and backslashes in paths.
func unescapeMountPath(path string) string {
	if !strings.Contains(path, `\`) {
		return path
	}
	var b strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] == '\\' && i+3 < len(path) {
			if v, err := strconv.ParseUint(path[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(v))
				i += 3
				continue
			}
		}
		b.WriteByte(path[i])
	}
	return b.String()
}
