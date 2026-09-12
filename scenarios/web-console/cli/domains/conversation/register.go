// Package conversation is the CLI's conversation-domain command surface. It
// mirrors the API's Connect-RPC ConversationService — session history, the
// read/listened cursor, and on-demand TTS summarization — and is built from
// the embedded manifest. File-reference resolution/preview moved to the
// file-preview domain.
package conversation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "conversation"

// Register builds the `conversation` subcommand group from the embedded
// manifest and wires Connect-RPC bindings to handlers.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ConversationService.Get":            h.get,
		"ConversationService.UpdateCursor":   h.cursorSet,
		"ConversationService.SummarizeEvent": h.summarize,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conversation: load from manifest: %w", err)
	}
	return group, nil
}
