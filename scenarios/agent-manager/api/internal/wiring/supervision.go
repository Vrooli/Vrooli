package wiring

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/supervision"
	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/google/uuid"
)

func effortDiscoveryConfig() supervision.EffortDiscoveryConfig {
	root := strings.TrimSpace(os.Getenv("AGENT_MANAGER_EFFORT_ROOT"))
	if root == "" {
		if contract, _, err := repocontract.LoadDefaultFromEnvOrCWD(); err == nil {
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
	return supervision.EffortDiscoveryConfig{Root: root, Interval: interval, StandingAllowanceRef: strings.TrimSpace(os.Getenv("AGENT_MANAGER_SUPERVISION_ALLOWANCE_REF"))}
}

type supervisionRunController struct{ orchestrator *orchestration.Orchestrator }

func (c supervisionRunController) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	return c.orchestrator.GetRun(ctx, id)
}

func (c supervisionRunController) ContinueRun(ctx context.Context, id uuid.UUID, message, idempotencyKey string) error {
	_, err := c.orchestrator.ContinueRun(ctx, orchestration.ContinueRunRequest{RunID: id, Message: message, IdempotencyKey: idempotencyKey})
	return err
}

func (c supervisionRunController) ContinuationAccepted(ctx context.Context, id uuid.UUID, message, key string) (bool, error) {
	return c.orchestrator.ContinuationAccepted(ctx, id, message, key)
}

func (c supervisionRunController) RecoverMissingSession(ctx context.Context, id uuid.UUID, message string) (*domain.Run, error) {
	return c.orchestrator.RecoverMissingSessionRun(ctx, orchestration.ResumeFromFailedRunRequest{RunID: id, CustomContext: message})
}

func (c supervisionRunController) FreshRecoveryAccepted(ctx context.Context, id uuid.UUID, message string) (*domain.Run, error) {
	return c.orchestrator.FreshRecoveryAccepted(ctx, id, message)
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
