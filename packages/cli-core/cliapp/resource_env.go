package cliapp

import "strings"

// ResourceEnvVars lists standard environment variable hooks for a resource CLI.
type ResourceEnvVars struct {
	SourceRootEnvVars   []string
	ControlPlaneEnvVars []string
}

// ResourceEnvOptions allows callers to append additional resource-specific env vars.
type ResourceEnvOptions struct {
	ExtraSourceRootEnvVars   []string
	ExtraControlPlaneEnvVars []string
}

// StandardResourceEnv derives a conventional set of env vars based on the
// resource name, keeping them consistent across CLIs while allowing extras.
func StandardResourceEnv(resourceName string, opts ResourceEnvOptions) ResourceEnvVars {
	slug := strings.ToUpper(strings.ReplaceAll(resourceName, "-", "_"))

	env := ResourceEnvVars{
		SourceRootEnvVars: []string{
			"VROOLI_CLI_SOURCE_ROOT",
			slug + "_CLI_SOURCE_ROOT",
		},
		ControlPlaneEnvVars: []string{
			"VROOLI_CLI_BIN",
			slug + "_VROOLI_CLI_BIN",
		},
	}

	env.SourceRootEnvVars = append(env.SourceRootEnvVars, opts.ExtraSourceRootEnvVars...)
	env.ControlPlaneEnvVars = append(env.ControlPlaneEnvVars, opts.ExtraControlPlaneEnvVars...)

	env.SourceRootEnvVars = dedupe(env.SourceRootEnvVars)
	env.ControlPlaneEnvVars = dedupe(env.ControlPlaneEnvVars)

	return env
}
