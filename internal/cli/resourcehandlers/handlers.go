package resourcehandlers

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqrun"
	"github.com/vrooli/vrooli/internal/resources"
)

type HandlerDeps[C any] struct {
	Stdout             func(C) io.Writer
	Stderr             func(C) io.Writer
	Globals            func(C) rootcli.GlobalOptions
	OutputFormat       func(C) (cliout.Format, error)
	EnsureCLI          func(C, string) error
	ResourceController func(C) (*resources.Controller, error)
}

var TimeNowForArchiveGC = func() time.Time {
	return time.Now().UTC()
}

// enforceHostRequirementsFn runs hostreqrun.Enforce before mutating resource
// operations (install/start/restart). Tests may override it with a stub.
var enforceHostRequirementsFn = hostreqrun.Enforce

// actionsRequiringHostRequirements enumerates CLI actions that must ensure
// declared tools/safeguards are present before the resource runs.
var actionsRequiringHostRequirements = map[string]struct{}{
	"install": {},
	"start":   {},
	"restart": {},
}

func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	commandHandlers := buildResourceCommandHandlers(deps)
	return func(ctx C, args []string) error {
		if len(args) == 0 || (len(args) == 1 && topcli.ListOrHelpWithoutRoot(args)) {
			showResourceHelp(deps.Stdout(ctx))
			return nil
		}
		controller, err := deps.ResourceController(ctx)
		if err != nil {
			return err
		}
		return runResourceSubcommandSet(ctx, controller, args, showResourceHelp, "resource", commandHandlers, deps.Stdout)
	}
}

func buildResourceCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceCommandTable := buildResourceCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceCommandTable)
}

func buildResourceBlueprintCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceBlueprintCommandTable := buildResourceBlueprintCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceBlueprintCommandTable)
}

func buildResourceArchiveCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceArchiveCommandTable := buildResourceArchiveCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceArchiveCommandTable)
}

func buildResourceTemplateCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceTemplateCommandTable := buildResourceTemplateCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceTemplateCommandTable)
}

func buildResourceSchemaCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceSchemaCommandTable := buildResourceSchemaCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceSchemaCommandTable)
}

func buildResourceCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	handlerMap := map[resourcecli.CommandID]rootcli.ResourceHandler[C]{
		resourcecli.CommandList: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceListRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resourceListResponse, error) {
				_ = req
				resp, err := newResourceCommandService(deps, ctx, controller).List()
				if err != nil {
					return "", resourceListResponse{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resourceListResponse{}, err
				}
				return format, resourceListResponse{
					Items:    resp.Items,
					Failures: append([]discovery.Failure(nil), resp.Failures...),
				}, nil
			},
			renderResourceListResponse,
		),
		resourcecli.CommandStatus: bindResourceCommand(deps,
			func(args []string) (resourcecli.StatusRequest, error) { return parseResourceStatusRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.StatusRequest) (cliout.Format, resourceStatusResponse, error) {
				if err := ensureNamedResourceCLI(deps, ctx, req.Name); err != nil {
					return "", resourceStatusResponse{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resourceStatusResponse{}, err
				}
				resp, err := newResourceCommandService(deps, ctx, controller).Status(req.Name, req.Fast)
				if err != nil {
					return "", resourceStatusResponse{}, err
				}
				if resp.Item != nil {
					return format, resourceStatusResponse{Item: resp.Item}, nil
				}
				return format, resourceStatusResponse{
					Items:    resp.Items,
					Failures: append([]discovery.Failure(nil), resp.Failures...),
				}, nil
			},
			renderResourceStatusResponse,
		),
		resourcecli.CommandValidate: bindResourceCommand(deps,
			func(args []string) (resourcecli.ValidateRequest, error) { return parseResourceValidateRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.ValidateRequest) (cliout.Format, resources.ResourceValidationReport, error) {
				report, err := newResourceCommandService(deps, ctx, controller).Validate(req.Name)
				if err != nil {
					return "", resources.ResourceValidationReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.ResourceValidationReport{}, err
				}
				return format, report, nil
			},
			renderResourceValidateResponse,
		),
		resourcecli.CommandInstall:   singleResourceControlHandler(deps, "install"),
		resourcecli.CommandUninstall: singleResourceControlHandler(deps, "uninstall"),
		resourcecli.CommandStart:     singleResourceControlHandler(deps, "start"),
		resourcecli.CommandRestart:   singleResourceControlHandler(deps, "restart"),
		resourcecli.CommandStop:      singleResourceControlHandler(deps, "stop"),
		resourcecli.CommandLogs:      singleResourceControlHandler(deps, "logs"),
		resourcecli.CommandStartAll: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceStartAllRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resourceapp.ControlReportResponse, error) {
				_ = req
				report, err := newResourceCommandService(deps, ctx, controller).StartAll()
				if err != nil {
					return "", resourceapp.ControlReportResponse{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resourceapp.ControlReportResponse{}, err
				}
				return format, report, nil
			},
			renderResourceControlReportResponse,
		),
		resourcecli.CommandStopAll: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceStopAllRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resourceapp.ControlReportResponse, error) {
				_ = req
				report, err := newResourceCommandService(deps, ctx, controller).StopAll()
				if err != nil {
					return "", resourceapp.ControlReportResponse{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resourceapp.ControlReportResponse{}, err
				}
				return format, report, nil
			},
			renderResourceControlReportResponse,
		),
		resourcecli.CommandEnable:  resourceToggleHandler(deps, true),
		resourcecli.CommandDisable: resourceToggleHandler(deps, false),
		resourcecli.CommandInfo: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) { return parseResourceInfoRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.Status, error) {
				if err := ensureNamedResourceCLI(deps, ctx, req.Name); err != nil {
					return "", resources.Status{}, err
				}
				item, err := newResourceCommandService(deps, ctx, controller).Info(req.Name)
				if err != nil {
					return "", resources.Status{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.Status{}, err
				}
				return format, item, nil
			},
			resourcecli.WriteInfo,
		),
		resourcecli.CommandDeprecate: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) { return parseResourceDeprecateRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.DeprecationReport, error) {
				report, err := newResourceCommandService(deps, ctx, controller).Deprecate(req.Name)
				if err != nil {
					return "", resources.DeprecationReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.DeprecationReport{}, err
				}
				return format, report, nil
			},
			resourcecli.WriteDeprecationReport,
		),
		resourcecli.CommandListDeprecated: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) {
				return parseResourceListDeprecatedRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, []resources.DeprecatedResource, error) {
				_ = req
				items, err := newResourceCommandService(deps, ctx, controller).ListDeprecated()
				if err != nil {
					return "", nil, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", nil, err
				}
				return format, items, nil
			},
			resourcecli.WriteDeprecatedList,
		),
		resourcecli.CommandArchiveToBlueprint: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) {
				return parseResourceArchiveToBlueprintRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.BlueprintArchiveReport, error) {
				report, err := newResourceCommandService(deps, ctx, controller).ArchiveToBlueprint(req.Name)
				if err != nil {
					return "", resources.BlueprintArchiveReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.BlueprintArchiveReport{}, err
				}
				return format, report, nil
			},
			resourcecli.WriteBlueprintArchiveReport,
		),
		resourcecli.CommandListBlueprintArchived: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) {
				return parseResourceListBlueprintArchivedRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, []resources.BlueprintArchivedResource, error) {
				_ = req
				items, err := newResourceCommandService(deps, ctx, controller).ListBlueprintArchived()
				if err != nil {
					return "", nil, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", nil, err
				}
				return format, items, nil
			},
			resourcecli.WriteBlueprintArchivedList,
		),
		resourcecli.CommandRestore: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) { return parseResourceRestoreRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.RestoreReport, error) {
				report, err := newResourceCommandService(deps, ctx, controller).Restore(req.Name)
				if err != nil {
					return "", resources.RestoreReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.RestoreReport{}, err
				}
				return format, report, nil
			},
			resourcecli.WriteRestoreReport,
		),
		resourcecli.CommandRestoreBlueprint: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) {
				return parseResourceRestoreBlueprintRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.BlueprintRestoreReport, error) {
				report, err := newResourceCommandService(deps, ctx, controller).RestoreBlueprint(req.Name)
				if err != nil {
					return "", resources.BlueprintRestoreReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.BlueprintRestoreReport{}, err
				}
				return format, report, nil
			},
			resourcecli.WriteBlueprintRestoreReport,
		),
		resourcecli.CommandArchive: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceArchiveHelp, "resource archive", buildResourceArchiveCommandHandlers(deps), deps.Stdout)
		},
		resourcecli.CommandBlueprint: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceBlueprintHelp, "resource blueprint", buildResourceBlueprintCommandHandlers(deps), deps.Stdout)
		},
		resourcecli.CommandTemplate: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceTemplateHelp, "resource template", buildResourceTemplateCommandHandlers(deps), deps.Stdout)
		},
		resourcecli.CommandSchema: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceSchemaHelp, "resource schema", buildResourceSchemaCommandHandlers(deps), deps.Stdout)
		},
	}
	return commandtree.BindSpecs(resourcecli.CommandSpecs(), handlerMap)
}

func buildResourceBlueprintCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	handlerMap := map[resourcecli.BlueprintCommandID]rootcli.ResourceHandler[C]{
		resourcecli.BlueprintCommandList: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceBlueprintListRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, []resources.Blueprint, error) {
				_ = req
				items, err := newResourceCommandService(deps, ctx, controller).BlueprintList()
				if err != nil {
					return "", nil, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", nil, err
				}
				return format, items, nil
			},
			resourcecli.WriteBlueprintList,
		),
		resourcecli.BlueprintCommandInfo: bindResourceCommand(deps,
			func(args []string) (resourcecli.NameRequest, error) { return parseResourceBlueprintInfoRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NameRequest) (cliout.Format, resources.Blueprint, error) {
				item, err := newResourceCommandService(deps, ctx, controller).BlueprintInfo(req.Name)
				if err != nil {
					return "", resources.Blueprint{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.Blueprint{}, err
				}
				return format, item, nil
			},
			resourcecli.WriteBlueprintInfo,
		),
		resourcecli.BlueprintCommandSearch: bindResourceCommand(deps,
			func(args []string) (resourcecli.BlueprintSearchRequest, error) {
				return parseResourceBlueprintSearchRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.BlueprintSearchRequest) (cliout.Format, resourceBlueprintSearchResponse, error) {
				items, err := newResourceCommandService(deps, ctx, controller).BlueprintSearch(req.Query)
				if err != nil {
					return "", resourceBlueprintSearchResponse{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resourceBlueprintSearchResponse{}, err
				}
				return format, resourceBlueprintSearchResponse{Query: req.Query, Items: items}, nil
			},
			renderResourceBlueprintSearchResponse,
		),
		resourcecli.BlueprintCommandValidate: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) {
				return parseResourceBlueprintValidateRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resources.BlueprintValidationReport, error) {
				_ = req
				report, err := newResourceCommandService(deps, ctx, controller).BlueprintValidate()
				if err != nil {
					return "", resources.BlueprintValidationReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.BlueprintValidationReport{}, err
				}
				return format, report, nil
			},
			resourcecli.WriteBlueprintValidationReport,
		),
	}
	return commandtree.BindSpecs(resourcecli.BlueprintCommandSpecs(), handlerMap)
}

func buildResourceArchiveCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	handlerMap := map[resourcecli.ArchiveCommandID]rootcli.ResourceHandler[C]{
		resourcecli.ArchiveCommandGC: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceArchiveGCRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
				_ = req
				report, err := controller.GarbageCollectDeprecatedArchives(TimeNowForArchiveGC())
				if err != nil {
					return "", resources.ArchiveGCReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.ArchiveGCReport{}, err
				}
				return format, report, nil
			},
			func(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
				return resourcecli.WriteArchiveGCReport(w, format, report, "deprecated resource")
			},
		),
		resourcecli.ArchiveCommandGCBlueprints: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) {
				return parseResourceArchiveBlueprintGCRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resources.ArchiveGCReport, error) {
				_ = req
				report, err := controller.GarbageCollectBlueprintArchives(TimeNowForArchiveGC())
				if err != nil {
					return "", resources.ArchiveGCReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.ArchiveGCReport{}, err
				}
				return format, report, nil
			},
			func(w io.Writer, format cliout.Format, report resources.ArchiveGCReport) error {
				return resourcecli.WriteArchiveGCReport(w, format, report, "blueprint resource")
			},
		),
	}
	return commandtree.BindSpecs(resourcecli.ArchiveCommandSpecs(), handlerMap)
}

func buildResourceTemplateCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	handlerMap := map[resourcecli.TemplateCommandID]rootcli.ResourceHandler[C]{
		resourcecli.TemplateCommandList: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) { return parseResourceTemplateListRequest(args) },
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, []resources.ResourceTemplateInfo, error) {
				_ = req
				items, err := newResourceCommandService(deps, ctx, controller).TemplateList()
				if err != nil {
					return "", nil, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", nil, err
				}
				return format, items, nil
			},
			renderResourceTemplateListResponse,
		),
		resourcecli.TemplateCommandShow: bindResourceCommand(deps,
			func(args []string) (resourcecli.TemplateNameRequest, error) {
				return parseResourceTemplateShowRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.TemplateNameRequest) (cliout.Format, resources.ResourceTemplateInfo, error) {
				item, err := newResourceCommandService(deps, ctx, controller).TemplateShow(req.Name)
				if err != nil {
					return "", resources.ResourceTemplateInfo{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.ResourceTemplateInfo{}, err
				}
				return format, item, nil
			},
			renderResourceTemplateShowResponse,
		),
		resourcecli.TemplateCommandValidate: bindResourceCommand(deps,
			func(args []string) (resourcecli.NoArgsRequest, error) {
				return parseResourceTemplateValidateRequest(args)
			},
			func(ctx C, controller *resources.Controller, req resourcecli.NoArgsRequest) (cliout.Format, resources.ResourceTemplateValidationReport, error) {
				_ = req
				report, err := newResourceCommandService(deps, ctx, controller).TemplateValidate()
				if err != nil {
					return "", resources.ResourceTemplateValidationReport{}, err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", resources.ResourceTemplateValidationReport{}, err
				}
				return format, report, nil
			},
			renderResourceTemplateValidateResponse,
		),
		resourcecli.TemplateCommandGenerate: func(ctx C, controller *resources.Controller, args []string) error {
			return bindResourceCommand(deps,
				func(args []string) (resourcecli.TemplateGenerateOptions, error) {
					return parseResourceTemplateGenerateRequest(controller, deps.Stderr(ctx), args)
				},
				func(ctx C, controller *resources.Controller, req resourcecli.TemplateGenerateOptions) (cliout.Format, resources.ResourceTemplateGenerateReport, error) {
					report, err := newResourceCommandService(deps, ctx, controller).TemplateGenerate(resources.ResourceTemplateGenerateRequest{
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
					format, err := deps.OutputFormat(ctx)
					if err != nil {
						return "", resources.ResourceTemplateGenerateReport{}, err
					}
					return format, report, nil
				},
				renderResourceTemplateGenerateResponse,
			)(ctx, controller, args)
		},
	}
	return commandtree.BindSpecs(resourcecli.TemplateCommandSpecs(), handlerMap)
}

func buildResourceSchemaCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	handlerMap := map[resourcecli.SchemaCommandID]rootcli.ResourceHandler[C]{
		resourcecli.SchemaCommandValidate: func(ctx C, controller *resources.Controller, args []string) error {
			if _, err := parseResourceSchemaValidateRequest(args); err != nil {
				return err
			}
			report, err := newResourceCommandService(deps, ctx, controller).SchemaValidate()
			if err != nil {
				return err
			}
			format, err := deps.OutputFormat(ctx)
			if err != nil {
				return err
			}
			if err := resourcecli.WriteSchemaValidationReport(deps.Stdout(ctx), format, report); err != nil {
				return err
			}
			if !report.Passed {
				return rootcli.ExitCodeError{Code: 1, Silent_: true}
			}
			return nil
		},
		resourcecli.SchemaCommandSync: func(ctx C, controller *resources.Controller, args []string) error {
			if _, err := parseResourceSchemaSyncRequest(args); err != nil {
				return err
			}
			report, err := newResourceCommandService(deps, ctx, controller).SchemaSync()
			if err != nil {
				return err
			}
			format, err := deps.OutputFormat(ctx)
			if err != nil {
				return err
			}
			if err := resourcecli.WriteSchemaSyncReport(deps.Stdout(ctx), format, report); err != nil {
				return err
			}
			if !report.Passed {
				return rootcli.ExitCodeError{Code: 1, Silent_: true}
			}
			return nil
		},
	}
	return commandtree.BindSpecs(resourcecli.SchemaCommandSpecs(), handlerMap)
}

func showResourceHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource <subcommand> [options]", "Resource Management", resourcecli.CommandSpecs())
}

func showResourceBlueprintHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource blueprint <subcommand> [options]", "Resource Blueprints", resourcecli.BlueprintCommandSpecs())
}

func showResourceArchiveHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource archive <subcommand> [options]", "Resource Archive", resourcecli.ArchiveCommandSpecs())
}

func showResourceTemplateHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource template <subcommand> [options]", "Resource Templates", resourcecli.TemplateCommandSpecs())
}

func showResourceSchemaHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource schema <subcommand> [options]", "Resource Schema", resourcecli.SchemaCommandSpecs())
}

type resourceStatusResponse struct {
	Item     *resources.Status
	Items    []resources.Status
	Failures []discovery.Failure
}

type resourceListResponse struct {
	Items    []resources.Resource
	Failures []discovery.Failure
}

type resourceBlueprintSearchResponse struct {
	Query string
	Items []resources.Blueprint
}

func newResourceCommandService[C any](deps HandlerDeps[C], ctx C, controller *resources.Controller) resourceapp.Service {
	return resourceapp.Service{
		Resources: controller,
		Stdout:    deps.Stdout(ctx),
		Stderr:    deps.Stderr(ctx),
	}
}

func bindResourceCommand[C any, Req any, Resp any](
	deps HandlerDeps[C],
	parse func(args []string) (Req, error),
	run func(ctx C, controller *resources.Controller, req Req) (cliout.Format, Resp, error),
	render func(w io.Writer, format cliout.Format, resp Resp) error,
) rootcli.ResourceHandler[C] {
	return rootcli.BindResourceCommand(deps.Stdout,
		func(ctx C, args []string) (Req, error) {
			return parse(args)
		},
		func(controller *resources.Controller, ctx C, req Req) (cliout.Format, Resp, error) {
			return run(ctx, controller, req)
		},
		render,
	)
}

func runResourceSubcommandSet[C any](
	ctx C,
	controller *resources.Controller,
	args []string,
	usage func(io.Writer),
	command string,
	handlers map[string]rootcli.ResourceHandler[C],
	stdout func(C) io.Writer,
) error {
	if len(args) == 0 || (len(args) == 1 && commandtree.WantsHelp(args)) {
		usage(stdout(ctx))
		return nil
	}
	handler, ok := handlers[commandtree.NormalizeName(args[0])]
	if !ok {
		return rootcli.UsageErrorf(command, "unknown %s command: %s", command, args[0])
	}
	return handler(ctx, controller, args[1:])
}

func singleResourceControlHandler[C any](deps HandlerDeps[C], action string) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		if len(args) != 1 {
			return rootcli.UsageErrorf("resource "+action, "resource %s requires exactly one resource name", action)
		}
		if err := ensureNamedResourceCLI(deps, ctx, args[0]); err != nil {
			return err
		}
		if err := enforceResourceHostRequirements(ctx, deps, controller, args[0], action); err != nil {
			return err
		}
		return controller.Run(args[0], []string{action}, deps.Stdout(ctx), deps.Stderr(ctx))
	}
}

// enforceResourceHostRequirements installs the declared hostTools and
// hostSafeguards for the target resource before install/start/restart actions
// so the resource is not invoked with missing tools. It is a no-op for other
// actions and for resources that declare nothing.
func enforceResourceHostRequirements[C any](ctx C, deps HandlerDeps[C], controller *resources.Controller, name, action string) error {
	if _, ok := actionsRequiringHostRequirements[action]; !ok {
		return nil
	}
	if enforceHostRequirementsFn == nil {
		return nil
	}
	if _, err := enforceHostRequirementsFn(hostreqrun.Options{
		Root:        controller.Root,
		Home:        controller.Home,
		Environment: hostreq.NormalizeEnvironment(controller.Environment),
		When:        "develop",
		Resources:   name,
		Scenarios:   "none",
		AutoInstall: true,
		Stdout:      deps.Stdout(ctx),
		Stderr:      deps.Stderr(ctx),
		Label:       "resource:" + name,
	}); err != nil {
		return err
	}
	return nil
}

func ensureNamedResourceCLI[C any](deps HandlerDeps[C], ctx C, name string) error {
	if deps.EnsureCLI == nil {
		return nil
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return deps.EnsureCLI(ctx, name)
}

func resourceToggleHandler[C any](deps HandlerDeps[C], enabled bool) rootcli.ResourceHandler[C] {
	return func(ctx C, controller *resources.Controller, args []string) error {
		action := "enable"
		if !enabled {
			action = "disable"
		}
		if len(args) != 1 {
			return rootcli.UsageErrorf("resource "+action, "resource %s requires exactly one resource name", action)
		}
		if err := controller.SetEnabled(args[0], enabled); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(deps.Stdout(ctx), "Updated %s: enabled=%t\n", args[0], enabled)
		return nil
	}
}

func parseResourceListRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseListRequest(args)
	return req, mapResourceParseError("resource list", err)
}

func parseResourceValidateRequest(args []string) (resourcecli.ValidateRequest, error) {
	req, err := resourcecli.ParseValidateRequest(args)
	return req, mapResourceParseError("resource validate", err)
}

func parseResourceStatusRequest(args []string) (resourcecli.StatusRequest, error) {
	req, err := resourcecli.ParseStatusRequest(args)
	return req, mapResourceParseError("resource status", err)
}

func parseResourceInfoRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseInfoRequest(args)
	return req, mapResourceParseError("resource info", err)
}

func parseResourceDeprecateRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseDeprecateRequest(args)
	return req, mapResourceParseError("resource deprecate", err)
}

func parseResourceListDeprecatedRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseListDeprecatedRequest(args)
	return req, mapResourceParseError("resource list-deprecated", err)
}

func parseResourceRestoreRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseRestoreRequest(args)
	return req, mapResourceParseError("resource restore", err)
}

func parseResourceArchiveToBlueprintRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseArchiveToBlueprintRequest(args)
	return req, mapResourceParseError("resource archive-to-blueprint", err)
}

func parseResourceListBlueprintArchivedRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseListBlueprintArchivedRequest(args)
	return req, mapResourceParseError("resource list-blueprint-archived", err)
}

func parseResourceRestoreBlueprintRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseRestoreBlueprintRequest(args)
	return req, mapResourceParseError("resource restore-blueprint", err)
}

func parseResourceArchiveGCRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseArchiveGCRequest(args)
	return req, mapResourceParseError("resource archive gc", err)
}

func parseResourceArchiveBlueprintGCRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseArchiveBlueprintGCRequest(args)
	return req, mapResourceParseError("resource archive gc-blueprints", err)
}

func parseResourceBlueprintListRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseBlueprintListRequest(args)
	return req, mapResourceParseError("resource blueprint list", err)
}

func parseResourceBlueprintInfoRequest(args []string) (resourcecli.NameRequest, error) {
	req, err := resourcecli.ParseBlueprintInfoRequest(args)
	return req, mapResourceParseError("resource blueprint info", err)
}

func parseResourceBlueprintSearchRequest(args []string) (resourcecli.BlueprintSearchRequest, error) {
	req, err := resourcecli.ParseBlueprintSearchRequest(args)
	return req, mapResourceParseError("resource blueprint search", err)
}

func parseResourceBlueprintValidateRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseBlueprintValidateRequest(args)
	return req, mapResourceParseError("resource blueprint validate", err)
}

func parseResourceStartAllRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseStartAllRequest(args)
	return req, mapResourceParseError("resource start-all", err)
}

func parseResourceStopAllRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseStopAllRequest(args)
	return req, mapResourceParseError("resource stop-all", err)
}

func parseResourceTemplateListRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseTemplateListRequest(args)
	return req, mapResourceParseError("resource template list", err)
}

func parseResourceTemplateShowRequest(args []string) (resourcecli.TemplateNameRequest, error) {
	req, err := resourcecli.ParseTemplateShowRequest(args)
	return req, mapResourceParseError("resource template show", err)
}

func parseResourceTemplateValidateRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseTemplateValidateRequest(args)
	return req, mapResourceParseError("resource template validate", err)
}

func parseResourceTemplateGenerateRequest(controller *resources.Controller, stderr io.Writer, args []string) (resourcecli.TemplateGenerateOptions, error) {
	req, err := resourcecli.ParseTemplateGenerateRequest(args, stderr, func(req resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateInfo, error) {
		return controller.ResolveTemplateGenerationRequest(req)
	})
	if err != nil {
		return resourcecli.TemplateGenerateOptions{}, mapResourceParseError("resource template generate", err)
	}
	return req, nil
}

func parseResourceSchemaValidateRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseSchemaValidateRequest(args)
	return req, mapResourceParseError("resource schema validate", err)
}

func parseResourceSchemaSyncRequest(args []string) (resourcecli.NoArgsRequest, error) {
	req, err := resourcecli.ParseSchemaSyncRequest(args)
	return req, mapResourceParseError("resource schema sync", err)
}

func mapResourceParseError(command string, err error) error {
	if err == nil {
		return nil
	}
	if helpErr, ok := err.(interface{ HelpText() string }); ok {
		return rootcli.CommandHelpOnly(helpErr.HelpText())
	}
	return rootcli.UsageErrorf(command, err.Error())
}

func renderResourceStatusResponse(w io.Writer, format cliout.Format, resp resourceStatusResponse) error {
	if resp.Item != nil {
		return resourcecli.WriteStatus(w, format, *resp.Item)
	}
	return resourcecli.WriteStatuses(w, format, resp.Items, resp.Failures)
}

func renderResourceListResponse(w io.Writer, format cliout.Format, resp resourceListResponse) error {
	return resourcecli.WriteList(w, format, resp.Items, resp.Failures)
}

func renderResourceValidateResponse(w io.Writer, format cliout.Format, report resources.ResourceValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteFieldsWithSuccess(w, report.Passed, map[string]any{"report": report})
	}
	status := "passed"
	if !report.Passed {
		status = "failed"
	}
	if _, err := io.WriteString(w, "Resource validation "+status+"\n"); err != nil {
		return err
	}
	for _, item := range report.Items {
		if len(item.Issues) == 0 {
			continue
		}
		if _, err := io.WriteString(w, "- "+item.Name+"\n"); err != nil {
			return err
		}
		for _, issue := range item.Issues {
			if _, err := io.WriteString(w, "  ["+issue.Severity+"] "+issue.Message+"\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderResourceControlReportResponse(w io.Writer, format cliout.Format, resp resourceapp.ControlReportResponse) error {
	switch {
	case resp.Start != nil:
		return resourcecli.WriteControlReport(w, format, "report", "Started", resp.Start, resp.Start.Started, resp.Start.Failed)
	case resp.Stop != nil:
		return resourcecli.WriteControlReport(w, format, "report", "Stopped", resp.Stop, resp.Stop.Stopped, resp.Stop.Failed)
	default:
		return nil
	}
}

func renderResourceBlueprintSearchResponse(w io.Writer, format cliout.Format, resp resourceBlueprintSearchResponse) error {
	return resourcecli.WriteBlueprintSearch(w, format, resp.Query, resp.Items)
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

func renderResourceTemplateValidateResponse(w io.Writer, format cliout.Format, report resources.ResourceTemplateValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteSuccessJSON(w, "report", report)
	}
	_, _ = fmt.Fprintf(w, "Validated %d resource templates\n", report.Count)
	return nil
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

func writeResourceTemplateVarTable(w io.Writer, title string, vars map[string]resources.ResourceTemplateVar) {
	if len(vars) == 0 {
		return
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s:\n", title)
	for _, key := range keys {
		item := vars[key]
		line := fmt.Sprintf("  - %s (--%s)", key, item.Flag)
		if item.Description != "" {
			line += ": " + item.Description
		}
		if item.Default != "" {
			line += " [default: " + item.Default + "]"
		}
		_, _ = fmt.Fprintln(w, line)
	}
}

func formatResourceTemplateRequiredVars(vars map[string]resources.ResourceTemplateVar) string {
	if len(vars) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s (--%s)", key, vars[key].Flag))
	}
	return strings.Join(parts, ", ")
}

func formatResourceTemplateRequiredFlags(vars map[string]resources.ResourceTemplateVar) string {
	if len(vars) == 0 {
		return ""
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(" --%s <%s>", vars[key].Flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}

func writeResourceTemplateValues(w io.Writer, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	_, _ = fmt.Fprintln(w, "Applied variables:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "  - %s=%s\n", key, values[key])
	}
}
