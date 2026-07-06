package message

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "messages"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"MessageService.GetTree":          h.tree,
		"MessageService.SendMessage":      h.send,
		"MessageService.EditMessage":      h.edit,
		"MessageService.Regenerate":       h.regenerate,
		"MessageService.StreamCompletion": h.stream,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("messages: load from manifest: %w", err)
	}
	return group, nil
}
