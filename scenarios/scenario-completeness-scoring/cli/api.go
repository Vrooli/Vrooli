package main

import (
	"context"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

type APIClient struct {
	client       *cliutil.HTTPClient
	baseResolver func() cliutil.APIBaseOptions
	tokenSource  func() string
}

func NewAPIClient(client *cliutil.HTTPClient, baseResolver func() cliutil.APIBaseOptions, tokenSource func() string) *APIClient {
	return &APIClient{
		client:       client,
		baseResolver: baseResolver,
		tokenSource:  tokenSource,
	}
}

func (c *APIClient) Get(path string, query url.Values) ([]byte, error) {
	return c.Request("GET", path, query, nil)
}

func (c *APIClient) Request(method, path string, query url.Values, body interface{}) ([]byte, error) {
	c.client.SetBaseOptions(c.baseResolver())
	c.client.SetToken(c.tokenSource())
	return c.client.Do(method, path, query, body)
}

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
