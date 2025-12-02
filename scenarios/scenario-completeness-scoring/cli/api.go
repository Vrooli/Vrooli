package main

import "github.com/vrooli/cli-core/cliutil"

func (a *App) buildAPIBaseOptions() cliutil.APIBaseOptions {
	return cliutil.APIBaseOptions{
		Override: a.apiOverride,
		EnvVars: []string{
			"SCENARIO_COMPLETENESS_SCORING_API_BASE",
			"SCORING_API_BASE",
		},
		ConfigBase: a.config.APIBase,
		PortEnvVars: []string{
			"API_PORT",
			"SCENARIO_COMPLETENESS_SCORING_API_PORT",
		},
		PortDetector: cliutil.DetectPortFromVrooli("scenario-completeness-scoring", "API_PORT"),
		DefaultBase:  defaultAPIBase,
	}
}
