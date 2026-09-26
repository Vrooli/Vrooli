//go:build linux

package sysmounts

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var lsblkProperty = regexp.MustCompile(`([A-Z]+)="([^"]*)"`)

// runLSBlk is the host-command seam. Production uses the read-only lsblk
// query; tests replace it with deterministic fixture output and never invoke
// a real device probe.
var runLSBlk = func(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "lsblk", args...).Output()
}

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
	out, err := runLSBlk(ctx, "-P", "-n", "-o", "LABEL,UUID,MODEL,SERIAL,FSTYPE", device)
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

// platformDeviceByIdentity inventories block devices without mounting them.
// lsblk's stable fields are compared in memory; no filesystem probe or write
// is performed. The caller still revalidates the identity immediately before
// any remediation action.
func platformDeviceByIdentity(ctx context.Context, uuid, serial string) (Volume, error) {
	out, err := runLSBlk(ctx, "-b", "-P", "-n", "-o", "PATH,LABEL,UUID,MODEL,SERIAL,FSTYPE,SIZE")
	if err != nil {
		return Volume{}, err
	}
	rows := make([]map[string]string, 0)
	byPath := make(map[string]map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		props := map[string]string{}
		for _, match := range lsblkProperty.FindAllStringSubmatch(line, -1) {
			props[match[1]] = match[2]
		}
		if props["PATH"] == "" {
			continue
		}
		rows = append(rows, props)
		byPath[props["PATH"]] = props
	}
	for _, props := range rows {
		// Partition rows commonly omit the parent disk serial/model. Enrich the
		// row from the whole disk before comparing the requested identity so a
		// plan bound to UUID+serial still resolves after the volume is unmounted.
		if props["SERIAL"] == "" || props["MODEL"] == "" {
			if parent := wholeDiskDevice(props["PATH"]); parent != "" {
				if disk := byPath[parent]; disk != nil {
					if props["SERIAL"] == "" {
						props["SERIAL"] = disk["SERIAL"]
					}
					if props["MODEL"] == "" {
						props["MODEL"] = disk["MODEL"]
					}
				}
			}
		}
		if !equalIdentityField(uuid, props["UUID"], true) || !equalIdentityField(serial, props["SERIAL"], false) {
			continue
		}
		device := strings.TrimSpace(props["PATH"])
		if device == "" {
			continue
		}
		size, _ := strconv.ParseInt(strings.TrimSpace(props["SIZE"]), 10, 64)
		class, removable := newClassifier().classify(mountInfo{Device: device, Fstype: props["FSTYPE"]})
		return Volume{DevicePath: device, Filesystem: props["FSTYPE"], MountDriver: props["FSTYPE"], Class: class, Removable: removable, TotalBytes: size, Label: props["LABEL"], UUID: props["UUID"], Model: props["MODEL"], Serial: props["SERIAL"]}, nil
	}
	return Volume{}, fmt.Errorf("%w: no device matched the supplied UUID or serial", ErrDeviceNotFound)
}

func equalIdentityField(expected, observed string, fold bool) bool {
	expected, observed = strings.TrimSpace(expected), strings.TrimSpace(observed)
	if expected == "" {
		return true
	}
	if observed == "" {
		return false
	}
	if fold {
		return strings.EqualFold(expected, observed)
	}
	return expected == observed
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
