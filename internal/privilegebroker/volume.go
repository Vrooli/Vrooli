package privilegebroker

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Volume evidence sources. They are package variables so tests can substitute
// fixture trees without the broker growing a general configuration surface.
var (
	volumeProcMounts    = "/proc/mounts"
	volumeDevDiskByUUID = "/dev/disk/by-uuid"
	volumeReadFile      = os.ReadFile
	volumeReadDir       = os.ReadDir
	volumeEvalSymlinks  = filepath.EvalSymlinks
)

// volumeSystemMountPrefixes are mountpoints whose volume keeps the host
// running. The broker refuses to touch them regardless of what the caller asks,
// because a caller mistake here takes the machine down.
var volumeSystemMountPrefixes = []string{"/", "/boot", "/usr", "/etc", "/var", "/home", "/root"}

// executeVolume runs a validated volume action behind the broker's own,
// independent checks. The caller's claims are never trusted: the broker
// re-reads mount state and re-resolves the device identity itself before any
// filesystem tool touches the disk.
func executeVolume(ctx context.Context, executor Executor, req Request) Result {
	fail := func(code string) Result { return NewFailure(req.RequestID, req.Action, code) }

	mountpoint, mounted := volumeMountpoint(req.Volume.Device)
	if mounted {
		if isVolumeSystemMount(mountpoint) {
			return fail("system_volume_refused")
		}
		// Every supported filesystem tool needs exclusive access. Running one
		// against a mounted volume risks corrupting the data the repair exists
		// to protect, so this is a refusal rather than an unmount-and-retry.
		return Result{
			Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action,
			Status: "failed", Code: "volume_mounted",
			Evidence: Evidence{Available: true, Mounted: true, Detail: "volume is mounted at " + mountpoint},
		}
	}

	if !volumeIdentityMatches(*req.Volume) {
		return fail("device_identity_mismatch")
	}

	tool, args, err := VolumeArgs(req)
	if err != nil {
		return fail(err.Error())
	}
	out, runErr := executor.Run(ctx, tool, args...)
	exitCode := volumeExitCode(runErr)
	if runErr != nil && errors.Is(runErr, exec.ErrNotFound) {
		return Result{
			Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action,
			Status: "unavailable", Code: "filesystem_tool_unavailable",
			Evidence: Evidence{IdentityVerified: true, Detail: tool + " is not installed"},
		}
	}

	evidence := Evidence{
		Available:        true,
		IdentityVerified: true,
		ExitCode:         exitCode,
		Detail:           boundedDetail(out),
	}
	if !volumeExitAcceptable(req.Volume.Filesystem, exitCode, runErr) {
		return Result{
			Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action,
			Status: "failed", Code: "filesystem_tool_failed", Evidence: evidence,
		}
	}
	if req.Action == ActionVolumeFilesystemCheck {
		// A check proves state; it never moves it.
		return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "verified", Evidence: evidence}
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "changed", Changed: true, Evidence: evidence}
}

// volumeMountpoint reports where a device is mounted, if it is.
func volumeMountpoint(device string) (string, bool) {
	data, err := volumeReadFile(volumeProcMounts)
	if err != nil {
		// An unreadable mount table means the broker cannot prove the volume is
		// detached. Reporting "mounted" is the safe direction: it refuses.
		return "unknown", true
	}
	want := resolveVolumePath(device)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if resolveVolumePath(fields[0]) == want {
			return fields[1], true
		}
	}
	return "", false
}

func isVolumeSystemMount(mountpoint string) bool {
	clean := filepath.Clean(strings.TrimSpace(mountpoint))
	if clean == "/" || clean == "." || clean == "unknown" {
		return true
	}
	for _, prefix := range volumeSystemMountPrefixes {
		if prefix == "/" {
			continue
		}
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}

// volumeIdentityMatches re-resolves the claimed identity against the host. A
// UUID that does not resolve to the claimed device means the disk was swapped
// or the caller is wrong; either way the request must not proceed.
func volumeIdentityMatches(subject VolumeSubject) bool {
	uuid := strings.TrimSpace(subject.UUID)
	if uuid == "" {
		// Validate already required an identifier. A serial-only claim cannot
		// be re-resolved from filesystem evidence alone, so the broker declines
		// rather than accepting an unverifiable identity.
		return false
	}
	if strings.ContainsAny(uuid, `/\`) {
		return false
	}
	link := filepath.Join(volumeDevDiskByUUID, uuid)
	if _, err := volumeReadDir(volumeDevDiskByUUID); err != nil {
		return false
	}
	return resolveVolumePath(link) == resolveVolumePath(subject.Device)
}

func resolveVolumePath(path string) string {
	path = strings.TrimSpace(path)
	if real, err := volumeEvalSymlinks(path); err == nil {
		return real
	}
	return path
}

// volumeExitCode extracts a tool's exit status, or -1 when there is none.
func volumeExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// volumeExitAcceptable interprets exit status per filesystem family. Treating
// every non-zero status as failure would report a successful repair as a
// failure: e2fsck returns 1 precisely when it corrected errors.
func volumeExitAcceptable(filesystem string, exitCode int, err error) bool {
	if err == nil {
		return true
	}
	if exitCode < 0 {
		return false
	}
	switch volumeFilesystems[strings.ToLower(strings.TrimSpace(filesystem))] {
	case "ext":
		// 0 clean, 1 errors corrected, 2 corrected and a reboot is advised.
		return exitCode <= 2
	case "vfat":
		// 0 clean, 1 errors corrected.
		return exitCode <= 1
	default:
		return exitCode == 0
	}
}

func boundedDetail(out []byte) string {
	const maxDetail = 1024
	text := strings.TrimSpace(string(out))
	if len(text) <= maxDetail {
		return text
	}
	return text[:maxDetail] + "… (truncated)"
}
