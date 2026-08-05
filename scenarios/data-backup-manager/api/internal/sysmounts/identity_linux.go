//go:build linux

package sysmounts

import (
	"context"
	"fmt"
	"os/exec"
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
	out, err := exec.CommandContext(ctx, "lsblk", "-P", "-n", "-o", "LABEL,UUID,MODEL,SERIAL", device).Output()
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
		}
	}
	return id, nil
}
