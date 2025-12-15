package workflow

import (
	"context"

	"github.com/vrooli/browser-automation-studio/database"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
)

// HydrateProject assembles the canonical proto Project from the DB index and on-disk metadata.
func (s *WorkflowService) HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error) {
	return hydrateProjectProto(ctx, project)
}

