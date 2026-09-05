//go:build darwin

package sysmounts

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// diskutil is the read-only native identity source on macOS. The parser is
// deliberately conservative; an unavailable field remains unknown.
func platformVolumeIdentity(ctx context.Context, device string) (VolumeIdentity, error) {
	if strings.TrimSpace(device) == "" {
		return VolumeIdentity{}, fmt.Errorf("device is empty")
	}
	out, err := exec.CommandContext(ctx, "diskutil", "info", device).Output()
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
			id.Serial = value
		}
	}
	return id, nil
}
