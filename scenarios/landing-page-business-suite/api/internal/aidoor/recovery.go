package aidoor

import (
	"context"
	"time"
)

// RecoverPending reconciles durable holds from Gateway's receipts, including
// operations whose public LPBS process disappeared while streaming.
func (d *Door) RecoverPending(ctx context.Context) error {
	rows, err := d.Keys.DB.QueryContext(ctx, `SELECT c.operation_id,k.id,k.business_account_id,k.actor_identity,k.app_bundle_key FROM ai_key_charges c JOIN ai_access_keys k ON k.id=c.key_id WHERE c.actual_credits IS NULL AND k.kind='session' OR (c.actual_credits IS NULL AND k.kind='device' AND NOT EXISTS(SELECT 1 FROM ai_key_charges child WHERE child.operation_id=c.operation_id AND child.key_id<>c.key_id))`)
	if err != nil {
		return err
	}
	type pending struct {
		operation string
		key       Key
	}
	operations := []pending{}
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.operation, &item.key.ID, &item.key.Account, &item.key.Actor, &item.key.App); err != nil {
			rows.Close()
			return err
		}
		operations = append(operations, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range operations {
		_ = d.recover(ctx, item.key, item.operation)
	}
	return nil
}
func (d *Door) RunRecovery(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		_ = d.RecoverPending(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
