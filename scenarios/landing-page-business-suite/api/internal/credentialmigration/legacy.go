// Package credentialmigration owns the compatibility bridge from retired
// cleartext LPBS settings columns to versioned credential-authority fields.
// It never removes source columns or prints source values.
package credentialmigration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const identity credentialauthority.Identity = "vrooli/landing-page-business-suite"

type Authority interface {
	Resolve(credentialauthority.Identity, string) (string, error)
	Status(credentialauthority.Identity, string) credentialauthority.Status
	PutCandidate(credentialauthority.Identity, string, string) (credentialauthority.CandidateRef, error)
	ActivateCandidate(credentialauthority.CandidateRef) error
	ActivateCandidateIfVersion(credentialauthority.CandidateRef, string) error
	Delete(credentialauthority.Identity, string) error
}

type Item struct {
	Source      string `json:"source"`
	Field       string `json:"field"`
	Disposition string `json:"disposition"`
}

type Plan struct {
	Items  []Item `json:"items"`
	values []plannedValue
}

type plannedValue struct {
	source string
	field  string
	value  string
}

type activation struct {
	plannedValue
	previous         string
	hadValue         bool
	expectedVersion  string
	activatedVersion string
}

type Result struct {
	Applied        []string `json:"applied_fields,omitempty"`
	Retained       []string `json:"retained_fields,omitempty"`
	CleanupPending []string `json:"cleanup_pending_fields,omitempty"`
}

// Prepare reads a legacy database snapshot and checks every authority target
// before allowing any write. Values stay in memory on Plan and never appear in
// its public JSON summary or errors.
func Prepare(ctx context.Context, db *sql.DB, mode string, authority Authority) (*Plan, error) {
	if db == nil || authority == nil {
		return nil, errors.New("legacy credential migration requires a database copy and credential authority")
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	sourceTx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin read-only legacy credential snapshot: %w", err)
	}
	defer func() { _ = sourceTx.Rollback() }()
	plan := &Plan{}
	var publishable, stripeSecret, webhook sql.NullString
	err = sourceTx.QueryRowContext(ctx, `SELECT publishable_key, secret_key, webhook_secret FROM payment_settings WHERE id = 1`).Scan(&publishable, &stripeSecret, &webhook)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("read legacy payment settings snapshot: %w", err)
	}
	if err == nil {
		pub, secret, hook := clean(publishable), clean(stripeSecret), clean(webhook)
		if pub != "" || secret != "" || hook != "" {
			if mode != "test" && mode != "live" {
				return nil, errors.New("legacy Stripe values exist; set STRIPE_MODE explicitly to test or live")
			}
			if (pub == "") != (secret == "") || (hook != "" && secret == "") {
				return nil, errors.New("legacy Stripe key set is incomplete; publishable and secret keys must be paired, with webhook secret optional")
			}
			if err := validateStripeMode(mode, pub, secret); err != nil {
				return nil, err
			}
			plan.add("payment_settings", "stripe-"+mode+"-publishable-key", pub)
			plan.add("payment_settings", "stripe-"+mode+"-secret-key", secret)
			plan.add("payment_settings", "stripe-"+mode+"-webhook-secret", hook)
		}
	}

	var access, storageSecret, session sql.NullString
	err = sourceTx.QueryRowContext(ctx, `SELECT access_key_id, secret_access_key, session_token FROM download_storage_settings WHERE bundle_key = 'business_suite' LIMIT 1`).Scan(&access, &storageSecret, &session)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("read legacy storage settings snapshot: %w", err)
	}
	if err == nil {
		accessValue, secretValue, sessionValue := clean(access), clean(storageSecret), clean(session)
		if (accessValue == "") != (secretValue == "") {
			return nil, errors.New("legacy S3 credential pair is incomplete; access key ID and secret key must be migrated together")
		}
		plan.add("download_storage_settings", "delivery-s3-access-key-id", accessValue)
		plan.add("download_storage_settings", "delivery-s3-secret-access-key", secretValue)
		plan.add("download_storage_settings", "delivery-s3-session-token", sessionValue)
	}
	if err := sourceTx.Commit(); err != nil {
		return nil, fmt.Errorf("finish read-only legacy credential snapshot: %w", err)
	}

	for i := range plan.values {
		item := &plan.Items[i]
		value := plan.values[i].value
		if value == "" {
			item.Disposition = "empty-source"
			continue
		}
		existing, readErr := authority.Resolve(identity, item.Field)
		switch {
		case readErr == nil && strings.TrimSpace(existing) == value:
			if strings.TrimSpace(authority.Status(identity, item.Field).Version) == "" {
				item.Disposition = "activate-version"
			} else {
				item.Disposition = "already-migrated"
			}
		case readErr == nil:
			return nil, fmt.Errorf("credential migration conflict at %s: authority already contains a different value", item.Field)
		case errors.Is(readErr, credentialauthority.ErrUnconfigured):
			item.Disposition = "backfill"
		default:
			return nil, fmt.Errorf("credential migration preflight failed for %s: %w", item.Field, readErr)
		}
	}
	return plan, nil
}

func (p *Plan) add(source, field, value string) {
	p.values = append(p.values, plannedValue{source: source, field: field, value: value})
	p.Items = append(p.Items, Item{Source: source, Field: field})
}

// Clear drops source credential strings after a plan is applied or printed.
// Go cannot guarantee zeroization of immutable string backing storage, but
// clearing references bounds their in-process lifetime.
func (p *Plan) Clear() {
	if p == nil {
		return
	}
	for i := range p.values {
		p.values[i].value = ""
	}
	p.values = nil
}

// Apply rechecks every target immediately before writing, activates candidates
// only after full conflict preflight, and removes newly activated fields if a
// later activation fails. Legacy SQL columns remain intact for inspection and
// rollback; schema cutover is a separate reviewed operation.
func Apply(ctx context.Context, plan *Plan, authority Authority) (Result, error) {
	if plan == nil || authority == nil {
		return Result{}, errors.New("credential migration plan and authority are required")
	}
	result := Result{}
	backfill := make([]activation, 0, len(plan.values))
	for _, item := range plan.values {
		if err := ctx.Err(); err != nil {
			return Result{}, fmt.Errorf("credential migration preflight canceled: %w", err)
		}
		if item.value == "" {
			result.Retained = append(result.Retained, item.field)
			continue
		}
		status := authority.Status(identity, item.field)
		current, err := authority.Resolve(identity, item.field)
		if err == nil {
			if strings.TrimSpace(current) != item.value {
				return Result{}, fmt.Errorf("credential migration conflict at %s: authority changed after preflight", item.field)
			}
			if strings.TrimSpace(status.Version) == "" {
				backfill = append(backfill, activation{plannedValue: item, previous: current, hadValue: true, expectedVersion: status.Version})
			} else {
				result.Retained = append(result.Retained, item.field)
			}
			continue
		}
		if !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return Result{}, fmt.Errorf("credential migration recheck failed for %s: %w", item.field, err)
		}
		backfill = append(backfill, activation{plannedValue: item, expectedVersion: status.Version})
	}

	activated := make([]activation, 0, len(backfill))
	for _, item := range backfill {
		if ctx.Err() != nil {
			err := ctx.Err()
			return rollback(ctx, authority, activated, item.field, err)
		}
		candidate, err := authority.PutCandidate(identity, item.field, item.value)
		if err == nil {
			err = authority.ActivateCandidateIfVersion(candidate, item.expectedVersion)
			if errors.Is(err, credentialauthority.ErrCandidateCleanupPending) {
				result.CleanupPending = append(result.CleanupPending, item.field)
				err = nil
			}
		}
		if err != nil {
			return rollback(ctx, authority, activated, item.field, err)
		}
		item.activatedVersion = candidate.Version
		activated = append(activated, item)
		result.Applied = append(result.Applied, item.field)
	}
	return result, nil
}

func Migrate(ctx context.Context, db *sql.DB, mode string, authority Authority) (Result, error) {
	plan, err := Prepare(ctx, db, mode, authority)
	if err != nil {
		return Result{}, err
	}
	defer plan.Clear()
	return Apply(ctx, plan, authority)
}

func rollback(_ context.Context, authority Authority, activated []activation, failedField string, cause error) (Result, error) {
	rollbackErrors := make([]string, 0)
	for i := len(activated) - 1; i >= 0; i-- {
		prior := activated[i]
		var rollbackErr error
		if prior.hadValue {
			var restore credentialauthority.CandidateRef
			restore, rollbackErr = authority.PutCandidate(identity, prior.field, prior.previous)
			if rollbackErr == nil {
				rollbackErr = authority.ActivateCandidateIfVersion(restore, prior.activatedVersion)
			}
		} else {
			rollbackErr = authority.Delete(identity, prior.field)
		}
		if rollbackErr != nil {
			rollbackErrors = append(rollbackErrors, prior.field)
		}
	}
	if len(rollbackErrors) > 0 {
		return Result{}, fmt.Errorf("activate credential migration field %s failed (%v); rollback incomplete for %s; legacy source values were retained", failedField, cause, strings.Join(rollbackErrors, ","))
	}
	return Result{}, fmt.Errorf("activate credential migration field %s failed (%v); prior active values restored and legacy source values were retained", failedField, cause)
}

func clean(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

func validateStripeMode(mode, publishable, secret string) error {
	for _, value := range []string{publishable, secret} {
		keyMode := ""
		switch {
		case strings.HasPrefix(value, "pk_test_"), strings.HasPrefix(value, "sk_test_"), strings.HasPrefix(value, "rk_test_"):
			keyMode = "test"
		case strings.HasPrefix(value, "pk_live_"), strings.HasPrefix(value, "sk_live_"), strings.HasPrefix(value, "rk_live_"):
			keyMode = "live"
		}
		if keyMode != "" && keyMode != mode {
			return fmt.Errorf("legacy Stripe credential mode does not match selected STRIPE_MODE=%s", mode)
		}
	}
	return nil
}
