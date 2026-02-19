package main

import (
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
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
	env := cliapp.StandardScenarioEnv(appName, cliapp.ScenarioEnvOptions{
		ExtraAPIEnvVars: []string{"API_BASE_URL", "VITE_API_BASE_URL"},
	})
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:               appName,
		Version:            appVersion,
		Description:        "Visited Tracker CLI",
		DefaultAPIBase:     defaultAPIBase,
		APIEnvVars:         env.APIEnvVars,
		APIPortEnvVars:     env.APIPortEnvVars,
		APIPortDetector:    cliutil.DetectPortFromVrooli(appName, "API_PORT"),
		ConfigDirEnvVars:   env.ConfigDirEnvVars,
		SourceRootEnvVars:  env.SourceRootEnvVars,
		TokenEnvVars:       env.TokenEnvVars,
		HTTPTimeoutEnvVars: env.HTTPTimeoutEnvVars,
		BuildFingerprint:   buildFingerprint,
		BuildTimestamp:     buildTimestamp,
		BuildSourceRoot:    buildSourceRoot,
		AllowAnonymous:     true,
	})
	if err != nil {
		return nil, err
	}

	app := &App{core: core}
	app.core.SetCommandsWithSubgroups(app.registerCommands(), app.registerSubcommands())
	return app, nil
}

func (a *App) Run(args []string) error {
	normalized, err := a.normalizeArgs(args)
	if err != nil {
		return err
	}
	return a.core.CLI.Run(normalized)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	health := cliapp.CommandGroup{
		Title: "Health",
		Commands: []cliapp.Command{
			{Name: "status", NeedsAPI: true, Description: "Check API health and coverage", Run: a.cmdStatus},
		},
	}

	tracking := cliapp.CommandGroup{
		Title: "Tracking",
		Commands: []cliapp.Command{
			{Name: "visit", NeedsAPI: true, Description: "Record file visits", Run: a.cmdVisit},
			{Name: "adjust-visit", NeedsAPI: true, Description: "Adjust a file visit count", Run: a.cmdAdjustVisit},
			{Name: "exclude", NeedsAPI: true, Description: "Bulk-exclude files from a campaign", Run: a.cmdExclude},
			{Name: "sync", NeedsAPI: true, Description: "Sync campaign file structure", Run: a.cmdSync},
		},
	}

	analytics := cliapp.CommandGroup{
		Title: "Analytics",
		Commands: []cliapp.Command{
			{Name: "least-visited", NeedsAPI: true, Description: "List least visited files", Run: a.cmdLeastVisited},
			{Name: "most-stale", NeedsAPI: true, Description: "List most stale files", Run: a.cmdMostStale},
			{Name: "coverage", NeedsAPI: true, Description: "Show coverage statistics", Run: a.cmdCoverage},
		},
	}

	data := cliapp.CommandGroup{
		Title: "Data",
		Commands: []cliapp.Command{
			{Name: "export", NeedsAPI: true, Description: "Export campaign data to a file", Run: a.cmdExport},
			{Name: "import", NeedsAPI: true, Description: "Import campaign data from a file", Run: a.cmdImport},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{health, tracking, analytics, data, config}
}

func (a *App) registerSubcommands() []cliapp.SubcommandGroup {
	campaigns := cliapp.SubcommandGroup{
		Name:        "campaigns",
		Description: "Manage visit campaigns",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "list", Description: "List campaigns", Run: a.cmdCampaignList},
			{Name: "create", Description: "Create a campaign", Run: a.cmdCampaignCreate},
			{Name: "get", Description: "Get a campaign by ID", Run: a.cmdCampaignGet},
			{Name: "update", Description: "Update campaign notes", Run: a.cmdCampaignUpdate},
			{Name: "note", Description: "Update campaign notes", Run: a.cmdCampaignNote},
			{Name: "reset", Description: "Reset campaign visits", Run: a.cmdCampaignReset},
			{Name: "delete", Description: "Delete a campaign", Run: a.cmdCampaignDelete},
			{Name: "find-or-create", Description: "Find or create by location and tag", Run: a.cmdCampaignFindOrCreate},
		},
	}

	files := cliapp.SubcommandGroup{
		Name:        "files",
		Description: "Manage tracked files",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "get-by-path", Description: "Lookup a tracked file by path", Run: a.cmdFileGetByPath},
			{Name: "note", Description: "Update notes for a tracked file", Run: a.cmdFileNote},
			{Name: "priority", Description: "Update priority weight for a tracked file", Run: a.cmdFilePriority},
			{Name: "exclude", Description: "Toggle exclusion for a tracked file", Run: a.cmdFileExclude},
		},
	}

	return []cliapp.SubcommandGroup{campaigns, files}
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
				return nil, errMissingFlagValue("--campaign-id")
			}
			a.setCampaignID(value)
		case strings.HasPrefix(arg, "--campaign="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--campaign="))
			if value == "" {
				return nil, errMissingFlagValue("--campaign")
			}
			a.setCampaignID(value)
		case arg == "--campaign-id" || arg == "--campaign":
			if i+1 >= len(args) {
				return nil, errMissingFlagValue(arg)
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
