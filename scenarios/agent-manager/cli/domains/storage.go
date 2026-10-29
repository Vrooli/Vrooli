package domains

import (
	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func storageGroup(deps support.Dependencies) cliapp.SubcommandGroup {
	group := support.SubcommandGroup("storage", "Inspect and reclaim Agent Manager's SQLite storage and the shared effort evidence home", deps.Storage,
		[2]string{"status", "Show file, free pages, WAL, and budget standing"},
		[2]string{"reclaim", "Return free pages online in bounded batches"},
		[2]string{"compact", "Fenced one-time rewrite into incremental auto-vacuum"},
		[2]string{"evidence", "Classify ~/.vrooli/evidence into regenerable, referenced and unknown bytes per effort (dry run)"},
		[2]string{"evidence-prune", "Preview or (--apply) remove regenerable intermediates of finished efforts with a receipt; an agent --apply asks the operator"})
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
		case "evidence":
			command.Usage = "agent-manager storage evidence [--json]"
			command.HelpText = "Walks the evidence home and the work records that cite it. Reports per effort directory its lifecycle (active, terminal, unenrolled) and bytes that are regenerable intermediates, cited intermediates, cited evidence and uncited evidence. Changes nothing."
		case "evidence-prune":
			command.Usage = "agent-manager storage evidence-prune [--effort <dir>]... [--class <class>]... [--apply --reason <text> [--local-owner]] [--status <decision-id>] [--json]"
			command.HelpText = "Without --effort, selects only efforts whose every matching enrollment is withdrawn or finalized past the 7-day grace. --effort names a finished effort directory on the operator's authority. Removes only uncited platform releases, release archives, dependency trees, copied ELF binaries and frames beside their encoded video; previews unless --apply. Every applied prune writes a receipt listing each removed path."
			command.Args.Flags = append(command.Args.Flags,
				cliapp.Flag{Name: "effort", Repeated: true, LocalOnly: true, Description: "Effort directory under the evidence home (repeatable)"},
				cliapp.Flag{Name: "class", Repeated: true, LocalOnly: true, Values: []string{"platform_release", "release_archive", "dependency_tree", "build_binary", "render_frames"}, Description: "Limit to a class (repeatable)"},
				cliapp.Flag{Name: "apply", Bool: true, LocalOnly: true, Description: "Remove the candidates; without it the prune is a preview"},
				cliapp.Flag{Name: "reason", LocalOnly: true, Description: "Why the named efforts are finished (required with --apply)"},
				cliapp.Flag{Name: "local-owner", Bool: true, LocalOnly: true, Description: "Explicit local owner exchange; unavailable inside identified agent runs"})
		}
	}
	return group
}
