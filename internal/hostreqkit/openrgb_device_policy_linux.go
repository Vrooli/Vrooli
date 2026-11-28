//go:build linux

package hostreqkit

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const OpenRGBLinuxRulePath = "/etc/udev/rules.d/68-vrooli-openrgb.rules"

type openRGBLinuxProbe struct {
	dev, sys, proc string
	machineID      func() (string, error)
	ruleDigest     func() (string, error)
	accountGroups  func(int) ([]uint32, error)
	stat           func(string, *unix.Stat_t) error
	access         func(string, uint32) error
	hasACL         func(string) (bool, error)
	incarnation    func(int) (string, error)
}

// ObserveLinuxOpenRGBDevicePolicy reads permissions and kernel metadata without
// opening a device node. The enrollment is supplied by the host permission
// owner, which qualifies the exact static recipe before installation. A live
// observation returns an explicit proof gap until the launcher can provide an
// in-context access witness for the child.
func ObserveLinuxOpenRGBDevicePolicy(ctx context.Context, policy OpenRGBLinuxPolicy, check OpenRGBLinuxCheck, incarnation func(int) (string, error)) (OpenRGBLinuxEvidence, error) {
	return newOpenRGBLinuxProbe(incarnation).observe(ctx, policy, check)
}

func newOpenRGBLinuxProbe(incarnation func(int) (string, error)) openRGBLinuxProbe {
	return openRGBLinuxProbe{
		dev: "/dev", sys: "/sys", proc: "/proc",
		machineID:     func() (string, error) { return readOpenRGBText("/etc/machine-id") },
		ruleDigest:    func() (string, error) { return digestOpenRGBRule(OpenRGBLinuxRulePath) },
		accountGroups: openRGBAccountGroups,
		stat:          unix.Lstat,
		access: func(path string, mode uint32) error {
			return unix.Faccessat2(unix.AT_FDCWD, path, mode, unix.AT_EACCESS|unix.AT_SYMLINK_NOFOLLOW)
		},
		hasACL: func(path string) (bool, error) {
			size, err := unix.Lgetxattr(path, "system.posix_acl_access", nil)
			if errors.Is(err, unix.ENODATA) || errors.Is(err, unix.EOPNOTSUPP) {
				return false, nil
			}
			return size > 0, err
		},
		incarnation: incarnation,
	}
}

func (p openRGBLinuxProbe) observe(ctx context.Context, policy OpenRGBLinuxPolicy, check OpenRGBLinuxCheck) (OpenRGBLinuxEvidence, error) {
	var evidence OpenRGBLinuxEvidence
	if err := ctx.Err(); err != nil {
		return evidence, err
	}
	id, err := policy.ID()
	if err != nil {
		return evidence, err
	}
	if err := validateOpenRGBDetectorScope(policy); err != nil {
		return evidence, err
	}
	if id != check.PolicyID || policy.ArtifactSHA256 != check.ArtifactSHA256 || policy.DetectorSHA256 != check.DetectorSHA256 || check.EffectiveUID <= 0 || (check.PID == 0) != (check.Incarnation == "") || check.PID < 0 {
		return evidence, errors.New("OpenRGB device policy request does not match the enrolled launch identity")
	}
	machine, err := p.machineID()
	if err != nil || machine != policy.MachineID {
		return evidence, errors.Join(errors.New("OpenRGB policy machine identity could not be confirmed"), err)
	}
	digest, err := p.ruleDigest()
	if err != nil || digest != policy.RuleSHA256 {
		return evidence, errors.Join(errors.New("installed OpenRGB static udev grant is unverified"), err)
	}
	if policy.OwnerUID != 0 && policy.OwnerUID != uint32(check.EffectiveUID) {
		return evidence, errors.New("OpenRGB static device owner differs from the launch account")
	}
	if policy.GroupID != 0 {
		groups, err := p.accountGroups(check.EffectiveUID)
		if err != nil || !slices.Contains(groups, policy.GroupID) {
			return evidence, errors.Join(errors.New("persistent OpenRGB device group membership is unverified"), err)
		}
	}
	self, err := readOpenRGBCredentials(filepath.Join(p.proc, "self", "status"))
	if err != nil {
		return evidence, err
	}
	if !self.unprivilegedAs(check.EffectiveUID) || policy.GroupID != 0 && (!slices.Contains(self.groups, policy.GroupID) && self.gid[0] != policy.GroupID) {
		return evidence, errors.New("OpenRGB launcher lacks the selected unprivileged account credentials")
	}
	evidence.StaticGrantConfigured = true
	nodes, generation, err := p.observeNodes(ctx, policy)
	if err != nil {
		return evidence, err
	}
	evidence.SelectedNodes = nodes
	evidence.DeviceGenerationSHA256 = generation
	evidence.CurrentScopeVerified = true
	if check.PID == 0 {
		return evidence, nil
	}
	return p.observeChild(check, self, evidence)
}

func (p openRGBLinuxProbe) observeNodes(ctx context.Context, policy OpenRGBLinuxPolicy) ([]string, string, error) {
	entries, err := os.ReadDir(p.dev)
	if err != nil {
		return nil, "", fmt.Errorf("enumerate Linux character device names: %w", err)
	}
	selected := make(map[OpenRGBLinuxDevice]string, len(policy.Devices))
	identities := map[string]string{}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		name := entry.Name()
		kind := ""
		switch {
		case numberedOpenRGBNode(name, "i2c-"):
			kind = "i2c"
		case numberedOpenRGBNode(name, "hidraw"):
			kind = "hidraw"
		case name != "port":
			continue
		}
		path := filepath.Join(p.dev, name)
		var stat unix.Stat_t
		if err := p.stat(path, &stat); err != nil {
			return nil, "", fmt.Errorf("inspect OpenRGB device node %s: %w", name, err)
		}
		if stat.Mode&unix.S_IFMT != unix.S_IFCHR {
			return nil, "", fmt.Errorf("%s is not an exact character-device node", name)
		}
		if kind == "" {
			continue
		}
		device, err := p.nodeDevice(name, kind, stat.Rdev)
		if err != nil {
			return nil, "", err
		}
		selector, matched := matchOpenRGBSelector(device, policy.Devices)
		if !matched {
			for _, enrolled := range policy.Devices {
				if openRGBDetectorMatches(enrolled.Detector, device) {
					if err := p.requireDenied(path, stat); err != nil {
						return nil, "", fmt.Errorf("enabled OpenRGB detector could transact with an unenrolled matching route: %w", err)
					}
					break
				}
			}
			if err := p.confirmNode(path, stat); err != nil {
				return nil, "", err
			}
			continue
		}
		if selected[selector] != "" {
			return nil, "", fmt.Errorf("OpenRGB physical selector %q matched multiple nodes", selector.Topology)
		}
		if policy.OwnerUID != 0 {
			if stat.Uid != policy.OwnerUID || stat.Mode&0o600 != 0o600 {
				return nil, "", fmt.Errorf("%s lacks the enrolled persistent owner read/write grant", name)
			}
		} else {
			if stat.Uid != 0 || stat.Gid != policy.GroupID || stat.Mode&0o777 != 0o660 {
				return nil, "", fmt.Errorf("%s lacks the enrolled root-owned static group mode", name)
			}
			if acl, err := p.hasACL(path); err != nil || acl {
				return nil, "", errors.Join(fmt.Errorf("%s access cannot be attributed solely to the static device group", name), err)
			}
		}
		if err := p.access(path, unix.R_OK|unix.W_OK); err != nil {
			return nil, "", fmt.Errorf("selected OpenRGB node %s does not have effective read/write access: %w", name, err)
		}
		if err := p.confirmNode(path, stat); err != nil {
			return nil, "", err
		}
		selected[selector] = path
		identities[path] = fmt.Sprintf("%s:%d:%d:%d", path, stat.Dev, stat.Ino, stat.Rdev)
	}
	if len(selected) != len(policy.Devices) {
		return nil, "", errors.New("one or more enrolled OpenRGB physical devices are absent")
	}
	nodes := make([]string, 0, len(selected))
	for _, path := range selected {
		nodes = append(nodes, path)
	}
	slices.Sort(nodes)
	ordered := make([]string, 0, len(nodes))
	for _, path := range nodes {
		ordered = append(ordered, identities[path])
	}
	generation := sha256.Sum256([]byte(strings.Join(ordered, "\x00")))
	return nodes, fmt.Sprintf("%x", generation), nil
}

func numberedOpenRGBNode(name, prefix string) bool {
	suffix, found := strings.CutPrefix(name, prefix)
	return found && suffix != "" && strings.IndexFunc(suffix, func(r rune) bool { return r < '0' || r > '9' }) == -1
}

func (p openRGBLinuxProbe) requireDenied(path string, before unix.Stat_t) error {
	for _, mode := range []uint32{unix.R_OK, unix.W_OK} {
		err := p.access(path, mode)
		if err == nil {
			return fmt.Errorf("unrelated device %s is accessible to the OpenRGB launch account", path)
		}
		if !errors.Is(err, unix.EACCES) && !errors.Is(err, unix.EPERM) {
			return fmt.Errorf("denial of unrelated device %s is unverified: %w", path, err)
		}
	}
	return p.confirmNode(path, before)
}

func (p openRGBLinuxProbe) confirmNode(path string, before unix.Stat_t) error {
	var after unix.Stat_t
	if err := p.stat(path, &after); err != nil {
		return fmt.Errorf("recheck OpenRGB device %s: %w", path, err)
	}
	if after.Dev != before.Dev || after.Ino != before.Ino || after.Rdev != before.Rdev || after.Mode != before.Mode ||
		after.Uid != before.Uid || after.Gid != before.Gid || after.Ctim != before.Ctim {
		return fmt.Errorf("OpenRGB device %s changed during access observation", path)
	}
	return nil
}

func (p openRGBLinuxProbe) nodeDevice(name, kind string, rdev uint64) (OpenRGBLinuxDevice, error) {
	class := kind
	if kind == "i2c" {
		class = "i2c-dev"
	}
	classPath := filepath.Join(p.sys, "class", class, name)
	devicePath, err := filepath.EvalSymlinks(filepath.Join(classPath, "device"))
	if err != nil {
		return OpenRGBLinuxDevice{}, fmt.Errorf("resolve physical topology for %s: %w", name, err)
	}
	deviceRoot := filepath.Join(p.sys, "devices")
	rel, err := filepath.Rel(deviceRoot, devicePath)
	if err != nil || !validOpenRGBTopology(rel) {
		return OpenRGBLinuxDevice{}, fmt.Errorf("%s has no confined physical sysfs topology", name)
	}
	// Both selected paths have one dynamic immediate child. I2C's child is
	// the adapter number. HID's child is the kernel HID instance.
	hardware, adapter, err := openRGBHardwareIdentity(devicePath, kind)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return OpenRGBLinuxDevice{}, err
	}
	if errors.Is(err, os.ErrNotExist) && kind == "hidraw" {
		hardware, err = openRGBHIDIdentity(devicePath)
		if err != nil {
			return OpenRGBLinuxDevice{}, err
		}
	}
	number, err := readOpenRGBText(filepath.Join(classPath, "dev"))
	if err != nil || number != fmt.Sprintf("%d:%d", unix.Major(rdev), unix.Minor(rdev)) {
		return OpenRGBLinuxDevice{}, errors.Join(fmt.Errorf("%s device number does not match its sysfs identity", name), err)
	}
	return OpenRGBLinuxDevice{Kind: kind, Topology: filepath.Dir(rel), HardwareID: hardware, Adapter: adapter}, nil
}

func openRGBHIDIdentity(devicePath string) (string, error) {
	uevent, err := readOpenRGBText(filepath.Join(devicePath, "uevent"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(uevent, "\n") {
		if value, ok := strings.CutPrefix(line, "HID_ID="); ok {
			parts := strings.Split(strings.ToLower(value), ":")
			if len(parts) == 3 && openRGBHex(parts[1], 8) && openRGBHex(parts[2], 8) {
				return parts[1][4:] + ":" + parts[2][4:] + ":00", nil
			}
		}
	}
	return "", errors.New("HID physical identity could not be qualified from kernel metadata")
}

func matchOpenRGBSelector(device OpenRGBLinuxDevice, selectors []OpenRGBLinuxDevice) (OpenRGBLinuxDevice, bool) {
	for _, selector := range selectors {
		if selector.Kind == device.Kind && selector.Topology == device.Topology && selector.HardwareID == device.HardwareID && selector.Adapter == device.Adapter {
			return selector, true
		}
	}
	return OpenRGBLinuxDevice{}, false
}

func openRGBHardwareIdentity(devicePath, kind string) (string, string, error) {
	parent := filepath.Dir(devicePath)
	var attrs []string
	adapter := ""
	if kind == "i2c" {
		var err error
		adapter, err = readOpenRGBText(filepath.Join(devicePath, "name"))
		if err != nil {
			return "", "", err
		}
		for _, attr := range []string{"vendor", "device", "subsystem_vendor", "subsystem_device"} {
			attrs = append(attrs, filepath.Join(parent, attr))
		}
	} else {
		usb := filepath.Dir(parent)
		attrs = []string{filepath.Join(usb, "idVendor"), filepath.Join(usb, "idProduct"), filepath.Join(parent, "bInterfaceNumber")}
	}
	parts := make([]string, 0, len(attrs))
	for _, attr := range attrs {
		value, err := readOpenRGBText(attr)
		if err != nil {
			return "", "", err
		}
		parts = append(parts, strings.TrimPrefix(strings.ToLower(value), "0x"))
	}
	return strings.Join(parts, ":"), adapter, nil
}

func readOpenRGBText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func digestOpenRGBRule(path string) (string, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return "", err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	var info unix.Stat_t
	if err := unix.Fstat(fd, &info); err != nil {
		return "", err
	}
	if info.Mode&unix.S_IFMT != unix.S_IFREG || info.Uid != 0 || info.Mode&0o022 != 0 {
		return "", errors.New("OpenRGB udev grant is not a root-owned regular file protected from non-root writes")
	}
	const limit = 64 << 10
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(data) == 0 || len(data) > limit {
		return "", errors.Join(errors.New("OpenRGB udev grant is empty, unreadable or oversized"), err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func openRGBAccountGroups(uid int) ([]uint32, error) {
	account, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return nil, err
	}
	groups, err := account.GroupIds()
	if err != nil {
		return nil, err
	}
	return parseOpenRGBIDs(groups)
}

type openRGBCredentials struct {
	uid, gid [4]uint32
	groups   []uint32
	capable  bool
}

func (c openRGBCredentials) unprivilegedAs(uid int) bool {
	return uid > 0 && !c.capable && c.uid == [4]uint32{uint32(uid), uint32(uid), uint32(uid), uint32(uid)} &&
		!slices.Contains(c.gid[:], 0) && !slices.Contains(c.groups, 0)
}

func parseOpenRGBIDs(values []string) ([]uint32, error) {
	ids := make([]uint32, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, err
		}
		ids = append(ids, uint32(id))
	}
	return ids, nil
}

func readOpenRGBCredentials(path string) (openRGBCredentials, error) {
	var out openRGBCredentials
	data, err := readOpenRGBText(path)
	if err != nil {
		return out, err
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(data, "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			fields[key] = strings.TrimSpace(value)
		}
	}
	for _, item := range []struct {
		name string
		ids  *[4]uint32
	}{{"Uid", &out.uid}, {"Gid", &out.gid}} {
		ids, err := parseOpenRGBIDs(strings.Fields(fields[item.name]))
		if err != nil || len(ids) != len(item.ids) {
			return out, fmt.Errorf("Linux process %s credentials are incomplete", item.name)
		}
		copy(item.ids[:], ids)
	}
	groups, ok := fields["Groups"]
	if !ok {
		return out, errors.New("Linux process supplementary groups are unavailable")
	}
	out.groups, err = parseOpenRGBIDs(strings.Fields(groups))
	if err != nil {
		return out, err
	}
	slices.Sort(out.groups)
	for _, key := range []string{"CapInh", "CapPrm", "CapEff", "CapAmb"} {
		value, err := strconv.ParseUint(fields[key], 16, 64)
		if err != nil {
			return out, fmt.Errorf("Linux process %s capability evidence is unavailable", key)
		}
		out.capable = out.capable || value != 0
	}
	return out, nil
}

func (p openRGBLinuxProbe) observeChild(check OpenRGBLinuxCheck, self openRGBCredentials, evidence OpenRGBLinuxEvidence) (OpenRGBLinuxEvidence, error) {
	if p.incarnation == nil {
		return evidence, errors.New("OpenRGB child incarnation observer is unavailable")
	}
	before, err := p.incarnation(check.PID)
	if err != nil || before != check.Incarnation {
		return evidence, errors.Join(errors.New("OpenRGB child incarnation could not be confirmed"), err)
	}
	pidDir := filepath.Join(p.proc, strconv.Itoa(check.PID))
	child, err := readOpenRGBCredentials(filepath.Join(pidDir, "status"))
	if err != nil {
		return evidence, err
	}
	if !child.unprivilegedAs(check.EffectiveUID) || child.uid != self.uid || child.gid != self.gid || !slices.Equal(child.groups, self.groups) {
		return evidence, errors.New("OpenRGB child credentials differ from the qualified launcher account")
	}
	selfScope, selfErr := readOpenRGBText(filepath.Join(p.proc, "self", "cgroup"))
	childScope, childErr := readOpenRGBText(filepath.Join(pidDir, "cgroup"))
	after, err := p.incarnation(check.PID)
	if err != nil || after != before {
		return evidence, errors.Join(errors.New("OpenRGB child incarnation changed during credential observation"), err)
	}
	evidence.LiveCredentialsMatch = true
	if selfErr != nil || childErr != nil || selfScope == "" || childScope == "" {
		return evidence, errors.Join(ErrOpenRGBLiveDeviceAccessUnverified, selfErr, childErr)
	}
	if selfScope != childScope {
		return evidence, fmt.Errorf("%w: the OpenRGB child runs in a different cgroup from the access checker", ErrOpenRGBLiveDeviceAccessUnverified)
	}
	// Matching account and scope evidence cannot rule out a distinct mount or
	// security context or a later restriction on the selected nodes.
	return evidence, ErrOpenRGBLiveDeviceAccessUnverified
}
