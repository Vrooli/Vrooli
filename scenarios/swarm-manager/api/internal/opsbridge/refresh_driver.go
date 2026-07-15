package opsbridge

import (
	"context"
	"log/slog"
	"time"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
)

// RunningOpLister lists persisted workflows so the driver can find active runner-
// owned runs. Satisfied by *opsrunner.WorkflowRepo.
type RunningOpLister interface {
	List() ([]agentops.WorkflowInstance, error)
}

// RunRefresher refreshes the round a live run belongs to, driving it toward a
// terminal status (which fires the terminal-round observer). Satisfied by
// *operatingmode.Service.
type RunRefresher interface {
	RefreshRunByID(ctx context.Context, runID string) (operatingmode.RoundEnvelope, bool, error)
}

// RefreshDriver polls active runner-owned target rounds and refreshes each, so a
// completed round is observed and delivered to CommitResult. It exists because a
// non-initiative target round is driven to completion by nothing else: the engine
// has no background poller, and its initiative-keyed refresh paths never see a
// target round. The driver's durable truth is the runner's own persisted
// workflows — a restart re-reads every running operation and resumes polling, so
// no in-memory timer state is load-bearing.
type RefreshDriver struct {
	lister    RunningOpLister
	refresher RunRefresher
	log       *slog.Logger
}

// NewRefreshDriver builds the driver. A nil logger defaults to slog.Default.
func NewRefreshDriver(lister RunningOpLister, refresher RunRefresher, log *slog.Logger) *RefreshDriver {
	if log == nil {
		log = slog.Default()
	}
	return &RefreshDriver{lister: lister, refresher: refresher, log: log}
}

// Tick refreshes every active runner-owned run exactly once. It reloads workflows
// from durable state each call, so it is inherently restart-safe. A per-run
// refresh error is logged and skipped rather than aborting the sweep — one wedged
// run must not stall the others.
func (d *RefreshDriver) Tick(ctx context.Context) error {
	workflows, err := d.lister.List()
	if err != nil {
		return err
	}
	for _, w := range workflows {
		for _, op := range w.Operations {
			if op.State != "running" || op.RunID == "" {
				continue
			}
			if _, _, err := d.refresher.RefreshRunByID(ctx, op.RunID); err != nil {
				d.log.Warn("opsbridge: refresh running operation failed",
					"err", err, "run_id", op.RunID, "execution_id", op.ExecutionID)
			}
		}
	}
	return nil
}

// Run ticks on an interval until ctx is canceled. It reloads durable state every
// tick, so the first tick after a restart resumes polling all running operations.
func (d *RefreshDriver) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.Tick(ctx); err != nil {
				d.log.Warn("opsbridge: refresh driver tick failed", "err", err)
			}
		}
	}
}
