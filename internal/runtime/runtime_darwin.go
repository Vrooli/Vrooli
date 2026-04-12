//go:build darwin

package runtime

func currentHost() Host {
	return Host{
		OS:              "darwin",
		PackageManager:  packageManager(),
		SupportsSetup:   false,
		SupportsDevelop: false,
		SupportsSysctl:  commandAvailable("sysctl"),
		SupportsSystemd: false,
		Notes: []string{
			"project-level setup/develop are native, but resource and scenario lifecycle support still assumes Linux-oriented tooling",
		},
	}
}

func packageManager() string {
	return detectFirstAvailable([]string{"brew"})
}
