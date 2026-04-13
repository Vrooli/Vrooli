package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	packageapp "github.com/vrooli/vrooli/internal/app/package"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/packagecli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
)

var packageCommandTable = buildPackageCommandTable()

var packageCommandHandlers = commandtree.BuildHandlerMap(packageCommandTable)

type packageListResponse struct {
	Packages []packagegov.Package `json:"packages"`
}

type packageDependentsResponse struct {
	PackageName string                       `json:"package_name"`
	Dependents  []packagegov.Dependent       `json:"dependents"`
	Issues      []packagegov.ValidationIssue `json:"issues,omitempty"`
}

type packageValidateResponse struct {
	Report packagegov.ValidationReport `json:"report"`
}

type packageRunResponse struct {
	PackageName string `json:"package_name"`
	Action      string `json:"action"`
}

type packageAuditResponse struct {
	Report packagegov.AuditReport `json:"report"`
}

func runPackageRootCommand(app *App, ctx *commandContext, args []string) error {
	return runAppSubcommandSet(app, ctx, args, showPackageHelp, "package", packageCommandHandlers)
}

func showPackageHelp(w io.Writer) {
	packagecli.RenderCommandHelp(w)
}

func runPackageListRequest(app *App, ctx *commandContext, req packagecli.ListRequest) (cliout.Format, packageListResponse, error) {
	_ = app
	_ = req
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageListResponse{}, err
	}
	items, issues, err := packageapp.Service{Root: ctx.Root}.List()
	if err != nil {
		return "", packageListResponse{}, err
	}
	if len(issues) > 0 && format != cliout.FormatJSON {
		for _, issue := range issues {
			_, _ = fmt.Fprintf(ctx.Stderr, "warning: %s\n", issue.Message)
		}
	}
	return format, packageListResponse{Packages: items}, nil
}

func runPackageInfoRequest(app *App, ctx *commandContext, req packagecli.InfoRequest) (cliout.Format, packagegov.Package, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packagegov.Package{}, err
	}
	item, err := packageapp.Service{Root: ctx.Root}.Info(req.Name)
	if err != nil {
		return "", packagegov.Package{}, usageErrorf("package info", err.Error())
	}
	return format, item, nil
}

func runPackageDependentsRequest(app *App, ctx *commandContext, req packagecli.DependentsRequest) (cliout.Format, packageDependentsResponse, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageDependentsResponse{}, err
	}
	item, report, err := packageapp.Service{Root: ctx.Root}.Dependents(req.Name)
	if err != nil {
		return "", packageDependentsResponse{}, usageErrorf("package dependents", err.Error())
	}
	return format, packageDependentsResponse{
		PackageName: item.Name,
		Dependents:  report.Dependents,
		Issues:      report.Issues,
	}, nil
}

func runPackageValidateRequest(app *App, ctx *commandContext, req packagecli.ValidateRequest) (cliout.Format, packageValidateResponse, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageValidateResponse{}, err
	}
	name := req.Name
	if req.All {
		name = ""
	}
	report, err := packageapp.Service{Root: ctx.Root}.Validate(name)
	if err != nil {
		return "", packageValidateResponse{}, err
	}
	return format, packageValidateResponse{Report: report}, nil
}

func runPackageBuildRequest(app *App, ctx *commandContext, req packagecli.RunRequest) (cliout.Format, packageRunResponse, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	writers := packageCommandOutputWriters(ctx, format)
	resp, err := packageapp.Service{Root: ctx.Root, Stdout: writers.stdout, Stderr: writers.stderr}.Build(req.Name)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	return format, packageRunResponse(resp), nil
}

func runPackageGenerateRequest(app *App, ctx *commandContext, req packagecli.RunRequest) (cliout.Format, packageRunResponse, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	writers := packageCommandOutputWriters(ctx, format)
	resp, err := packageapp.Service{Root: ctx.Root, Stdout: writers.stdout, Stderr: writers.stderr}.Generate(req.Name)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	return format, packageRunResponse(resp), nil
}

func runPackageRefreshRequest(app *App, ctx *commandContext, req packagecli.RefreshRequest) (cliout.Format, packageapp.RefreshResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageapp.RefreshResponse{}, err
	}
	opCtx := ctx
	if format == cliout.FormatJSON {
		cloned := *ctx
		cloned.Stdout = ctx.Stderr
		opCtx = &cloned
	}
	writers := packageCommandOutputWriters(opCtx, format)
	resp, err := packageapp.Service{
		Root:   ctx.Root,
		Stdout: writers.stdout,
		Stderr: writers.stderr,
		ScenarioService: func() (packageapp.ScenarioRuntime, error) {
			return app.newScenarioService(opCtx)
		},
		ScenarioRunner: func() (packageapp.ScenarioPhaseRunner, error) {
			return app.newScenarioLifecycleRunner(opCtx)
		},
	}.Refresh(packageapp.RefreshRequest{
		PackageName: req.Name,
		Target:      req.Target,
		NoRestart:   req.NoRestart,
	})
	if err != nil {
		return "", packageapp.RefreshResponse{}, err
	}
	return format, resp, nil
}

type packageCommandWriters struct {
	stdout io.Writer
	stderr io.Writer
}

func packageCommandOutputWriters(ctx *commandContext, format cliout.Format) packageCommandWriters {
	if format == cliout.FormatJSON {
		return packageCommandWriters{stdout: ctx.Stderr, stderr: ctx.Stderr}
	}
	return packageCommandWriters{stdout: ctx.Stdout, stderr: ctx.Stderr}
}

func runPackageAuditRequest(app *App, ctx *commandContext, req packagecli.AuditRequest) (cliout.Format, packageAuditResponse, error) {
	_ = app
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageAuditResponse{}, err
	}
	name := req.Name
	if req.All {
		name = ""
	}
	report, err := packageapp.Service{Root: ctx.Root}.Audit(name)
	if err != nil {
		return "", packageAuditResponse{}, err
	}
	return format, packageAuditResponse{Report: report}, nil
}

func renderPackageListResponse(w io.Writer, format cliout.Format, resp packageListResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "packages", resp.Packages)
	}
	for _, item := range resp.Packages {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, item.Manifest.Package.Kind, filepath.ToSlash(item.RootPath))
	}
	return nil
}

func renderPackageInfoResponse(w io.Writer, format cliout.Format, resp packagegov.Package) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "package", resp)
	}
	_, _ = fmt.Fprintf(w, "name: %s\n", resp.Name)
	_, _ = fmt.Fprintf(w, "kind: %s\n", resp.Manifest.Package.Kind)
	_, _ = fmt.Fprintf(w, "root: %s\n", filepath.ToSlash(resp.RootPath))
	_, _ = fmt.Fprintf(w, "display: %s\n", resp.Manifest.Package.DisplayName)
	_, _ = fmt.Fprintf(w, "adoptable: %t\n", resp.Manifest.Package.Adoption.ScenarioAdoptable)
	return nil
}

func renderPackageDependentsResponse(w io.Writer, format cliout.Format, resp packageDependentsResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "dependents", resp)
	}
	for _, dep := range resp.Dependents {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", dep.ConsumerName, dep.ConsumerClass, dep.AdoptionMode, filepath.ToSlash(dep.DependencyFile))
	}
	if len(resp.Issues) > 0 {
		_, _ = fmt.Fprintln(w)
		for _, issue := range resp.Issues {
			_, _ = fmt.Fprintf(w, "%s: %s (%s)\n", issue.Severity, issue.Message, filepath.ToSlash(issue.Path))
		}
	}
	return nil
}

func renderPackageValidateResponse(w io.Writer, format cliout.Format, resp packageValidateResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "report", resp.Report)
	}
	if len(resp.Report.Issues) == 0 {
		_, _ = fmt.Fprintln(w, "package governance validation passed")
		return nil
	}
	for _, issue := range resp.Report.Issues {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Severity, issue.Code, filepath.ToSlash(issue.Path), issue.Message)
	}
	return nil
}

func renderPackageRunResponse(w io.Writer, format cliout.Format, resp packageRunResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "result", resp)
	}
	_, _ = fmt.Fprintf(w, "%s %s completed\n", resp.Action, resp.PackageName)
	return nil
}

func renderPackageRefreshResponse(w io.Writer, format cliout.Format, resp packageapp.RefreshResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "refresh", resp)
	}
	if len(resp.Items) == 0 {
		_, _ = fmt.Fprintf(w, "refreshed %s with no affected governed consumers\n", resp.PackageName)
		return nil
	}
	for _, item := range resp.Items {
		classText := string(item.Class)
		if len(item.Classes) > 1 {
			parts := make([]string, 0, len(item.Classes))
			for _, class := range item.Classes {
				parts = append(parts, string(class))
			}
			classText = strings.Join(parts, ",")
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.Consumer, classText, item.Action, item.Status)
	}
	return nil
}

func renderPackageAuditResponse(w io.Writer, format cliout.Format, resp packageAuditResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "audit", resp.Report)
	}
	if len(resp.Report.Issues) == 0 {
		_, _ = fmt.Fprintln(w, "package governance audit passed")
		return nil
	}
	for _, issue := range resp.Report.Issues {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Severity, issue.Code, filepath.ToSlash(issue.Path), issue.Message)
	}
	return nil
}

func buildPackageCommandTable() []commandtree.Spec[appCommandHandler] {
	handlerMap := map[packagecli.CommandID]appCommandHandler{
		packagecli.CommandList: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.ListRequest, error) {
				_ = globals
				return packagecli.ParseListRequest(args)
			},
			runPackageListRequest,
			renderPackageListResponse,
		),
		packagecli.CommandInfo: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.InfoRequest, error) {
				_ = globals
				return packagecli.ParseInfoRequest(args)
			},
			runPackageInfoRequest,
			renderPackageInfoResponse,
		),
		packagecli.CommandDependents: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.DependentsRequest, error) {
				_ = globals
				return packagecli.ParseDependentsRequest(args)
			},
			runPackageDependentsRequest,
			renderPackageDependentsResponse,
		),
		packagecli.CommandValidate: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.ValidateRequest, error) {
				_ = globals
				return packagecli.ParseValidateRequest(args)
			},
			runPackageValidateRequest,
			renderPackageValidateResponse,
		),
		packagecli.CommandBuild: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.RunRequest, error) {
				_ = globals
				return packagecli.ParseRunRequest("build", args)
			},
			runPackageBuildRequest,
			renderPackageRunResponse,
		),
		packagecli.CommandGenerate: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.RunRequest, error) {
				_ = globals
				return packagecli.ParseRunRequest("generate", args)
			},
			runPackageGenerateRequest,
			renderPackageRunResponse,
		),
		packagecli.CommandRefresh: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.RefreshRequest, error) {
				_ = globals
				return packagecli.ParseRefreshRequest(args)
			},
			runPackageRefreshRequest,
			renderPackageRefreshResponse,
		),
		packagecli.CommandAudit: bindGlobalCommand(
			func(globals globalOptions, args []string) (packagecli.AuditRequest, error) {
				_ = globals
				return packagecli.ParseAuditRequest(args)
			},
			runPackageAuditRequest,
			renderPackageAuditResponse,
		),
	}

	source := packagecli.CommandSpecs()
	specs := make([]commandtree.Spec[appCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[appCommandHandler]{
			Name:        spec.Name,
			Aliases:     append([]string(nil), spec.Aliases...),
			Group:       spec.Group,
			Summary:     spec.Summary,
			Hidden:      spec.Hidden,
			Suggestable: spec.Suggestable,
			RootPolicy:  spec.RootPolicy,
			Help:        spec.Help,
			Handler:     handler,
		})
	}
	return specs
}
