package main

import "github.com/vrooli/cli-core/cliapp"

const (
	appName        = "git-control-tower"
	appVersion     = "0.1.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func boolPtr(v bool) *bool { return &v }

type App struct {
	core *cliapp.ScenarioApp
}

func NewApp() (*App, error) {
	app := &App{}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                    appName,
		Version:                 appVersion,
		Description:             "Git Control Tower CLI",
		DefaultAPIBase:          defaultAPIBase,
		ExtraAPIEnvVars:         []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		BuildFingerprint:        buildFingerprint,
		BuildTimestamp:          buildTimestamp,
		BuildSourceRoot:         buildSourceRoot,
		AllowAnonymous:          true,
		IncludeConfigureCommand: boolPtr(false),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			app.core = core
			return app.registerCommands()
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) Run(args []string) error {
	return a.core.CLI.Run(args)
}

func (a *App) registerCommands() []cliapp.CommandGroup {
	repo := cliapp.CommandGroup{
		Title: "Repository",
		Commands: []cliapp.Command{
			{Name: "repo-status", NeedsAPI: true, Description: "Show repository status (branch + changed files)", Run: a.cmdRepoStatus},
			{Name: "diff", NeedsAPI: true, Description: "Show git diff (--path=FILE --staged)", Run: a.cmdDiff},
			{Name: "stage", NeedsAPI: true, Description: "Stage files (FILE... or --scope=scenario:name)", Run: a.cmdStage},
			{Name: "unstage", NeedsAPI: true, Description: "Unstage files (FILE... or --scope=scenario:name)", Run: a.cmdUnstage},
			{Name: "commit", NeedsAPI: true, Description: "Create a commit (-m MESSAGE [--conventional])", Run: a.cmdCommit},
			{Name: "sync-status", NeedsAPI: true, Description: "Check push/pull status ([--fetch] [--remote=NAME])", Run: a.cmdSyncStatus},
			{Name: "branch-list", NeedsAPI: true, Description: "List branches", Run: a.cmdBranchList},
			{Name: "branch-create", NeedsAPI: true, Description: "Create branch NAME [--from=BASE] [--no-checkout] [--allow-dirty]", Run: a.cmdBranchCreate},
			{Name: "branch-switch", NeedsAPI: true, Description: "Switch branch NAME [--allow-dirty] [--track-remote]", Run: a.cmdBranchSwitch},
			{Name: "branch-publish", NeedsAPI: true, Description: "Publish current branch ([--remote=NAME] [--branch=NAME] [--fetch])", Run: a.cmdBranchPublish},
		},
	}

	review := cliapp.CommandGroup{
		Title: "Review",
		Commands: []cliapp.Command{
			{Name: "review-summary", NeedsAPI: true, Description: "Show readiness review for a scenario", Run: a.cmdReviewSummary},
			{Name: "review-run", NeedsAPI: true, Description: "Run readiness checks and show results", Run: a.cmdReviewRun},
			{Name: "review-status", NeedsAPI: true, Description: "Check status of a review run job", Run: a.cmdReviewStatus},
		},
	}

	audit := cliapp.CommandGroup{
		Title: "Audit",
		Commands: []cliapp.Command{
			{Name: "audit", NeedsAPI: true, Description: "Query audit logs ([--operation=TYPE] [--limit=N])", Run: a.cmdAudit},
		},
	}

	config := cliapp.CommandGroup{
		Title: "Configuration",
		Commands: []cliapp.Command{
			a.core.ConfigureCommand([]string{"api_base"}, []string{"token", "api_token"}),
		},
	}

	return []cliapp.CommandGroup{repo, review, audit, config}
}
