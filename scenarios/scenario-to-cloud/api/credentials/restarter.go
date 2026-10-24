package credentials

import (
	"context"
	"fmt"
	"strings"
)

// ArgvRestarter restarts consumers on the target through the control plane's
// lifecycle verbs: `vrooli scenario restart <id>` for scenario consumers and
// `vrooli resource restart <id>` for resource consumers. Consumer refs are
// closure component ids (`scenario:<id>`, `resource:<id>`); a bare id is
// treated as a scenario.
type ArgvRestarter struct {
	Runner ArgvRunner
	Label  string
}

// Restart issues one restart per consumer and stops at the first failure.
func (r ArgvRestarter) Restart(ctx context.Context, _ Target, consumers []string) error {
	for _, consumer := range consumers {
		args, ok := restartArgv(consumer)
		if !ok {
			continue
		}
		if _, err := r.Runner.Run(ctx, r.Label, args, nil); err != nil {
			return fmt.Errorf("restart %s: %w", consumer, err)
		}
	}
	return nil
}

func restartArgv(consumer string) ([]string, bool) {
	kind, id, found := strings.Cut(consumer, ":")
	if !found {
		kind, id = "scenario", kind
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false
	}
	switch kind {
	case "scenario":
		return []string{"vrooli", "scenario", "restart", id, "--json"}, true
	case "resource":
		return []string{"vrooli", "resource", "restart", id, "--json"}, true
	}
	return nil, false
}
