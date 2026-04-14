package packagehandlers

import (
	"io"

	packageapp "github.com/vrooli/vrooli/internal/app/package"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/packagecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
)

type HandlerDeps[C any] struct {
	Stdout             func(C) io.Writer
	Stderr             func(C) io.Writer
	Root               func(C) string
	OutputFormat       func(C) (cliout.Format, error)
	ScenarioOperations func(C) (packageapp.ScenarioRuntime, error)
	LifecycleRunner    func(C) (packageapp.ScenarioPhaseRunner, error)
}

func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	handlers := buildCommandHandlers(deps)
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, packagecli.RenderCommandHelp, "package", handlers, deps.Stdout)
	}
}

func buildCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.Handler[C] {
	return commandtree.BuildHandlerMap(buildCommandTable(deps))
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[packagecli.CommandID]rootcli.Handler[C]{
		packagecli.CommandList: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.ListRequest, error) { return packagecli.ParseListRequest(args) },
			func(ctx C, req packagecli.ListRequest) (cliout.Format, packagecli.ListResponse, error) {
				_ = req
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.ListResponse{}, err
				}
				items, issues, err := packageapp.Service{Root: deps.Root(ctx)}.List()
				if err != nil {
					return "", packagecli.ListResponse{}, err
				}
				if len(issues) > 0 && format != cliout.FormatJSON {
					for _, issue := range issues {
						_, _ = io.WriteString(deps.Stderr(ctx), "warning: "+issue.Message+"\n")
					}
				}
				return format, packagecli.ListResponse{Packages: items}, nil
			},
			packagecli.RenderList,
		),
		packagecli.CommandInfo: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.InfoRequest, error) { return packagecli.ParseInfoRequest(args) },
			func(ctx C, req packagecli.InfoRequest) (cliout.Format, packagegov.Package, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagegov.Package{}, err
				}
				item, err := packageapp.Service{Root: deps.Root(ctx)}.Info(req.Name)
				if err != nil {
					return "", packagegov.Package{}, rootcli.UsageErrorf("package info", err.Error())
				}
				return format, item, nil
			},
			packagecli.RenderInfo,
		),
		packagecli.CommandDependents: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.DependentsRequest, error) { return packagecli.ParseDependentsRequest(args) },
			func(ctx C, req packagecli.DependentsRequest) (cliout.Format, packagecli.DependentsResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.DependentsResponse{}, err
				}
				item, report, err := packageapp.Service{Root: deps.Root(ctx)}.Dependents(req.Name)
				if err != nil {
					return "", packagecli.DependentsResponse{}, rootcli.UsageErrorf("package dependents", err.Error())
				}
				return format, packagecli.DependentsResponse{
					PackageName: item.Name,
					Dependents:  report.Dependents,
					Issues:      report.Issues,
				}, nil
			},
			packagecli.RenderDependents,
		),
		packagecli.CommandValidate: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.ValidateRequest, error) { return packagecli.ParseValidateRequest(args) },
			func(ctx C, req packagecli.ValidateRequest) (cliout.Format, packagecli.ValidateResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.ValidateResponse{}, err
				}
				name := req.Name
				if req.All {
					name = ""
				}
				report, err := packageapp.Service{Root: deps.Root(ctx)}.Validate(name)
				if err != nil {
					return "", packagecli.ValidateResponse{}, err
				}
				return format, packagecli.ValidateResponse{Report: report}, nil
			},
			packagecli.RenderValidate,
		),
		packagecli.CommandBuild: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.RunRequest, error) { return packagecli.ParseRunRequest("build", args) },
			func(ctx C, req packagecli.RunRequest) (cliout.Format, packagecli.RunResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.RunResponse{}, err
				}
				resp, err := newService(deps, ctx, format).Build(req.Name)
				if err != nil {
					return "", packagecli.RunResponse{}, err
				}
				return format, packagecli.RunResponse(resp), nil
			},
			packagecli.RenderRun,
		),
		packagecli.CommandGenerate: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.RunRequest, error) { return packagecli.ParseRunRequest("generate", args) },
			func(ctx C, req packagecli.RunRequest) (cliout.Format, packagecli.RunResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.RunResponse{}, err
				}
				resp, err := newService(deps, ctx, format).Generate(req.Name)
				if err != nil {
					return "", packagecli.RunResponse{}, err
				}
				return format, packagecli.RunResponse(resp), nil
			},
			packagecli.RenderRun,
		),
		packagecli.CommandRefresh: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.RefreshRequest, error) { return packagecli.ParseRefreshRequest(args) },
			func(ctx C, req packagecli.RefreshRequest) (cliout.Format, packagecli.RefreshResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.RefreshResponse{}, err
				}
				resp, err := newService(deps, ctx, format).Refresh(packageapp.RefreshRequest{
					PackageName: req.Name,
					Target:      req.Target,
					NoRestart:   req.NoRestart,
				})
				if err != nil {
					return "", packagecli.RefreshResponse{}, err
				}
				return format, packagecli.RefreshResponse{
					PackageName: resp.PackageName,
					Items:       toCLIRefreshItems(resp.Items),
				}, nil
			},
			packagecli.RenderRefresh,
		),
		packagecli.CommandAudit: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (packagecli.AuditRequest, error) { return packagecli.ParseAuditRequest(args) },
			func(ctx C, req packagecli.AuditRequest) (cliout.Format, packagecli.AuditResponse, error) {
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", packagecli.AuditResponse{}, err
				}
				name := req.Name
				if req.All {
					name = ""
				}
				report, err := packageapp.Service{Root: deps.Root(ctx)}.Audit(name)
				if err != nil {
					return "", packagecli.AuditResponse{}, err
				}
				return format, packagecli.AuditResponse{Report: report}, nil
			},
			packagecli.RenderAudit,
		),
	}

	source := packagecli.CommandSpecs()
	specs := make([]commandtree.Spec[rootcli.Handler[C]], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[rootcli.Handler[C]]{
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

func newService[C any](deps HandlerDeps[C], ctx C, format cliout.Format) packageapp.Service {
	stdout := deps.Stdout(ctx)
	stderr := deps.Stderr(ctx)
	if format == cliout.FormatJSON {
		stdout = stderr
	}
	return packageapp.Service{
		Root:   deps.Root(ctx),
		Stdout: stdout,
		Stderr: stderr,
		ScenarioService: func() (packageapp.ScenarioRuntime, error) {
			if deps.ScenarioOperations == nil {
				return nil, nil
			}
			return deps.ScenarioOperations(ctx)
		},
		ScenarioRunner: func() (packageapp.ScenarioPhaseRunner, error) {
			if deps.LifecycleRunner == nil {
				return nil, nil
			}
			return deps.LifecycleRunner(ctx)
		},
	}
}

func toCLIRefreshItems(items []packageapp.RefreshItem) []packagecli.RefreshItem {
	result := make([]packagecli.RefreshItem, 0, len(items))
	for _, item := range items {
		result = append(result, packagecli.RefreshItem{
			Consumer: item.Consumer,
			Class:    item.Class,
			Classes:  append([]packagegov.ConsumerClass(nil), item.Classes...),
			Action:   item.Action,
			Status:   item.Status,
		})
	}
	return result
}
