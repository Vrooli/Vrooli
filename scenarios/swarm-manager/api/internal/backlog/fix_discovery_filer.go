package backlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/execution"
	"swarm-manager/internal/identity"
)

// FixDiscoveryFiler implements execution.RemediationFiler. It persists one fix
// backlog item per discovery finding via the canonical backlog creation path,
// so discovery-filed items are indistinguishable from hand-filed ones except
// for their provenance tag. Lives in the backlog package (which owns item
// creation and already imports execution) to avoid an import cycle.
type FixDiscoveryFiler struct {
	service *Service
}

// NewFixDiscoveryFiler wraps a backlog Service as a RemediationFiler.
func NewFixDiscoveryFiler(service *Service) *FixDiscoveryFiler {
	return &FixDiscoveryFiler{service: service}
}

// fixDiscoveryTag marks items the on-demand readiness discovery created, both
// as a tag (for filtering) and as the provenance source.
const fixDiscoveryTag = "fix-before-feature-discovery"

// FileRemediation creates a fix item per finding for the scenario. It is
// idempotent: an item whose stable name already exists is skipped, so repeated
// discovery runs do not duplicate work. Returns the count of newly created items.
func (f *FixDiscoveryFiler) FileRemediation(ctx context.Context, scenario string, findings []execution.DiscoveryFinding) (int, error) {
	if f.service == nil {
		return 0, errors.New("FixDiscoveryFiler: service is nil")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return 0, errors.New("FixDiscoveryFiler: scenario is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	created := 0
	var firstErr error
	for _, finding := range findings {
		dimSlug := slugify(finding.Dimension)
		if dimSlug == "" {
			dimSlug = "general"
		}
		name := fmt.Sprintf("fix-discovery-%s-%s", slugify(scenario), dimSlug)

		item := BacklogItem{
			Name:        name,
			Title:       fmt.Sprintf("[%s] readiness: %s", scenario, finding.Dimension),
			Description: discoveryDescription(scenario, finding),
			Status:      StatusBacklog,
			Kind:        KindFix,
			Priority:    3,
			Tags:        []string{fixDiscoveryTag},
			Created:     now,
			Updated:     now,
			AcceptanceAllow: []string{
				"scenarios/" + scenario + "/**",
			},
			CreatedBy: &identity.Provenance{
				Type:   identity.TypeAgent,
				Source: fixDiscoveryTag,
			},
		}

		err := f.service.Create(item, CreationContext{
			Context:    ctx,
			Source:     SourceFixDiscovery,
			Entrypoint: "execution.fix-before-feature-discovery",
		})
		if err != nil {
			if errors.Is(err, ErrItemExists) {
				continue // already filed by a prior run — idempotent
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		created++
	}
	return created, firstErr
}

func discoveryDescription(scenario string, finding execution.DiscoveryFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "On-demand readiness discovery flagged the %q dimension as %s for `%s`.\n\n",
		finding.Dimension, strings.ToUpper(finding.Status), scenario)
	if strings.TrimSpace(finding.Details) != "" {
		b.WriteString("Details:\n")
		b.WriteString(finding.Details)
		b.WriteString("\n\n")
	}
	b.WriteString("Filed automatically by the fix-before-feature gate so this pre-existing issue is resolved before feature work stacks on it. Refine via the normal workshop flow.")
	return b.String()
}
