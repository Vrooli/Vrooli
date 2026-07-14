//go:build darwin

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	return Host{
		OS:              "darwin",
		PackageManager:  packageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  hostreqkit.CommandAvailable("sysctl"),
		SupportsSystemd: false,
		Notes: []string{
			"docker-service resources require Docker Desktop (or a docker-compatible runtime) to be installed and running",
			"workspace-sandbox protected mode and X11 desktop automation are unavailable on macOS",
		},
	}
}

func packageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"brew"})
}
