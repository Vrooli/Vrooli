package volumeremediation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	observerParameterA = 4
	observerParameterB = 512
	observerParameterC = 62
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
			return shell.NewCommandContext(ctx, argv[0], argv[1:]...).Output()
		},
	}
}

// Observe reports the current state of one device.
func (o *HostObserver) Observe(ctx context.Context, devicePath string) (State, error) {
	if err := validateDevicePath(devicePath); err != nil {
		return State{}, err
	}
	switch o.goos {
	case string(hostreqspec.PlatformLinux):
		return o.observeLinux(ctx, devicePath)
	case string(hostreqspec.PlatformDarwin):
		return o.observeDarwin(ctx, devicePath)
	case string(hostreqspec.PlatformWindows):
		return o.observeWindows(ctx, devicePath)
	default:
		return State{}, ErrUnsupported{
			Reason:          "no volume observation adapter for " + o.goos,
			OperatorCommand: "Get-Volume | Format-List",
		}
	}
}

// observeWindows reads the volume and backing-disk identity through
// PowerShell's storage cmdlets. Get-Volume is read-only and returns the
// volume's stable UniqueId even when the filesystem is not mounted. Windows
// does not expose a portable clean/dirty filesystem verdict through this
// inventory, so Dirty deliberately remains unknown.
func (o *HostObserver) observeWindows(ctx context.Context, devicePath string) (State, error) {
	selector := ""
	query := ""
	if len(devicePath) == 2 && devicePath[1] == ':' {
		letter := strings.ToUpper(devicePath[:1])
		selector = letter + ":"
		query = "Get-Volume -DriveLetter '" + letter + "'"
	} else if windowsVolumeGUIDPattern.MatchString(devicePath) {
		selector = devicePath
		query = "Get-Volume -UniqueId '" + devicePath + "'"
	} else {
		return State{}, ErrInvalid{Field: "device.path", Reason: "Windows volume observation requires a drive letter or stable volume GUID"}
	}
	state := State{Device: Device{Path: selector}, Dirty: TristateUnknown}
	if o.run == nil {
		return state, fmt.Errorf("PowerShell volume observer is unavailable")
	}
	// The drive-letter and volume-GUID selectors are restricted by
	// validateDevicePath. Get-Volume by UniqueId keeps this lookup stable even
	// when Windows has not assigned a drive letter after reboot.
	script := fmt.Sprintf("$v = %s -ErrorAction Stop; $p = Get-Partition -Volume $v -ErrorAction SilentlyContinue | Select-Object -First 1; $d = if ($p) { Get-Disk -Number $p.DiskNumber -ErrorAction SilentlyContinue }; [pscustomobject]@{ DriveLetter=$v.DriveLetter; FileSystem=$v.FileSystem; UniqueId=$v.UniqueId; Size=$v.Size; FileSystemLabel=$v.FileSystemLabel; SerialNumber=$d.SerialNumber; IsReadOnly=$d.IsReadOnly } | ConvertTo-Json -Compress", query)
	out, err := o.run(ctx, []string{"powershell", "-NoProfile", "-NonInteractive", "-Command", script})
	if err != nil {
		return state, fmt.Errorf("query Windows volume inventory: %w", err)
	}
	var record struct {
		DriveLetter     string `json:"DriveLetter"`
		FileSystem      string `json:"FileSystem"`
		UniqueID        string `json:"UniqueId"`
		Size            int64  `json:"Size"`
		FileSystemLabel string `json:"FileSystemLabel"`
		SerialNumber    string `json:"SerialNumber"`
		IsReadOnly      *bool  `json:"IsReadOnly"`
	}
	if err := json.Unmarshal(out, &record); err != nil {
		return state, fmt.Errorf("decode Windows volume inventory: %w", err)
	}
	if strings.TrimSpace(record.UniqueID) == "" {
		return state, fmt.Errorf("Windows volume inventory returned no stable volume identity")
	}
	if record.IsReadOnly == nil {
		return state, fmt.Errorf("Windows volume inventory could not determine whether the backing disk is read-only")
	}
	state.Device.Filesystem = strings.TrimSpace(record.FileSystem)
	state.Device.UUID = strings.TrimSpace(record.UniqueID)
	state.Device.Serial = strings.TrimSpace(record.SerialNumber)
	state.Device.TotalBytes = record.Size
	state.Mounted = strings.TrimSpace(record.DriveLetter) != ""
	state.ReadOnly = *record.IsReadOnly
	if state.Mounted {
		state.Device.Mountpoint = strings.ToUpper(strings.TrimSpace(record.DriveLetter)) + `:\`
		state.Observations = append(state.Observations, "volume mounted at "+state.Device.Mountpoint)
	} else {
		state.Observations = append(state.Observations, "volume is visible but not mounted; Windows manages attachment")
	}
	state.Observations = append(state.Observations, "filesystem consistency is unknown; Windows volume inventory does not publish a clean/dirty verdict")
	if state.ReadOnly {
		state.Observations = append(state.Observations, "backing disk is read-only")
	}
	return state, nil
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
		state.Device.MountDriver = fstype
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
		if len(fields) < observerParameterA {
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
	if err != nil || sectors <= 0 || sectors > (1<<observerParameterC)/512 {
		return 0
	}
	return sectors * observerParameterB
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

// enrichWithLsblk fills stable identity and on-disk filesystem fields that the
// mount table cannot supply — notably the serial and the format hidden behind
// generic userspace drivers such as fuseblk. MountDriver remains the active
// mount-table value so safety policy can still distinguish NTFS-3G from ntfs3.
// A failure leaves fields unset, which safety gates treat as unproven.
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
			if device.Filesystem == "" || strings.EqualFold(strings.TrimSpace(device.MountDriver), "fuseblk") {
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
	info, infoErr := o.run(ctx, []string{"diskutil", "info", devicePath})
	if infoErr == nil {
		fields := diskutilInfoFields(string(info))
		state.Device.UUID = firstNonEmptyVolumeValue(fields["Volume UUID"], fields["Disk / Partition UUID"])
		state.Device.Filesystem = firstNonEmptyVolumeValue(fields["File System Personality"], fields["File System"])
		mountpoint := strings.TrimSpace(fields["Mount Point"])
		if strings.Contains(strings.ToLower(mountpoint), "not mounted") || strings.Contains(strings.ToLower(mountpoint), "not applicable") {
			mountpoint = ""
		}
		state.Device.Mountpoint = mountpoint
		state.Mounted = strings.EqualFold(strings.TrimSpace(fields["Mounted"]), "yes") || mountpoint != ""
		state.ReadOnly = strings.EqualFold(strings.TrimSpace(fields["Read-Only Volume"]), "yes") || strings.EqualFold(strings.TrimSpace(fields["Read-Only Media"]), "yes")
	}
	out, err := o.run(ctx, []string{"mount"})
	if err != nil {
		if infoErr != nil {
			return state, fmt.Errorf("inspect macOS volume with diskutil and mount: diskutil: %v; mount: %w", infoErr, err)
		}
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
				state.Device.MountDriver = strings.TrimSpace(parts[0])
				if state.Device.Filesystem == "" {
					state.Device.Filesystem = state.Device.MountDriver
				}
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
	if state.Mounted {
		state.Observations = append(state.Observations, "mounted at "+state.Device.Mountpoint+" (diskutil info)")
	} else {
		state.Device.Mountpoint = ""
		state.Observations = append(state.Observations, "not mounted")
	}
	if infoErr != nil {
		return state, fmt.Errorf("inspect unmounted macOS volume identity: %w", infoErr)
	}
	if state.Device.UUID == "" {
		return state, fmt.Errorf("diskutil did not report a stable volume UUID")
	}
	return state, nil
}

func diskutilInfoFields(output string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok {
			fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return fields
}

func firstNonEmptyVolumeValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
