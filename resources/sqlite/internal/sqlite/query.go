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
	vals := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid pair: %s", p)
		}
		cols = append(cols, parts[0])
		vals = append(vals, fmt.Sprintf("'%s'", strings.ReplaceAll(parts[1], "'", "''")))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cols, ","), strings.Join(vals, ","))
	_, err := s.Execute(ctx, dbName, query)
	return err
}

// Update helper.
func (s *Service) QueryUpdate(ctx context.Context, dbName, table string, pairs []string, where string) error {
	if err := s.ValidateName(table); err != nil {
		return err
	}
	assignments := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid pair: %s", p)
		}
		assignments = append(assignments, fmt.Sprintf("%s='%s'", parts[0], strings.ReplaceAll(parts[1], "'", "''")))
	}
	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(assignments, ","))
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	_, err := s.Execute(ctx, dbName, query)
	return err
}
