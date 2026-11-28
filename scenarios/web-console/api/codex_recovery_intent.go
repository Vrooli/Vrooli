package main

import (
	"context"
	"errors"
	"fmt"

	"web-console/internal/dbx"
)

const codexRecoveryIntentSchema = `
CREATE TABLE IF NOT EXISTS codex_recovery_intents (
	session_id TEXT PRIMARY KEY,
	thread_id TEXT NOT NULL,
	boot_id TEXT NOT NULL,
	owner_id TEXT NOT NULL,
	phase TEXT NOT NULL CHECK (phase IN ('claimed', 'owned'))
);`

var errCodexReopenUnconfirmed = errors.New("native reopen withheld: session changed or a prior launch has no confirmed safe handoff")

type codexRecoveryIntents struct {
	db   dbx.Handle
	boot string
}

func (s *codexRecoveryIntents) claim(ctx context.Context, session, thread, owner string) error {
	if s == nil {
		return nil
	}
	// The row check and launch claim happen in one write, before any provider
	// RPC. A registered owner from an earlier verified boot cannot still run.
	// An unconfirmed claim survives even that reboot for explicit resolution.
	result, err := s.db.ExecContext(ctx, `INSERT INTO codex_recovery_intents(session_id,thread_id,boot_id,owner_id,phase)
		SELECT id,native_thread_id,?,?,'claimed' FROM sessions
		WHERE id=? AND native_thread_id=? AND launch_mode='codex_app_server'
		  AND status='live' AND archived_at='' AND recovered_into=''
		ON CONFLICT(session_id) DO UPDATE SET thread_id=excluded.thread_id,
		  boot_id=excluded.boot_id, owner_id=excluded.owner_id, phase='claimed'
		WHERE codex_recovery_intents.phase='owned'
		  AND excluded.boot_id!='unverified' AND codex_recovery_intents.boot_id!='unverified'
		  AND codex_recovery_intents.boot_id!=excluded.boot_id`, s.boot, owner, session, thread)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errCodexReopenUnconfirmed
	}
	return nil
}

func (s *codexRecoveryIntents) confirm(ctx context.Context, session, thread, owner string) error {
	if s == nil {
		return nil
	}
	result, err := s.db.ExecContext(ctx, `UPDATE codex_recovery_intents SET phase='owned'
		WHERE session_id=? AND thread_id=? AND boot_id=? AND owner_id=? AND phase='claimed'`, session, thread, s.boot, owner)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return fmt.Errorf("native reopen claim changed before registration was confirmed")
	}
	return err
}

func (s *codexRecoveryIntents) release(ctx context.Context, session, owner string) error {
	if s == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM codex_recovery_intents
		WHERE session_id=? AND boot_id=? AND owner_id=? AND phase='owned'`, session, s.boot, owner)
	return err
}
