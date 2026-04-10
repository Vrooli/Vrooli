package docaccessstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"knowledge-observatory/internal/ports"
)

type Postgres struct {
	DB *sql.DB
}

func (p *Postgres) LogAccess(ctx context.Context, row ports.DocAccessRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	_, err := p.DB.ExecContext(ctx,
		`INSERT INTO knowledge_observatory.doc_access_log (scenario_name, doc_type, operation) VALUES ($1, $2, $3)`,
		strings.TrimSpace(row.ScenarioName),
		strings.TrimSpace(row.DocType),
		strings.TrimSpace(row.Operation),
	)
	return err
}

func (p *Postgres) QueryStats(ctx context.Context, filter ports.DocAccessFilter) ([]ports.DocAccessStat, error) {
	if p == nil || p.DB == nil {
		return nil, nil
	}

	query := `SELECT scenario_name, doc_type,
		COUNT(*) FILTER (WHERE operation = 'read') AS read_count,
		COUNT(*) FILTER (WHERE operation = 'write') AS write_count,
		COUNT(*) FILTER (WHERE operation = 'reset') AS reset_count
		FROM knowledge_observatory.doc_access_log`

	var conditions []string
	var args []interface{}
	idx := 1

	if s := strings.TrimSpace(filter.ScenarioName); s != "" {
		conditions = append(conditions, fmt.Sprintf("scenario_name = $%d", idx))
		args = append(args, s)
		idx++
	}
	if d := strings.TrimSpace(filter.DocType); d != "" {
		conditions = append(conditions, fmt.Sprintf("doc_type = $%d", idx))
		args = append(args, d)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY scenario_name, doc_type ORDER BY scenario_name, doc_type"

	rows, err := p.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ports.DocAccessStat
	for rows.Next() {
		var s ports.DocAccessStat
		if err := rows.Scan(&s.ScenarioName, &s.DocType, &s.ReadCount, &s.WriteCount, &s.ResetCount); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}
