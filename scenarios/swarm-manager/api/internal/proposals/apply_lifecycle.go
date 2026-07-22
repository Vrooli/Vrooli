package proposals

import (
	"context"
	"fmt"

	"swarm-manager/internal/backlog"
)

func (a *Applier) applyRecreateItem(ctx context.Context, ref string) error {
	if a.itemLifecycle == nil {
		return fmt.Errorf("recreate_item unavailable: lifecycle service is not wired")
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	_, err = a.itemLifecycle.RecreateItem(ctx, kind, name)
	return err
}

func (a *Applier) applyResetArtifacts(ctx context.Context, ref string, scopes []ResetArtifactScope) error {
	if a.itemLifecycle == nil {
		return fmt.Errorf("reset_artifacts unavailable: lifecycle service is not wired")
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	translated := make([]backlog.ResetArtifactScope, len(scopes))
	for i, scope := range scopes {
		translated[i] = backlog.ResetArtifactScope(scope)
	}
	_, err = a.itemLifecycle.ResetArtifacts(ctx, kind, name, translated)
	return err
}

func (a *Applier) applyRecreateMilestone(ctx context.Context, name string) error {
	if a.milestoneLifecycle == nil {
		return fmt.Errorf("recreate_milestone unavailable: lifecycle service is not wired")
	}
	return a.milestoneLifecycle.RecreateMilestone(ctx, name)
}
