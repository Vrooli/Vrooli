//go:build linux

package runtime

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func currentHost() Host {
	facts := currentPlatformFacts()
	return Host{
		OS:              string(hostreqspec.PlatformLinux),
		PackageManager:  detectPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  facts.SupportsSysctl,
		SupportsSystemd: facts.SupportsSystemd,
	}
}

func detectPackageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"apt-get", "dnf", "yum", "zypper", "pacman", "apk", "brew"})
}
