package main

import (
	"os"
	"strings"
	"visited-tracker/cli/domains"
	"visited-tracker/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "visited-tracker"
	appVersion     = "1.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core       *cliapp.ScenarioApp
	campaignID string
}

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                 appName,
		Version:              appVersion,
		Description:          "Visited Tracker CLI",
		DefaultAPIBase:       defaultAPIBase,
		ExtraAPIEnvVars:      []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:     buildFingerprint,
		BuildTimestamp:       buildTimestamp,
		BuildSourceRoot:      buildSourceRoot,
		AllowAnonymous:       true,
		IncludeStatusCommand: boolPtr(false),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			state := domains.State{CampaignID: &app.campaignID}
			return domains.CommandGroups(core, state)
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			state := domains.State{CampaignID: &app.campaignID}
			return domains.SubcommandGroups(core, state)
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	normalized, err := a.normalizeArgs(args)
	if err != nil {
		return err
	}
	return a.core.CLI.Run(normalized)
}

func (a *App) normalizeArgs(args []string) ([]string, error) {
	var cleaned []string
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		switch {
		case strings.HasPrefix(arg, "--campaign-id="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--campaign-id="))
			if value == "" {
				return nil, support.ErrMissingFlagValue("--campaign-id")
			}
			a.setCampaignID(value)
		case strings.HasPrefix(arg, "--campaign="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--campaign="))
			if value == "" {
				return nil, support.ErrMissingFlagValue("--campaign")
			}
			a.setCampaignID(value)
		case arg == "--campaign-id" || arg == "--campaign":
			if i+1 >= len(args) {
				return nil, support.ErrMissingFlagValue(arg)
			}
			a.setCampaignID(strings.TrimSpace(args[i+1]))
			i++
		default:
			cleaned = append(cleaned, arg)
		}
	}

	if len(cleaned) == 0 {
		return cleaned, nil
	}
	if cleaned[0] == "campaign" {
		cleaned[0] = "campaigns"
	}
	if cleaned[0] == "campaigns" && len(cleaned) == 1 {
		cleaned = append(cleaned, "list")
	}
	return cleaned, nil
}

func (a *App) setCampaignID(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	a.campaignID = id
	_ = os.Setenv("VISITED_TRACKER_CAMPAIGN_ID", id)
}

func boolPtr(v bool) *bool {
	return &v
}
