//go:build windows

package sysmounts

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var runPowerShell = func(ctx context.Context, script string) ([]byte, error) {
	return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Output()
}

type windowsVolumeRecord struct {
	DriveLetter     string `json:"DriveLetter"`
	FileSystem      string `json:"FileSystem"`
	FileSystemLabel string `json:"FileSystemLabel"`
	UniqueID        string `json:"UniqueId"`
	Size            int64  `json:"Size"`
}

// Windows identity probing is kept behind this seam until a native volume
// adapter is available. Returning uncertainty is safer than treating a reused
// drive letter as the same device.
func platformVolumeIdentity(ctx context.Context, device string) (VolumeIdentity, error) {
	record, err := windowsVolume(ctx, device)
	if err != nil {
		return VolumeIdentity{}, err
	}
	return VolumeIdentity{Label: record.FileSystemLabel, UUID: record.UniqueID, Filesystem: record.FileSystem}, nil
}

func platformDeviceByIdentity(ctx context.Context, uuid, serial string) (Volume, error) {
	if strings.TrimSpace(uuid) == "" {
		return Volume{}, fmt.Errorf("Windows identity lookup requires a volume unique id")
	}
	if strings.TrimSpace(serial) != "" {
		return Volume{}, fmt.Errorf("Windows volume inventory does not expose a portable disk serial through the safe adapter")
	}
	records, err := windowsVolumes(ctx)
	if err != nil {
		return Volume{}, err
	}
	var record windowsVolumeRecord
	for _, candidate := range records {
		if strings.EqualFold(strings.TrimSpace(candidate.UniqueID), strings.TrimSpace(uuid)) {
			record = candidate
			break
		}
	}
	if record.UniqueID == "" {
		return Volume{}, fmt.Errorf("device unique id did not match PowerShell response")
	}
	device := strings.TrimSpace(record.DriveLetter)
	if device != "" && !strings.HasSuffix(device, ":") {
		device += ":"
	}
	return Volume{DevicePath: device, Mountpoint: device + `\`, Filesystem: record.FileSystem, MountDriver: record.FileSystem, TotalBytes: record.Size, Label: record.FileSystemLabel, UUID: record.UniqueID}, nil
}

func windowsVolume(ctx context.Context, device string) (windowsVolumeRecord, error) {
	selector := "Get-Volume"
	if strings.TrimSpace(device) != "" {
		device = strings.TrimSuffix(strings.TrimSpace(device), ":")
		selector = fmt.Sprintf("Get-Volume -DriveLetter '%s'", strings.ReplaceAll(device, "'", "''"))
	}
	script := selector + " | Select-Object DriveLetter,FileSystem,FileSystemLabel,UniqueId,Size | ConvertTo-Json -Compress"
	out, err := runPowerShell(ctx, script)
	if err != nil {
		return windowsVolumeRecord{}, err
	}
	var records []windowsVolumeRecord
	if err := json.Unmarshal(out, &records); err == nil {
		if len(records) == 0 {
			return windowsVolumeRecord{}, fmt.Errorf("PowerShell returned no volumes")
		}
		return records[0], nil
	}
	var record windowsVolumeRecord
	if err := json.Unmarshal(out, &record); err != nil {
		return windowsVolumeRecord{}, fmt.Errorf("decode PowerShell volume inventory: %w", err)
	}
	return record, nil
}

func windowsVolumes(ctx context.Context) ([]windowsVolumeRecord, error) {
	out, err := runPowerShell(ctx, "Get-Volume | Select-Object DriveLetter,FileSystem,FileSystemLabel,UniqueId,Size | ConvertTo-Json -Compress")
	if err != nil {
		return nil, err
	}
	var records []windowsVolumeRecord
	if err := json.Unmarshal(out, &records); err == nil {
		return records, nil
	}
	var record windowsVolumeRecord
	if err := json.Unmarshal(out, &record); err != nil {
		return nil, fmt.Errorf("decode PowerShell volume inventory: %w", err)
	}
	return []windowsVolumeRecord{record}, nil
}
