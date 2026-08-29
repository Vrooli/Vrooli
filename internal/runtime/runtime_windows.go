//go:build windows

package runtime

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func currentHost() Host {
	facts := currentPlatformFacts()
	return Host{
		OS:              string(hostreqspec.PlatformWindows),
		PackageManager:  detectWindowsPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  facts.SupportsSysctl,
		SupportsSystemd: facts.SupportsSystemd,
		Notes:           []string{"Windows uses native process, lock, and SCM lifecycle backends; host-tool availability remains capability-driven."},
	}
}

func detectWindowsPackageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"winget", "choco", "scoop"})
}
