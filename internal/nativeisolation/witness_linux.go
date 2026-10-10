//go:build linux

package nativeisolation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/api-core/effortauthority"
)

type liveWitnessReader struct{}

func newWitnessReader() witnessReader    { return liveWitnessReader{} }
func (liveWitnessReader) now() time.Time { return time.Now().UTC() }
func (liveWitnessReader) qualify(ctx context.Context, q *effortauthority.NativeUIDIsolation) error {
	return q.Qualify(ctx)
}

var witnessProperties = []string{"Id", "LoadState", "ActiveState", "SubState", "MainPID", "ExecMainStartTimestampMonotonic", "InvocationID", "NRestarts", "TasksCurrent", "User", "Group", "RootDirectory", "WorkingDirectory", "BindPaths", "ReadWritePaths", "NoNewPrivileges", "CapabilityBoundingSet", "AmbientCapabilities", "Restart", "KillMode", "Delegate", "PrivateMounts", "PrivateDevices", "ProtectSystem", "ProtectHome", "RestrictNamespaces", "RestrictSUIDSGID", "ProtectKernelTunables", "ProtectKernelModules", "ProtectKernelLogs", "ProtectControlGroups", "RestrictAddressFamilies", "IPAddressDeny", "IPAddressAllow", "RuntimeMaxUSec", "TimeoutStopUSec"}

func witnessShow(raw []byte, s ProbeSpec) (int, string, error) {
	values := map[string]string{}
	allowed := map[string]bool{}
	for _, key := range witnessProperties {
		allowed[key] = true
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || !allowed[key] {
			return 0, "", ErrRefused
		}
		if _, duplicate := values[key]; duplicate {
			return 0, "", ErrRefused
		}
		values[key] = value
	}
	if len(values) != len(allowed) {
		return 0, "", ErrRefused
	}
	expected := map[string]string{"Id": s.unit(), "LoadState": "loaded", "ActiveState": "active", "SubState": "running", "NRestarts": "0", "TasksCurrent": "1", "User": fmt.Sprint(s.Manifest.NativeUID), "Group": fmt.Sprint(s.Manifest.NativeGID), "RootDirectory": s.Manifest.RootFS, "WorkingDirectory": s.repository(), "BindPaths": s.Manifest.WorkspaceSource + ":" + s.repository(), "ReadWritePaths": s.repository(), "NoNewPrivileges": "yes", "CapabilityBoundingSet": "", "AmbientCapabilities": "", "Restart": "no", "KillMode": "control-group", "Delegate": "no", "PrivateMounts": "yes", "PrivateDevices": "yes", "ProtectSystem": "strict", "ProtectHome": "tmpfs", "RestrictNamespaces": "yes", "RestrictSUIDSGID": "yes", "ProtectKernelTunables": "yes", "ProtectKernelModules": "yes", "ProtectKernelLogs": "yes", "ProtectControlGroups": "yes"}
	for k, v := range expected {
		if values[k] != v {
			return 0, "", ErrRefused
		}
	}
	if strings.ToUpper(values["RestrictAddressFamilies"]) != "AF_INET AF_INET6" || (values["IPAddressDeny"] != "any" && values["IPAddressDeny"] != "0.0.0.0/0 ::/0") || values["IPAddressAllow"] != strings.Join(s.Manifest.NetworkAllow, " ") {
		return 0, "", ErrRefused
	}
	max, de := systemdWitnessDuration(values["RuntimeMaxUSec"])
	stop, se := systemdWitnessDuration(values["TimeoutStopUSec"])
	limit := 8 * time.Hour
	if s.Purpose == FirstQualification {
		limit = 5 * time.Minute
	}
	if de != nil || se != nil || max <= 0 || max > limit || stop != 5*time.Second {
		return 0, "", ErrRefused
	}
	if _, e := strictMicrosecondTimestamp(values["ExecMainStartTimestampMonotonic"]); e != nil {
		return 0, "", ErrRefused
	}
	pid, e := strconv.Atoi(values["MainPID"])
	id, ie := hex.DecodeString(values["InvocationID"])
	if e != nil || pid <= 0 || strconv.Itoa(pid) != values["MainPID"] || ie != nil || len(id) != 16 || values["InvocationID"] == strings.Repeat("0", 32) || strings.ToLower(values["InvocationID"]) != values["InvocationID"] {
		return 0, "", ErrRefused
	}
	return pid, values["InvocationID"], nil
}

func startIdentity(raw []byte, pid int) (string, error) {
	text := strings.TrimSpace(string(raw))
	first := strings.Index(text, " (")
	last := strings.LastIndex(text, ") ")
	if first < 1 || last <= first || text[:first] != strconv.Itoa(pid) {
		return "", ErrRefused
	}
	fields := strings.Fields(text[last+2:])
	// State is field3; starttime is field22. Dead/zombie states are not evidence.
	if len(fields) < 20 || (fields[0] != "S" && fields[0] != "R" && fields[0] != "I") {
		return "", ErrRefused
	}
	ticks, e := strconv.ParseUint(fields[19], 10, 64)
	if e != nil || ticks == 0 {
		return "", ErrRefused
	}
	return strconv.FormatUint(ticks, 10), nil
}

func witnessGID(raw []byte, gid uint32) error {
	seen := false
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "Gid:") {
			continue
		}
		if seen {
			return ErrRefused
		}
		seen = true
		values := strings.Fields(strings.TrimPrefix(line, "Gid:"))
		if len(values) != 4 {
			return ErrRefused
		}
		for _, v := range values {
			n, e := strconv.ParseUint(v, 10, 32)
			if e != nil || uint32(n) != gid {
				return ErrRefused
			}
		}
	}
	if !seen {
		return ErrRefused
	}
	return nil
}

func cgroupIdentity(raw []byte, unit string) (string, error) {
	line := strings.TrimSpace(string(raw))
	if strings.Contains(line, "\n") || !strings.HasPrefix(line, "0::/system.slice/") {
		return "", ErrRefused
	}
	path := strings.TrimPrefix(line, "0::")
	if path != "/system.slice/"+unit {
		return "", ErrRefused
	}
	return path, nil
}

func inodeIdentity(path string) (string, error) {
	info, e := os.Stat(path)
	if e != nil || !info.IsDir() {
		return "", ErrRefused
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ErrRefused
	}
	return fmt.Sprintf("%d:%d", st.Dev, st.Ino), nil
}

func (liveWitnessReader) read(ctx context.Context, s ProbeSpec) (witnessSnapshot, error) {
	var out witnessSnapshot
	if s.validate(ctx, time.Now().UTC()) != nil || checkExecutable(s.Manifest.Systemctl, s.Manifest.NativeUID) != nil || trustedTree(s.Manifest.RootFS, s.Manifest.NativeUID) != nil || trustedPath(filepath.Dir(s.Manifest.WorkspaceSource), s.Manifest.NativeUID) != nil || checkWorkspace(s.Manifest.WorkspaceSource, s.Manifest.Isolation.ProtectedPaths) != nil {
		return out, ErrRefused
	}
	helper := Executable{Path: filepath.Join(s.Manifest.RootFS, ProbeHelperPath), SHA256: s.HelperSHA256}
	if checkExecutable(helper, s.Manifest.NativeUID) != nil {
		return out, ErrRefused
	}
	image, e := treeDigest(s.Manifest.RootFS)
	if e != nil || image != s.Manifest.RootFSDigest {
		return out, ErrRefused
	}
	// Read-only exact pinned executable; no shell, status wrapper, refresh or retry.
	cmd := exec.CommandContext(ctx, s.Manifest.Systemctl.Path, "--system", "show", s.unit(), "--property="+strings.Join(witnessProperties, ","), "--no-pager")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	raw, e := cmd.Output()
	if e != nil {
		return out, ErrRefused
	}
	pid, invocation, e := witnessShow(raw, s)
	if e != nil {
		return out, ErrRefused
	}
	uptime, e := os.ReadFile("/proc/uptime")
	if e != nil {
		return out, ErrRefused
	}
	unitValues := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		k, v, _ := strings.Cut(line, "=")
		unitValues[k] = v
	}
	if witnessRuntimeDeadline(time.Now().UTC(), s.Deadline, unitValues["ExecMainStartTimestampMonotonic"], unitValues["RuntimeMaxUSec"], uptime) != nil {
		return out, ErrRefused
	}
	base := filepath.Join("/proc", strconv.Itoa(pid))
	status, e := os.ReadFile(filepath.Join(base, "status"))
	if e != nil || validatePosture(status, s.Manifest.NativeUID) != nil || witnessGID(status, s.Manifest.NativeGID) != nil {
		return out, ErrRefused
	}
	stat, e := os.ReadFile(filepath.Join(base, "stat"))
	if e != nil {
		return out, ErrRefused
	}
	start, e := startIdentity(stat, pid)
	if e != nil {
		return out, ErrRefused
	}
	group, e := os.ReadFile(filepath.Join(base, "cgroup"))
	if e != nil {
		return out, ErrRefused
	}
	cgroup, e := cgroupIdentity(group, s.unit())
	if e != nil {
		return out, ErrRefused
	}
	command, e := os.ReadFile(filepath.Join(base, "cmdline"))
	if e != nil || string(command) != ProbeHelperPath+"\x00--until\x00"+s.Deadline.UTC().Format(time.RFC3339Nano)+"\x00" {
		return out, ErrRefused
	}
	executable, e := os.ReadFile(filepath.Join(base, "exe"))
	if e != nil {
		return out, ErrRefused
	}
	hash := sha256.Sum256(executable)
	if hex.EncodeToString(hash[:]) != s.HelperSHA256 {
		return out, ErrRefused
	}
	root, e := inodeIdentity(filepath.Join(base, "root"))
	if e != nil {
		return out, ErrRefused
	}
	expectedRoot, e := inodeIdentity(s.Manifest.RootFS)
	if e != nil || root != expectedRoot {
		return out, ErrRefused
	}
	workspace, e := inodeIdentity(s.Manifest.WorkspaceSource)
	if e != nil {
		return out, ErrRefused
	}
	boundWorkspace, e := inodeIdentity(filepath.Join(base, "root", s.repository()))
	if e != nil || boundWorkspace != workspace {
		return out, ErrRefused
	}
	namespace, e := os.Readlink(filepath.Join(base, "ns", "mnt"))
	if e != nil || mountNamespaceIdentity(namespace) != nil {
		return out, ErrRefused
	}
	mountInfo, e := os.ReadFile(filepath.Join(base, "mountinfo"))
	if e != nil || len(mountInfo) == 0 {
		return out, ErrRefused
	}
	mountHash := sha256.Sum256(mountInfo)
	// Reject a changing mount view during this read as well as between reads.
	namespaceAfter, ne := os.Readlink(filepath.Join(base, "ns", "mnt"))
	mountInfoAfter, me := os.ReadFile(filepath.Join(base, "mountinfo"))
	workspaceAfter, we := inodeIdentity(filepath.Join(base, "root", s.repository()))
	if ne != nil || me != nil || we != nil || namespaceAfter != namespace || sha256.Sum256(mountInfoAfter) != mountHash || workspaceAfter != workspace {
		return out, ErrRefused
	}
	statAfter, e := os.ReadFile(filepath.Join(base, "stat"))
	if e != nil {
		return out, ErrRefused
	}
	startAfter, e := startIdentity(statAfter, pid)
	if e != nil || startAfter != start {
		return out, ErrRefused
	}
	return witnessSnapshot{PID: pid, StartTicks: start, InvocationID: invocation, Cgroup: cgroup, ExecutableHash: s.HelperSHA256, ImageHash: image, RootIdentity: root, WorkspaceIdentity: workspace, MountNamespace: namespace, MountInfoHash: hex.EncodeToString(mountHash[:]), UnitPostureHash: digest(string(raw))}, nil
}

func mountNamespaceIdentity(raw string) error {
	if !strings.HasPrefix(raw, "mnt:[") || !strings.HasSuffix(raw, "]") {
		return ErrRefused
	}
	value := strings.TrimSuffix(strings.TrimPrefix(raw, "mnt:["), "]")
	n, e := strconv.ParseUint(value, 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != value {
		return ErrRefused
	}
	return nil
}

// Accept only bounded systemctl duration syntax, never infinity or an implicit unit.
func systemdWitnessDuration(raw string) (time.Duration, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 || len(fields) > 6 {
		return 0, ErrRefused
	}
	var total time.Duration
	for _, field := range fields {
		d, e := time.ParseDuration(strings.ReplaceAll(field, "min", "m"))
		if e != nil || d <= 0 || d > 8*time.Hour || total > 8*time.Hour-d {
			return 0, ErrRefused
		}
		total += d
	}
	return total, nil
}

func strictMicrosecondTimestamp(raw string) (time.Duration, error) {
	n, e := strconv.ParseUint(raw, 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != raw || n > uint64((1<<63-1)/int64(time.Microsecond)) {
		return 0, ErrRefused
	}
	return time.Duration(n) * time.Microsecond, nil
}

// /proc/uptime includes suspend time. Subtracting it from systemd's monotonic
// start is conservative: suspend can only shorten the accepted remaining life.
func witnessRemainingRuntime(startRaw, maxRaw string, uptimeRaw []byte) (time.Duration, error) {
	start, e := strictMicrosecondTimestamp(startRaw)
	max, me := systemdWitnessDuration(maxRaw)
	fields := strings.Fields(string(uptimeRaw))
	if e != nil || me != nil || len(fields) != 2 {
		return 0, ErrRefused
	}
	// Kernel uptime uses decimal seconds, never signs, exponent or units.
	for _, c := range fields[0] {
		if (c < '0' || c > '9') && c != '.' {
			return 0, ErrRefused
		}
	}
	uptime, ue := time.ParseDuration(fields[0] + "s")
	if ue != nil || uptime < start || uptime-start >= max {
		return 0, ErrRefused
	}
	return max - (uptime - start), nil
}

func witnessRuntimeDeadline(now, deadline time.Time, startRaw, maxRaw string, uptimeRaw []byte) error {
	remaining, e := witnessRemainingRuntime(startRaw, maxRaw, uptimeRaw)
	if e != nil || !now.Before(deadline) || now.Add(remaining).After(deadline) {
		return ErrRefused
	}
	return nil
}
