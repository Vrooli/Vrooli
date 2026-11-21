package hub

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrEnrollmentChanged = errors.New("enrollment changed; retry")

func (s *Service) RootEnrollmentGeneration(ctx context.Context, recipient, cloudflare string) (int64, error) {
	_, err := s.db.ExecContext(ctx, `INSERT INTO root_enrollment_generations(recipient, cloudflare_subject, generation) VALUES(?,?,0) ON CONFLICT DO NOTHING`, recipient, cloudflare)
	if err != nil {
		return 0, err
	}
	var generation int64
	err = s.db.QueryRowContext(ctx, `SELECT generation FROM root_enrollment_generations WHERE recipient=? AND cloudflare_subject=?`, recipient, cloudflare).Scan(&generation)
	return generation, err
}
func (s *Service) EnrollRootAccount(ctx context.Context, recipient, cloudflare, account string, generation int64) error {
	result, err := s.db.ExecContext(ctx, `INSERT INTO root_enrollments(recipient, cloudflare_subject, authenticator_subject) SELECT ?,?,? WHERE EXISTS (SELECT 1 FROM root_enrollment_generations WHERE recipient=? AND cloudflare_subject=? AND generation=?) ON CONFLICT DO NOTHING`, recipient, cloudflare, account, recipient, cloudflare, generation)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 { // Existing association is an idempotent success only if generation still matches.
		var current int64
		if err := s.db.QueryRowContext(ctx, `SELECT generation FROM root_enrollment_generations WHERE recipient=? AND cloudflare_subject=?`, recipient, cloudflare).Scan(&current); err != nil {
			return err
		}
		if current != generation {
			return ErrEnrollmentChanged
		}
	}
	return nil
}
func (s *Service) RevokeRootEnrollments(ctx context.Context, recipient, cloudflare string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO root_enrollment_generations(recipient, cloudflare_subject, generation) VALUES(?,?,1) ON CONFLICT(recipient, cloudflare_subject) DO UPDATE SET generation=generation+1`, recipient, cloudflare); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM root_enrollments WHERE recipient=? AND cloudflare_subject=?`, recipient, cloudflare); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) RootEnrollmentOwners(ctx context.Context, recipient, account string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT cloudflare_subject FROM root_enrollments WHERE recipient=? AND authenticator_subject=?`, recipient, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var owners []string
	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			return nil, err
		}
		owners = append(owners, owner)
	}
	return owners, rows.Err()
}

// RootEnrollmentGrant identifies the revocation generation observed before proof.
type RootEnrollmentGrant struct {
	Owner      string
	Generation int64
}

func (s *Service) RootEnrollmentSnapshot(ctx context.Context, recipient string) ([]RootEnrollmentGrant, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT cloudflare_subject, generation FROM root_enrollment_generations WHERE recipient=?`, recipient)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	grants := []RootEnrollmentGrant{}
	for rows.Next() {
		var g RootEnrollmentGrant
		if err := rows.Scan(&g.Owner, &g.Generation); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}
func (s *Service) RootEnrollmentActive(ctx context.Context, recipient, account string, grants []RootEnrollmentGrant) (bool, error) {
	return rootEnrollmentActive(ctx, s.db, recipient, account, grants)
}

type enrollmentQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func rootEnrollmentActive(ctx context.Context, db enrollmentQueryer, recipient, account string, grants []RootEnrollmentGrant) (bool, error) {
	if recipient == "" || account == "" || len(grants) == 0 {
		return false, nil
	}
	args := []any{recipient, account}
	clauses := make([]string, 0, len(grants))
	for _, grant := range grants {
		clauses = append(clauses, "(e.cloudflare_subject=? AND g.generation=?)")
		args = append(args, grant.Owner, grant.Generation)
	}
	query := `SELECT EXISTS(SELECT 1 FROM root_enrollments e JOIN root_enrollment_generations g ON g.recipient=e.recipient AND g.cloudflare_subject=e.cloudflare_subject WHERE e.recipient=? AND e.authenticator_subject=? AND (` + strings.Join(clauses, " OR ") + `))`
	var active bool
	err := db.QueryRowContext(ctx, query, args...).Scan(&active)
	return active, err
}
