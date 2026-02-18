// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md
package scenarios

import (
	"context"

	"swarm-manager/internal/execution"
)

// executionServiceAdapter adapts execution.Service to the ExecutionQueuer interface.
type executionServiceAdapter struct {
	service *execution.Service
}

// NewExecutionQueuer creates an ExecutionQueuer from an execution.Service.
func NewExecutionQueuer(service *execution.Service) ExecutionQueuer {
	return &executionServiceAdapter{service: service}
}

func (a *executionServiceAdapter) QueueSpecSyncArchive(ctx context.Context, ac SpecSyncArchiveContext) (SpecSyncArchiveRecord, error) {
	record, err := a.service.QueueSpecSyncArchive(ctx, execution.ArchiveContext{
		ScenarioName:   ac.ScenarioName,
		ScenarioPath:   ac.ScenarioPath,
		PresetOrCustom: ac.PresetOrCustom,
		PreservePaths:  ac.PreservePaths,
		PreservePreset: ac.PreservePreset,
	})
	if err != nil {
		return SpecSyncArchiveRecord{}, err
	}
	return SpecSyncArchiveRecord{
		ExecutionID: record.ExecutionID,
		Status:      string(record.Status),
	}, nil
}
