package database

import (
	"context"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

var _ repository.ManagedRunProcessReader = (*runRepository)(nil)

func (r *runRepository) ListManagedRunProcesses(ctx context.Context) ([]repository.RunProcessBinding, error) {
	// Match the canonical nonterminal filter, then perform only the exclusions
	// managedAuthorizationRun previously applied after decoding the rich list.
	where, args, err := runListWhere(repository.RunListFilter{NonTerminalOnly: true})
	if err != nil {
		return nil, err
	}
	args = append(args, string(domain.ExecutionModeAttached))
	var rows []struct {
		ID        uuid.UUID `db:"id"`
		RunnerPID int       `db:"runner_pid"`
	}
	if err := r.db.SelectContext(ctx, &rows, "SELECT runs.id, runs.runner_pid FROM runs"+where+" AND runs.runner_pid > 1 AND COALESCE(runs.execution_mode, '') <> ?", args...); err != nil {
		return nil, wrapDBError("list process bindings", "Run", "", err)
	}
	result := make([]repository.RunProcessBinding, len(rows))
	for i, row := range rows {
		result[i] = repository.RunProcessBinding{ID: row.ID, RunnerPID: row.RunnerPID}
	}
	return result, nil
}
