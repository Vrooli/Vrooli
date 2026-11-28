//go:build !linux && !darwin && !windows

package runtime

import "github.com/vrooli/vrooli/internal/hostreqkit"

func currentHost() Host {
	facts := currentPlatformFacts()
	os := facts.OS
	if os == "" {
		os = "other"
	}
	return hostreqkit.Host{
		OS:              os,
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  false,
		SupportsSystemd: false,
		Notes: []string{
			"no project-level runtime implementation is defined for this platform",
		},
	}
}
