package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"compute-manager/internal/provider"
)

const (
	adoptedVPSProvider      = "adopted-vps"
	adoptedVPSRegion        = "manual"
	adoptedVPSOfferVersion  = "operator-declared-monthly-total-v1"
	adoptedVPSOfferValidity = 30 * 24 * time.Hour
)

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// RecordAdoptedVPSOffer stores an operator-declared total monthly bill under a
// one-way offer identity derived from the request idempotency key. It never
// invents a provider price: the fact source explicitly identifies it as a
// self-reported declaration. The declared total occupies the server component;
// IPv4 is zero here because Compute Manager does not purchase an attachment
// for an already-owned VPS (the operator's total may include their IPv4 line).
func (c Catalog) RecordAdoptedVPSOffer(ctx context.Context, idempotencyKey string, monthlyMinor int64, currency, taxBasis string) (provider.PriceFact, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	taxBasis = strings.TrimSpace(taxBasis)
	if idempotencyKey == "" || len(idempotencyKey) > 200 {
		return provider.PriceFact{}, errors.New("adopted-vps requires a bounded idempotency key")
	}
	if monthlyMinor < 0 {
		return provider.PriceFact{}, errors.New("adopted-vps monthly declared cost cannot be negative")
	}
	if !currencyCode.MatchString(currency) {
		return provider.PriceFact{}, errors.New("adopted-vps currency must be an ISO 4217 alpha code")
	}
	if taxBasis != "pre-tax" && taxBasis != "tax-inclusive" {
		return provider.PriceFact{}, errors.New("adopted-vps tax basis must be explicitly pre-tax or tax-inclusive")
	}
	if c.Repo.DB == nil {
		return provider.PriceFact{}, errors.New("adopted-vps price repository is unavailable")
	}
	sum := sha256.Sum256([]byte(idempotencyKey))
	key := hex.EncodeToString(sum[:12])
	now := c.now()
	fact := provider.PriceFact{
		Provider: adoptedVPSProvider, OfferID: "adopted-vps:declared:" + key,
		Region: adoptedVPSRegion, Size: "declared-" + key,
		Currency: currency, ServerMinor: monthlyMinor, IPv4Minor: 0,
		TaxBasis: taxBasis, SourceURL: "urn:vrooli:compute-manager:operator-declared/adopted-vps/" + key,
		SourceVersion: adoptedVPSOfferVersion, ObservedAt: now,
		ExpiresAt: now.Add(adoptedVPSOfferValidity),
	}
	_, err := c.Repo.DB.ExecContext(ctx, `INSERT INTO provider_setup_prices(provider_id,region,size,offer_id,currency,server_minor,ipv4_minor,tax_basis,source_url,source_version,observed_at,expires_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(provider_id,region,size) DO NOTHING`, fact.Provider, fact.Region, fact.Size, fact.OfferID, fact.Currency, fact.ServerMinor, fact.IPv4Minor, fact.TaxBasis, fact.SourceURL, fact.SourceVersion, fact.ObservedAt.Format(time.RFC3339Nano), fact.ExpiresAt.Format(time.RFC3339Nano))
	if err != nil {
		return provider.PriceFact{}, fmt.Errorf("persist declared adopted-vps offer: %w", err)
	}
	stored, err := c.LookupPrice(ctx, adoptedVPSProvider, fact.Region, fact.Size, true)
	if err != nil {
		return provider.PriceFact{}, err
	}
	if stored.OfferID != fact.OfferID || stored.Currency != fact.Currency || stored.ServerMinor != fact.ServerMinor || stored.IPv4Minor != fact.IPv4Minor || stored.TaxBasis != fact.TaxBasis || stored.SourceURL != fact.SourceURL || stored.SourceVersion != fact.SourceVersion {
		return provider.PriceFact{}, errors.New("adopted-vps idempotency key is already bound to different declared billing facts")
	}
	return stored, nil
}
