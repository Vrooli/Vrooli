package domains

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func storageGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	group := support.SubcommandGroup("storage", "Inspect and reclaim Agent Manager's SQLite storage", deps.Storage,
		[2]string{"status", "Show file, free pages, WAL, and budget standing"},
		[2]string{"reclaim", "Return free pages online in bounded batches"},
		[2]string{"compact", "Fenced one-time rewrite into incremental auto-vacuum"})
	jsonFlag := cliapp.Flag{Name: "json", Bool: true, LocalOnly: true, Description: "Print the owner projection"}
	for i := range group.Subcommands {
		command := &group.Subcommands[i]
		command.Args.Flags = []cliapp.Flag{jsonFlag}
		switch command.Name {
		case "status":
			command.Usage = "agent-manager storage status [--json]"
			command.HelpText = "Reads page, freelist, WAL, and declared-budget standing with the next owner action. Never scans pages."
		case "reclaim":
			command.Usage = "agent-manager storage reclaim [--dry-run] [--json]"
			command.HelpText = "Returns freelist pages to the filesystem without an outage when the file is in incremental auto-vacuum mode. A none-mode file only reports that compaction is required."
			command.Args.Flags = append(command.Args.Flags, cliapp.Flag{Name: "dry-run", Bool: true, LocalOnly: true, Description: "Preview without changing the file"})
		case "compact":
			command.Usage = "agent-manager storage compact --reason <text> [--local-owner] [--no-wait] [--timeout 45m] [--json]"
			command.HelpText = "Requires a closed and drained maintenance fence (maintenance begin, then drain). Rewrites the database with VACUUM into incremental auto-vacuum mode, verifies quick_check, and reports bytes before and after. Background writers pause for the rewrite."
			command.Args.Flags = append(command.Args.Flags,
				cliapp.Flag{Name: "reason", Required: true, Description: "Required nonblank reason, 1-512 bytes"},
				cliapp.Flag{Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local owner exchange; unavailable inside identified agent runs"},
				cliapp.Flag{Name: "no-wait", Bool: true, LocalOnly: true, Description: "Return once the compaction starts"},
				cliapp.Flag{Name: "timeout", LocalOnly: true, Description: "How long to wait for the result (default 45m)"})
		}
	}
	return group
}
