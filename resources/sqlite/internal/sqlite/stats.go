package sqlite

import (
	"context"
	"fmt"
	"os"
	"strings"
)

var ErrDbstatUnavailable = fmt.Errorf("dbstat module not available in this SQLite build")

func (s *Service) Analyze(ctx context.Context, dbName string) error {
	_, err := s.Execute(ctx, dbName, "ANALYZE;")
	return err
}

func (s *Service) ShowStats(ctx context.Context, dbName string) (*QueryResult, error) {
	ok, err := s.dbstatAvailable(ctx, dbName)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDbstatUnavailable
	}
	query := `
SELECT name, path, pageno, pagetype, ncell, payload, unused, mx_payload
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

func (s *Service) dbstatAvailable(ctx context.Context, dbName string) (bool, error) {
	path, err := s.databasePath(dbName)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); err != nil {
		return false, fmt.Errorf("database not found: %s", dbName)
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return false, err
	}
	defer db.Close()
	val := querySingleTextArgs(ctx, db, "SELECT name FROM pragma_module_list WHERE name='dbstat';")
	return strings.TrimSpace(val) == "dbstat", nil
}
