//go:build linux

package runtime

import "os/exec"

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
	for _, candidate := range []string{"apt-get", "dnf", "yum", "pacman", "apk", "brew"} {
		if commandAvailable(candidate) {
			return candidate
		}
	}
	return ""
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
