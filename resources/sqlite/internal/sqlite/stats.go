package sqlite

import (
	"context"
	"fmt"
)

func (s *Service) Analyze(ctx context.Context, dbName string) error {
	_, err := s.Execute(ctx, dbName, "ANALYZE;")
	return err
}

func (s *Service) ShowStats(ctx context.Context, dbName string) (*QueryResult, error) {
	query := `
SELECT name, path, pageno, pagetype, ncell, payload, unused, mx_payload, npage
FROM dbstat
LIMIT 50;
`
	return s.Execute(ctx, dbName, query)
}

func (s *Service) EnableStats(ctx context.Context, dbName string) error {
	_, err := s.Execute(ctx, dbName, "PRAGMA analysis_limit=400;")
	return err
}

func (s *Service) Vacuum(ctx context.Context, dbName string) error {
	_, err := s.Execute(ctx, dbName, "VACUUM;")
	return err
}

func (s *Service) StatsSummary(ctx context.Context, dbName string) (string, error) {
	res, err := s.Execute(ctx, dbName, "SELECT COUNT(*) as tables FROM sqlite_master WHERE type='table';")
	if err != nil {
		return "", err
	}
	if len(res.Rows) == 0 {
		return "No stats", nil
	}
	return fmt.Sprintf("Tables: %s", res.Rows[0][0]), nil
}
