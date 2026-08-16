package manifestvalidation

import (
	"context"
	"fmt"
	"strings"
)

// ProjectTargetID is the stable validation identity for the repository root.
// It is deliberately not a scenario name: the project owns cli/manifest.json
// and packages/proto/schemas/cli, while scenarios own their own subtrees.
const ProjectTargetID = "repo"

// ProjectCLIBinary is the executable whose registered command surface is the
// project target's runtime authority.
const ProjectCLIBinary = "vrooli"

// TargetKind identifies the validation ownership axis understood by this
// provider. Other target kinds belong to other validation providers.
type TargetKind string

const (
	TargetKindScenario TargetKind = "scenario"
	TargetKindProject  TargetKind = "project"
)

// Target is the transport-neutral target shape used by the validation service.
// Root is an optional explicit filesystem root supplied by an owning caller;
// callers that use the shared proto contract should pass the resolved physical
// path through WithScenarioPath instead.
type Target struct {
	Kind TargetKind
	ID   string
	Root string
}

// ValidateTarget validates the target kinds owned by cli-health. Project
// validation deliberately routes through the same manifest/proto/architecture
// pipeline as scenario validation, but the loaders select the repository-root
// assets for ProjectTargetID.
func (s *Service) ValidateTarget(ctx context.Context, target Target) (Report, error) {
	kind := TargetKind(strings.ToLower(strings.TrimSpace(string(target.Kind))))
	id := strings.TrimSpace(target.ID)
	switch kind {
	case TargetKindProject:
		if id == "" {
			id = ProjectTargetID
		}
		if !isProjectTarget(id) {
			return Report{}, fmt.Errorf("unsupported project target %q; want %q", target.ID, ProjectTargetID)
		}
		// An absolute root is useful to direct callers and tests. The Connect
		// handler supplies the physical request path through the context, so the
		// target's repository-relative identity is never mistaken for a path.
		if scenarioPathFrom(ctx) == "" && strings.TrimSpace(target.Root) != "" {
			ctx = WithScenarioPath(ctx, target.Root)
		}
		return s.ValidateScenario(ctx, ProjectTargetID)
	case TargetKindScenario:
		if id == "" {
			return Report{}, fmt.Errorf("scenario target id is required")
		}
		if scenarioPathFrom(ctx) == "" && strings.TrimSpace(target.Root) != "" {
			ctx = WithScenarioPath(ctx, target.Root)
		}
		return s.ValidateScenario(ctx, id)
	default:
		return Report{}, fmt.Errorf("unsupported validation target kind %q", target.Kind)
	}
}

func isProjectTarget(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return id == ProjectTargetID || id == ProjectCLIBinary
}
