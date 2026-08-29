//go:build darwin

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	facts := currentPlatformFacts()
	return Host{
		OS:              "darwin",
		PackageManager:  packageManager(),
		SupportsSetup:   true,
		SupportsDevelop: true,
		SupportsSysctl:  facts.SupportsSysctl,
		SupportsSystemd: facts.SupportsSystemd,
		Notes: []string{
			"managed-service resources use the container-runtime provider ladder: existing daemon, remote DOCKER_HOST, OrbStack/Rancher/Docker Desktop, or headless Colima",
			"workspace-sandbox protected mode is partial on macOS via Seatbelt (sandbox-exec): filesystem write-containment and network denial are enforced; path illusion and pid-namespace isolation are not",
			"X11 desktop automation is unavailable on macOS",
		},
	}
}

func packageManager() string {
	return hostreqkit.DetectFirstAvailable([]string{"brew"})
}
