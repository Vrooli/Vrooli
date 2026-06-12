package scenarios

import (
	"context"
	"fmt"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// cliClient is the shared typed Vrooli CLI client. It is a package var so tests
// can substitute a fake runner via vroolicli.WithRunner.
var cliClient = vroolicli.New()

// VrooliScenarioLister discovers every scenario on disk via the typed Vrooli CLI
// contract (`vrooli scenario list --json`).
type VrooliScenarioLister struct{}

// NewVrooliScenarioLister constructs the default CLI-backed lister.
func NewVrooliScenarioLister() *VrooliScenarioLister {
	return &VrooliScenarioLister{}
}

// ListScenarios returns every scenario with its metadata.
func (l *VrooliScenarioLister) ListScenarios(ctx context.Context) ([]ScenarioMetadata, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := cliClient.ListScenarios(ctx)
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario list failed: %w", err)
	}

	scenarios := make([]ScenarioMetadata, 0, len(resp.GetScenarios()))
	for _, item := range resp.GetScenarios() {
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			continue
		}
		scenarios = append(scenarios, ScenarioMetadata{
			Name:        name,
			Description: strings.TrimSpace(item.GetDescription()),
			Status:      strings.TrimSpace(item.GetStatus()),
			Tags:        append([]string(nil), item.GetTags()...),
		})
	}
	return scenarios, nil
}

var _ ScenarioLister = (*VrooliScenarioLister)(nil)
