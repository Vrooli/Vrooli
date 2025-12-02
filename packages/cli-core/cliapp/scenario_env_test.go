package cliapp

import "testing"

func TestStandardScenarioEnv(t *testing.T) {
	env := StandardScenarioEnv("scenario-completeness-scoring", ScenarioEnvOptions{
		ExtraAPIEnvVars:        []string{"SCORING_API_BASE", "SCENARIO_COMPLETENESS_SCORING_API_BASE"},
		ExtraAPIPortEnvVars:    []string{"SCENARIO_COMPLETENESS_SCORING_API_PORT"},
		ExtraConfigDirEnvVars:  []string{"CUSTOM_CONFIG"},
		ExtraSourceRootEnvVars: []string{"CUSTOM_SRC"},
		ExtraTokenEnvVars:      []string{"CUSTOM_TOKEN"},
		ExtraHTTPTimeoutEnvVars: []string{
			"SCENARIO_COMPLETENESS_SCORING_HTTP_TIMEOUT",
		},
	})

	expectContains := func(list []string, value string) {
		for _, v := range list {
			if v == value {
				return
			}
		}
		t.Fatalf("expected %s to contain %s", list, value)
	}

	expectContains(env.APIEnvVars, "SCENARIO_COMPLETENESS_SCORING_API_BASE")
	expectContains(env.APIEnvVars, "SCENARIO_COMPLETENESS_SCORING_API_URL")
	expectContains(env.APIEnvVars, "SCORING_API_BASE")

	expectContains(env.APIPortEnvVars, "API_PORT")
	expectContains(env.APIPortEnvVars, "SCENARIO_COMPLETENESS_SCORING_API_PORT")
	if env.APIPortEnvVars[0] != "SCENARIO_COMPLETENESS_SCORING_API_PORT" {
		t.Fatalf("expected scenario-specific API port env first, got %s", env.APIPortEnvVars[0])
	}

	expectContains(env.ConfigDirEnvVars, "SCENARIO_COMPLETENESS_SCORING_CONFIG_DIR")
	expectContains(env.ConfigDirEnvVars, "VROOLI_CLI_CONFIG_DIR")
	expectContains(env.ConfigDirEnvVars, "CUSTOM_CONFIG")

	expectContains(env.SourceRootEnvVars, "VROOLI_CLI_SOURCE_ROOT")
	expectContains(env.SourceRootEnvVars, "SCENARIO_COMPLETENESS_SCORING_CLI_SOURCE_ROOT")
	expectContains(env.SourceRootEnvVars, "CUSTOM_SRC")

	expectContains(env.TokenEnvVars, "SCENARIO_COMPLETENESS_SCORING_API_TOKEN")
	expectContains(env.TokenEnvVars, "VROOLI_API_TOKEN")
	expectContains(env.TokenEnvVars, "CUSTOM_TOKEN")

	expectContains(env.HTTPTimeoutEnvVars, "SCENARIO_COMPLETENESS_SCORING_HTTP_TIMEOUT")
	expectContains(env.HTTPTimeoutEnvVars, "VROOLI_HTTP_TIMEOUT")
}
