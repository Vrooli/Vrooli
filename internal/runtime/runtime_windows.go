//go:build windows

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	facts := currentPlatformFacts()
	return Host{
		OS:              "windows",
		PackageManager:  detectWindowsPackageManager(),
		SupportsSetup:   true,
		SupportsDevelop: false,
		SupportsSysctl:  facts.SupportsSysctl,
		SupportsSystemd: facts.SupportsSystemd,
		Notes: []string{
			"Windows host-provisioning and scenario-runtime support are separate rungs; see docs/configuration/host/tools.md",
		},
	}
}

func detectWindowsPackageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"winget", "choco", "scoop"})
}
