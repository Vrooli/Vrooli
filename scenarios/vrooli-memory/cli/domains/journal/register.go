package journal

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "journal"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{"JournalService.AppendEntry": cliapp.ProtoMutation(h.noteCall, h.noteReport), "JournalService.ProcessClassificationRetries": cliapp.ProtoMutation(h.retryCall, h.retryReport), "JournalService.ProcessEmbeddingRetries": cliapp.ProtoMutation(h.retryEmbeddingsCall, h.retryEmbeddingsReport)})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("journal: load manifest: %w", err)
	}
	return g, nil
}

func Command(core *cliapp.ScenarioApp) cliapp.Command {
	h := newHandlers(core)
	return cliapp.Command{Name: "note", Description: "Append an immutable memory entry", Args: cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "body", Required: true, Description: "Memory prose"}},
		Flags:       []cliapp.Flag{{Name: "kind", Description: "Entry kind"}},
	}}.WithPrimitive(cliapp.ProtoMutation(h.noteCall, h.noteReport))
}

func Commands(core *cliapp.ScenarioApp) []cliapp.Command {
	h := newHandlers(core)
	return []cliapp.Command{Command(core), cliapp.Command{Name: "retry-classifications", Description: "Replay queued classification failures", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit", Description: "Maximum queue items (up to 500)"}}}}.WithPrimitive(cliapp.ProtoMutation(h.retryCall, h.retryReport)), cliapp.Command{Name: "retry-embeddings", Description: "Replay queued embedding failures", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "limit", Description: "Maximum queue items (up to 500)"}}}}.WithPrimitive(cliapp.ProtoMutation(h.retryEmbeddingsCall, h.retryEmbeddingsReport))}
}
