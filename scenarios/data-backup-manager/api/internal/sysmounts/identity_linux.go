//go:build linux

package sysmounts

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

var lsblkProperty = regexp.MustCompile(`([A-Z]+)="([^"]*)"`)

// platformVolumeIdentity uses lsblk's read-only metadata query. Failure is
// intentionally non-fatal: readiness reports uncertainty instead of inventing
// a mutable identity.
func platformVolumeIdentity(ctx context.Context, device string) (VolumeIdentity, error) {
	device = strings.TrimSpace(device)
	if device == "" {
		return VolumeIdentity{}, fmt.Errorf("device is empty")
	}
	id, err := lsblkIdentity(ctx, device)
	if err != nil {
		return VolumeIdentity{}, err
	}
	if id.Serial == "" || id.Model == "" {
		// lsblk reports MODEL and SERIAL for the whole disk, not for a
		// partition, so a partition-only query leaves the identity resting on
		// the UUID alone. Ask the backing disk so a plan is bound by two
		// independent anchors rather than one.
		if parent := wholeDiskDevice(device); parent != "" {
			if disk, derr := lsblkIdentity(ctx, parent); derr == nil {
				if id.Serial == "" {
					id.Serial = disk.Serial
				}
				if id.Model == "" {
					id.Model = disk.Model
				}
			}
		}
	}
	return id, nil
}

func lsblkIdentity(ctx context.Context, device string) (VolumeIdentity, error) {
	out, err := exec.CommandContext(ctx, "lsblk", "-P", "-n", "-o", "LABEL,UUID,MODEL,SERIAL,FSTYPE", device).Output()
	if err != nil {
		return VolumeIdentity{}, err
	}
	var id VolumeIdentity
	for _, match := range lsblkProperty.FindAllStringSubmatch(string(out), -1) {
		switch match[1] {
		case "LABEL":
			id.Label = match[2]
		case "UUID":
			id.UUID = match[2]
		case "MODEL":
			id.Model = match[2]
		case "SERIAL":
			id.Serial = match[2]
		case "FSTYPE":
			id.Filesystem = match[2]
		}
	}
	return id, nil
}

// wholeDiskDevice maps a partition device path to its backing whole-disk path
// (/dev/sda1 -> /dev/sda, /dev/nvme0n1p2 -> /dev/nvme0n1). It returns empty
// when the path is already a whole disk.
func wholeDiskDevice(device string) string {
	dir, name := path.Split(device)
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return ""
	}
	if strings.HasSuffix(trimmed, "p") && len(trimmed) > 1 {
		trimmed = strings.TrimSuffix(trimmed, "p")
	}
	if trimmed == "" || trimmed == name {
		return ""
	}
	return dir + trimmed
}
