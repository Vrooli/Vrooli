//go:build linux

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	facts := currentPlatformFacts()
	return Host{
		OS:              "linux",
		PackageManager:  detectPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  facts.SupportsSysctl,
		SupportsSystemd: facts.SupportsSystemd,
	}
}

func detectPackageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"apt-get", "dnf", "yum", "pacman", "apk", "brew"})
}
