//go:build linux

package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var dockerDiskIdentityTimeout = 15 * time.Second

type dockerDiskInspect struct {
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
	GraphDriver struct {
		Data map[string]string `json:"Data"`
	} `json:"GraphDriver"`
}

// resolveDockerDiskPath asks the Docker daemon for the exact running container
// mount or writable layer, then uses the kernel mount table (not caller data)
// to bind that path to a device identity and an accessible statfs probe.
func resolveDockerDiskPath(ctx context.Context, p DiskJobPath) (string, string, int64, string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, dockerDiskIdentityTimeout)
	defer cancel()
	command := exec.CommandContext(queryCtx, "docker", "inspect", "--type", "container", p.DockerContainer)
	output, err := command.Output()
	if err != nil {
		return "", "", 0, "", fmt.Errorf("inspect Docker container %q for disk path: %w", p.DockerContainer, err)
	}
	var inspected []dockerDiskInspect
	if err := json.Unmarshal(output, &inspected); err != nil {
		return "", "", 0, "", fmt.Errorf("decode Docker container disk-path identity: %w", err)
	}
	if len(inspected) != 1 || !inspected[0].State.Running {
		return "", "", 0, "", fmt.Errorf("Docker disk path requires exactly one running container")
	}
	var actual string
	if p.DockerUpperDir {
		actual = inspected[0].GraphDriver.Data["UpperDir"]
	} else {
		for _, mount := range inspected[0].Mounts {
			if filepath.Clean(mount.Destination) == filepath.Clean(p.DockerDestination) {
				if !mount.RW {
					return "", "", 0, "", fmt.Errorf("Docker mount %q is read-only", mount.Destination)
				}
				actual = mount.Source
				break
			}
		}
	}
	if actual == "" {
		return "", "", 0, "", fmt.Errorf("Docker container does not expose requested disk path")
	}
	actual, err = filepath.Abs(filepath.Clean(actual))
	if err != nil {
		return "", "", 0, "", err
	}
	wanted, err := filepath.Abs(filepath.Clean(p.Path))
	if err != nil {
		return "", "", 0, "", err
	}
	if wanted != actual {
		return "", "", 0, "", fmt.Errorf("declared Docker disk path %s does not match daemon source %s", wanted, actual)
	}
	fsid, mountpoint, err := filesystemFromMountInfo(actual)
	if err != nil {
		return "", "", 0, "", err
	}
	probe, verifiedID, free, err := diskFilesystem(mountpoint)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("probe Docker path filesystem at %s: %w", mountpoint, err)
	}
	if verifiedID != fsid {
		return "", "", 0, "", fmt.Errorf("kernel mount identity mismatch for Docker path %s: mount table=%s statfs=%s", actual, fsid, verifiedID)
	}
	return actual, fsid, free, probe, nil
}

// resolveDockerRootDir binds an admission to Docker's daemon-reported storage
// root. This lets a caller reserve the Docker filesystem before creating the
// container, volume, or writable layer whose exact paths do not exist yet.
// The request must name the same root; filesystem identity and free space come
// from the host mount table and an accessible mountpoint probe.
func resolveDockerRootDir(ctx context.Context, p DiskJobPath) (string, string, int64, string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, dockerDiskIdentityTimeout)
	defer cancel()
	command := exec.CommandContext(queryCtx, "docker", "info", "--format", "{{.DockerRootDir}}")
	output, err := command.Output()
	if err != nil {
		return "", "", 0, "", fmt.Errorf("query Docker daemon root for disk path: %w", err)
	}
	actual := strings.TrimSpace(string(output))
	if actual == "" {
		return "", "", 0, "", fmt.Errorf("Docker daemon returned an empty root directory")
	}
	actual, err = filepath.Abs(filepath.Clean(actual))
	if err != nil {
		return "", "", 0, "", err
	}
	wanted, err := filepath.Abs(filepath.Clean(p.Path))
	if err != nil {
		return "", "", 0, "", err
	}
	if wanted != actual {
		return "", "", 0, "", fmt.Errorf("declared Docker root %s does not match daemon root %s", wanted, actual)
	}
	rootInfo, err := os.Lstat(actual)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("inspect Docker root path %s: %w", actual, err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return "", "", 0, "", fmt.Errorf("Docker root path must be a real directory, not a symlink or non-directory")
	}
	fsid, mountpoint, err := filesystemFromMountInfo(actual)
	if err != nil {
		return "", "", 0, "", err
	}
	probe, verifiedID, free, err := diskFilesystem(mountpoint)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("probe Docker root filesystem at %s: %w", mountpoint, err)
	}
	if verifiedID != fsid {
		return "", "", 0, "", fmt.Errorf("kernel mount identity mismatch for Docker root %s: mount table=%s statfs=%s", actual, fsid, verifiedID)
	}
	return actual, fsid, free, probe, nil
}

func filesystemFromMountInfo(path string) (string, string, error) {
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", "", fmt.Errorf("read host mount table for Docker disk path: %w", err)
	}
	best, bestID := "", ""
	decode := strings.NewReplacer("\\040", " ", "\\011", "\t", "\\012", "\n", "\\134", "\\")
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mountpoint := decode.Replace(fields[4])
		mountpoint = filepath.Clean(mountpoint)
		if clean != mountpoint && !strings.HasPrefix(clean, strings.TrimSuffix(mountpoint, string(filepath.Separator))+string(filepath.Separator)) {
			continue
		}
		if len(mountpoint) < len(best) {
			continue
		}
		parts := strings.SplitN(fields[2], ":", 2)
		if len(parts) != 2 {
			continue
		}
		major, e1 := strconv.ParseUint(parts[0], 10, 32)
		minor, e2 := strconv.ParseUint(parts[1], 10, 32)
		if e1 != nil || e2 != nil {
			continue
		}
		dev := linuxMakeDevice(uint32(major), uint32(minor))
		best, bestID = mountpoint, fmt.Sprintf("dev:%d", dev)
	}
	if best == "" {
		return "", "", fmt.Errorf("host mount table has no filesystem for Docker path %s", clean)
	}
	return bestID, best, nil
}

func linuxMakeDevice(major, minor uint32) uint64 {
	return uint64(minor&0xff) | uint64(major&0xfff)<<8 | uint64(minor&^uint32(0xff))<<12 | uint64(major&^uint32(0xfff))<<32
}
