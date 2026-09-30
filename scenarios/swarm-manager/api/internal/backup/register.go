// Package backup performs boot-time, idempotent registration of swarm-manager's
// durable runtime data with the data-backup-manager scenario.
//
// swarm-manager's domain data (backlog items, initiatives, agent sessions) now
// lives under the api-core/storage data class root (the operator runtime home,
// ~/.vrooli/data/vrooli/swarm-manager) rather than the git tree. That data is
// the operator's accumulated work and must be backed up. We self-register a
// single filesystem backup target covering the whole data base, mirroring the
// agentmanager EnsureProfile reconcile pattern: the upstream `targets register`
// is the idempotency authority (re-register by (owner,name) is a no-op/update),
// so this is safe to run on every boot.
//
// captures live in the cache class (disposable) and are deliberately NOT
// registered. Backup destinations and plans are operator configuration — this
// helper only declares WHAT is worth protecting, never WHERE it is stored, so a
// missing destination never blocks boot.
package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"swarm-manager/internal/runtimepaths"
)

const (
	// Owner is the owning scenario id used for the backup target.
	Owner = "swarm-manager"
	// DomainTargetName is the owner-scoped name of the single filesystem target
	// covering swarm-manager's whole data base.
	DomainTargetName = "domain-data"

	cliBinary  = "data-backup-manager"
	cliTimeout = 30 * time.Second
)

// CLIRunner runs the data-backup-manager CLI and returns its combined output.
// Injected so tests can record invocations without a real CLI/RPC backend.
type CLIRunner func(ctx context.Context, args ...string) ([]byte, error)

func defaultRunner(ctx context.Context, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, cliBinary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w (%s)", cliBinary, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// EnsureBackupTargets idempotently registers swarm-manager's durable data base
// as a single filesystem backup target with data-backup-manager. The locator is
// the resolved api-core/storage data root, so data-backup-manager's
// regenerable==false discovery sees it. captures (cache class) are NOT
// registered. Returns an error only for genuine failures; callers at boot should
// treat any error as non-fatal (data-backup-manager may not be running yet).
func EnsureBackupTargets(ctx context.Context, runner CLIRunner) error {
	if runner == nil {
		runner = defaultRunner
	}
	dataBase, err := runtimepaths.DataPath("")
	if err != nil {
		return fmt.Errorf("resolve data base: %w", err)
	}
	if _, err := runner(ctx,
		"targets", "register",
		"--owner", Owner,
		"--name", DomainTargetName,
		"--kind", "filesystem",
		"--locator", dataBase,
	); err != nil {
		return fmt.Errorf("register backup target: %w", err)
	}
	slog.Info("backup target registered",
		"owner", Owner, "name", DomainTargetName, "kind", "filesystem", "locator", dataBase)
	return nil
}
