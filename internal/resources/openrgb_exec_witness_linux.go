//go:build linux

package resources

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/process"
	"golang.org/x/sys/unix"
)

// Linux applies no-new-privileges to an OS thread. Keep the checking thread
// pinned through exec, which preserves that thread's restriction in the image.
func confineOpenRGBExecThread() (func(), error) {
	runtime.LockOSThread()
	previous, err := unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	if err == nil && previous == 1 {
		return runtime.UnlockOSThread, nil
	}
	if err == nil {
		err = unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
	}
	if err != nil {
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("confine OpenRGB exec thread: %w", err)
	}
	// This restriction is irreversible. If exec returns, the native launcher
	// command exits, so the restricted thread stays pinned until termination.
	return func() {}, nil
}

func openRGBExecContext(ref platform.ServiceRef, incarnation string) (string, error) {
	if ref.RootPID != os.Getpid() {
		return "", errors.New("OpenRGB launcher is outside the admitted native root")
	}
	const threadDir = "/proc/thread-self"
	thread, err := openRGBExecCredentialLines(threadDir + "/status")
	if err != nil {
		return "", err
	}
	leader, err := openRGBExecCredentialLines("/proc/self/status")
	if err != nil || thread != leader {
		return "", errors.Join(errors.New("OpenRGB device checker and exec thread credentials differ"), err)
	}
	return openRGBNativeContext(ref, incarnation, threadDir)
}

func openRGBServingContext(ref platform.ServiceRef, incarnation string) (string, error) {
	return openRGBNativeContext(ref, incarnation, filepath.Join("/proc", strconv.Itoa(ref.RootPID)))
}

func openRGBNativeContext(ref platform.ServiceRef, incarnation, procDir string) (string, error) {
	if ref.ValidateIdentity() != nil || ref.RootPID <= 0 || ref.ScopeDevice == 0 || ref.ScopeInode == 0 {
		return "", errors.New("OpenRGB launch lacks exact native root and cgroup inode evidence")
	}
	current, err := process.Incarnation(ref.RootPID)
	if err != nil || current != incarnation {
		return "", errors.Join(errors.New("OpenRGB launcher incarnation changed"), err)
	}
	cgroups, err := os.ReadFile(filepath.Join(procDir, "cgroup"))
	if err != nil || !strings.Contains("\n"+string(cgroups)+"\n", "\n0::"+ref.Scope.Path+"\n") {
		return "", errors.Join(errors.New("OpenRGB exec thread is outside its admitted native cgroup"), err)
	}
	var scope unix.Stat_t
	if err := unix.Stat(filepath.Join("/sys/fs/cgroup", ref.Scope.Path), &scope); err != nil || uint64(scope.Dev) != ref.ScopeDevice || scope.Ino != ref.ScopeInode {
		return "", errors.Join(errors.New("OpenRGB launcher cgroup inode changed"), err)
	}
	status, err := os.ReadFile(filepath.Join(procDir, "status"))
	if err != nil || openRGBExecStatusFields(status)["NoNewPrivs"] != "1" {
		return "", errors.Join(errors.New("OpenRGB process has no verified no-new-privileges restriction"), err)
	}
	digest, err := openRGBExecContextDigest(procDir)
	if err != nil {
		return "", err
	}
	current, err = process.Incarnation(ref.RootPID)
	if err != nil || current != incarnation {
		return "", errors.Join(errors.New("OpenRGB native root changed during context observation"), err)
	}
	return digest, nil
}

func openRGBExecContextDigest(procDir string) (string, error) {
	var evidence []string
	for _, namespace := range []string{"mnt", "user", "cgroup", "net"} {
		id, err := os.Readlink(filepath.Join(procDir, "ns", namespace))
		if err != nil || id == "" {
			return "", errors.Join(fmt.Errorf("OpenRGB exec thread has no %s namespace identity", namespace), err)
		}
		evidence = append(evidence, id)
	}
	label, err := os.ReadFile(filepath.Join(procDir, "attr", "current"))
	if err != nil || len(label) == 0 {
		return "", errors.Join(errors.New("OpenRGB exec thread has no security context evidence"), err)
	}
	evidence = append(evidence, string(label))
	credentials, err := openRGBExecCredentialLines(filepath.Join(procDir, "status"))
	if err != nil {
		return "", err
	}
	evidence = append(evidence, credentials)
	data, err := json.Marshal(evidence)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func openRGBExecCredentialLines(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	fields := openRGBExecStatusFields(data)
	var values []string
	for _, key := range []string{"Uid", "Gid", "Groups", "CapInh", "CapPrm", "CapEff", "CapBnd", "CapAmb"} {
		value, ok := fields[key]
		if !ok || value == "" && key != "Groups" {
			return "", fmt.Errorf("OpenRGB exec thread lacks %s evidence", key)
		}
		values = append(values, value)
	}
	return strings.Join(values, "\n"), nil
}

func openRGBExecStatusFields(data []byte) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok {
			fields[key] = strings.TrimSpace(value)
		}
	}
	return fields
}
