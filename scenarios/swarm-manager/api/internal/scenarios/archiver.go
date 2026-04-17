// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package scenarios

import (
	"context"
	"swarm-manager/internal/execution"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// Archiver performs scenario archive operations.
// It implements execution.Archiver so it can be called from the
// execution service's post-completion hook.
type Archiver struct {
	handler *Handler
}

// NewArchiver creates an Archiver backed by the given Handler.
func NewArchiver(handler *Handler) *Archiver {
	return &Archiver{handler: handler}
}

// ArchiveScenario archives a scenario to a backlog idea.
// Implements execution.Archiver.
func (a *Archiver) ArchiveScenario(_ context.Context, ac execution.ArchiveContext) error {
	scenario, err := a.handler.loadScenarioByPath(ac.ScenarioName, ac.ScenarioPath)
	if err != nil {
		return err
	}

	var preserveFiles *apipb.PreserveFilesRequest
	if len(ac.PreservePaths) > 0 || ac.PreservePreset != "" {
		preserveFiles = &apipb.PreserveFilesRequest{
			Paths: ac.PreservePaths,
		}
		if ac.PreservePreset != "" {
			preserveFiles.Preset = &ac.PreservePreset
		}
		normalizePreserveFilesRequest(preserveFiles)
	}

	_, _, _, err = a.handler.archiveToBacklogIdea(scenario, ac.ScenarioPath, preserveFiles)
	return err
}
