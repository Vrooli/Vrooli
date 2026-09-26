package main

import (
	"context"
	"net/url"
	"os"
	"strings"
	"time"

	"scenario-to-cloud/cli/domains"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "scenario-to-cloud"
	appVersion     = "0.2.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

var exchangeLocalMachinePrincipal = authn.ExchangeLocalMachinePrincipal

// App is the main CLI application container.
type App struct {
	core *cliapp.ScenarioApp
}

// NewApp creates a new CLI application instance.
func NewApp() (*App, error) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "scenario-to-cloud CLI",
		DefaultAPIBase:     defaultAPIBase,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		DefaultHTTPTimeout: 2 * time.Minute,
		AllowAnonymous:     true,
		CommandGroups:      domains.CommandGroups,
	})
	if err != nil {
		return nil, err
	}

	return &App{core: core}, nil
}

// Run executes the CLI with the given arguments.
func (a *App) Run(args []string) error {
	// Local scenario-to-cloud management is protected even though it runs on
	// loopback. If the operator has not supplied a token, exchange the current
	// process identity through scenario-authenticator. The token stays in this
	// process memory and is never written to shell state, config, logs, or
	// deployment payloads. Remote/shared API bases still require their normal
	// explicit bearer token.
	if err := a.ensureLocalOperatorToken(args); err != nil {
		return err
	}
	return a.core.CLI.Run(args)
}

func (a *App) ensureLocalOperatorToken(args []string) error {
	if a == nil || a.core == nil || hasConfiguredOperatorToken(a.core) || isNonAPICommand(args) {
		return nil
	}
	base := strings.TrimSpace(a.core.APIRootBase())
	parsed, err := url.Parse(base)
	if err != nil || !isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	login, err := exchangeLocalMachinePrincipal(context.Background())
	if err != nil || login == nil || login.Tokens == nil || strings.TrimSpace(login.Tokens.AccessToken) == "" {
		// Leave the normal cli-core authentication error to explain configured
		// remote tokens. This error is only for a local runtime with no exchange.
		return nil
	}
	a.core.Config.Token = strings.TrimSpace(login.Tokens.AccessToken)
	return nil
}

func hasConfiguredOperatorToken(core *cliapp.ScenarioApp) bool {
	if core == nil {
		return false
	}
	for _, key := range []string{"SCENARIO_TO_CLOUD_API_TOKEN", "VROOLI_API_TOKEN"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return strings.TrimSpace(core.Config.Token) != ""
}

func isNonAPICommand(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch strings.TrimSpace(args[0]) {
	case "help", "--help", "-h", "configure", "version", "--version":
		return true
	default:
		return false
	}
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
