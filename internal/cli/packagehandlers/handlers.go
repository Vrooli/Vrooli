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
	TestGenieRunner    func(C, string, io.Writer, io.Writer) error
}

type packageService struct {
	packageapp.Service
	format cliout.Format
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
		packagecli.CommandList: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.ListRequest, error) { return packagecli.ParseListRequest(args) },
			func(service packageService, req packagecli.ListRequest) (packagecli.ListResponse, error) {
				items, issues, err := service.List()
				if err != nil {
					return packagecli.ListResponse{}, err
				}
				if len(issues) > 0 && service.format != cliout.FormatJSON {
					for _, issue := range issues {
						_, _ = io.WriteString(service.Stderr, "warning: "+issue.Message+"\n")
					}
				}
				return packagecli.ListResponse{Packages: items}, nil
			},
			packagecli.RenderList,
		),
		packagecli.CommandInfo: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.InfoRequest, error) { return packagecli.ParseInfoRequest(args) },
			func(service packageService, req packagecli.InfoRequest) (packagegov.Package, error) {
				item, err := service.Info(req.Name)
				if err != nil {
					return packagegov.Package{}, rootcli.UsageErrorf("package info", "%s", err.Error())
				}
				return item, nil
			},
			packagecli.RenderInfo,
		),
		packagecli.CommandDependents: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.DependentsRequest, error) {
				return packagecli.ParseDependentsRequest(args)
			},
			func(service packageService, req packagecli.DependentsRequest) (packagecli.DependentsResponse, error) {
				item, report, err := service.Dependents(req.Name)
				if err != nil {
					return packagecli.DependentsResponse{}, rootcli.UsageErrorf("package dependents", "%s", err.Error())
				}
				return packagecli.DependentsResponse{
					PackageName: item.Name,
					Dependents:  report.Dependents,
					Issues:      report.Issues,
				}, nil
			},
			packagecli.RenderDependents,
		),
		packagecli.CommandBuild: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.RunRequest, error) {
				return packagecli.ParseRunRequest("build", args)
			},
			func(service packageService, req packagecli.RunRequest) (packagecli.RunResponse, error) {
				resp, err := service.Build(req.Name)
				if err != nil {
					return packagecli.RunResponse{}, err
				}
				return packagecli.RunResponse(resp), nil
			},
			packagecli.RenderRun,
		),
		packagecli.CommandGenerate: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.RunRequest, error) {
				return packagecli.ParseRunRequest("generate", args)
			},
			func(service packageService, req packagecli.RunRequest) (packagecli.RunResponse, error) {
				resp, err := service.Generate(req.Name)
				if err != nil {
					return packagecli.RunResponse{}, err
				}
				return packagecli.RunResponse(resp), nil
			},
			packagecli.RenderRun,
		),
		packagecli.CommandTest: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.RunRequest, error) {
				return packagecli.ParseRunRequest("test", args)
			},
			func(service packageService, req packagecli.RunRequest) (packagecli.RunResponse, error) {
				resp, err := service.Test(req.Name)
				if err != nil {
					return packagecli.RunResponse{}, err
				}
				return packagecli.RunResponse(resp), nil
			},
			packagecli.RenderRun,
		),
		packagecli.CommandRefresh: rootcli.BindService(deps.Stdout, deps.OutputFormat, newServiceFor(deps),
			func(ctx C, args []string) (packagecli.RefreshRequest, error) {
				return packagecli.ParseRefreshRequest(args)
			},
			func(service packageService, req packagecli.RefreshRequest) (packagecli.RefreshResponse, error) {
				resp, err := service.Refresh(packageapp.RefreshRequest{
					PackageName: req.Name,
					Target:      req.Target,
					NoRestart:   req.NoRestart,
					Interactive: req.Interactive,
				})
				if err != nil {
					return packagecli.RefreshResponse{}, err
				}
				return packagecli.RefreshResponse{
					PackageName: resp.PackageName,
					Items:       toCLIRefreshItems(resp.Items),
				}, nil
			},
			packagecli.RenderRefresh,
		),
	}

	return commandtree.BindSpecs(packagecli.CommandSpecs(), handlerMap)
}

func newServiceFor[C any](deps HandlerDeps[C]) func(C, cliout.Format) (packageService, error) {
	return func(ctx C, format cliout.Format) (packageService, error) {
		return packageService{Service: newService(deps, ctx, format), format: format}, nil
	}
}

func newService[C any](deps HandlerDeps[C], ctx C, format cliout.Format) packageapp.Service {
	stdout := deps.Stdout(ctx)
	stderr := deps.Stderr(ctx)
	if format != cliout.FormatHuman {
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
		TestGenieRunner: func(target string, stdout, stderr io.Writer) error {
			if deps.TestGenieRunner == nil {
				return nil
			}
			return deps.TestGenieRunner(ctx, target, stdout, stderr)
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
