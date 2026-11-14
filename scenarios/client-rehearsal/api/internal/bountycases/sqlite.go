package bountycases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository {
	if db == nil {
		panic("bountycases.NewSQLiteRepository requires database")
	}
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) Register(ctx context.Context, bundle CaseBundle) (CaseBundle, error) {
	var latestVersion int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(bundle_version),0) FROM bounty_replay_cases WHERE case_id = ?`, bundle.CaseID).Scan(&latestVersion)
	if err != nil {
		return CaseBundle{}, fmt.Errorf("read latest bounty case version: %w", err)
	}
	if int64(bundle.BundleVersion) != latestVersion+1 {
		return CaseBundle{}, ErrConflict{Reason: "bundle_version must append exactly one version after the current immutable bundle"}
	}
	var otherCaseID string
	err = r.db.QueryRowContext(ctx, `SELECT case_id FROM bounty_replay_cases WHERE public_url = ? AND case_id <> ? LIMIT 1`, bundle.PublicURL, bundle.CaseID).Scan(&otherCaseID)
	if err == nil {
		return CaseBundle{}, ErrConflict{Reason: "public issue URL is already registered to another case"}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CaseBundle{}, fmt.Errorf("check bounty issue URL ownership: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `SELECT case_id FROM bounty_replay_cases WHERE pull_request_url = ? AND case_id <> ? LIMIT 1`, bundle.PullRequestURL, bundle.CaseID).Scan(&otherCaseID)
	if err == nil {
		return CaseBundle{}, ErrConflict{Reason: "winning pull request URL is already registered to another case"}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CaseBundle{}, fmt.Errorf("check bounty pull request ownership: %w", err)
	}
	if latestVersion > 0 {
		var previousURL string
		if err := r.db.QueryRowContext(ctx, `SELECT public_url FROM bounty_replay_cases WHERE case_id = ? ORDER BY bundle_version DESC LIMIT 1`, bundle.CaseID).Scan(&previousURL); err != nil {
			return CaseBundle{}, fmt.Errorf("read prior bounty issue URL: %w", err)
		}
		if previousURL != bundle.PublicURL {
			return CaseBundle{}, ErrConflict{Reason: "new immutable bundle versions must remain bound to the original issue URL"}
		}
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return CaseBundle{}, fmt.Errorf("encode bounty case bundle: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO bounty_replay_cases(case_id,public_url,pull_request_url,bundle_version,bundle_digest,bundle_json) VALUES(?,?,?,?,?,?)`,
		bundle.CaseID, bundle.PublicURL, bundle.PullRequestURL, bundle.BundleVersion, bundle.BundleDigest, string(encoded))
	if err != nil {
		if isUniqueConstraint(err) {
			return CaseBundle{}, ErrConflict{Reason: "bundle version or immutable content digest is already registered"}
		}
		return CaseBundle{}, fmt.Errorf("register bounty case %q: %w", bundle.CaseID, err)
	}
	return bundle, nil
}

func (r *sqliteRepository) Resolve(ctx context.Context, lookup Lookup) (CaseBundle, error) {
	var encoded string
	var query string
	var arg string
	if lookup.CaseID != "" {
		query, arg = `SELECT bundle_json FROM bounty_replay_cases WHERE case_id = ? ORDER BY bundle_version DESC LIMIT 1`, lookup.CaseID
	} else {
		query, arg = `SELECT bundle_json FROM bounty_replay_cases WHERE public_url = ? ORDER BY bundle_version DESC LIMIT 1`, lookup.PublicURL
	}
	err := r.db.QueryRowContext(ctx, query, arg).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		key := lookup.CaseID
		if key == "" {
			key = lookup.PublicURL
		}
		return CaseBundle{}, ErrNotFound{Lookup: key}
	}
	if err != nil {
		return CaseBundle{}, fmt.Errorf("resolve registered bounty case: %w", err)
	}
	var bundle CaseBundle
	if err := json.Unmarshal([]byte(encoded), &bundle); err != nil {
		return CaseBundle{}, ErrCorrupt{Reason: "stored bundle JSON cannot be decoded"}
	}
	if bundle.CaseID == "" || bundle.BundleDigest == "" || strings.TrimSpace(bundle.PublicURL) == "" {
		return CaseBundle{}, ErrCorrupt{Reason: "stored bundle is missing indexed identity fields"}
	}
	return bundle, nil
}

func (r *sqliteRepository) GetVersion(ctx context.Context, caseID string, version uint32) (CaseBundle, error) {
	var encoded string
	err := r.db.QueryRowContext(ctx, `SELECT bundle_json FROM bounty_replay_cases WHERE case_id = ? AND bundle_version = ?`, caseID, version).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return CaseBundle{}, ErrNotFound{Lookup: fmt.Sprintf("%s@%d", caseID, version)}
	}
	if err != nil {
		return CaseBundle{}, fmt.Errorf("resolve bounty case bundle version: %w", err)
	}
	var bundle CaseBundle
	if err := json.Unmarshal([]byte(encoded), &bundle); err != nil {
		return CaseBundle{}, ErrCorrupt{Reason: "stored bundle JSON cannot be decoded"}
	}
	return bundle, nil
}

func isUniqueConstraint(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "primary key")
}
