// DOC: docs/internal/CLI_AUDIT.md
// DOC: docs/internal/SEAMS.md#cli-api-parity
//
// Package main implements the development-toolchain-validator CLI.
package main

import (
	"development-toolchain-validator/cli/domains"
	"development-toolchain-validator/cli/domains/connections"
	"development-toolchain-validator/cli/domains/references"
	"development-toolchain-validator/cli/internal/textutil"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "development-toolchain-validator"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:             appName,
		Version:          appVersion,
		Description:      "Development Toolchain Validator CLI",
		DefaultAPIBase:   defaultAPIBase,
		ExtraAPIEnvVars:  []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint: buildFingerprint,
		BuildTimestamp:   buildTimestamp,
		BuildSourceRoot:  buildSourceRoot,
		AllowAnonymous:   true,
		SubcommandGroups: domains.SubcommandGroups,
	})
	if err != nil {
		return nil, err
	}
	return &App{core: core}, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

// Thin delegations keep the package-local test surface stable while the CLI
// itself moves to domain packages and subcommand groups.
func (a *App) cmdReference(args []string) error {
	return references.Run(a.core, args)
}

func (a *App) cmdConnection(args []string) error {
	return connections.Run(a.core, args)
}

func truncate(s string, maxLen int) string {
	return textutil.Truncate(s, maxLen)
}

type (
	referenceResponse      = references.ReferenceResponse
	referenceListResponse  = references.ReferenceListResponse
	referenceCreateRequest = references.ReferenceCreateRequest
	referenceUpdateRequest = references.ReferenceUpdateRequest
)

type (
	connectionResponse       = connections.ConnectionResponse
	connectionListResponse   = connections.ConnectionListResponse
	connectionConnectRequest = connections.ConnectionConnectRequest
	driftCheckRequest        = connections.DriftCheckRequest
	driftStatusResponse      = connections.DriftStatusResponse
)
