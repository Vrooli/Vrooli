package optimize

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "optimize"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"OptimizationService.CreateOptimizationRun": h.run,
		"OptimizationService.RunCandidate":          h.candidate,
		"OptimizationService.ScoreCandidates":       h.score,
		"OptimizationService.ApproveCandidate":      h.approve,
		"OptimizationService.RollbackOptimization":  h.rollback,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("optimize: load from manifest: %w", err)
	}
	return group, nil
}
