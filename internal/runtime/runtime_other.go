//go:build !linux && !darwin && !windows

package runtime

func currentHost() Host {
	return Host{
		OS:              "other",
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  false,
		SupportsSystemd: false,
		Notes: []string{
			"no project-level runtime implementation is defined for this platform",
		},
	}
}
