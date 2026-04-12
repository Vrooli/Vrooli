//go:build linux

package runtime

func currentHost() Host {
	return Host{
		OS:              "linux",
		PackageManager:  detectPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  commandAvailable("sysctl"),
		SupportsSystemd: commandAvailable("systemctl"),
	}
}

func detectPackageManager() string {
	return detectFirstAvailable([]string{"apt-get", "dnf", "yum", "pacman", "apk", "brew"})
}
