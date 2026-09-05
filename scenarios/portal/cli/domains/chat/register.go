package chat

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "chats"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ChatService.ListChats":   h.list,
		"ChatService.CreateChat":  h.create,
		"ChatService.GetChat":     h.show,
		"ChatService.UpdateChat":  h.update,
		"ChatService.DeleteChat":  h.delete,
		"ChatService.ListGroups":  h.groups,
		"ChatService.CreateGroup": h.createGroup,
		"ChatService.UpdateGroup": h.updateGroup,
		"ChatService.DeleteGroup": h.deleteGroup,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("chats: load from manifest: %w", err)
	}
	return group, nil
}
