package database

import (
	"context"
	"database/sql"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
)

const (
	fallbackRowLimit = 20
	fallbackEvents   = "('runner.fallback.attempted','runner.fallback.exhausted','model.fallback.attempted','model.fallback.exhausted','policy.candidate.attempt')"
)

// FallbackInsights counts the fallback and policy-candidate events retained in
// the filter's window. Event retention bounds how far back it can see.
func (r *invocationReadModelRepository) FallbackInsights(ctx context.Context, filter invocationreadmodel.Filter) (invocationreadmodel.FallbackInsights, error) {
	var out invocationreadmodel.FallbackInsights
	var from, to any = "0000", "9999"
	if filter.From != nil {
		from = SQLiteTime(*filter.From)
	}
	if filter.To != nil {
		to = SQLiteTime(*filter.To)
	}
	window := " WHERE event_type IN " + fallbackEvents + " AND timestamp >= ? AND timestamp < ?"

	counts, err := r.fallbackGroups(ctx, `SELECT event_type, '', '', COUNT(*) FROM run_events`+window+` GROUP BY event_type`, from, to)
	if err != nil {
		return out, err
	}
	for _, row := range counts {
		switch domain.RunEventType(row.from) {
		case domain.EventTypeRunnerFallbackAttempted:
			out.RunnerAttempts = row.count
		case domain.EventTypeRunnerFallbackExhausted:
			out.RunnerExhausted = row.count
		case domain.EventTypeModelFallbackAttempted:
			out.ModelAttempts = row.count
		case domain.EventTypeModelFallbackExhausted:
			out.ModelExhausted = row.count
		case domain.EventTypePolicyCandidateAttempt:
			out.PolicyCandidateEvents = row.count
		}
	}

	pairs, err := r.fallbackGroups(ctx, `SELECT event_type, COALESCE(json_extract(data, '$.from'), '') || char(31) || COALESCE(json_extract(data, '$.to'), ''), COALESCE(json_extract(data, '$.reason'), ''), COUNT(*)
		FROM run_events`+window+` AND event_type IN ('runner.fallback.attempted','model.fallback.attempted') GROUP BY 1, 2, 3 ORDER BY 4 DESC, 2 ASC`, from, to)
	if err != nil {
		return out, err
	}
	reasons, err := r.fallbackGroups(ctx, `SELECT event_type, COALESCE(json_extract(data, '$.reason'), json_extract(data, '$.last_reason'), ''), '', COUNT(*)
		FROM run_events`+window+` AND event_type IN ('runner.fallback.attempted','runner.fallback.exhausted','model.fallback.attempted','model.fallback.exhausted') GROUP BY 1, 2 ORDER BY 4 DESC, 2 ASC`, from, to)
	if err != nil {
		return out, err
	}
	policy, err := r.fallbackGroups(ctx, `SELECT 'outcome', COALESCE(json_extract(data, '$.outcome'), ''), '', COUNT(*) FROM run_events`+window+` AND event_type = 'policy.candidate.attempt' GROUP BY 2
		UNION ALL SELECT 'failure_class', json_extract(data, '$.failure_class'), '', COUNT(*) FROM run_events`+window+` AND event_type = 'policy.candidate.attempt' AND COALESCE(json_extract(data, '$.failure_class'), '') <> '' GROUP BY 2 ORDER BY 4 DESC, 2 ASC`, from, to, from, to)
	if err != nil {
		return out, err
	}
	for _, row := range pairs {
		pair := invocationreadmodel.FallbackPair{Reason: row.reason, Count: row.count}
		pair.From, pair.To = splitFallbackPair(row.key)
		if domain.RunEventType(row.from) == domain.EventTypeRunnerFallbackAttempted {
			out.RunnerPairs = appendLimited(out.RunnerPairs, pair)
		} else {
			out.ModelPairs = appendLimited(out.ModelPairs, pair)
		}
	}
	for _, row := range reasons {
		item := invocationreadmodel.CountRow{Key: row.key, Count: row.count}
		if len(row.from) >= 6 && row.from[:6] == "runner" {
			out.RunnerReasons = appendLimited(out.RunnerReasons, item)
		} else {
			out.ModelReasons = appendLimited(out.ModelReasons, item)
		}
	}
	for _, row := range policy {
		item := invocationreadmodel.CountRow{Key: row.key, Count: row.count}
		if row.from == "outcome" {
			out.PolicyOutcomes = append(out.PolicyOutcomes, item)
		} else {
			out.PolicyFailureClasses = appendLimited(out.PolicyFailureClasses, item)
		}
	}
	return out, nil
}

type fallbackGroup struct {
	from, key, reason string
	count             int64
}

func (r *invocationReadModelRepository) fallbackGroups(ctx context.Context, query string, args ...any) ([]fallbackGroup, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []fallbackGroup
	for rows.Next() {
		var row fallbackGroup
		var key sql.NullString
		if err := rows.Scan(&row.from, &key, &row.reason, &row.count); err != nil {
			return nil, err
		}
		row.key = key.String
		out = append(out, row)
	}
	return out, rows.Err()
}

func appendLimited[T any](rows []T, row T) []T {
	if len(rows) >= fallbackRowLimit {
		return rows
	}
	return append(rows, row)
}

func splitFallbackPair(joined string) (string, string) {
	for i := 0; i < len(joined); i++ {
		if joined[i] == 0x1f {
			return joined[:i], joined[i+1:]
		}
	}
	return joined, ""
}
