//go:build darwin

package sysmounts

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var runDiskutil = func(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "diskutil", args...).Output()
}

// diskutil is the read-only native identity source on macOS. The parser is
// deliberately conservative; an unavailable field remains unknown.
func platformVolumeIdentity(ctx context.Context, device string) (VolumeIdentity, error) {
	if strings.TrimSpace(device) == "" {
		return VolumeIdentity{}, fmt.Errorf("device is empty")
	}
	out, err := runDiskutil(ctx, "info", device)
	if err != nil {
		return VolumeIdentity{}, err
	}
	var id VolumeIdentity
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch key {
		case "Volume Name":
			id.Label = value
		case "Volume UUID":
			id.UUID = value
		case "Device / Media Name":
			id.Model = value
		case "Device Identifier":
			// Device Identifier is stable for the current partition but may
			// change after repartitioning. Keep it only as a descriptive model;
			// Volume UUID is the safety identity.
			id.Model = value
		case "Disk / Partition UUID":
			if id.UUID == "" {
				id.UUID = value
			}
		}
	}
	return id, nil
}

func platformDeviceByIdentity(ctx context.Context, uuid, serial string) (Volume, error) {
	if strings.TrimSpace(uuid) == "" {
		return Volume{}, fmt.Errorf("macOS identity lookup requires a volume UUID")
	}
	out, err := runDiskutil(ctx, "info", uuid)
	if err != nil {
		return Volume{}, err
	}
	fields := diskutilFields(string(out))
	if !strings.EqualFold(strings.TrimSpace(fields["Volume UUID"]), strings.TrimSpace(uuid)) &&
		!strings.EqualFold(strings.TrimSpace(fields["Disk / Partition UUID"]), strings.TrimSpace(uuid)) {
		return Volume{}, fmt.Errorf("device UUID did not match diskutil response")
	}
	if strings.TrimSpace(serial) != "" {
		return Volume{}, fmt.Errorf("macOS does not expose a portable disk serial through diskutil info")
	}
	device := fields["Device Node"]
	if device == "" {
		return Volume{}, fmt.Errorf("diskutil response omitted device node")
	}
	size, _ := strconv.ParseInt(strings.ReplaceAll(fields["Disk Size"], ",", ""), 10, 64)
	return Volume{
		DevicePath: device, Mountpoint: fields["Mount Point"], Filesystem: fields["File System Personality"],
		MountDriver: fields["Type (Bundle)"], TotalBytes: size, Label: fields["Volume Name"],
		UUID: firstNonEmpty(fields["Volume UUID"], fields["Disk / Partition UUID"]), Model: fields["Device / Media Name"],
	}, nil
}

func diskutilFields(output string) map[string]string {
	fields := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			fields[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return fields
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
