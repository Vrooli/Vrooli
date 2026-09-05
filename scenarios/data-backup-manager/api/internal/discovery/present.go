package discovery

import (
	"fmt"
	"path/filepath"

	"data-backup-manager/internal/sysmounts"
)

// destinationLabel builds a default, human-facing destination name for a
// volume. The operator can rename it when accepting; this is only the default.
func destinationLabel(v Volume) string {
	base := filepath.Base(v.Mountpoint)
	if base == "" || base == "/" || base == "." {
		base = "system"
	}
	switch v.Class {
	case sysmounts.ClassRemovable:
		return "Removable drive — " + base
	case sysmounts.ClassNetwork:
		return "Network mount — " + base
	default:
		return "Volume — " + base
	}
}

// destinationRationale explains why a volume is (or is not) a good destination.
func destinationRationale(v Volume, separateRootOK bool) string {
	if !separateRootOK {
		return "Overlaps Vrooli's protected data, so it can't be a backup destination — pick a separate disk or drive."
	}
	free := humanizeBytes(v.FreeBytes)
	switch v.Class {
	case sysmounts.ClassRemovable:
		return fmt.Sprintf("Plugged-in removable drive with %s free — a good offsite-style backup destination.", free)
	case sysmounts.ClassNetwork:
		return fmt.Sprintf("Network mount with %s free.", free)
	default:
		return fmt.Sprintf("Fixed volume with %s free.", free)
	}
}

// humanizeBytes renders a byte count as a short human string. Local (no new
// dependency); good enough for a one-line rationale.
func humanizeBytes(b int64) string {
	if b <= 0 {
		return "unknown space"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
