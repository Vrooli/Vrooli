package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/shell"
)

var packageCommandTable = []appSubcommandDescriptor{
	{Name: "list", Summary: "List governed packages", Handler: bindGlobalCommand(parsePackageListRequest, runPackageListRequest, renderPackageListResponse)},
	{Name: "info", Summary: "Show package manifest metadata", Handler: bindGlobalCommand(parsePackageInfoRequest, runPackageInfoRequest, renderPackageInfoResponse)},
	{Name: "dependents", Summary: "List package consumers", Handler: bindGlobalCommand(parsePackageDependentsRequest, runPackageDependentsRequest, renderPackageDependentsResponse)},
	{Name: "validate", Summary: "Validate package manifests and package adoption policy", Handler: bindGlobalCommand(parsePackageValidateRequest, runPackageValidateRequest, renderPackageValidateResponse)},
	{Name: "build", Summary: "Run the package build lifecycle", Handler: bindGlobalCommand(parsePackageRunRequest("build"), runPackageBuildRequest, renderPackageRunResponse)},
	{Name: "generate", Summary: "Run the package generation lifecycle", Handler: bindGlobalCommand(parsePackageRunRequest("generate"), runPackageGenerateRequest, renderPackageRunResponse)},
	{Name: "refresh", Summary: "Rebuild/regenerate a package and propagate to affected consumers", Handler: bindGlobalCommand(parsePackageRefreshRequest, runPackageRefreshRequest, renderPackageRefreshResponse)},
	{Name: "audit", Summary: "Report governance drift and unsupported package adoption", Handler: bindGlobalCommand(parsePackageAuditRequest, runPackageAuditRequest, renderPackageAuditResponse)},
}

var packageCommandHandlers = buildAppSubcommandMap(packageCommandTable)

type packageListRequest struct{}

type packageInfoRequest struct {
	Name string
}

type packageDependentsRequest struct {
	Name string
}

type packageValidateRequest struct {
	Name string
	All  bool
}

type packageRunRequest struct {
	Name string
}

type packageRefreshRequest struct {
	Name      string
	Target    string
	NoRestart bool
}

type packageAuditRequest struct {
	Name string
	All  bool
}

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

type packageRefreshItem struct {
	Scenario string `json:"scenario"`
	Status   string `json:"status"`
}

type packageRefreshResponse struct {
	PackageName string               `json:"package_name"`
	Items       []packageRefreshItem `json:"items"`
}

type packageAuditResponse struct {
	Report packagegov.AuditReport `json:"report"`
}

func runPackageRootCommand(app *App, ctx *commandContext, args []string) error {
	return runAppSubcommandSet(app, ctx, args, showPackageHelp, "package", packageCommandHandlers)
}

func showPackageHelp(w io.Writer) {
	renderSubcommandHelp(w, "Vrooli Package Commands", "vrooli package <subcommand> [options]", "Package Governance", packageCommandTable)
}

func parsePackageListRequest(globals globalOptions, args []string) (packageListRequest, error) {
	if len(args) > 0 {
		return packageListRequest{}, unknownOptionError("package list", args[0])
	}
	return packageListRequest{}, nil
}

func runPackageListRequest(app *App, ctx *commandContext, req packageListRequest) (cliout.Format, packageListResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageListResponse{}, err
	}
	items, issues, err := packagegov.LoadAll(ctx.Root)
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

func parsePackageInfoRequest(globals globalOptions, args []string) (packageInfoRequest, error) {
	if len(args) != 1 {
		return packageInfoRequest{}, usageErrorf("package info", "package info requires exactly one package name")
	}
	return packageInfoRequest{Name: args[0]}, nil
}

func runPackageInfoRequest(app *App, ctx *commandContext, req packageInfoRequest) (cliout.Format, packagegov.Package, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packagegov.Package{}, err
	}
	items, _, err := packagegov.LoadAll(ctx.Root)
	if err != nil {
		return "", packagegov.Package{}, err
	}
	item, ok := packagegov.FindByName(items, req.Name)
	if !ok {
		return "", packagegov.Package{}, usageErrorf("package info", "package %q not found", req.Name)
	}
	return format, item, nil
}

func parsePackageDependentsRequest(globals globalOptions, args []string) (packageDependentsRequest, error) {
	if len(args) != 1 {
		return packageDependentsRequest{}, usageErrorf("package dependents", "package dependents requires exactly one package name")
	}
	return packageDependentsRequest{Name: args[0]}, nil
}

func runPackageDependentsRequest(app *App, ctx *commandContext, req packageDependentsRequest) (cliout.Format, packageDependentsResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageDependentsResponse{}, err
	}
	items, _, err := packagegov.LoadAll(ctx.Root)
	if err != nil {
		return "", packageDependentsResponse{}, err
	}
	item, ok := packagegov.FindByName(items, req.Name)
	if !ok {
		return "", packageDependentsResponse{}, usageErrorf("package dependents", "package %q not found", req.Name)
	}
	report, err := packagegov.DiscoverDependents(ctx.Root, item)
	if err != nil {
		return "", packageDependentsResponse{}, err
	}
	return format, packageDependentsResponse{
		PackageName: item.Name,
		Dependents:  report.Dependents,
		Issues:      report.Issues,
	}, nil
}

func parsePackageValidateRequest(globals globalOptions, args []string) (packageValidateRequest, error) {
	req := packageValidateRequest{}
	for _, arg := range args {
		switch arg {
		case "--all":
			req.All = true
		default:
			if strings.HasPrefix(arg, "-") {
				return packageValidateRequest{}, unknownOptionError("package validate", arg)
			}
			if req.Name != "" {
				return packageValidateRequest{}, usageErrorf("package validate", "package validate accepts at most one package name")
			}
			req.Name = arg
		}
	}
	if !req.All && req.Name == "" {
		req.All = true
	}
	return req, nil
}

func runPackageValidateRequest(app *App, ctx *commandContext, req packageValidateRequest) (cliout.Format, packageValidateResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageValidateResponse{}, err
	}
	name := req.Name
	if req.All {
		name = ""
	}
	report, err := packagegov.Validate(ctx.Root, name)
	if err != nil {
		return "", packageValidateResponse{}, err
	}
	return format, packageValidateResponse{Report: report}, nil
}

func parsePackageRunRequest(action string) func(globalOptions, []string) (packageRunRequest, error) {
	return func(globals globalOptions, args []string) (packageRunRequest, error) {
		if len(args) != 1 {
			return packageRunRequest{}, usageErrorf("package "+action, "package %s requires exactly one package name", action)
		}
		return packageRunRequest{Name: args[0]}, nil
	}
}

func runPackageBuildRequest(app *App, ctx *commandContext, req packageRunRequest) (cliout.Format, packageRunResponse, error) {
	return runPackageLifecycle(ctx, req.Name, "build")
}

func runPackageGenerateRequest(app *App, ctx *commandContext, req packageRunRequest) (cliout.Format, packageRunResponse, error) {
	return runPackageLifecycle(ctx, req.Name, "generate")
}

func runPackageLifecycle(ctx *commandContext, name, action string) (cliout.Format, packageRunResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	item, err := loadNamedPackage(ctx.Root, name)
	if err != nil {
		return "", packageRunResponse{}, err
	}
	var commands []packagegov.CommandSpec
	switch action {
	case "build":
		commands = item.Manifest.Package.Lifecycle.Build
	case "generate":
		commands = item.Manifest.Package.Lifecycle.Generate
	}
	if err := packagegov.RunCommands(item.RootPath, commands, ctx.Stdout, ctx.Stderr); err != nil {
		return "", packageRunResponse{}, err
	}
	return format, packageRunResponse{PackageName: item.Name, Action: action}, nil
}

func parsePackageRefreshRequest(globals globalOptions, args []string) (packageRefreshRequest, error) {
	req := packageRefreshRequest{Target: "all"}
	for _, arg := range args {
		switch arg {
		case "--no-restart":
			req.NoRestart = true
		default:
			if strings.HasPrefix(arg, "-") {
				return packageRefreshRequest{}, unknownOptionError("package refresh", arg)
			}
			if req.Name == "" {
				req.Name = arg
				continue
			}
			if req.Target == "all" {
				req.Target = arg
				continue
			}
			return packageRefreshRequest{}, usageErrorf("package refresh", "package refresh accepts at most a package name and one target scenario")
		}
	}
	if req.Name == "" {
		return packageRefreshRequest{}, usageErrorf("package refresh", "package refresh requires a package name")
	}
	return req, nil
}

func runPackageRefreshRequest(app *App, ctx *commandContext, req packageRefreshRequest) (cliout.Format, packageRefreshResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageRefreshResponse{}, err
	}
	item, err := loadNamedPackage(ctx.Root, req.Name)
	if err != nil {
		return "", packageRefreshResponse{}, err
	}

	if item.Manifest.Package.Refresh.Strategy == packagegov.RefreshGenerateThenSetup {
		if err := packagegov.RunCommands(item.RootPath, item.Manifest.Package.Lifecycle.Generate, ctx.Stdout, ctx.Stderr); err != nil {
			return "", packageRefreshResponse{}, err
		}
	}
	if err := packagegov.RunCommands(item.RootPath, item.Manifest.Package.Lifecycle.Build, ctx.Stdout, ctx.Stderr); err != nil {
		return "", packageRefreshResponse{}, err
	}

	discovery, err := packagegov.DiscoverDependents(ctx.Root, item)
	if err != nil {
		return "", packageRefreshResponse{}, err
	}
	targets := packagegov.MatchDependents(discovery.Dependents, req.Target)
	byScenario := make(map[string]struct{})
	for _, dep := range targets {
		if strings.HasPrefix(string(dep.ConsumerClass), "scenario_") {
			byScenario[dep.ConsumerName] = struct{}{}
		}
	}
	names := make([]string, 0, len(byScenario))
	for name := range byScenario {
		names = append(names, name)
	}
	sort.Strings(names)

	service, err := app.newScenarioService(ctx)
	if err != nil {
		return "", packageRefreshResponse{}, err
	}
	runner, err := app.newScenarioLifecycleRunner(ctx)
	if err != nil {
		return "", packageRefreshResponse{}, err
	}

	resp := packageRefreshResponse{PackageName: item.Name}
	for _, scenarioName := range names {
		depsForScenario := make([]packagegov.Dependent, 0, len(targets))
		for _, dep := range targets {
			if dep.ConsumerName == scenarioName {
				depsForScenario = append(depsForScenario, dep)
			}
		}

		status := "no_action"

		switch item.Manifest.Package.Refresh.Strategy {
		case packagegov.RefreshScenarioSetup, packagegov.RefreshGenerateThenSetup:
			detail, _, err := service.Lookup(scenarioName)
			if err != nil {
				return "", packageRefreshResponse{}, err
			}
			wasRunning := detail.Runtime.ProcessCount > 0
			if wasRunning {
				if err := runner.Stop(scenarioName, lifecycle.StopOptions{}); err != nil {
					return "", packageRefreshResponse{}, err
				}
			}
			if _, err := runner.RunPhaseDetailed(scenarioName, "setup", lifecycle.PhaseOptions{}); err != nil {
				return "", packageRefreshResponse{}, err
			}
			status = "setup_only"
			if wasRunning && !req.NoRestart && item.Manifest.Package.Refresh.RestartRunningConsumers {
				if _, err := service.StartDetailed(scenarioName, lifecycle.StartOptions{}); err != nil {
					return "", packageRefreshResponse{}, err
				}
				status = "restarted"
			} else if wasRunning {
				status = "stopped_after_setup"
			}
		case packagegov.RefreshRestartConsumers:
			detail, _, err := service.Lookup(scenarioName)
			if err != nil {
				return "", packageRefreshResponse{}, err
			}
			wasRunning := detail.Runtime.ProcessCount > 0
			if !wasRunning {
				status = "not_running"
				break
			}
			if req.NoRestart || !item.Manifest.Package.Refresh.RestartRunningConsumers {
				status = "running_not_restarted"
				break
			}
			if err := runner.Stop(scenarioName, lifecycle.StopOptions{}); err != nil {
				return "", packageRefreshResponse{}, err
			}
			if _, err := service.StartDetailed(scenarioName, lifecycle.StartOptions{}); err != nil {
				return "", packageRefreshResponse{}, err
			}
			status = "restarted"
		case packagegov.RefreshRebuildCLI:
			rebuilt, err := rebuildGoConsumerTargets(depsForScenario, ctx.Stdout, ctx.Stderr)
			if err != nil {
				return "", packageRefreshResponse{}, err
			}
			if rebuilt {
				status = "rebuilt"
			}
		case packagegov.RefreshNone:
			status = "no_action"
		}

		resp.Items = append(resp.Items, packageRefreshItem{Scenario: scenarioName, Status: status})
	}
	return format, resp, nil
}

func rebuildGoConsumerTargets(dependents []packagegov.Dependent, stdout, stderr io.Writer) (bool, error) {
	seen := make(map[string]struct{}, len(dependents))
	rebuilt := false
	for _, dep := range dependents {
		buildPath := filepath.Clean(dep.ConsumerPath)
		if strings.EqualFold(filepath.Base(dep.DependencyFile), "go.mod") {
			buildPath = filepath.Dir(dep.DependencyFile)
		}
		if buildPath == "." || buildPath == "" {
			continue
		}
		if _, ok := seen[buildPath]; ok {
			continue
		}
		seen[buildPath] = struct{}{}
		if _, err := os.Stat(filepath.Join(buildPath, "go.mod")); err != nil {
			continue
		}
		spec := shell.Spec{
			Name:   "go",
			Args:   []string{"build", "./..."},
			Dir:    buildPath,
			Env:    append(os.Environ(), "GOWORK=off"),
			Stdout: stdout,
			Stderr: stderr,
		}
		if err := shell.Run(spec); err != nil {
			return rebuilt, err
		}
		rebuilt = true
	}
	return rebuilt, nil
}

func parsePackageAuditRequest(globals globalOptions, args []string) (packageAuditRequest, error) {
	req := packageAuditRequest{}
	for _, arg := range args {
		switch arg {
		case "--all":
			req.All = true
		default:
			if strings.HasPrefix(arg, "-") {
				return packageAuditRequest{}, unknownOptionError("package audit", arg)
			}
			if req.Name != "" {
				return packageAuditRequest{}, usageErrorf("package audit", "package audit accepts at most one package name")
			}
			req.Name = arg
		}
	}
	if req.Name == "" {
		req.All = true
	}
	return req, nil
}

func runPackageAuditRequest(app *App, ctx *commandContext, req packageAuditRequest) (cliout.Format, packageAuditResponse, error) {
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return "", packageAuditResponse{}, err
	}
	name := req.Name
	if req.All {
		name = ""
	}
	report, err := packagegov.Audit(ctx.Root, name)
	if err != nil {
		return "", packageAuditResponse{}, err
	}
	return format, packageAuditResponse{Report: report}, nil
}

func loadNamedPackage(root, name string) (packagegov.Package, error) {
	items, _, err := packagegov.LoadAll(root)
	if err != nil {
		return packagegov.Package{}, err
	}
	item, ok := packagegov.FindByName(items, name)
	if !ok {
		return packagegov.Package{}, usageErrorf("package", "package %q not found", name)
	}
	return item, nil
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

func renderPackageRefreshResponse(w io.Writer, format cliout.Format, resp packageRefreshResponse) error {
	if format == cliout.FormatJSON {
		return writeSuccessData(w, "refresh", resp)
	}
	if len(resp.Items) == 0 {
		_, _ = fmt.Fprintf(w, "refreshed %s with no affected scenario consumers\n", resp.PackageName)
		return nil
	}
	for _, item := range resp.Items {
		_, _ = fmt.Fprintf(w, "%s\t%s\n", item.Scenario, item.Status)
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
