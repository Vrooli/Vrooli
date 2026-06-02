package queue

import (
	"context"

	"github.com/ecosystem-manager/api/pkg/agentmanager"
	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
)

// runDiffAdapter adapts agent-manager's run-diff API to the autosteer
// RunDiffProvider seam: it fetches a run's diff and reshapes the proto RunDiff
// into the classifier's stack-agnostic gameguard.Diff. A nil diff (in-place run
// with no sandbox) maps to an empty diff, which the classifier treats as clean.
type runDiffAdapter struct {
	svc agentmanager.AgentServiceAPI
}

// compile-time assertion that the adapter satisfies the autosteer seam.
var _ autosteer.RunDiffProvider = runDiffAdapter{}

func (a runDiffAdapter) GetRunDiff(ctx context.Context, runID string) (gameguard.Diff, error) {
	rd, err := a.svc.GetRunDiff(ctx, runID)
	if err != nil {
		return gameguard.Diff{}, err
	}
	if rd == nil {
		return gameguard.Diff{}, nil
	}
	files := make([]gameguard.FileChange, 0, len(rd.GetFiles()))
	for _, f := range rd.GetFiles() {
		files = append(files, gameguard.FileChange{
			Path:       f.GetPath(),
			ChangeType: f.GetChangeType(),
			Additions:  int(f.GetAdditions()),
			Deletions:  int(f.GetDeletions()),
		})
	}
	return gameguard.Diff{Content: rd.GetContent(), Files: files}, nil
}
