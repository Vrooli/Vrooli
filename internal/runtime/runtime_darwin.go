//go:build darwin

package runtime

import "os/exec"

func currentHost() Host {
	return Host{
		OS:              "darwin",
		PackageManager:  packageManager(),
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  commandAvailable("sysctl"),
		SupportsSystemd: false,
		Notes: []string{
			"project-level setup/develop still depend on Linux-oriented shell steps",
		},
	}
}

func packageManager() string {
	if commandAvailable("brew") {
		return "brew"
	}
	return ""
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
