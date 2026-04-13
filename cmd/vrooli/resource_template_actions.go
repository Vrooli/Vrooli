package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
)

type (
	resourceTemplateNoArgsRequest = resourcecli.NoArgsRequest
	resourceTemplateNameRequest   = resourcecli.TemplateNameRequest
)

func parseResourceTemplateListRequest(globals globalOptions, args []string) (resourceTemplateNoArgsRequest, error) {
	_ = globals
	req, err := resourcecli.ParseTemplateListRequest(args)
	return req, mapResourceParseError("resource template list", err)
}

func runResourceTemplateListRequest(controller *resources.Controller, ctx *commandContext, req resourceTemplateNoArgsRequest) (cliout.Format, []resources.ResourceTemplateInfo, error) {
	_ = req
	items, err := newResourceTemplateCommandService(ctx, controller).TemplateList()
	if err != nil {
		return "", nil, err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", nil, err
	}
	return format, items, nil
}

func renderResourceTemplateListResponse(w io.Writer, format cliout.Format, items []resources.ResourceTemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "templates", items)
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

func parseResourceTemplateShowRequest(globals globalOptions, args []string) (resourceTemplateNameRequest, error) {
	_ = globals
	req, err := resourcecli.ParseTemplateShowRequest(args)
	return req, mapResourceParseError("resource template show", err)
}

func runResourceTemplateShowRequest(controller *resources.Controller, ctx *commandContext, req resourceTemplateNameRequest) (cliout.Format, resources.ResourceTemplateInfo, error) {
	info, err := newResourceTemplateCommandService(ctx, controller).TemplateShow(req.Name)
	if err != nil {
		return "", resources.ResourceTemplateInfo{}, err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", resources.ResourceTemplateInfo{}, err
	}
	return format, info, nil
}

func renderResourceTemplateShowResponse(w io.Writer, format cliout.Format, info resources.ResourceTemplateInfo) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "template", info)
	}
	manifest := info.Manifest
	rows := [][]string{
		{"Name", info.Name},
		{"Display Name", manifest.DisplayName},
		{"Driver", manifest.Driver},
		{"Transitional", cliout.BoolLabel(manifest.Transitional)},
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

func parseResourceTemplateValidateRequest(globals globalOptions, args []string) (resourceTemplateNoArgsRequest, error) {
	_ = globals
	req, err := resourcecli.ParseTemplateValidateRequest(args)
	return req, mapResourceParseError("resource template validate", err)
}

func runResourceTemplateValidateRequest(controller *resources.Controller, ctx *commandContext, req resourceTemplateNoArgsRequest) (cliout.Format, resources.ResourceTemplateValidationReport, error) {
	_ = req
	report, err := newResourceTemplateCommandService(ctx, controller).TemplateValidate()
	if err != nil {
		return "", resources.ResourceTemplateValidationReport{}, err
	}
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", resources.ResourceTemplateValidationReport{}, err
	}
	return format, report, nil
}

func renderResourceTemplateValidateResponse(w io.Writer, format cliout.Format, report resources.ResourceTemplateValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Validated %d resource templates\n", report.Count)
	return nil
}

func parseResourceTemplateGenerateRequest(controller *resources.Controller, globals globalOptions, args []string, stderr io.Writer) (resourcecli.TemplateGenerateOptions, error) {
	_ = globals
	req, err := resourcecli.ParseTemplateGenerateRequest(args, stderr, func(req resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateInfo, error) {
		return controller.ResolveTemplateGenerationRequest(req)
	})
	if err != nil {
		return resourcecli.TemplateGenerateOptions{}, mapResourceParseError("resource template generate", err)
	}
	return req, nil
}

func runResourceTemplateGenerateRequest(controller *resources.Controller, ctx *commandContext, req resourcecli.TemplateGenerateOptions) (cliout.Format, resources.ResourceTemplateGenerateReport, error) {
	report, err := newResourceTemplateCommandService(ctx, controller).TemplateGenerate(resources.ResourceTemplateGenerateRequest{
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
	format, err := formatFromJSON(ctx.Globals.json)
	if err != nil {
		return "", resources.ResourceTemplateGenerateReport{}, err
	}
	return format, report, nil
}

func newResourceTemplateCommandService(ctx *commandContext, controller *resources.Controller) resourceapp.Service {
	return resourceapp.Service{
		Resources: controller,
		Stdout:    ctx.Stdout,
		Stderr:    ctx.Stderr,
	}
}

func renderResourceTemplateGenerateResponse(w io.Writer, format cliout.Format, report resources.ResourceTemplateGenerateReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "report", report)
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
