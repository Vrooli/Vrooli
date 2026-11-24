package provision

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"compute-manager/internal/intent"
	"compute-manager/internal/provider/akash"
)

var ErrSpendReceiptNotFound = errors.New("spend receipt not found")

// SpendReceipt is the persisted quote joined with the cap decision that
// admission recorded before any provider effect.
type SpendReceipt struct {
	IdempotencyKey             string
	Provider                   string
	OfferID                    string
	Region                     string
	Size                       string
	IPv4                       bool
	Term                       string
	Currency                   string
	ServerMinor                int64
	IPv4Minor                  int64
	MonthlyMinor               int64
	TaxBasis                   string
	SourceURL                  string
	SourceVersion              string
	ObservedAt                 time.Time
	ExpiresAt                  time.Time
	QuoteState                 string
	QuoteReason                string
	CapCurrency                string
	CapMonthly                 int64
	CapTaxBasis                string
	Decision                   string
	AdmissionReason            string
	CreatedAt                  time.Time
	RequestState               string
	InstanceID                 string
	PriceProvenanceJSON        string
	PriceProvenanceFingerprint string
}

func (s Service) GetSpendReceipt(ctx context.Context, idempotencyKey string) (SpendReceipt, error) {
	queryer, ok := s.DB.(interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	})
	if !ok || s.DB == nil {
		return SpendReceipt{}, errors.New("spend receipt storage is unavailable")
	}
	var receipt SpendReceipt
	var observedAt, expiresAt, createdAt string
	var quoteJSON string
	err := queryer.QueryRowContext(ctx, `SELECT q.idempotency_key,q.provider,q.offer_id,q.region,q.size,q.ipv4,q.term,q.currency,q.server_minor,q.ipv4_minor,q.monthly_minor,q.tax_basis,q.source_url,q.source_version,q.observed_at,q.expires_at,q.state,q.reason,a.cap_currency,a.cap_monthly_minor,a.cap_tax_basis,a.decision,a.reason,q.created_at,COALESCE(i.state,''),COALESCE(i.instance_id,''),q.quote_json FROM spend_quotes q JOIN spend_admissions a ON a.idempotency_key=q.idempotency_key LEFT JOIN instance_intents i ON i.idempotency_key=q.idempotency_key WHERE q.idempotency_key=?`, idempotencyKey).Scan(&receipt.IdempotencyKey, &receipt.Provider, &receipt.OfferID, &receipt.Region, &receipt.Size, &receipt.IPv4, &receipt.Term, &receipt.Currency, &receipt.ServerMinor, &receipt.IPv4Minor, &receipt.MonthlyMinor, &receipt.TaxBasis, &receipt.SourceURL, &receipt.SourceVersion, &observedAt, &expiresAt, &receipt.QuoteState, &receipt.QuoteReason, &receipt.CapCurrency, &receipt.CapMonthly, &receipt.CapTaxBasis, &receipt.Decision, &receipt.AdmissionReason, &createdAt, &receipt.RequestState, &receipt.InstanceID, &quoteJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return SpendReceipt{}, ErrSpendReceiptNotFound
	}
	if err != nil {
		return SpendReceipt{}, err
	}
	var storedQuote StoredQuote
	if err := json.Unmarshal([]byte(quoteJSON), &storedQuote); err != nil {
		return SpendReceipt{}, errors.New("stored spend quote JSON is invalid")
	}
	if storedQuote.IdempotencyKey != receipt.IdempotencyKey || storedQuote.Provider != receipt.Provider || storedQuote.OfferID != receipt.OfferID || storedQuote.Region != receipt.Region || storedQuote.Currency != receipt.Currency || storedQuote.MonthlyMinor != receipt.MonthlyMinor || storedQuote.SourceURL != receipt.SourceURL || storedQuote.SourceVersion != receipt.SourceVersion {
		return SpendReceipt{}, errors.New("stored spend quote JSON does not match indexed quote columns")
	}
	receipt.PriceProvenanceJSON = storedQuote.PriceProvenanceJSON
	receipt.PriceProvenanceFingerprint = storedQuote.PriceProvenanceFingerprint
	receipt.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
	if err != nil {
		return SpendReceipt{}, errors.New("stored spend receipt has an invalid observation time")
	}
	receipt.ExpiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return SpendReceipt{}, errors.New("stored spend receipt has an invalid expiry time")
	}
	receipt.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return SpendReceipt{}, errors.New("stored spend receipt has an invalid creation time")
	}
	return receipt, nil
}

// ValidateAkashQuoteReplay ensures a successful request replay still observes
// the identical fresh quote inputs and admission policy before returning the
// stored receipt. It is read-only and never acquires capacity or calls a provider.
func (s Service) ValidateAkashQuoteReplay(ctx context.Context, record intent.Record, receipt SpendReceipt) error {
	if record.Provider != akash.Name || receipt.Provider != akash.Name {
		return errors.New("Akash quote replay provider mismatch")
	}
	if s.MoneyCap.Currency != receipt.CapCurrency || s.MoneyCap.MonthlyMinor != receipt.CapMonthly || s.MoneyCap.TaxBasis != receipt.CapTaxBasis {
		return errors.New("Akash quote replay admission cap changed")
	}
	region, _ := record.Spec["region"].(string)
	size, _ := record.Spec["size"].(string)
	ipv4, _ := record.Spec["ipv4"].(bool)
	quote, err := priceSelection(ctx, s.Prices, akash.Name, region, size, record.IdempotencyKey, ipv4, s.MoneyCap, s.now())
	if err != nil {
		return fmt.Errorf("Akash quote replay source is stale or unavailable: %w", err)
	}
	if err := akash.ValidateIntentBinding(quote.PriceProvenanceJSON, record.Spec); err != nil {
		return err
	}
	if quote.PriceProvenanceFingerprint != receipt.PriceProvenanceFingerprint || quote.PriceProvenanceJSON != receipt.PriceProvenanceJSON || quote.MonthlyMinor != receipt.MonthlyMinor || quote.Currency != receipt.Currency || quote.OfferID != receipt.OfferID || quote.Region != receipt.Region || quote.SourceURL != receipt.SourceURL || quote.SourceVersion != receipt.SourceVersion || !quote.ObservedAt.Equal(receipt.ObservedAt) || !quote.ExpiresAt.Equal(receipt.ExpiresAt) || quote.State != receipt.QuoteState {
		return errors.New("Akash quote replay source, conversion, target, freshness, or admission facts changed")
	}
	return nil
}
