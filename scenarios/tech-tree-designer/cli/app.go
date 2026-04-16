package main

import (
	"fmt"
	"strings"

	"tech-tree-designer/cli/domains"
	"tech-tree-designer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName        = "tech-tree-designer"
	appVersion     = "1.0.0"
	defaultAPIBase = ""
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

type App struct {
	core     *cliapp.ScenarioApp
	selector *support.TreeSelector
}

func NewApp() (*App, error) {
	app := &App{
		selector: support.NewTreeSelector(),
	}
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "Tech Tree Designer CLI",
		DefaultAPIBase:      defaultAPIBase,
		ExtraAPIEnvVars:     []string{"API_BASE_URL", "VITE_API_BASE_URL"},
		ExtraAPIPortEnvVars: []string{"TECH_TREE_DESIGNER_API_PORT"},
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
		AllowAnonymous:      true,
		IncludeStatusCommand: func() *bool {
			value := false
			return &value
		}(),
		CommandGroups: func(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
			return domains.CommandGroups(app.dependencies(core))
		},
		SubcommandGroups: func(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
			return domains.SubcommandGroups(app.dependencies(core))
		},
	})
	if err != nil {
		return nil, err
	}
	app.core = core
	return app, nil
}

func (a *App) dependencies(core *cliapp.ScenarioApp) support.Dependencies {
	return support.Dependencies{
		Core:     core,
		Selector: a.selector,
	}
}

func (a *App) Run(args []string) error {
	normalized, treeID, treeSlug, err := a.extractTreeArgs(args)
	if err != nil {
		return err
	}
	a.selector.Set(treeID, treeSlug)
	return a.core.CLI.Run(a.normalizeArgs(normalized))
}

func (a *App) normalizeArgs(args []string) []string {
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg != "" {
			cleaned = append(cleaned, arg)
		}
	}
	if len(cleaned) == 0 {
		return cleaned
	}

	switch cleaned[0] {
	case "config":
		cleaned[0] = "configure"
	case "status":
		cleaned[0] = "overview"
	case "dependencies":
		cleaned = append([]string{"graph", "dependencies"}, cleaned[1:]...)
	}

	if cleaned[0] == "progress" && len(cleaned) > 1 {
		if contains(cleaned[1:], "--list") {
			cleaned = append([]string{"progress", "list"}, removeToken(cleaned[1:], "--list")...)
		} else if contains(cleaned[1:], "--scenario") && contains(cleaned[1:], "--status") {
			cleaned = append([]string{"progress", "set-status"}, cleaned[1:]...)
		}
	}

	switch cleaned[0] {
	case "trees", "sectors", "stages", "progress", "milestones", "catalog":
		if shouldInsertList(cleaned[1:]) {
			cleaned = append([]string{cleaned[0], "list"}, cleaned[1:]...)
		}
	case "graph":
		if len(cleaned) == 1 {
			cleaned = append(cleaned, "export")
		}
	}

	return cleaned
}

func (a *App) extractTreeArgs(args []string) ([]string, string, string, error) {
	cleaned := make([]string, 0, len(args))
	var treeID string
	var treeSlug string
	for i := 0; i < len(args); i++ {
		switch strings.TrimSpace(args[i]) {
		case "--tree":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--tree requires a tree ID")
			}
			treeID = strings.TrimSpace(args[i+1])
			i++
		case "--tree-slug":
			if i+1 >= len(args) {
				return nil, "", "", fmt.Errorf("--tree-slug requires a slug")
			}
			treeSlug = strings.TrimSpace(args[i+1])
			i++
		default:
			cleaned = append(cleaned, args[i])
		}
	}
	return cleaned, treeID, treeSlug, nil
}

func shouldInsertList(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "list", "get", "create", "update", "delete", "clone", "children", "set-maturity", "link", "unlink", "set-status", "refresh", "hide", "show", "dependencies", "connections", "export", "help", "-h", "--help":
		return false
	default:
		return strings.HasPrefix(args[0], "-")
	}
}

func contains(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func removeToken(args []string, target string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == target {
			continue
		}
		out = append(out, arg)
	}
	return out
}
