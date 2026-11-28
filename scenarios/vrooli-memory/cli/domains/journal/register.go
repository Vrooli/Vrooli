package journal

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "journal"

func RegisterCapture(core *cliapp.ScenarioApp, manifest []byte) (cliapp.CommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "capture", map[string]cliapp.PrimitiveHandler{
		"JournalService.Remember": cliapp.ProtoMutation(h.rememberCall, h.rememberReport),
	})
	if err != nil {
		return cliapp.CommandGroup{}, fmt.Errorf("capture: load manifest: %w", err)
	}
	return cliapp.CommandGroup{Title: group.Description, Commands: group.Subcommands}, nil
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"JournalService.AppendEntry":                  cliapp.ProtoMutation(h.noteCall, h.noteReport),
		"JournalService.ProcessClassificationRetries": cliapp.ProtoMutation(h.retryCall, h.retryReport),
		"JournalService.ProcessEmbeddingRetries":      cliapp.ProtoMutation(h.retryEmbeddingsCall, h.retryEmbeddingsReport),
		"JournalService.ListClassificationNotices":    cliapp.ProtoList(h.noticesCall, h.noticesReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("journal: load manifest: %w", err)
	}
	return g, nil
}

func Command(core *cliapp.ScenarioApp) cliapp.Command {
	h := newHandlers(core)
	return cliapp.Command{Name: "note", Description: "Append an immutable memory entry", Args: cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "body", Required: false, Description: "Required memory prose, separate from work-record fields; use --body-file for multiline or shell-sensitive text"}},
		Flags:       []cliapp.Flag{{Name: "body-file", Description: "Read memory prose from a file, or - for standard input"}, {Name: "scope", Description: "Ledger scope (default agent-memory)", Default: "agent-memory"}, {Name: "kind", Description: "Entry kind"}, {Name: "trigger", Description: "Work-record trigger"}, {Name: "approach", Description: "Work-record approach"}, {Name: "evidence", Description: "Work-record evidence"}, {Name: "outcome", Description: "Work-record outcome"}},
	}}.WithPrimitive(cliapp.ProtoMutation(h.noteCall, h.noteReport))
}

func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{Command(core), cliapp.Command{Name: "retry-classifications", Description: "Replay queued classification failures", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope", Description: "Ledger scope (default agent-memory)", Default: "agent-memory"}, {Name: "limit", Description: "Maximum queue items (up to 500)"}}}}.WithPrimitive(cliapp.ProtoMutation(h.retryCall, h.retryReport)), cliapp.Command{Name: "retry-embeddings", Description: "Replay queued embedding failures", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scope", Description: "Ledger scope (default agent-memory)", Default: "agent-memory"}, {Name: "limit", Description: "Maximum queue items (up to 500)"}}}}.WithPrimitive(cliapp.ProtoMutation(h.retryEmbeddingsCall, h.retryEmbeddingsReport))}
}
