package planshandlers

import (
	"fmt"
	"io"

	planapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/planscli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type HandlerDeps[C any] struct {
	Stdout       func(C) io.Writer
	Stdin        func(C) io.Reader
	Root         func(C) string
	Home         func(C) (string, error)
	OutputFormat func(C) (cliout.Format, error)
}

func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	handlers := commandtree.BuildHandlerMap(buildCommandTable(deps))
	return func(ctx C, args []string) error {
		return rootcli.RunSubcommandSet(ctx, args, planscli.RenderCommandHelp, "plans", handlers, deps.Stdout)
	}
}

func buildCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.Handler[C]] {
	handlerMap := map[planscli.CommandID]rootcli.Handler[C]{
		planscli.CommandAdd: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.AddRequest, error) { return planscli.ParseAddRequest(args) },
			func(ctx C, req planscli.AddRequest) (cliout.Format, planapp.AddOutput, error) {
				if !req.Stdin {
					return "", planapp.AddOutput{}, rootcli.UsageErrorf("plans add", "plans add requires --stdin")
				}
				content, err := io.ReadAll(deps.Stdin(ctx))
				if err != nil {
					return "", planapp.AddOutput{}, fmt.Errorf("read stdin: %w", err)
				}
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.AddOutput{}, err
				}
				resp, err := service.Add(planscli.ToAppAdd(req, string(content)))
				return format, resp, err
			},
			planscli.RenderAdd,
		),
		planscli.CommandList: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.ListRequest, error) { return planscli.ParseListRequest(args) },
			func(ctx C, req planscli.ListRequest) (cliout.Format, planapp.ListOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.ListOutput{}, err
				}
				resp, err := service.List(planapp.ListRequest{Repo: req.Repo, IncludeAll: req.AllRepos, IncludeArchived: req.IncludeArchived})
				return format, resp, err
			},
			planscli.RenderList,
		),
		planscli.CommandShow: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.RefRequest, error) {
				return planscli.ParseRefRequest("plans show", planscli.CommandShow, args)
			},
			func(ctx C, req planscli.RefRequest) (cliout.Format, planapp.ShowOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.ShowOutput{}, err
				}
				resp, err := service.Show(planapp.ShowRequest{Ref: req.Ref, Repo: req.Repo})
				return format, resp, err
			},
			planscli.RenderShow,
		),
		planscli.CommandPath: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.RefRequest, error) {
				return planscli.ParseRefRequest("plans path", planscli.CommandPath, args)
			},
			func(ctx C, req planscli.RefRequest) (cliout.Format, planapp.PathOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.PathOutput{}, err
				}
				resp, err := service.Path(planapp.ShowRequest{Ref: req.Ref, Repo: req.Repo})
				return format, resp, err
			},
			planscli.RenderPath,
		),
		planscli.CommandArchive: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.RefRequest, error) {
				return planscli.ParseRefRequest("plans archive", planscli.CommandArchive, args)
			},
			func(ctx C, req planscli.RefRequest) (cliout.Format, planapp.ArchiveOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.ArchiveOutput{}, err
				}
				resp, err := service.Archive(planapp.ArchiveRequest{Ref: req.Ref, Repo: req.Repo})
				return format, resp, err
			},
			planscli.RenderArchive,
		),
		planscli.CommandImport: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.ImportRequest, error) { return planscli.ParseImportRequest(args) },
			func(ctx C, req planscli.ImportRequest) (cliout.Format, planapp.ImportOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.ImportOutput{}, err
				}
				resp, err := service.Import(planapp.ImportRequest{
					Path:         req.Path,
					Title:        req.Title,
					Slug:         req.Slug,
					Repo:         req.Repo,
					DeleteSource: req.DeleteSource,
				})
				return format, resp, err
			},
			planscli.RenderImport,
		),
		planscli.CommandExport: rootcli.BindGlobalCommand(deps.Stdout,
			func(ctx C, args []string) (planscli.ExportRequest, error) { return planscli.ParseExportRequest(args) },
			func(ctx C, req planscli.ExportRequest) (cliout.Format, planapp.ExportOutput, error) {
				format, service, err := newService(deps, ctx)
				if err != nil {
					return "", planapp.ExportOutput{}, err
				}
				resp, err := service.Export(planapp.ExportRequest{Ref: req.Ref, Repo: req.Repo, To: req.To})
				return format, resp, err
			},
			planscli.RenderExport,
		),
	}
	return commandtree.BindSpecs(planscli.CommandSpecs(), handlerMap)
}

func newService[C any](deps HandlerDeps[C], ctx C) (cliout.Format, planapp.Service, error) {
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", planapp.Service{}, err
	}
	home, err := deps.Home(ctx)
	if err != nil {
		return "", planapp.Service{}, err
	}
	return format, planapp.Service{Root: deps.Root(ctx), Home: home}, nil
}
