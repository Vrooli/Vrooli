package cliapp

import (
	"strings"
)

// ScenarioEnvVars lists standard environment variable hooks for a scenario CLI.
type ScenarioEnvVars struct {
	APIEnvVars         []string
	APIPortEnvVars     []string
	ConfigDirEnvVars   []string
	SourceRootEnvVars  []string
	TokenEnvVars       []string
	HTTPTimeoutEnvVars []string
}

// ScenarioEnvOptions allows callers to append additional scenario-specific env vars.
type ScenarioEnvOptions struct {
	ExtraAPIEnvVars         []string
	ExtraAPIPortEnvVars     []string
	ExtraConfigDirEnvVars   []string
	ExtraSourceRootEnvVars  []string
	ExtraTokenEnvVars       []string
	ExtraHTTPTimeoutEnvVars []string
}

// StandardScenarioEnv derives a conventional set of env vars based on the
// scenario name, keeping them consistent across CLIs while allowing extras.
func StandardScenarioEnv(appName string, opts ScenarioEnvOptions) ScenarioEnvVars {
	slug := strings.ToUpper(strings.ReplaceAll(appName, "-", "_"))

	env := ScenarioEnvVars{
		APIEnvVars: []string{
			slug + "_API_BASE",
			slug + "_API_URL",
		},
		APIPortEnvVars: []string{
			slug + "_API_PORT",
			"API_PORT",
		},
		ConfigDirEnvVars: []string{
			slug + "_CONFIG_DIR",
			"VROOLI_CLI_CONFIG_DIR",
		},
		SourceRootEnvVars: []string{
			"VROOLI_CLI_SOURCE_ROOT",
			slug + "_CLI_SOURCE_ROOT",
		},
		TokenEnvVars: []string{
			slug + "_API_TOKEN",
			"VROOLI_API_TOKEN",
		},
		HTTPTimeoutEnvVars: []string{
			slug + "_HTTP_TIMEOUT",
			"VROOLI_HTTP_TIMEOUT",
		},
	}

	env.APIEnvVars = append(env.APIEnvVars, opts.ExtraAPIEnvVars...)
	env.APIPortEnvVars = append(env.APIPortEnvVars, opts.ExtraAPIPortEnvVars...)
	env.ConfigDirEnvVars = append(env.ConfigDirEnvVars, opts.ExtraConfigDirEnvVars...)
	env.SourceRootEnvVars = append(env.SourceRootEnvVars, opts.ExtraSourceRootEnvVars...)
	env.TokenEnvVars = append(env.TokenEnvVars, opts.ExtraTokenEnvVars...)
	env.HTTPTimeoutEnvVars = append(env.HTTPTimeoutEnvVars, opts.ExtraHTTPTimeoutEnvVars...)

	env.APIEnvVars = dedupe(env.APIEnvVars)
	env.APIPortEnvVars = dedupe(env.APIPortEnvVars)
	env.ConfigDirEnvVars = dedupe(env.ConfigDirEnvVars)
	env.SourceRootEnvVars = dedupe(env.SourceRootEnvVars)
	env.TokenEnvVars = dedupe(env.TokenEnvVars)
	env.HTTPTimeoutEnvVars = dedupe(env.HTTPTimeoutEnvVars)

	return env
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		if _, exists := seen[val]; exists {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	return out
}
