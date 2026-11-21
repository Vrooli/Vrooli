package commerce

import (
	"context"
	"database/sql"
	"errors"
)

func checkSettlementFeature(ctx context.Context, tx *sql.Tx, dialect, account, operation, feature string) error {
	if feature == "" {
		return nil
	}
	query := `SELECT feature_id FROM business_account_usage_events WHERE business_account_id=$1 AND operation_id=$2`
	if dialect == "sqlite" {
		query = `SELECT feature_id FROM business_account_usage_events WHERE business_account_id=? AND operation_id=?`
	}
	var previous string
	err := tx.QueryRowContext(ctx, query, account, operation).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if previous != feature {
		return errors.New("settlement feature attribution changed")
	}
	return nil
}

func (s *BusinessAccountUsageService) writeSettlementEvent(ctx context.Context, tx *sql.Tx, operation string, event businessAccountUsageEvent, feature string) error {
	columns := `operation_id,business_account_id,actor_identity,app_bundle_key,model,credits,cost_micros,prompt_tokens,completion_tokens,provider`
	placeholders := `$1,$2,$3,$4,$5,$6,$7,$8,$9,$10`
	args := []any{operation, event.accountID, event.actor, event.appKey, event.model, event.credits, event.costMicros, event.promptTokens, event.completionTokens, event.provider}
	if s.dialect == "sqlite" {
		placeholders = `?,?,?,?,?,?,?,?,?,?`
	}
	if feature != "" {
		columns += `,feature_id`
		args = append(args, feature)
		if s.dialect == "sqlite" {
			placeholders += `,?`
		} else {
			placeholders += `,$11`
		}
	}
	columns += `,created_at`
	if s.dialect == "sqlite" {
		placeholders += `,?`
		args = append(args, s.now())
	} else {
		placeholders += `,NOW()`
	}
	query := `INSERT INTO business_account_usage_events (` + columns + `) VALUES (` + placeholders + `)`
	if s.dialect == "sqlite" {
		query += ` ON CONFLICT (operation_id) DO NOTHING`
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}
