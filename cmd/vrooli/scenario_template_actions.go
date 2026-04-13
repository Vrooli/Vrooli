package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
)

func runScenarioTemplateListRequest(app *App, ctx *commandContext, req scenariocli.TemplateListRequest) (cliout.Format, []scenariocli.TemplateInfo, error) {
	_ = app
	_ = req
	templates, err := loadScenarioTemplates(ctx.Root)
	if err != nil {
		return "", nil, err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", nil, err
	}
	return format, templates, nil
}

func runScenarioTemplateShowRequest(app *App, ctx *commandContext, req scenariocli.TemplateShowRequest) (cliout.Format, scenariocli.TemplateInfo, error) {
	_ = app
	info, err := loadScenarioTemplate(ctx.Root, req.Name)
	if err != nil {
		return "", scenariocli.TemplateInfo{}, err
	}
	return cliout.FormatHuman, info, nil
}

func runScenarioGenerateRequest(app *App, ctx *commandContext, req scenariocli.GenerateRequest) (cliout.Format, scenariocli.GenerateResult, error) {
	info := req.TemplateInfo
	opts := req.Options

	currentDate := currentDateUTC()
	randomToken, err := randomTemplateToken()
	if err != nil {
		return "", scenariocli.GenerateResult{}, err
	}
	values := copyStringMap(opts.Values)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken

	optionalKeys := make([]string, 0, len(info.Manifest.OptionalVars))
	for key := range info.Manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) != "" {
			continue
		}
		values[key] = renderScenarioTemplateString(info.Manifest.OptionalVars[key].Default, values)
	}

	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(ctx.Root, "scenarios", values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(ctx.Root, filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)

	if opts.DryRun {
		return cliout.FormatHuman, scenariocli.GenerateResult{
			TemplateName: info.Name,
			DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
			Destination:  destination,
			Values:       values,
			Manifest:     info.Manifest,
			DryRun:       true,
			RunHooks:     false,
		}, nil
	}

	if stat, err := os.Stat(destination); err == nil && stat != nil {
		if !opts.Force {
			return "", scenariocli.GenerateResult{}, fmt.Errorf("destination already exists: %s (use --force to overwrite)", destination)
		}
		if err := os.RemoveAll(destination); err != nil {
			return "", scenariocli.GenerateResult{}, err
		}
	}

	if err := copyScenarioTemplate(info.Path, destination, values); err != nil {
		return "", scenariocli.GenerateResult{}, err
	}
	if err := verifyScenarioTemplate(destination); err != nil {
		return "", scenariocli.GenerateResult{}, err
	}

	result := scenariocli.GenerateResult{
		TemplateName: info.Name,
		DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
		Destination:  destination,
		Values:       values,
		Manifest:     info.Manifest,
		DryRun:       false,
		RunHooks:     opts.RunHooks,
	}
	if opts.RunHooks {
		if err := runScenarioTemplateHooksWithApp(app, ctx.Root, ctx.Globals, destination, info.Manifest, ctx.Stdout, ctx.Stderr); err != nil {
			return "", scenariocli.GenerateResult{}, err
		}
	}
	return cliout.FormatHuman, result, nil
}
