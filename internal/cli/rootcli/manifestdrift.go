package rootcli

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

const minimumNestedCommandSegments = 2

// RegisteredLeafPaths validates and returns the leaf paths exposed by the
// bound root registry. Child parsers contribute their paths from the same
// specs or binding-name sets used to build their handler maps; the registry
// verifies that every contributed path is reachable through a bound root (and
// through a bound scenario child for the scenario domain).
func (r *Registry[C]) RegisteredLeafPaths(childPaths []string) ([]string, error) {
	validated := make([]string, 0, len(childPaths))
	for _, raw := range childPaths {
		parts := strings.Fields(commandtree.NormalizeName(raw))
		if len(parts) == 0 {
			continue
		}
		if _, ok := r.TopLevelHandler(parts[0]); !ok {
			return nil, fmt.Errorf("unwalkable:%s: root handler is not bound", parts[0])
		}
		if parts[0] == "scenario" {
			if len(parts) < minimumNestedCommandSegments {
				return nil, fmt.Errorf("unwalkable:scenario: missing child command")
			}
			if _, ok := r.ScenarioHandler(parts[1]); !ok {
				return nil, fmt.Errorf("unwalkable:scenario/%s: child handler is not bound", parts[1])
			}
		}
		validated = append(validated, strings.Join(parts, " "))
	}
	return commandtree.WalkCommandTree(commandtree.CommandTreeFromPaths(validated)), nil
}

// WalkCommandTree is the manifest-check view of the live registry.
func WalkCommandTree[C any](registry *Registry[C], childPaths []string) ([]string, error) {
	if registry == nil {
		return nil, fmt.Errorf("unwalkable:root: registry is nil")
	}
	return registry.RegisteredLeafPaths(childPaths)
}
