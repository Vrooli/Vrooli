package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type resourceTemplateCommandAction[Req any, Resp any] struct {
	parse  func(args []string) (Req, error)
	run    func(controller *resources.Controller, globals globalOptions, stderr io.Writer, req Req) (cliout.Format, Resp, error)
	render func(w io.Writer, format cliout.Format, resp Resp) error
}

func executeResourceTemplateCommand[Req any, Resp any](controller *resources.Controller, globals globalOptions, args []string, stdout, stderr io.Writer, action resourceTemplateCommandAction[Req, Resp]) error {
	req, err := action.parse(args)
	if err != nil {
		var helpErr commandHelpError
		if errors.As(err, &helpErr) {
			_, _ = fmt.Fprintln(stdout, helpErr.message)
			return nil
		}
		return err
	}
	format, resp, err := action.run(controller, globals, stderr, req)
	if err != nil {
		return err
	}
	return action.render(stdout, format, resp)
}

type (
	resourceTemplateNoArgsRequest struct{}
	resourceTemplateNameRequest   struct{ Name string }
)

func parseResourceTemplateNoArgs(help, command string, args []string) (resourceTemplateNoArgsRequest, error) {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return resourceTemplateNoArgsRequest{}, commandHelpOnly(help)
	}
	if len(args) > 0 {
		return resourceTemplateNoArgsRequest{}, usageErrorf(command, "%s does not accept positional arguments", command)
	}
	return resourceTemplateNoArgsRequest{}, nil
}

func parseResourceTemplateName(help, command string, args []string) (resourceTemplateNameRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return resourceTemplateNameRequest{}, commandHelpOnly(help)
		}
	}
	if len(args) != 1 {
		return resourceTemplateNameRequest{}, usageErrorf(command, "%s requires exactly one template name", command)
	}
	return resourceTemplateNameRequest{Name: args[0]}, nil
}

func parseResourceTemplateListRequest(args []string) (resourceTemplateNoArgsRequest, error) {
	return parseResourceTemplateNoArgs("Usage: vrooli resource template list", "resource template list", args)
}

func runResourceTemplateListRequest(controller *resources.Controller, globals globalOptions, stderr io.Writer, req resourceTemplateNoArgsRequest) (cliout.Format, []resources.ResourceTemplateInfo, error) {
	items, err := controller.ListResourceTemplates()
	if err != nil {
		return "", nil, err
	}
	format, err := formatFromJSON(globals.json)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceTemplateListResponse(w io.Writer, format cliout.Format, items []resources.ResourceTemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": true, "templates": items})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Name,
			item.Manifest.DisplayName,
			item.Manifest.Driver,
			formatResourceTemplateRequiredVars(item.Manifest.RequiredVars),
		})
	}
	if err := cliout.RenderTable(w, []string{"Name", "Display Name", "Driver", "Required Vars"}, rows); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Tip: vrooli resource template show <name>")
	return nil
}

func parseResourceTemplateShowRequest(args []string) (resourceTemplateNameRequest, error) {
	return parseResourceTemplateName("Usage: vrooli resource template show <name>", "resource template show", args)
}

func runResourceTemplateShowRequest(controller *resources.Controller, globals globalOptions, stderr io.Writer, req resourceTemplateNameRequest) (cliout.Format, resources.ResourceTemplateInfo, error) {
	info, err := controller.ResourceTemplate(req.Name)
	if err != nil {
		return "", resources.ResourceTemplateInfo{}, err
	}
	format, err := formatFromJSON(globals.json)
	if err != nil {
		return "", resources.ResourceTemplateInfo{}, err
	}
	return format, info, nil
}

func renderResourceTemplateShowResponse(w io.Writer, format cliout.Format, info resources.ResourceTemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": true, "template": info})
	}
	manifest := info.Manifest
	rows := [][]string{
		{"Name", info.Name},
		{"Display Name", manifest.DisplayName},
		{"Driver", manifest.Driver},
		{"Transitional", boolLabel(manifest.Transitional)},
		{"Description", manifest.Description},
	}
	if err := cliout.RenderTable(w, []string{"Field", "Value"}, rows); err != nil {
		return err
	}
	writeResourceTemplateVarTable(w, "Required Variables", manifest.RequiredVars)
	writeResourceTemplateVarTable(w, "Optional Variables", manifest.OptionalVars)
	if len(manifest.PlatformExpectations) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Platform Expectations:")
		for _, line := range manifest.PlatformExpectations {
			_, _ = fmt.Fprintf(w, "  - %s\n", line)
		}
	}
	if len(manifest.Docs) > 0 {
		keys := make([]string, 0, len(manifest.Docs))
		for key := range manifest.Docs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Docs:")
		for _, key := range keys {
			_, _ = fmt.Fprintf(w, "  - %s: %s\n", key, manifest.Docs[key])
		}
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Tip: vrooli resource template generate %s%s\n", info.Name, formatResourceTemplateRequiredFlags(manifest.RequiredVars))
	return nil
}

func parseResourceTemplateValidateRequest(args []string) (resourceTemplateNoArgsRequest, error) {
	return parseResourceTemplateNoArgs("Usage: vrooli resource template validate", "resource template validate", args)
}

func runResourceTemplateValidateRequest(controller *resources.Controller, globals globalOptions, stderr io.Writer, req resourceTemplateNoArgsRequest) (cliout.Format, resources.ResourceTemplateValidationReport, error) {
	report, err := controller.ValidateResourceTemplates()
	if err != nil {
		return "", resources.ResourceTemplateValidationReport{}, err
	}
	format, err := formatFromJSON(globals.json)
	if err != nil {
		return "", resources.ResourceTemplateValidationReport{}, err
	}
	return format, report, nil
}

func renderResourceTemplateValidateResponse(w io.Writer, format cliout.Format, report resources.ResourceTemplateValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": true, "report": report})
	}
	_, _ = fmt.Fprintf(w, "Validated %d resource templates\n", report.Count)
	return nil
}

func parseResourceTemplateGenerateRequest(controller *resources.Controller, args []string, stderr io.Writer) (resourceTemplateGenerateOptions, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return resourceTemplateGenerateOptions{}, commandHelpOnly("Usage: vrooli resource template generate <template> [options]\n       vrooli resource template generate --from-blueprint <name> [options]")
		}
	}
	return parseResourceTemplateGenerateArgs(controller, args, stderr)
}

func runResourceTemplateGenerateRequest(controller *resources.Controller, globals globalOptions, stderr io.Writer, req resourceTemplateGenerateOptions) (cliout.Format, resources.ResourceTemplateGenerateReport, error) {
	report, err := controller.GenerateResourceTemplate(resources.ResourceTemplateGenerateRequest{
		TemplateName:  req.TemplateName,
		BlueprintName: req.BlueprintName,
		Destination:   req.Destination,
		Force:         req.Force,
		DryRun:        req.DryRun,
		Values:        req.Values,
	})
	if err != nil {
		return "", resources.ResourceTemplateGenerateReport{}, err
	}
	format, err := formatFromJSON(globals.json)
	if err != nil {
		return "", resources.ResourceTemplateGenerateReport{}, err
	}
	return format, report, nil
}

func renderResourceTemplateGenerateResponse(w io.Writer, format cliout.Format, report resources.ResourceTemplateGenerateReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": true, "report": report})
	}
	if report.DryRun {
		_, _ = fmt.Fprintf(w, "[DRY-RUN] Would generate resource template %s at %s\n", report.Template.Name, report.Destination)
	} else {
		_, _ = fmt.Fprintf(w, "Generated resource template %s at %s\n", report.Template.Name, report.Destination)
	}
	if strings.TrimSpace(report.BlueprintName) != "" {
		_, _ = fmt.Fprintf(w, "Blueprint: %s\n", report.BlueprintName)
	}
	writeResourceTemplateValues(w, report.Values)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Files:")
	for _, path := range report.Files {
		_, _ = fmt.Fprintf(w, "  - %s\n", path)
	}
	return nil
}
