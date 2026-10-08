package componenttests

import (
	"context"
	"fmt"
	"time"

	"react-component-library/internal/components"
)

func (r Runner) runPage(ctx context.Context, request Request) (Report, error) {
	if request.Version != components.PageStoryVersion || request.IncludeClosure {
		return Report{}, ValidationError{Code: "page_workspace_required", Detail: "page subjects use --version workspace and do not have a released dependency closure"}
	}
	if r.Pages == nil {
		return Report{}, fmt.Errorf("page story discovery is not configured")
	}
	subject, err := r.Pages(ctx, request.ComponentID)
	if err != nil {
		return Report{}, err
	}
	if subject.Contract == nil || len(components.StoryContractErrors(components.ValidateStoryContract(subject.Contract))) > 0 {
		return Report{}, ValidationError{Code: "invalid_page_contract", Detail: "page story must satisfy the shared story contract"}
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now().UTC()
	}
	asset := components.Component{ID: request.ComponentID, LibraryID: request.ComponentID}
	version := components.ComponentVersion{Version: request.Version}
	results, artifacts, err := r.contractStoryResults(ctx, asset, version, *subject.Contract)
	if err != nil {
		return Report{}, err
	}
	report := Report{RootComponentID: request.ComponentID, RootLibraryID: request.ComponentID, RootVersion: request.Version, SourceRevision: subject.Revision, CreatedAt: now, Results: results, Artifacts: artifacts, Verdict: VerdictPassed}
	// Workspace observations must remain distinct even when source is unchanged.
	report.ID = reportID(report) + "_" + fmt.Sprint(now.UnixNano())
	for _, result := range results {
		if result.Verdict == VerdictFailed {
			report.Verdict = VerdictFailed
			break
		}
		if result.Verdict == VerdictBlocked {
			report.Verdict = VerdictBlocked
		}
	}
	return report, nil
}
