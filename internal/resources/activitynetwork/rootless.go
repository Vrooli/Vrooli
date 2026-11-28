package activitynetwork

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config describes a rootless network boundary for a managed resource. A
// caller may publish one loopback edge for generic owners, while the LM1
// resource contract publishes nothing and uses the host edge's authorized
// private Unix bridge instead.
type Config struct {
	StateDir     string
	HostPort     int
	GuestPort    int
	PublishEdge  bool
	ChildCommand []string
	UID          int
	GID          int
}

func (c Config) Validate() error {
	if !filepath.IsAbs(strings.TrimSpace(c.StateDir)) || filepath.Clean(c.StateDir) != c.StateDir {
		return errors.New("activity network state directory must be absolute and clean")
	}
	if c.PublishEdge {
		if c.HostPort < 1 || c.HostPort > 65535 || c.GuestPort < 1 || c.GuestPort > 65535 {
			return errors.New("published activity edge ports must be in range 1..65535")
		}
	} else if c.HostPort != 0 || c.GuestPort != 0 {
		return errors.New("unpublished activity network must not declare host or guest ports")
	}
	if c.UID < 0 || c.GID < 0 {
		return errors.New("activity network owner uid and gid must be nonnegative")
	}
	if len(c.ChildCommand) == 0 || strings.TrimSpace(c.ChildCommand[0]) == "" || !filepath.IsAbs(c.ChildCommand[0]) {
		return errors.New("activity network child command must start with an absolute executable path")
	}
	for _, arg := range c.ChildCommand {
		if strings.ContainsRune(arg, '\x00') {
			return errors.New("activity network child command contains a NUL byte")
		}
	}
	return nil
}

// RootlessKitArgs returns a closed argv vector. It never accepts a bind IP,
// port protocol, or network driver beyond the exact optional loopback mapping
// declared by the resource owner.
func RootlessKitArgs(c Config) ([]string, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	args := []string{
		"--state-dir", c.StateDir,
		"--subid-source=static",
		"--copy-up=/etc",
		"--net=slirp4netns",
		"--disable-host-loopback",
		"--slirp4netns-sandbox=true",
		"--slirp4netns-seccomp=true",
		"--port-driver=builtin",
	}
	if c.PublishEdge {
		args = append(args, "--publish", fmt.Sprintf("127.0.0.1:%d:%d/tcp", c.HostPort, c.GuestPort))
	}
	args = append(args, "--",
		c.ChildCommand[0],
	)
	args = append(args, c.ChildCommand[1:]...)
	return args, nil
}

// ReadChildPID reads RootlessKit's child PID only from its exact private state
// directory. The directory itself is owner-controlled by the resource owner.
func ReadChildPID(stateDir string) (int, error) {
	if !filepath.IsAbs(stateDir) || filepath.Clean(stateDir) != stateDir {
		return 0, errors.New("activity network state directory must be absolute and clean")
	}
	path := filepath.Join(stateDir, "child_pid")
	info, err := os.Lstat(path)
	if err != nil {
		return 0, fmt.Errorf("read RootlessKit child PID: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > 32 {
		return 0, errors.New("RootlessKit child PID file is not a small regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read RootlessKit child PID: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return 0, errors.New("RootlessKit child PID is invalid")
	}
	return pid, nil
}

// WaitForChildPID waits for RootlessKit to publish its child PID. The managed
// service supervisor returns as soon as RootlessKit is spawned, before the
// child_pid file exists, so a single read races the namespace setup. Only a
// missing file is retried; a malformed file fails immediately.
func WaitForChildPID(stateDir string, timeout, interval time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := ReadChildPID(stateDir)
		if err == nil || !errors.Is(err, os.ErrNotExist) || !time.Now().Before(deadline) {
			return pid, err
		}
		time.Sleep(interval)
	}
}

// OwnerNamespaceArgs constructs the only supported administrative bridge to
// a service's private network namespace. Workload callers must use the
// published activity edge; resource-owner CLI operations (for example model
// unload or health inspection) may join the exact RootlessKit child namespace.
func OwnerNamespaceArgs(childPID int, command []string) ([]string, error) {
	if childPID <= 1 {
		return nil, errors.New("activity network child PID must be greater than one")
	}
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" || !filepath.IsAbs(command[0]) {
		return nil, errors.New("activity network owner command must start with an absolute executable path")
	}
	for _, arg := range command {
		if strings.ContainsRune(arg, '\x00') {
			return nil, errors.New("activity network owner command contains a NUL byte")
		}
	}
	args := []string{"--user", "--preserve-credentials", "--net", "--target", strconv.Itoa(childPID), "--", command[0]}
	return append(args, command[1:]...), nil
}

// PrepareSingleIDMapHelpers writes narrowly scoped RootlessKit helper
// executables. They map namespace root to the current Vrooli owner only; they
// never map subordinate IDs or grant capabilities in the host namespace.
// RootlessKit otherwise requires setuid newuidmap/newgidmap helpers even when
// the managed process needs only the caller's own UID and GID.
func PrepareSingleIDMapHelpers(dir string, uid, gid int) error {
	if !filepath.IsAbs(strings.TrimSpace(dir)) || filepath.Clean(dir) != dir {
		return errors.New("activity network helper directory must be absolute and clean")
	}
	if uid < 0 || gid < 0 {
		return errors.New("activity network owner uid and gid must be nonnegative")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create activity network helper directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("restrict activity network helper directory: %w", err)
	}
	uidText, gidText := strconv.Itoa(uid), strconv.Itoa(gid)
	uidScript := idMapScript("uid_map", "", uidText)
	gidScript := idMapScript("gid_map", "setgroups", gidText)
	for name, content := range map[string]string{"newuidmap": uidScript, "newgidmap": gidScript} {
		path := filepath.Join(dir, name)
		if err := writePrivateExecutable(path, []byte(content)); err != nil {
			return fmt.Errorf("write activity network %s: %w", name, err)
		}
	}
	return nil
}

func idMapScript(mapping, prewrite, ownerID string) string {
	var before string
	if prewrite != "" {
		before = "printf deny > \"/proc/$1/" + prewrite + "\"\n"
	}
	return "#!/bin/sh\n" +
		"set -eu\n" +
		// RootlessKit supplies its configured subordinate ranges after the
		// owner mapping. They are intentionally ignored: the only map we write
		// is 0 -> owner, length 1.
		"[ \"$#\" -ge 4 ] || exit 65\n" +
		"case \"$1\" in ''|*[!0-9]*) exit 65 ;; esac\n" +
		"[ \"$2\" = 0 ] && [ \"$3\" = " + ownerID + " ] && [ \"$4\" = 1 ] || exit 65\n" +
		before +
		"printf '0 %s 1\\n' " + ownerID + " > \"/proc/$1/" + mapping + "\"\n"
}

func writePrivateExecutable(path string, body []byte) error {
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Chmod(path, 0o700)
}
