//go:build darwin

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	return Host{
		OS:              "darwin",
		PackageManager:  packageManager(),
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  hostreqkit.CommandAvailable("sysctl"),
		SupportsSystemd: false,
		Notes: []string{
			"project-level setup/develop are native, but resource and scenario lifecycle support still assumes Linux-oriented tooling",
		},
	}
}

func packageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"brew"})
}
