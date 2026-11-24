package provision

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"compute-manager/internal/provider"
	"compute-manager/internal/provider/akash"
	"github.com/google/uuid"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
)

var ErrQuoteUnavailable = errors.New("billing quote unavailable")

type SourceEffect = effectgate.Mode

const (
	SourceEffectStub = effectgate.Stub
	SourceEffectLive = effectgate.Live
)

type PriceCatalog = provider.PriceCatalogue
type PriceFact = provider.PriceFact

type MoneyCap struct {
	Currency     string
	MonthlyMinor int64
	TaxBasis     string
}

type StoredQuote struct {
	ID                         string    `json:"id"`
	IdempotencyKey             string    `json:"idempotency_key"`
	Provider                   string    `json:"provider"`
	OfferID                    string    `json:"offer_id"`
	Region                     string    `json:"region"`
	Size                       string    `json:"size"`
	IPv4                       bool      `json:"ipv4"`
	Term                       string    `json:"term"`
	Currency                   string    `json:"currency"`
	ServerMinor                int64     `json:"server_minor_per_month"`
	IPv4Minor                  int64     `json:"ipv4_minor_per_month"`
	MonthlyMinor               int64     `json:"monthly_minor"`
	TaxBasis                   string    `json:"tax_basis"`
	SourceURL                  string    `json:"source_url"`
	SourceVersion              string    `json:"source_version"`
	ObservedAt                 time.Time `json:"observed_at"`
	ExpiresAt                  time.Time `json:"expires_at"`
	PriceProvenanceJSON        string    `json:"price_provenance_json,omitempty"`
	PriceProvenanceFingerprint string    `json:"price_provenance_fingerprint,omitempty"`
	State                      string    `json:"state"`
	Reason                     string    `json:"reason,omitempty"`
	CreatedAt                  time.Time `json:"created_at"`
}

func priceSelection(ctx context.Context, catalog PriceCatalog, provider, region, size, key string, ipv4 bool, cap MoneyCap, now time.Time) (StoredQuote, error) {
	if catalog == nil {
		return StoredQuote{}, fmt.Errorf("%w: catalogue is not configured", ErrQuoteUnavailable)
	}
	fact, err := catalog.LookupPrice(ctx, provider, region, size, ipv4)
	if err != nil {
		return StoredQuote{}, fmt.Errorf("%w: %v", ErrQuoteUnavailable, err)
	}
	if fact.Provider != provider || fact.Region != region || fact.Size != size || fact.OfferID == "" || fact.SourceURL == "" || fact.SourceVersion == "" || fact.Currency == "" || fact.TaxBasis == "" || fact.ObservedAt.IsZero() || fact.ExpiresAt.IsZero() {
		return StoredQuote{}, fmt.Errorf("%w: incomplete or mismatched catalogue facts", ErrQuoteUnavailable)
	}
	if !now.Before(fact.ExpiresAt) || fact.ObservedAt.After(now) {
		return StoredQuote{}, fmt.Errorf("%w: catalogue facts are stale or from the future", ErrQuoteUnavailable)
	}
	if provider == akash.Name {
		if err := akash.ValidateCatalogueFact(fact, now); err != nil {
			return StoredQuote{}, fmt.Errorf("%w: %v", ErrQuoteUnavailable, err)
		}
	}
	if cap.Currency != fact.Currency || cap.TaxBasis != fact.TaxBasis || cap.MonthlyMinor <= 0 {
		return StoredQuote{}, fmt.Errorf("money cap does not match catalogue currency and tax basis")
	}
	if fact.ServerMinor < 0 || fact.IPv4Minor < 0 {
		return StoredQuote{}, fmt.Errorf("%w: negative price", ErrQuoteUnavailable)
	}
	if fact.ServerMinor > int64(^uint64(0)>>1)-fact.IPv4Minor {
		return StoredQuote{}, fmt.Errorf("%w: total price overflows integer minor units", ErrQuoteUnavailable)
	}
	total := fact.ServerMinor + fact.IPv4Minor
	state, reason := "approved", ""
	if total > cap.MonthlyMinor {
		state, reason = "refused_over_cap", fmt.Sprintf("monthly quote %s %d exceeds cap %d", fact.Currency, total, cap.MonthlyMinor)
	}
	return StoredQuote{IdempotencyKey: key, Provider: provider, OfferID: fact.OfferID, Region: region, Size: size, IPv4: ipv4, Term: "monthly", Currency: fact.Currency, ServerMinor: fact.ServerMinor, IPv4Minor: fact.IPv4Minor, MonthlyMinor: total, TaxBasis: fact.TaxBasis, SourceURL: fact.SourceURL, SourceVersion: fact.SourceVersion, ObservedAt: fact.ObservedAt.UTC(), ExpiresAt: fact.ExpiresAt.UTC(), PriceProvenanceJSON: fact.ProvenanceJSON, PriceProvenanceFingerprint: fact.ProvenanceFingerprint, State: state, Reason: reason, CreatedAt: now.UTC()}, nil
}

func persistQuote(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, quote *StoredQuote, cap MoneyCap) (bool, error) {
	quote.ID = uuid.NewString()
	encoded, err := json.Marshal(quote)
	if err != nil {
		return false, err
	}
	result, err := db.ExecContext(ctx, `INSERT INTO spend_quotes (id,idempotency_key,provider,offer_id,region,size,ipv4,term,currency,server_minor,ipv4_minor,monthly_minor,tax_basis,source_url,source_version,observed_at,expires_at,state,reason,quote_json,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, quote.ID, quote.IdempotencyKey, quote.Provider, quote.OfferID, quote.Region, quote.Size, quote.IPv4, quote.Term, quote.Currency, quote.ServerMinor, quote.IPv4Minor, quote.MonthlyMinor, quote.TaxBasis, quote.SourceURL, quote.SourceVersion, quote.ObservedAt.Format(time.RFC3339Nano), quote.ExpiresAt.Format(time.RFC3339Nano), quote.State, quote.Reason, string(encoded), quote.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return false, fmt.Errorf("store spend quote: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check spend quote insert: %w", err)
	}
	if rows == 0 {
		return false, nil
	}
	_, err = db.ExecContext(ctx, `INSERT INTO spend_admissions (idempotency_key,quote_id,cap_currency,cap_monthly_minor,cap_tax_basis,decision,reason,created_at) SELECT ?,id,?,?,?,?,?,? FROM spend_quotes WHERE idempotency_key=? ON CONFLICT(idempotency_key) DO NOTHING`, quote.IdempotencyKey, cap.Currency, cap.MonthlyMinor, cap.TaxBasis, quote.State, quote.Reason, quote.CreatedAt.Format(time.RFC3339Nano), quote.IdempotencyKey)
	if err != nil {
		return false, fmt.Errorf("store spend admission: %w", err)
	}
	return true, nil
}

func persistQuoteUnavailable(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, quote *StoredQuote, cap MoneyCap) error {
	quote.ID = uuid.NewString()
	encoded, err := json.Marshal(quote)
	if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `INSERT INTO spend_quotes (id,idempotency_key,provider,offer_id,region,size,ipv4,term,currency,server_minor,ipv4_minor,monthly_minor,tax_basis,source_url,source_version,observed_at,expires_at,state,reason,quote_json,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, quote.ID, quote.IdempotencyKey, quote.Provider, quote.OfferID, quote.Region, quote.Size, quote.IPv4, quote.Term, quote.Currency, quote.ServerMinor, quote.IPv4Minor, quote.MonthlyMinor, quote.TaxBasis, quote.SourceURL, quote.SourceVersion, quote.ObservedAt.Format(time.RFC3339Nano), quote.ExpiresAt.Format(time.RFC3339Nano), quote.State, quote.Reason, string(encoded), quote.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("store unavailable spend quote: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check unavailable spend quote insert: %w", err)
	}
	if rows == 0 {
		return nil
	}
	_, err = db.ExecContext(ctx, `INSERT INTO spend_admissions (idempotency_key,quote_id,cap_currency,cap_monthly_minor,cap_tax_basis,decision,reason,created_at) SELECT ?,id,?,?,?,?,?,? FROM spend_quotes WHERE idempotency_key=? ON CONFLICT(idempotency_key) DO NOTHING`, quote.IdempotencyKey, cap.Currency, cap.MonthlyMinor, cap.TaxBasis, quote.State, quote.Reason, quote.CreatedAt.Format(time.RFC3339Nano), quote.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("store unavailable spend admission: %w", err)
	}
	return nil
}
