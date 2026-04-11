//go:build windows

package runtime

func currentHost() Host {
	return Host{
		OS:              "windows",
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  false,
		SupportsSystemd: false,
		Notes: []string{
			"project-level setup/develop are native, but resource and scenario lifecycle support still assumes Linux-oriented tooling",
		},
	}
}
