package main

import (
	"context"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) apiGet(path string, query url.Values) ([]byte, error) {
	return a.apiRequest("GET", path, query, nil)
}

func (a *App) apiRequest(method, path string, query url.Values, body interface{}) ([]byte, error) {
	a.client.SetBaseOptions(a.buildAPIBaseOptions())
	a.client.SetToken(a.config.Token)
	return a.client.Do(method, path, query, body)
}

func (a *App) buildAPIBaseOptions() cliutil.APIBaseOptions {
	return cliutil.APIBaseOptions{
		Override:   a.apiOverride,
		EnvVars:    []string{"SCORING_API_BASE"},
		ConfigBase: a.config.APIBase,
		PortEnvVars: []string{
			"API_PORT",
			"SCENARIO_COMPLETENESS_PORT",
		},
		PortDetector: a.detectPortFromVrooli,
		DefaultBase:  defaultAPIBase,
	}
}

func (a *App) detectPortFromVrooli() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "port", "scenario-completeness-scoring", "API_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
