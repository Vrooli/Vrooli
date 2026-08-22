package resourcehandlers

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/hostinventory"
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

func buildResourceSchemaCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	resourceSchemaCommandTable := buildResourceSchemaCommandTable(deps)
	return commandtree.BuildHandlerMap(resourceSchemaCommandTable)
}

func buildResourceAcquisitionCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	return commandtree.BuildHandlerMap(buildResourceAcquisitionCommandTable(deps))
}

func buildResourceAccelerationCommandHandlers[C any](deps HandlerDeps[C]) map[string]rootcli.ResourceHandler[C] {
	return commandtree.BuildHandlerMap(buildResourceAccelerationCommandTable(deps))
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
		resourcecli.CommandInstall:   installResourceHandler(deps),
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
		resourcecli.CommandUpstreamCheck: bindResourceCommand(deps,
			parseResourceUpstreamCheckRequest,
			func(ctx C, controller *resources.Controller, req resourcecli.UpstreamCheckRequest) (cliout.Format, upstreamcheck.AggregateReport, error) {
				_ = controller
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return "", upstreamcheck.AggregateReport{}, err
				}
				agg, ok := runUpstreamCheck(req)
				if !ok {
					return "", upstreamcheck.AggregateReport{}, rootcli.UsageErrorf(
						"resource upstream-check",
						"unknown coding-agent resource %q (known: %s)", req.Name, knownCodingAgentResourceNames())
				}
				return format, agg, nil
			},
			resourcecli.WriteUpstreamCheck,
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
		resourcecli.CommandSchema: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceSchemaHelp, "resource schema", buildResourceSchemaCommandHandlers(deps), deps.Stdout)
		},
		resourcecli.CommandAcquisition: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceAcquisitionHelp, "resource acquisition", buildResourceAcquisitionCommandHandlers(deps), deps.Stdout)
		},
		resourcecli.CommandAcceleration: func(ctx C, controller *resources.Controller, args []string) error {
			return runResourceSubcommandSet(ctx, controller, args, showResourceAccelerationHelp, "resource acceleration", buildResourceAccelerationCommandHandlers(deps), deps.Stdout)
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

func buildResourceAccelerationCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	return []commandtree.Spec[rootcli.ResourceHandler[C]]{
		{
			Name:    string(resourcecli.AccelerationCommandExplain),
			Summary: "Explain a resource's declared backends, host readiness and observed placement",
			Args:    commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}},
			Handler: func(ctx C, controller *resources.Controller, args []string) error {
				if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
					return rootcli.UsageErrorf("resource acceleration explain", "resource acceleration explain requires exactly one resource name")
				}
				name := strings.TrimSpace(args[0])
				manifest, err := controller.LoadManifest(filepath.Join(controller.Root, "resources", name, "resource.json"))
				if err != nil {
					return err
				}
				explanation, err := controller.ExplainAcceleration(context.Background(), manifest)
				if err != nil {
					return err
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return err
				}
				return writeAccelerationExplanation(deps.Stdout(ctx), format, explanation)
			},
		},
	}
}

// writeAccelerationExplanation renders one table an operator can read top to
// bottom: what was declared, what the host reaches, what was selected and why,
// where the process landed, and the command that repairs a failing row.
func writeAccelerationExplanation(w io.Writer, format cliout.Format, result resources.AccelerationExplanation) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, result)
	}
	_, _ = fmt.Fprintf(w, "Resource: %s\n", result.Resource)
	if len(result.Declared) == 0 {
		_, _ = fmt.Fprintln(w, "Declared backends: none (this resource does no accelerated work)")
	} else {
		_, _ = fmt.Fprintf(w, "Declared backends: %s (require: %s)\n", strings.Join(result.Declared, ", "), result.Require)
	}
	_, _ = fmt.Fprintf(w, "Host backends:     %s\n", strings.Join(result.HostBackends, ", "))
	_, _ = fmt.Fprintln(w, "Host facts:")
	for _, key := range sortedKeys(result.Facts) {
		_, _ = fmt.Fprintf(w, "- %s=%s\n", key, result.Facts[key])
	}
	if len(result.Considered) > 0 {
		_, _ = fmt.Fprintln(w, "Backend verdicts:")
		for _, verdict := range result.Considered {
			mark := "unreachable"
			if verdict.Ready {
				mark = "ready"
			}
			_, _ = fmt.Fprintf(w, "- %-7s %-11s %s\n", verdict.Backend, mark, verdict.Reason)
		}
		_, _ = fmt.Fprintf(w, "Selected: %s\n", result.Selected)
	}
	if result.Placement != nil {
		_, _ = fmt.Fprintf(w, "Placement: declared=%s observed=%s state=%s\n", result.Placement.Declared, orUnknown(string(result.Placement.Observed)), result.Placement.State)
		_, _ = fmt.Fprintf(w, "  target: %s\n", result.Placement.Target)
		_, _ = fmt.Fprintf(w, "  reason: %s\n", result.Placement.Reason)
	} else if len(result.Declared) > 0 {
		_, _ = fmt.Fprintln(w, "Placement: not running, so there is nothing to verify")
	}
	if result.Claim != nil {
		_, _ = fmt.Fprintf(w, "Claim: %s preferred=%d floor=%d priority=%s\n", result.Claim.ResourceKind, result.Claim.PreferredBytes, result.Claim.FloorBytes, result.Claim.Priority)
	}
	if result.Remediation != "" {
		_, _ = fmt.Fprintf(w, "Remediation: %s\n", result.Remediation)
	}
	return nil
}

func orUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func buildResourceAcquisitionCommandTable[C any](deps HandlerDeps[C]) []commandtree.Spec[rootcli.ResourceHandler[C]] {
	return []commandtree.Spec[rootcli.ResourceHandler[C]]{
		{
			Name:    string(resourcecli.AcquisitionCommandExplain),
			Summary: "Explain host-fact acquisition target selection",
			Args:    commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}},
			Handler: func(ctx C, controller *resources.Controller, args []string) error {
				if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
					return rootcli.UsageErrorf("resource acquisition explain", "resource acquisition explain requires exactly one resource name")
				}
				name := strings.TrimSpace(args[0])
				manifest, err := controller.LoadManifest(filepath.Join(controller.Root, "resources", name, "resource.json"))
				if err != nil {
					return err
				}
				snapshot, err := hostinventory.Collect(context.Background())
				if err != nil {
					return fmt.Errorf("collect host facts: %w", err)
				}
				facts := snapshot.AcceleratorFacts()
				result := resourceAcquisitionExplanation{
					Resource:       name,
					Facts:          facts,
					FactProvenance: acquisitionFactProvenance(snapshot),
				}
				if manifest.ManagedService != nil && manifest.ManagedService.Acquisition != nil {
					result.Acquisition = manifest.ManagedService.Acquisition
					result.Resolution = manifest.ManagedService.Acquisition.Explain(facts)
					if verdict, ok := controller.StagedArtifactClosure(manifest); ok {
						result.Closure = &verdict
					}
				}
				format, err := deps.OutputFormat(ctx)
				if err != nil {
					return err
				}
				return writeResourceAcquisitionExplanation(deps.Stdout(ctx), format, result)
			},
		},
		{
			Name:    string(resourcecli.AcquisitionCommandPrune),
			Summary: "Remove superseded managed-resource artifact versions",
			Args:    commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name"}}},
			Handler: func(ctx C, controller *resources.Controller, args []string) error {
				if len(args) > 1 {
					return rootcli.UsageErrorf("resource acquisition prune", "resource acquisition prune accepts at most one resource name")
				}
				names := make([]string, 0, 1)
				if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
					names = append(names, strings.TrimSpace(args[0]))
				} else {
					items, err := controller.Discover()
					if err != nil {
						return err
					}
					for _, item := range items {
						manifest, err := controller.LoadManifest(filepath.Join(controller.Root, "resources", item.Name, "resource.json"))
						if err == nil && manifest.ManagedService != nil && manifest.ManagedService.Acquisition != nil {
							names = append(names, item.Name)
						}
					}
				}
				removed := 0
				for _, name := range names {
					count, err := controller.PruneManagedServiceArtifacts(name)
					if err != nil {
						return err
					}
					removed += count
				}
				_, err := fmt.Fprintf(deps.Stdout(ctx), "pruned %d superseded artifact version(s)\n", removed)
				return err
			},
		},
	}
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

func showResourceSchemaHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource schema <subcommand> [options]", "Resource Schema", resourcecli.SchemaCommandSpecs())
}

func showResourceAcquisitionHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource acquisition <subcommand> [options]", "Resource Acquisition", resourcecli.AcquisitionCommandSpecs())
}

func showResourceAccelerationHelp(w io.Writer) {
	resourcecli.RenderCommandHelp(w, "", "vrooli resource acceleration <subcommand> [options]", "Resource Acceleration", resourcecli.AccelerationCommandSpecs())
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

type resourceAcquisitionExplanation struct {
	Resource       string                              `json:"resource"`
	Facts          map[string]string                   `json:"facts"`
	FactProvenance map[string]hostinventory.Provenance `json:"fact_provenance,omitempty"`
	Acquisition    *binaryfetch.Acquisition            `json:"acquisition,omitempty"`
	Resolution     binaryfetch.ResolutionExplanation   `json:"resolution,omitempty"`
	// Closure is the runtime-closure verdict for the artifact staged on this
	// host, when one is staged. A digest-correct artifact whose libraries do
	// not resolve is the failure this field exists to make visible.
	Closure *resources.ClosureVerdict `json:"runtime_closure,omitempty"`
}

// acquisitionFactProvenance answers "where did each fact that selected this
// artifact come from". Every emitted accelerator fact must appear, otherwise an
// operator reading `vrooli resource acquisition explain` sees a selection with
// no way to check the input that drove it.
func acquisitionFactProvenance(snapshot hostinventory.Snapshot) map[string]hostinventory.Provenance {
	result := map[string]hostinventory.Provenance{}
	for _, key := range []string{"os", "arch"} {
		if provenance, ok := snapshot.FieldProvenance[key]; ok {
			result[key] = provenance
		}
	}
	for key, provenance := range snapshot.AcceleratorFactProvenance() {
		result[key] = provenance
	}
	return result
}

func writeResourceAcquisitionExplanation(w io.Writer, format cliout.Format, result resourceAcquisitionExplanation) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, result)
	}
	_, _ = fmt.Fprintf(w, "Resource: %s\n", result.Resource)
	_, _ = fmt.Fprintln(w, "Facts:")
	for key, value := range result.Facts {
		_, _ = fmt.Fprintf(w, "- %s=%s\n", key, value)
	}
	if result.Acquisition == nil {
		_, _ = fmt.Fprintln(w, "Acquisition: not declared")
		return nil
	}
	_, _ = fmt.Fprintln(w, "Candidates:")
	for _, candidate := range result.Resolution.Candidates {
		selection := ""
		if candidate.Selected {
			selection = " [selected]"
		}
		_, _ = fmt.Fprintf(w, "- #%d when=%v: %s%s\n", candidate.Index, candidate.When, candidate.Reason, selection)
	}
	if result.Closure != nil {
		_, _ = fmt.Fprintf(w, "Runtime closure: %s\n", result.Closure.State)
		if len(result.Closure.Unresolved) > 0 {
			_, _ = fmt.Fprintf(w, "  unresolved: %s\n", strings.Join(result.Closure.Unresolved, ", "))
		}
		if result.Closure.Reason != "" {
			_, _ = fmt.Fprintf(w, "  reason: %s\n", result.Closure.Reason)
		}
	}
	if result.Resolution.Selected >= 0 {
		_, _ = fmt.Fprintf(w, "Selected candidate: #%d\n", result.Resolution.Selected)
	} else {
		_, _ = fmt.Fprintln(w, "Selected candidate: none")
	}
	return nil
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

// installResourceHandler is install plus the one flag that distinguishes
// "stage this if it is missing" from "the host changed, replace what is
// staged". Without the second, a needs_reacquire resource has a diagnosis and
// no cure.
func installResourceHandler[C any](deps HandlerDeps[C]) rootcli.ResourceHandler[C] {
	base := singleResourceControlHandler(deps, "install")
	return func(ctx C, controller *resources.Controller, args []string) error {
		names := make([]string, 0, len(args))
		reacquire := false
		for _, arg := range args {
			if strings.TrimSpace(arg) == "--reacquire" {
				reacquire = true
				continue
			}
			names = append(names, arg)
		}
		if !reacquire {
			return base(ctx, controller, names)
		}
		if len(names) != 1 {
			return rootcli.UsageErrorf("resource install", "resource install --reacquire requires exactly one resource name")
		}
		if err := controller.DiscardStagedArtifact(names[0], deps.Stderr(ctx)); err != nil {
			return err
		}
		return base(ctx, controller, names)
	}
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
		// Admit before launching the resource so a declared companion can find
		// the manifest-derived claim on its first heartbeat. Admission is
		// advisory and non-blocking; controller.Run remains the authority for
		// the actual lifecycle action.
		admitResourceCapacityCLI(controller.Root, args[0], action, deps.Stderr(ctx))
		if err := controller.Run(args[0], []string{action}, deps.Stdout(ctx), deps.Stderr(ctx)); err != nil {
			return err
		}
		return nil
	}
}

// actionsAdmittingCapacity enumerates the standalone CLI actions that bring a
// resource onto the host's contended capacity (GPU VRAM / RAM) and so should
// record an advisory capacity claim.
var actionsAdmittingCapacity = map[string]struct{}{
	"start":   {},
	"restart": {},
}

// admitResourceCapacityCLI runs the advisory capacity broker admission after a
// standalone `vrooli resource start|restart` brings a resource up, mirroring the
// lifecycle Runner's admitResourceCapacity for the dependency-start path. It is
// ALWAYS advisory and non-fatal: a resource with no `capacity` block, disabled
// enforcement, or any operational error is a silent no-op (AdmitResource returns
// before touching the ledger), so the command's behaviour and exit code are
// unchanged. Only warnings surface (to stderr); a clean claim is silent.
func admitResourceCapacityCLI(root, name, action string, stderr io.Writer) {
	if _, ok := actionsAdmittingCapacity[action]; !ok {
		return
	}
	result, err := capacity.AdmitResource(context.Background(), capacity.AdmitOptions{
		Root:         root,
		ResourceName: name,
	})
	if err != nil {
		fmt.Fprintf(stderr, "capacity admission skipped (advisory): %v\n", err)
		return
	}
	if result.Skipped {
		return
	}
	for _, warn := range result.Verdict.Warnings {
		fmt.Fprintf(stderr, "capacity admission warning (%s): %s\n", name, warn)
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
	return rootcli.UsageErrorf(command, "%s", err.Error())
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
		return resourcecli.WriteValidationReportJSON(w, report)
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
