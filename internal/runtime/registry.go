package runtime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
)

type handler interface {
	Name() string
	Kind() hostreq.Kind
	Inspect(host Host, requirement hostreq.ResolvedRequirement) ItemStatus
	Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
}

type registry struct {
	tools      map[string]handler
	safeguards map[string]handler
}

func newRegistry(items ...handler) registry {
	r := registry{
		tools:      map[string]handler{},
		safeguards: map[string]handler{},
	}
	for _, item := range items {
		r.register(item)
	}
	return r
}

func (r *registry) register(item handler) {
	if item == nil {
		panic("runtime registry: nil handler")
	}
	name := strings.TrimSpace(item.Name())
	if name == "" {
		panic("runtime registry: handler name is required")
	}
	target := r.handlersForKind(item.Kind())
	if _, exists := target[name]; exists {
		panic(fmt.Sprintf("runtime registry: duplicate %s handler %q", item.Kind(), name))
	}
	target[name] = item
}

func (r registry) lookup(kind hostreq.Kind, name string) handler {
	return r.handlersForKind(kind)[strings.TrimSpace(name)]
}

func (r registry) names(kind hostreq.Kind) []string {
	target := r.handlersForKind(kind)
	names := make([]string, 0, len(target))
	for name := range target {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r registry) handlersForKind(kind hostreq.Kind) map[string]handler {
	switch kind {
	case hostreq.KindSafeguard:
		return r.safeguards
	default:
		return r.tools
	}
}

var runtimeRegistry = newRegistry(
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
	newFFmpegTool(),
	newBatsTool(),
	newYQTool(),
	newRemoteSessionProtectionSafeguard(),
)

func lookupHandler(kind hostreq.Kind, name string) handler {
	return runtimeRegistry.lookup(kind, name)
}

func HasHandler(kind hostreq.Kind, name string) bool {
	return lookupHandler(kind, name) != nil
}

func RegisteredNames(kind hostreq.Kind) []string {
	return runtimeRegistry.names(kind)
}
