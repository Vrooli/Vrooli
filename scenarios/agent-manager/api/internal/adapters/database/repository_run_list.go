package database

import (
	"context"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

// runListWhere renders the shared WHERE clause for List and Count so a
// page and its total always describe the same filtered population. The
// clause may reference the joined tasks table as t.
func runListWhere(filter repository.RunListFilter) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}
	if filter.NonTerminalOnly {
		// Keep this set aligned with RunStatus.IsTerminal: unknown is terminal,
		// while parked and needs_review remain resumable/nonterminal.
		conditions = append(conditions, "runs.status IN (?, ?, ?, ?, ?)")
		args = append(args,
			string(domain.RunStatusPending), string(domain.RunStatusStarting), string(domain.RunStatusRunning),
			string(domain.RunStatusNeedsReview), string(domain.RunStatusParked),
		)
	}

	if len(filter.RunnerPIDs) > 0 {
		placeholders := make([]string, len(filter.RunnerPIDs))
		for i, pid := range filter.RunnerPIDs {
			if pid <= 1 {
				return "", nil, fmt.Errorf("invalid runner PID filter")
			}
			placeholders[i] = "?"
			args = append(args, pid)
		}
		conditions = append(conditions, "runs.runner_pid IN ("+strings.Join(placeholders, ",")+")")
	}

	if filter.TaskID != nil {
		conditions = append(conditions, "runs.task_id = ?")
		args = append(args, *filter.TaskID)
	}
	if filter.AgentProfileID != nil {
		conditions = append(conditions, "runs.agent_profile_id = ?")
		args = append(args, *filter.AgentProfileID)
	}
	if filter.Status != nil {
		conditions = append(conditions, "runs.status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.EndedFrom != nil {
		conditions = append(conditions, "runs.ended_at >= ?")
		args = append(args, SQLiteTime(*filter.EndedFrom))
	}
	if filter.EndedTo != nil {
		conditions = append(conditions, "runs.ended_at < ?")
		args = append(args, SQLiteTime(*filter.EndedTo))
	}
	if filter.TagPrefix != "" {
		conditions = append(conditions, "runs.tag LIKE ?")
		args = append(args, filter.TagPrefix+"%")
	}
	if filter.ScopePrefix != "" {
		// Enumerate by the joined task's scope_path rather than an ad-hoc tag
		// LIKE: this is what the promote-quiesce drain uses to find every run
		// targeting a scenario (scope "scenarios/<name>"). The exact-boundary
		// refinement (scenarios/foo vs scenarios/foo-bar) is the caller's job;
		// the SQL is a cheap prefix narrowing on the existing tasks JOIN.
		conditions = append(conditions, "t.scope_path LIKE ?")
		args = append(args, filter.ScopePrefix+"%")
	}
	if filter.InvestigatesRunID != nil {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM json_each(runs.source_run_ids) WHERE json_each.value = ?)")
		args = append(args, (*filter.InvestigatesRunID).String())
	}
	if filter.AppliesInvestigationRunID != nil {
		conditions = append(conditions, "runs.source_investigation_run_id = ?")
		args = append(args, *filter.AppliesInvestigationRunID)
	}
	if filter.ParentRunID != nil {
		conditions = append(conditions, "runs.parent_run_id = ?")
		args = append(args, *filter.ParentRunID)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}
	return whereClause, args, nil
}

func (r *runRepository) List(ctx context.Context, filter repository.RunListFilter) ([]*domain.Run, error) {
	whereClause, args, err := runListWhere(filter)
	if err != nil {
		return nil, err
	}

	// Use pruned column set — omits heavy fields (summary, resolved_config, etc.)
	colList := strings.ReplaceAll(listRunColumns, "\n", "")
	colList = strings.ReplaceAll(colList, "\t", " ")
	// Prefix each column with "runs." for the JOIN
	var prefixed []string
	for _, col := range strings.Split(colList, ",") {
		col = strings.TrimSpace(col)
		if col != "" {
			prefixed = append(prefixed, "runs."+col)
		}
	}

	base := fmt.Sprintf(
		"SELECT %s, SUBSTR(t.description, 1, 120) AS prompt_preview FROM runs LEFT JOIN tasks t ON runs.task_id = t.id%s ORDER BY runs.created_at DESC, runs.id DESC",
		strings.Join(prefixed, ", "),
		whereClause,
	)
	queryWithPaging, pagingArgs := appendLimitOffset(base, filter.Limit, filter.Offset)
	args = append(args, pagingArgs...)
	query := queryWithPaging

	var rows []listRunLiteRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list", "Run", "", err)
	}

	result := make([]*domain.Run, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// Count returns how many runs match filter, ignoring Limit and Offset. It
// uses the same WHERE clause as List; the tasks join is added only when the
// filter needs it, so status/tag/profile counts stay on the runs indexes.
func (r *runRepository) Count(ctx context.Context, filter repository.RunListFilter) (int, error) {
	whereClause, args, err := runListWhere(filter)
	if err != nil {
		return 0, err
	}
	from := "runs"
	if filter.ScopePrefix != "" {
		from = "runs LEFT JOIN tasks t ON runs.task_id = t.id"
	}
	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM "+from+whereClause, args...); err != nil {
		return 0, wrapDBError("count", "Run", "", err)
	}
	return total, nil
}
