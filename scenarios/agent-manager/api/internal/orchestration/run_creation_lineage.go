// Responsibility: admit a run's place in a run tree (parent, depth, charter)
// before any creation effect, and render its lineage header.
package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
)

// maxChildDepth is the current runExecution.maxChildDepth. Without a settings
// store the default applies, so tests and minimal wiring keep the same cap.
func (o *Orchestrator) maxChildDepth() int {
	if o.orchestrationSettings == nil {
		return agentconfig.DefaultMaxChildDepth
	}
	return o.orchestrationSettings.Get().RunExecution.MaxChildDepth
}

// admitLineage reads the parent once for the whole creation flow, refuses a
// child deeper than runExecution.maxChildDepth, and validates the charter. It
// runs before any reservation, so a refusal leaves nothing behind and the
// caller can fall back to its harness's native sub-agents.
//
// A fresh recovery replaces an already admitted child in the same place in its
// tree. The depth cap governs new nesting, so it does not refuse the
// replacement after the setting was lowered, and a parent row that has since
// been deleted leaves the replacement where the repository puts any child of a
// missing parent (depth 1).
func (f *createRunFlow) admitLineage(ctx context.Context) error {
	o, req := f.o, &f.req
	f.maxDepth = o.maxChildDepth()
	if req.CharterPath != "" {
		charter, err := validateCharterPath(req.CharterPath)
		if err != nil {
			return err
		}
		req.CharterPath = charter
	}
	if req.ParentRunID == nil {
		return nil
	}
	if o.runs == nil {
		return domain.NewConfigMissingError("runs", "run repository not configured", nil)
	}
	parent, err := o.runs.Get(ctx, *req.ParentRunID)
	if err != nil {
		return domain.NewValidationErrorWithHint("parentRunId",
			"parent run could not be read for child admission: "+err.Error(),
			"retry after the parent run is durable")
	}
	if parent == nil && f.recovery != nil {
		f.depth = 1
		return nil
	}
	if parent == nil {
		return domain.NewValidationErrorWithHint("parentRunId",
			fmt.Sprintf("parent run %s does not exist", *req.ParentRunID),
			"pass the creating run's own ID (agent-manager run identity)")
	}
	depth := parent.Depth + 1
	if depth > f.maxDepth && f.recovery == nil {
		return &domain.ChildDepthExceededError{ParentRunID: parent.ID, Depth: depth, MaxDepth: f.maxDepth}
	}
	f.parent, f.depth = parent, depth
	return nil
}

// validateCharterPath accepts only an absolute path to an existing regular
// file, read from the host Agent Manager launches the run on.
func validateCharterPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if !filepath.IsAbs(trimmed) {
		return "", domain.NewValidationErrorWithHint("charterPath", fmt.Sprintf("charter path %q is not absolute", path), "pass the charter's absolute path")
	}
	cleaned := filepath.Clean(trimmed)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", domain.NewValidationErrorWithHint("charterPath", fmt.Sprintf("charter %s is not readable: %v", cleaned, err), "write the charter before creating the run")
	}
	if !info.Mode().IsRegular() {
		return "", domain.NewValidationErrorWithHint("charterPath", fmt.Sprintf("charter %s is not a regular file", cleaned), "name the charter file itself, not its directory")
	}
	return cleaned, nil
}

// lineageHeader is the header this run's task prompt starts with, or "".
func (f *createRunFlow) lineageHeader() string {
	return domain.RunLineage{
		RunID:       f.runID,
		TaskID:      f.task.ID,
		ParentRunID: f.req.ParentRunID,
		Depth:       f.depth,
		MaxDepth:    f.maxDepth,
		CharterPath: f.req.CharterPath,
	}.Header()
}

// admitDependentChild applies the dependent-delegation gate to the parent
// admitLineage read. A fresh recovery runs the source's retained contract,
// which passed this gate when the source was admitted; its parent is often
// parked waiting for that very child, or finished, and so no longer holds a
// live qualification. The replacement still must keep or narrow the parent's
// execution restrictions.
func (f *createRunFlow) admitDependentChild() error {
	if f.recovery == nil {
		return dependentDelegationAdmissionError(f.parent, f.resolvedConfig)
	}
	if f.parent.ResolvedConfig == nil || f.resolvedConfig == nil {
		return nil
	}
	return dependentExecutionPermissionsError(f.parent.ResolvedConfig, f.resolvedConfig)
}
