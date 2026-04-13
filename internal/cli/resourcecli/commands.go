package resourcecli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	CommandID          string
	BlueprintCommandID string
	ArchiveCommandID   string
	TemplateCommandID  string
)

const (
	CommandList                  CommandID = "list"
	CommandStatus                CommandID = "status"
	CommandValidate              CommandID = "validate"
	CommandInstall               CommandID = "install"
	CommandStart                 CommandID = "start"
	CommandStartAll              CommandID = "start-all"
	CommandStop                  CommandID = "stop"
	CommandStopAll               CommandID = "stop-all"
	CommandEnable                CommandID = "enable"
	CommandDisable               CommandID = "disable"
	CommandInfo                  CommandID = "info"
	CommandDeprecate             CommandID = "deprecate"
	CommandListDeprecated        CommandID = "list-deprecated"
	CommandArchiveToBlueprint    CommandID = "archive-to-blueprint"
	CommandListBlueprintArchived CommandID = "list-blueprint-archived"
	CommandRestore               CommandID = "restore"
	CommandRestoreBlueprint      CommandID = "restore-blueprint"
	CommandArchive               CommandID = "archive"
	CommandBlueprint             CommandID = "blueprint"
	CommandTemplate              CommandID = "template"
)

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
	TemplateCommandList     TemplateCommandID = "list"
	TemplateCommandShow     TemplateCommandID = "show"
	TemplateCommandValidate TemplateCommandID = "validate"
	TemplateCommandGenerate TemplateCommandID = "generate"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandList), Summary: "List discovered resources", Handler: CommandList},
		{Name: string(CommandStatus), Summary: "Show resource status", Handler: CommandStatus},
		{Name: string(CommandValidate), Summary: "Validate resource manifests and export contracts", Handler: CommandValidate},
		{Name: string(CommandInstall), Summary: "Install a resource", Handler: CommandInstall},
		{Name: string(CommandStart), Summary: "Start a resource", Handler: CommandStart},
		{Name: string(CommandStartAll), Summary: "Start all enabled resources", Handler: CommandStartAll},
		{Name: string(CommandStop), Summary: "Stop a resource", Handler: CommandStop},
		{Name: string(CommandStopAll), Summary: "Stop all running resources", Handler: CommandStopAll},
		{Name: string(CommandEnable), Summary: "Enable a resource in configuration", Handler: CommandEnable},
		{Name: string(CommandDisable), Summary: "Disable a resource in configuration", Handler: CommandDisable},
		{Name: string(CommandInfo), Summary: "Show resource metadata", Handler: CommandInfo},
		{Name: string(CommandDeprecate), Summary: "Deprecate a resource", Handler: CommandDeprecate},
		{Name: string(CommandListDeprecated), Summary: "List deprecated resources", Handler: CommandListDeprecated},
		{Name: string(CommandArchiveToBlueprint), Summary: "Archive a resource into blueprint-only state", Handler: CommandArchiveToBlueprint},
		{Name: string(CommandListBlueprintArchived), Summary: "List blueprint-archived resources", Handler: CommandListBlueprintArchived},
		{Name: string(CommandRestore), Summary: "Restore a deprecated resource", Handler: CommandRestore},
		{Name: string(CommandRestoreBlueprint), Summary: "Restore a blueprint-archived resource", Handler: CommandRestoreBlueprint},
		{Name: string(CommandArchive), Summary: "Manage resource archive maintenance", Handler: CommandArchive},
		{Name: string(CommandBlueprint), Summary: "Inspect resource blueprints", Handler: CommandBlueprint},
		{Name: string(CommandTemplate), Summary: "Manage resource templates", Handler: CommandTemplate},
	}
}

func BlueprintCommandSpecs() []commandtree.Spec[BlueprintCommandID] {
	return []commandtree.Spec[BlueprintCommandID]{
		{Name: string(BlueprintCommandList), Summary: "List resource blueprints", Handler: BlueprintCommandList},
		{Name: string(BlueprintCommandInfo), Summary: "Show a resource blueprint", Handler: BlueprintCommandInfo},
		{Name: string(BlueprintCommandSearch), Summary: "Search resource blueprints", Handler: BlueprintCommandSearch},
		{Name: string(BlueprintCommandValidate), Summary: "Validate blueprint metadata", Handler: BlueprintCommandValidate},
	}
}

func ArchiveCommandSpecs() []commandtree.Spec[ArchiveCommandID] {
	return []commandtree.Spec[ArchiveCommandID]{
		{Name: string(ArchiveCommandGC), Summary: "Purge expired deprecated-resource archives", Handler: ArchiveCommandGC},
		{Name: string(ArchiveCommandGCBlueprints), Summary: "Purge expired blueprint-resource archives", Handler: ArchiveCommandGCBlueprints},
	}
}

func TemplateCommandSpecs() []commandtree.Spec[TemplateCommandID] {
	return []commandtree.Spec[TemplateCommandID]{
		{Name: string(TemplateCommandList), Summary: "List resource templates", Handler: TemplateCommandList},
		{Name: string(TemplateCommandShow), Summary: "Show template details", Handler: TemplateCommandShow},
		{Name: string(TemplateCommandValidate), Summary: "Validate template manifests", Handler: TemplateCommandValidate},
		{Name: string(TemplateCommandGenerate), Summary: "Generate files from a template", Handler: TemplateCommandGenerate},
	}
}

func RenderCommandHelp[ID any](w io.Writer, title, usage, defaultGroup string, specs []commandtree.Spec[ID]) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        title,
		Usage:        usage,
		DefaultGroup: defaultGroup,
	}, specs)
}
