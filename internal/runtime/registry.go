package runtime

import (
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

type handler interface {
	Name() string
	Kind() hostreq.Kind
	Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus
	Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
}

var handlers = []handler{
	newDockerTool(),
	newGitTool(),
	newCurlTool(),
	newJQTool(),
	newGoTool(),
	newNodeTool(),
	newPythonTool(),
	newHelmTool(),
	newTmuxTool(),
	newSQLiteTool(),
	newRemoteSessionProtectionSafeguard(),
}

func lookupHandler(kind hostreq.Kind, name string) handler {
	name = strings.TrimSpace(name)
	for _, item := range handlers {
		if item.Kind() == kind && item.Name() == name {
			return item
		}
	}
	return nil
}
