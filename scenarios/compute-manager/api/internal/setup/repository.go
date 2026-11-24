package setup

import (
	"compute-manager/internal/provider"
	"context"
	"database/sql"
	"errors"
	"time"
)

type db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type StepRecord struct {
	ProviderID string
	StepID     string
	AskID      string
	EventTime  time.Time
	State      string
	Answer     string
	Detail     string
	UpdatedAt  time.Time
}

type Repository struct{ DB db }

func NewRepository(database db) Repository { return Repository{DB: database} }

func (r Repository) Step(ctx context.Context, providerID, stepID string) (StepRecord, error) {
	var item StepRecord
	var eventTime, updatedAt string
	err := r.DB.QueryRowContext(ctx, `SELECT provider_id,step_id,ask_id,event_time,state,answer,detail,updated_at FROM provider_setup_steps WHERE provider_id=? AND step_id=?`, providerID, stepID).Scan(&item.ProviderID, &item.StepID, &item.AskID, &eventTime, &item.State, &item.Answer, &item.Detail, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return StepRecord{}, nil
	}
	if err != nil {
		return StepRecord{}, err
	}
	item.EventTime, _ = time.Parse(time.RFC3339Nano, eventTime)
	item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return item, nil
}

func (r Repository) Steps(ctx context.Context, providerID string) ([]StepRecord, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT provider_id,step_id,ask_id,event_time,state,answer,detail,updated_at FROM provider_setup_steps WHERE provider_id=? ORDER BY rowid`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StepRecord
	for rows.Next() {
		var item StepRecord
		var eventTime, updatedAt string
		if err := rows.Scan(&item.ProviderID, &item.StepID, &item.AskID, &eventTime, &item.State, &item.Answer, &item.Detail, &updatedAt); err != nil {
			return nil, err
		}
		item.EventTime, _ = time.Parse(time.RFC3339Nano, eventTime)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) StartStep(ctx context.Context, item StepRecord) (StepRecord, error) {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO provider_setup_steps(provider_id,step_id,ask_id,event_time,state,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(provider_id,step_id) DO UPDATE SET ask_id=excluded.ask_id,event_time=excluded.event_time,state=excluded.state,answer='',detail='',updated_at=excluded.updated_at WHERE provider_setup_steps.state NOT IN ('pending','complete')`, item.ProviderID, item.StepID, item.AskID, item.EventTime.UTC().Format(time.RFC3339Nano), item.State, item.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return StepRecord{}, err
	}
	return r.Step(ctx, item.ProviderID, item.StepID)
}

func (r Repository) ResolveStep(ctx context.Context, item StepRecord) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE provider_setup_steps SET state=?,answer=?,detail=?,updated_at=? WHERE provider_id=? AND step_id=? AND ask_id=? AND state='pending'`, item.State, item.Answer, item.Detail, item.UpdatedAt.UTC().Format(time.RFC3339Nano), item.ProviderID, item.StepID, item.AskID)
	return err
}

func (r Repository) PutEvidence(ctx context.Context, providerID string, e Evidence) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO provider_setup_evidence(provider_id,evidence_id,state,verified,detail,checked_at,source_url,source_version) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(provider_id,evidence_id) DO UPDATE SET state=excluded.state,verified=excluded.verified,detail=excluded.detail,checked_at=excluded.checked_at,source_url=excluded.source_url,source_version=excluded.source_version`, providerID, e.ID, e.State, e.Verified, e.Detail, e.CheckedAt.UTC().Format(time.RFC3339Nano), e.SourceURL, e.SourceVersion)
	return err
}

func (r Repository) Evidence(ctx context.Context, providerID string) ([]Evidence, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT evidence_id,state,verified,detail,checked_at,source_url,source_version FROM provider_setup_evidence WHERE provider_id=? ORDER BY checked_at`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Evidence
	for rows.Next() {
		var e Evidence
		var checkedAt string
		if err := rows.Scan(&e.ID, &e.State, &e.Verified, &e.Detail, &checkedAt, &e.SourceURL, &e.SourceVersion); err != nil {
			return nil, err
		}
		e.CheckedAt, _ = time.Parse(time.RFC3339Nano, checkedAt)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r Repository) PutPrice(ctx context.Context, fact provider.PriceFact) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO provider_setup_prices(provider_id,region,size,offer_id,currency,server_minor,ipv4_minor,tax_basis,source_url,source_version,observed_at,expires_at,provenance_json,provenance_fingerprint) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(provider_id,region,size) DO UPDATE SET offer_id=excluded.offer_id,currency=excluded.currency,server_minor=excluded.server_minor,ipv4_minor=excluded.ipv4_minor,tax_basis=excluded.tax_basis,source_url=excluded.source_url,source_version=excluded.source_version,observed_at=excluded.observed_at,expires_at=excluded.expires_at,provenance_json=excluded.provenance_json,provenance_fingerprint=excluded.provenance_fingerprint`, fact.Provider, fact.Region, fact.Size, fact.OfferID, fact.Currency, fact.ServerMinor, fact.IPv4Minor, fact.TaxBasis, fact.SourceURL, fact.SourceVersion, fact.ObservedAt.UTC().Format(time.RFC3339Nano), fact.ExpiresAt.UTC().Format(time.RFC3339Nano), fact.ProvenanceJSON, fact.ProvenanceFingerprint)
	return err
}

func (r Repository) Prices(ctx context.Context, providerID string) ([]provider.PriceFact, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT provider_id,region,size,offer_id,currency,server_minor,ipv4_minor,tax_basis,source_url,source_version,observed_at,expires_at,provenance_json,provenance_fingerprint FROM provider_setup_prices WHERE provider_id=? ORDER BY observed_at DESC`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []provider.PriceFact
	for rows.Next() {
		var f provider.PriceFact
		var observed, expires string
		if err := rows.Scan(&f.Provider, &f.Region, &f.Size, &f.OfferID, &f.Currency, &f.ServerMinor, &f.IPv4Minor, &f.TaxBasis, &f.SourceURL, &f.SourceVersion, &observed, &expires, &f.ProvenanceJSON, &f.ProvenanceFingerprint); err != nil {
			return nil, err
		}
		f.ObservedAt, _ = time.Parse(time.RFC3339Nano, observed)
		f.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		out = append(out, f)
	}
	return out, rows.Err()
}
