//go:build linux

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	return Host{
		OS:              "linux",
		PackageManager:  detectPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  hostreqkit.CommandAvailable("sysctl"),
		SupportsSystemd: hostreqkit.CommandAvailable("systemctl"),
	}
}

func detectPackageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"apt-get", "dnf", "yum", "pacman", "apk", "brew"})
}
