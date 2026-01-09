package sqlite

import (
	"context"
	"fmt"
	"strings"
)

// Build and execute helper SELECT.
func (s *Service) QuerySelect(ctx context.Context, dbName, table string, where string, order string, limit int) (*QueryResult, error) {
	if err := s.ValidateName(table); err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT * FROM %s", table)
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	if strings.TrimSpace(order) != "" {
		query += " ORDER BY " + order
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	return s.Execute(ctx, dbName, query)
}

// Insert helper: kv pairs "col=val".
func (s *Service) QueryInsert(ctx context.Context, dbName, table string, pairs []string) error {
	if err := s.ValidateName(table); err != nil {
		return err
	}
	cols := make([]string, 0, len(pairs))
	vals := make([]any, 0, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid pair: %s", p)
		}
		if err := validateIdentifier(parts[0]); err != nil {
			return err
		}
		cols = append(cols, parts[0])
		vals = append(vals, parts[1])
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(cols)), ",")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ","), placeholders)
	path, err := s.databasePath(dbName)
	if err != nil {
		return err
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, query, vals...)
	return err
}

// Update helper.
func (s *Service) QueryUpdate(ctx context.Context, dbName, table string, pairs []string, where string) error {
	if err := s.ValidateName(table); err != nil {
		return err
	}
	assignments := make([]string, 0, len(pairs))
	args := make([]any, 0, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid pair: %s", p)
		}
		if err := validateIdentifier(parts[0]); err != nil {
			return err
		}
		assignments = append(assignments, fmt.Sprintf("%s = ?", parts[0]))
		args = append(args, parts[1])
	}
	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(assignments, ","))
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	path, err := s.databasePath(dbName)
	if err != nil {
		return err
	}
	db, err := s.openDatabase(ctx, path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

func validateIdentifier(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("identifier required")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return fmt.Errorf("invalid identifier: %s", name)
		}
	}
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("identifier cannot start with a digit: %s", name)
	}
	return nil
}
