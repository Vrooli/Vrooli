package resourcecli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	CommandID            string
	AcquisitionCommandID string
	BlueprintCommandID   string
	ArchiveCommandID     string
	SchemaCommandID      string
)

const (
	CommandList                  CommandID = "list"
	CommandCensus                CommandID = "census"
	CommandScaffold              CommandID = "scaffold"
	CommandCLISync               CommandID = "cli-sync"
	CommandStatus                CommandID = "status"
	CommandValidate              CommandID = "validate"
	CommandInstall               CommandID = "install"
	CommandUninstall             CommandID = "uninstall"
	CommandStart                 CommandID = "start"
	CommandRestart               CommandID = "restart"
	CommandStartAll              CommandID = "start-all"
	CommandStop                  CommandID = "stop"
	CommandStopAll               CommandID = "stop-all"
	CommandLogs                  CommandID = "logs"
	CommandEnable                CommandID = "enable"
	CommandDisable               CommandID = "disable"
	CommandInfo                  CommandID = "info"
	CommandUpstreamCheck         CommandID = "upstream-check"
	CommandUpdate                CommandID = "update"
	CommandUpdateCheck           CommandID = "update-check"
	CommandDeprecate             CommandID = "deprecate"
	CommandListDeprecated        CommandID = "list-deprecated"
	CommandArchiveToBlueprint    CommandID = "archive-to-blueprint"
	CommandListBlueprintArchived CommandID = "list-blueprint-archived"
	CommandRestore               CommandID = "restore"
	CommandRestoreBlueprint      CommandID = "restore-blueprint"
	CommandArchive               CommandID = "archive"
	CommandBlueprint             CommandID = "blueprint"
	CommandSchema                CommandID = "schema"
	CommandAcquisition           CommandID = "acquisition"
	CommandAcceleration          CommandID = "acceleration"
)

const (
	AcquisitionCommandRuntimeProbe     AcquisitionCommandID = "runtime-probe"
	AcquisitionCommandExplain          AcquisitionCommandID = "explain"
	AcquisitionCommandInspectData      AcquisitionCommandID = "inspect-data"
	AcquisitionCommandComposeCandidate AcquisitionCommandID = "compose-candidate"
	AcquisitionCommandPrune            AcquisitionCommandID = "prune"
	AcquisitionCommandModelCache       AcquisitionCommandID = "model-cache"
	AcquisitionCommandDataGroup        AcquisitionCommandID = "data"
)

// AccelerationCommandID names a subcommand of `vrooli resource acceleration`.
type AccelerationCommandID string

const (
	AccelerationCommandExplain AccelerationCommandID = "explain"
)

// AccelerationCommandSpecs declares the accelerator inspection surface.
func AccelerationCommandSpecs() []commandtree.Spec[AccelerationCommandID] {
	return []commandtree.Spec[AccelerationCommandID]{
		{Name: string(AccelerationCommandExplain), Summary: "Explain a resource's declared backends, host readiness and observed placement", Args: nameArgSchema("name"), Handler: AccelerationCommandExplain},
	}
}

const (
	BlueprintCommandList     BlueprintCommandID = "list"
	BlueprintCommandInfo     BlueprintCommandID = "info"
	BlueprintCommandSearch   BlueprintCommandID = "search"
	BlueprintCommandValidate BlueprintCommandID = "validate"
)

const (
	ArchiveCommandGC           ArchiveCommandID = "gc"
	ArchiveCommandGCBlueprints ArchiveCommandID = "gc-blueprints"
)

const (
	SchemaCommandValidate SchemaCommandID = "validate"
	SchemaCommandSync     SchemaCommandID = "sync"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandList), Summary: "List discovered resources", Handler: CommandList},
		{Name: string(CommandCensus), Summary: "Report resource declarations and installed CLI state", Handler: CommandCensus},
		{Name: string(CommandScaffold), Summary: "Generate a resource scaffold", Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--name", ValueName: "name", Description: "Resource name"}, {Name: "--driver", ValueName: "archetype", Description: "Resource archetype"}}}, Handler: CommandScaffold},
		{Name: string(CommandCLISync), Summary: "Reconcile declared resource CLIs", Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--dry-run", Description: "Report actions without installing"}}}, Handler: CommandCLISync},
		{
			Name:    string(CommandStatus),
			Summary: "Show resource status",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name"}},
				Options: []commandtree.OptionArg{
					{Name: "--fast", Description: "Use fast status probes"},
					{Name: "--no-fast", Description: "Disable fast status probes"},
					{Name: "--health-only", Description: "Observe process/health without auditing current artifact bytes; reports artifact_verification=not_evaluated"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandStatus,
		},
		{
			Name:    string(CommandValidate),
			Summary: "Validate resource manifests and export contracts",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name"}},
			},
			Handler: CommandValidate,
		},
		{Name: string(CommandInstall), Summary: "Install a resource", Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}},
			Options: []commandtree.OptionArg{{
				Name:        "--reacquire",
				Description: "Discard the staged artifact and re-resolve, re-download and re-verify it under the host's current facts. Use this when status reports needs_reacquire.",
			}},
		}, Handler: CommandInstall},
		{Name: string(CommandUninstall), Summary: "Uninstall a resource", Args: nameArgSchema("name"), Handler: CommandUninstall},
		{Name: string(CommandStart), Summary: "Start a resource", Args: resourceStartArgs(), Handler: CommandStart},
		{Name: string(CommandRestart), Summary: "Restart a resource", Args: nameArgSchema("name"), Handler: CommandRestart},
		{Name: string(CommandStartAll), Summary: "Start all enabled resources", Handler: CommandStartAll},
		{Name: string(CommandStop), Summary: "Stop a resource", Args: nameArgSchema("name"), Handler: CommandStop},
		{Name: string(CommandStopAll), Summary: "Stop all running resources", Handler: CommandStopAll},
		{Name: string(CommandLogs), Summary: "Show a bounded tail of managed resource logs (default 200, maximum 1000 lines and 1 MiB)", Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}},
			Options:     []commandtree.OptionArg{{Name: "--tail", ValueName: "lines", Description: "Number of trailing lines to show (1..1000; output also capped at 1 MiB)"}},
		}, Handler: CommandLogs},
		{Name: string(CommandEnable), Summary: "Enable a resource in configuration", Args: nameArgSchema("name"), Handler: CommandEnable},
		{Name: string(CommandDisable), Summary: "Disable a resource in configuration", Args: nameArgSchema("name"), Handler: CommandDisable},
		{Name: string(CommandInfo), Summary: "Show resource metadata", Args: nameArgSchema("name"), Handler: CommandInfo},
		{
			Name:    string(CommandUpstreamCheck),
			Summary: "Check resource acquisition references and coding-agent versions (read-only)",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name"}},
				Options: []commandtree.OptionArg{
					{Name: "--all", Description: "Check every resource acquisition reference and coding-agent CLI"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandUpstreamCheck,
		},
		{
			Name:    string(CommandUpdate),
			Summary: "Update a coding agent or prepare and reconcile a managed-service update",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--check", Description: "Report state and eligibility without changing anything"},
					{Name: "--on-behalf-of", ValueName: "subject", Description: "The verified operator a trusted caller acts for, recorded as the actor. An agent's claim is recorded as a claim."},
					{Name: "--action", ValueName: "action", Description: "Coding agents: install, refresh-catalog, use-managed-copy, link-managed-copy, rollback, doctor, preview-cleanup, uninstall. Managed services: prepare (default), rollback, status, resume, abort"},
					{Name: "--cleanup-plan", ValueName: "id", Description: "Exact preview identity required for uninstall"},
					{Name: "--request", ValueName: "id", Description: "Managed service request identity; reuse after an interrupted call"},
					{Name: "--rollback-of", ValueName: "request", Description: "Qualified terminal receipt whose retained previous image is selected for rollback"},
					{Name: "--version", ValueName: "version", Description: "Exact staged managed-service release for prepare"},
					{Name: "--sha256", ValueName: "digest", Description: "Exact staged managed-service checksum for prepare"},
					{Name: "--retry", Description: "Explicit managed-service retry with a new request after a failed release"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandUpdate,
		},
		{
			Name:    string(CommandUpdateCheck),
			Summary: "Report a resource's update state and eligibility (read-only)",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandUpdateCheck,
		},
		{Name: string(CommandDeprecate), Summary: "Deprecate a resource", Args: nameArgSchema("name"), Handler: CommandDeprecate},
		{Name: string(CommandListDeprecated), Summary: "List deprecated resources", Handler: CommandListDeprecated},
		{Name: string(CommandArchiveToBlueprint), Summary: "Archive a resource into blueprint-only state", Args: nameArgSchema("name"), Handler: CommandArchiveToBlueprint},
		{Name: string(CommandListBlueprintArchived), Summary: "List blueprint-archived resources", Handler: CommandListBlueprintArchived},
		{Name: string(CommandRestore), Summary: "Restore a deprecated resource", Args: nameArgSchema("name"), Handler: CommandRestore},
		{Name: string(CommandRestoreBlueprint), Summary: "Restore a blueprint-archived resource", Args: nameArgSchema("name"), Handler: CommandRestoreBlueprint},
		{Name: string(CommandArchive), Summary: "Manage resource archive maintenance", Handler: CommandArchive},
		{Name: string(CommandBlueprint), Summary: "Inspect resource blueprints", Handler: CommandBlueprint},
		{Name: string(CommandSchema), Summary: "Manage resource-derived schema artifacts", Handler: CommandSchema},
		{Name: string(CommandAcquisition), Summary: "Inspect declared resource acquisition", Handler: CommandAcquisition},
		{Name: string(CommandAcceleration), Summary: "Inspect accelerator declaration, readiness and placement", Handler: CommandAcceleration},
	}
}

func AcquisitionCommandSpecs() []commandtree.Spec[AcquisitionCommandID] {
	return []commandtree.Spec[AcquisitionCommandID]{
		{Name: string(AcquisitionCommandRuntimeProbe), Summary: "Qualify managed Python imports in an owned offline read-only runtime", Args: AcquisitionRuntimeProbeArgs(), Handler: AcquisitionCommandRuntimeProbe},
		{Name: string(AcquisitionCommandComposeCandidate), Summary: "Prepare an uninstalled composed artifact candidate and actual digest", Args: AcquisitionComposeCandidateArgs(), Handler: AcquisitionCommandComposeCandidate},
		{Name: string(AcquisitionCommandInspectData), Summary: "Verify bounded pinned upstream metadata and report its SHA-256 (never admit execution)", Args: AcquisitionInspectDataArgs(), Handler: AcquisitionCommandInspectData},
		{Name: string(AcquisitionCommandExplain), Summary: "Explain host-fact acquisition target selection", Args: nameArgSchema("name"), Handler: AcquisitionCommandExplain},
		{Name: string(AcquisitionCommandPrune), Summary: "Preview or apply managed-resource artifact version retention", Args: AcquisitionPruneArgs(), Handler: AcquisitionCommandPrune},
		{Name: string(AcquisitionCommandDataGroup), Summary: "Acquire one explicitly declared data group", Args: AcquisitionDataGroupArgs(), Handler: AcquisitionCommandDataGroup},
		{Name: string(AcquisitionCommandModelCache), Summary: "Preview or apply owner-fenced model-cache retention", Args: AcquisitionModelCacheArgs(), Handler: AcquisitionCommandModelCache},
	}
}

func BlueprintCommandSpecs() []commandtree.Spec[BlueprintCommandID] {
	return []commandtree.Spec[BlueprintCommandID]{
		{Name: string(BlueprintCommandList), Summary: "List resource blueprints", Handler: BlueprintCommandList},
		{Name: string(BlueprintCommandInfo), Summary: "Show a resource blueprint", Args: nameArgSchema("name"), Handler: BlueprintCommandInfo},
		{Name: string(BlueprintCommandSearch), Summary: "Search resource blueprints", Args: nameArgSchema("query"), Handler: BlueprintCommandSearch},
		{Name: string(BlueprintCommandValidate), Summary: "Validate blueprint metadata", Handler: BlueprintCommandValidate},
	}
}

func ArchiveCommandSpecs() []commandtree.Spec[ArchiveCommandID] {
	return []commandtree.Spec[ArchiveCommandID]{
		{Name: string(ArchiveCommandGC), Summary: "Purge expired deprecated-resource archives", Handler: ArchiveCommandGC},
		{Name: string(ArchiveCommandGCBlueprints), Summary: "Purge expired blueprint-resource archives", Handler: ArchiveCommandGCBlueprints},
	}
}

func SchemaCommandSpecs() []commandtree.Spec[SchemaCommandID] {
	return []commandtree.Spec[SchemaCommandID]{
		{Name: string(SchemaCommandValidate), Summary: "Validate generated resource schema artifacts and scenario references", Handler: SchemaCommandValidate},
		{Name: string(SchemaCommandSync), Summary: "Regenerate resource schema artifacts from manifests", Handler: SchemaCommandSync},
	}
}

func RenderCommandHelp[ID any](w io.Writer, title, usage, defaultGroup string, specs []commandtree.Spec[ID]) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        title,
		Usage:        usage,
		DefaultGroup: defaultGroup,
	}, specs)
}

func nameArgSchema(name string) commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: name, Required: true}},
	}
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown resource command spec: " + string(id))
}

func blueprintCommandSpec(id BlueprintCommandID) commandtree.Spec[BlueprintCommandID] {
	for _, spec := range BlueprintCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown resource blueprint command spec: " + string(id))
}

func archiveCommandSpec(id ArchiveCommandID) commandtree.Spec[ArchiveCommandID] {
	for _, spec := range ArchiveCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown resource archive command spec: " + string(id))
}

func schemaCommandSpec(id SchemaCommandID) commandtree.Spec[SchemaCommandID] {
	for _, spec := range SchemaCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown resource schema command spec: " + string(id))
}

func CommandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli resource "+spec.Name, spec)
}

func BlueprintCommandHelpText(id BlueprintCommandID) string {
	spec := blueprintCommandSpec(id)
	return commandtree.SpecHelpText("", "vrooli resource blueprint "+spec.Name, spec)
}

func ArchiveCommandHelpText(id ArchiveCommandID) string {
	spec := archiveCommandSpec(id)
	return commandtree.SpecHelpText("", "vrooli resource archive "+spec.Name, spec)
}

func SchemaCommandHelpText(id SchemaCommandID) string {
	spec := schemaCommandSpec(id)
	return commandtree.SpecHelpText("", "vrooli resource schema "+spec.Name, spec)
}

func AcquisitionCommandHelpText(id AcquisitionCommandID) string {
	for _, spec := range AcquisitionCommandSpecs() {
		if spec.Handler == id {
			return commandtree.SpecHelpText("", "vrooli resource acquisition "+spec.Name, spec)
		}
	}
	panic("unknown resource acquisition command spec: " + string(id))
}

// AcquisitionInspectDataArgs is shared by actual dispatch and maintained help.
func AcquisitionInspectDataArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}, Options: []commandtree.OptionArg{
		{Name: "--url", ValueName: "https-url", Description: "Pinned upstream metadata URL"},
		{Name: "--git-blob-sha1", ValueName: "40hex", Description: "Expected Git object identity"},
		{Name: "--max-bytes", ValueName: "bytes", Description: "Explicit positive bound, at most1048576"},
		{Name: "--out", ValueName: "new-file", Description: "New non-executable file; never overwrite"},
		{Name: "--logical-id", ValueName: "identity", Description: "Resource-declared credential identity"},
		{Name: "--field", ValueName: "field", Description: "Resource-declared credential field"},
	}}
}

func AcquisitionComposeCandidateArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}, Options: []commandtree.OptionArg{{Name: "--out", ValueName: "new-directory", Description: "New candidate bundle; never overwrite"}, {Name: "--offline", Description: "Require verified cached external parts; refuse network acquisition"}}}
}

func AcquisitionModelCacheArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}, Options: []commandtree.OptionArg{
		{Name: "--entry", ValueName: "entry", Description: "Declared storage entry (default models)"},
		{Name: "--apply-preview", ValueName: "file", Description: "Apply one exact saved preview; never stop a resource"},
		{Name: "--pin", ValueName: "true|false", Description: "Pin or unpin the resource's model bundle"},
	}}
}

func AcquisitionPruneArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name"}}, Options: []commandtree.OptionArg{
		commandtree.JSONOption(),
		{Name: "--keep-previous", ValueName: "n", Description: "Previous versions kept for rollback beside the active one (default 1)"},
		{Name: "--apply", Description: "Remove the versions the preview marks superseded; without it nothing is deleted"},
	}}
}

func AcquisitionDataGroupArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}, Options: []commandtree.OptionArg{{Name: "--group", ValueName: "group", Description: "Exact declared data group; never start a resource"}}}
}

func resourceStartArgs() commandtree.ArgSchema {
	args := nameArgSchema("name")
	args.Options = append(args.Options, commandtree.OptionArg{Name: "--no-reclaim", Description: "Require manifest-derived admission using already-free capacity; refuse startup if unavailable"})
	return args
}

func AcquisitionRuntimeProbeArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "name", Required: true}}, Options: []commandtree.OptionArg{{Name: "--imports", ValueName: "modules", Description: "Comma-separated import-only Python modules"}, {Name: "--out", ValueName: "directory", Description: "New receipt directory outside the artifact"}}}
}
