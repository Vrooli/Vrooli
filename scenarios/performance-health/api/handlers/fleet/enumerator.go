package fleet

import (
	"context"
	"sort"
	"strings"

	"performance-health/internal/fleet"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliEnumerator lists every discovered scenario via the typed `vrooli scenario
// list --json` contract — the standard scenario enumeration, not a bespoke
// filesystem walk. This keeps the fleet scan consistent with the rest of the
// platform's notion of "the fleet".
type cliEnumerator struct {
	client *vroolicli.Client
}

func newCLIEnumerator() *cliEnumerator { return &cliEnumerator{client: vroolicli.New()} }

var _ fleet.Enumerator = (*cliEnumerator)(nil)

// List returns the slugs of every discovered scenario, sorted and de-duplicated.
func (e *cliEnumerator) List(ctx context.Context) ([]string, error) {
	resp, err := e.client.ListScenarios(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range resp.GetScenarios() {
		name := strings.TrimSpace(s.GetName())
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}
