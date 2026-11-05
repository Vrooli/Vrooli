package signals

import (
	"context"
	"fmt"

	"architecture-cartographer/internal/graph"
)

// NewGraphSnapshotProvider returns a SnapshotProvider that delegates to
// the graph.Service. GetLatestSnapshot uses ListSnapshots(PageSize=1)
// because the underlying repository sorts by extracted_at DESC.
func NewGraphSnapshotProvider(svc graph.Service) SnapshotProvider {
	return &graphSnapshotProvider{svc: svc}
}

type graphSnapshotProvider struct {
	svc graph.Service
}

func (p *graphSnapshotProvider) GetLatestSnapshot(ctx context.Context, scenario string) (graph.GraphSnapshot, error) {
	page, err := p.svc.ListSnapshots(ctx, graph.ListSnapshotsFilter{Scenario: scenario, PageSize: 1})
	if err != nil {
		return graph.GraphSnapshot{}, err
	}
	if len(page.Snapshots) == 0 {
		return graph.GraphSnapshot{}, graph.ErrSnapshotNotFound{ID: fmt.Sprintf("scenario=%s", scenario)}
	}
	return page.Snapshots[0], nil
}
