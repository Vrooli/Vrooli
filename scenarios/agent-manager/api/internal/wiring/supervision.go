package wiring

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/stepback"
	"agent-manager/internal/supervision"

	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/google/uuid"
)

func effortDiscoveryConfig() (supervision.EffortDiscoveryConfig, error) {
	projectRoot, err := agentconfig.RepoRoot()
	if err != nil {
		return supervision.EffortDiscoveryConfig{}, err
	}
	root := strings.TrimSpace(os.Getenv("AGENT_MANAGER_EFFORT_ROOT"))
	if root == "" {
		if contract, err := repocontract.LoadDefault(projectRoot); err == nil {
			if runtimeRoot := strings.TrimSpace(os.Getenv("AGENT_MANAGER_RUNTIME_ROOT")); runtimeRoot != "" {
				// Contract resolves the protected class; the runtime root itself is configurable.
				if entry, err := contract.RuntimeHomeEntry(string(filepath.Separator), repocontract.HomeKeyPlanArtifacts); err == nil {
					root = filepath.Join(runtimeRoot, entry.RelPath, "efforts")
				}
			} else if home, err := os.UserHomeDir(); err == nil {
				if entry, err := contract.RuntimeHomeEntry(home, repocontract.HomeKeyPlanArtifacts); err == nil {
					root = filepath.Join(entry.AbsPath, "efforts")
				}
			}
		}
	}
	interval, _ := time.ParseDuration(os.Getenv("AGENT_MANAGER_EFFORT_SCAN_INTERVAL"))
	return supervision.EffortDiscoveryConfig{Root: root, Interval: interval, StandingAllowanceRef: strings.TrimSpace(os.Getenv("AGENT_MANAGER_SUPERVISION_ALLOWANCE_REF")), ProjectRoot: projectRoot}, nil
}

type supervisionRunController struct{ orchestrator *orchestration.Orchestrator }

// GetRun reads stored run state. Supervision decides from status, lineage
// and admission fields, never from the derived goal status or action views.
func (c supervisionRunController) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	return c.orchestrator.GetStoredRun(ctx, id)
}

func (c supervisionRunController) ReadEffortBoardRuns(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Run, map[uuid.UUID]supervision.EffortRunAccounting, error) {
	runs, accounting, err := c.orchestrator.RunAccountingBatch(ctx, ids)
	usage := make(map[uuid.UUID]supervision.EffortRunAccounting, len(accounting))
	for id, account := range accounting {
		usage[id] = supervision.EffortRunAccounting{Tokens: account.Tokens, TokensKnown: account.TokensKnown, WeightedTokens: account.WeightedTokens, WeightedPresent: account.WeightedPresent, WeightedKnown: account.WeightedKnown, TokenCap: account.TokenCap, TokenCapMode: account.TokenCapMode}
	}
	return runs, usage, err
}

func (c supervisionRunController) ReadRunStepBack(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]stepback.Trigger, error) {
	return c.orchestrator.ReadRunStepBack(ctx, ids)
}

func (c supervisionRunController) ContinueRun(ctx context.Context, id uuid.UUID, message, idempotencyKey string) error {
	_, err := c.orchestrator.ContinueRun(ctx, orchestration.ContinueRunRequest{RunID: id, Message: message, IdempotencyKey: idempotencyKey})
	return err
}

func (c supervisionRunController) StopRun(ctx context.Context, id uuid.UUID) error {
	return c.orchestrator.StopRun(ctx, id)
}

func (c supervisionRunController) ParkRun(ctx context.Context, id uuid.UUID, watchID string, deadline *time.Time) error {
	_, err := c.orchestrator.ParkRun(ctx, orchestration.ParkRunInput{RunID: id, Producer: orchestration.ProducerSupervision, Key: watchID, Deadline: deadline})
	return err
}

func (c supervisionRunController) WakeRun(ctx context.Context, id uuid.UUID, result string) error {
	_, err := c.orchestrator.WakeRun(ctx, orchestration.WakeRunInput{RunID: id, Result: result})
	return err
}

func (c supervisionRunController) WakeEffortCheckin(ctx context.Context, id uuid.UUID, effortRef string, through int64) error {
	return c.orchestrator.WakeEffortCheckin(ctx, id, effortRef, through)
}
