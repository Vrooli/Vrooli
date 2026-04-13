package scenariocli

const (
	StartHelpText            = "Usage: vrooli scenario start <name> [name2...] [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]"
	RestartHelpText          = "Usage: vrooli scenario restart <name> [--path <path>] [--best-effort] [--clean-stale] [--open] [--json]"
	ListHelpText             = "Usage: vrooli scenario list [--json] [--include-ports]"
	InfoHelpText             = "Usage: vrooli scenario info <name> [--json]"
	StatusHelpText           = "Usage: vrooli scenario status [name] [--json]"
	SetupHelpText            = "Usage: vrooli scenario setup <name> [--path <path>]"
	TestHelpText             = "Usage: vrooli scenario test <name> [phase|all|e2e] [--allow-skip-missing-runtime] [--manage-runtime]"
	StartAllHelpText         = "Usage: vrooli scenario start-all [--json]"
	StopAllHelpText          = "Usage: vrooli scenario stop-all [--json]"
	PortHelpText             = "Usage: vrooli scenario port <scenario-name> [<port-name>] [--json]"
	OpenHelpText             = "Usage: vrooli scenario open <scenario-name> [--port <name>] [--print-url]"
	RequirementsHelpTextBody = "Usage: vrooli scenario requirements <subcommand> [options]\n\nSubcommands:\n  report <name> [options]          Generate requirement coverage summary\n  validate <name> [--quiet]        Validate requirement files\n  sync <name>                      Sync requirement statuses from local evidence\n  manual-log <name> <req> [opts]   Record manual validation evidence\n  snapshot <name>                  Show latest requirements sync snapshot\n  lint-prd <name> [--json]         Check PRD to requirements mapping\n  phase <name> --phase <phase>     Inspect validations for a single phase\n  init <name> [options]            Scaffold a requirements registry\n"
	HealFromSandboxHelpText  = "Usage: vrooli scenario heal-from-sandbox [--merged-path <path>] [--dry-run]"
	TemplateCommandHelpText  = "Scenario Template Commands:\n  vrooli scenario template list\n  vrooli scenario template show <template>\n  vrooli scenario generate <template> [options]"
	TemplateGenerateHelpText = "Usage: vrooli scenario generate <template> --id <slug> --display-name <name> --description <text> [options]\nOptions:\n  --dest <path>         Destination directory (defaults to scenarios/<id>)\n  --var KEY=VALUE       Additional placeholder override (repeatable)\n  --force               Overwrite destination if it already exists\n  --dry-run             Print the planned actions without writing files\n  --run-hooks           Execute template post hooks after generation"
)
