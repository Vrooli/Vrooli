package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"compute-manager/internal/provider"
)

// Catalog is the persistent, freshness-checked price catalogue. Provider prices
// are refreshed only by VerifyProvider; admission reads observations only.
type Catalog struct {
	Repo Repository
	Now  func() time.Time
}

const (
	localQEMUPriceSource  = "urn:vrooli:compute-manager:host-local-qemu-zero-provider-compute-ipv4-charge"
	localQEMUPriceVersion = "declared-host-local-zero-provider-compute-ipv4-charge-v1"
)

func (c Catalog) LookupPrice(ctx context.Context, providerID, region, size string, includeIPv4 bool) (provider.PriceFact, error) {
	if providerID == "local-qemu" {
		if region != "local" || (size != "headless-linux" && size != "desktop-linux") {
			return provider.PriceFact{}, errors.New("local-qemu quote requires region local and an exact supported profile as size")
		}
		if c.Repo.DB == nil {
			return provider.PriceFact{}, errors.New("local-qemu quote repository is unavailable")
		}
		fact, err := c.localQEMUFact(ctx, size)
		if err != nil {
			return provider.PriceFact{}, err
		}
		if err := validateLocalQEMUFact(fact, region, size, c.now()); err != nil {
			return provider.PriceFact{}, err
		}
		// The zero is the provider-billed compute and IPv4 amount only. It does
		// not claim that host capacity, electricity, or operation has no cost.
		// Both amounts are present even for IPv4 requests: local SSH forwarding
		// is not a separately billed public IPv4 attachment.
		_ = includeIPv4
		return fact, nil
	}
	facts, err := c.Repo.Prices(ctx, providerID)
	if err != nil {
		return provider.PriceFact{}, err
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	for _, fact := range facts {
		if fact.Region == region && fact.Size == size && fact.SourceURL != "" && fact.SourceVersion != "" && now.Before(fact.ExpiresAt) {
			if includeIPv4 && fact.IPv4Minor < 0 {
				return provider.PriceFact{}, errors.New("IPv4 price component is unavailable")
			}
			return fact, nil
		}
	}
	return provider.PriceFact{}, fmt.Errorf("fresh provider price fact unavailable for %s/%s/%s", providerID, region, size)
}

func (c Catalog) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c Catalog) localQEMUFact(ctx context.Context, profile string) (provider.PriceFact, error) {
	facts, err := c.Repo.Prices(ctx, "local-qemu")
	if err != nil {
		return provider.PriceFact{}, err
	}
	for _, fact := range facts {
		if fact.Region == "local" && fact.Size == profile {
			return fact, nil
		}
	}
	return provider.PriceFact{}, fmt.Errorf("local-qemu declared billing fact is missing for profile %s", profile)
}

// RecordLocalQEMUOffer writes the explicit host-local zero-provider-charge
// declaration. Startup/readiness may call this; admission itself only reads
// the stored observation and refuses if it is missing or stale.
func (c Catalog) RecordLocalQEMUOffer(ctx context.Context, profile string) (provider.PriceFact, error) {
	if profile != "headless-linux" && profile != "desktop-linux" {
		return provider.PriceFact{}, fmt.Errorf("unsupported local-qemu profile %q", profile)
	}
	if c.Repo.DB == nil {
		return provider.PriceFact{}, errors.New("local-qemu quote repository is unavailable")
	}
	now := c.now()
	fact := provider.PriceFact{
		Provider: "local-qemu", OfferID: "host-local-qemu:" + profile,
		Region: "local", Size: profile, Currency: "USD", ServerMinor: 0,
		IPv4Minor: 0, TaxBasis: "pre-tax", SourceURL: localQEMUPriceSource,
		SourceVersion: localQEMUPriceVersion, ObservedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	if err := c.Repo.PutPrice(ctx, fact); err != nil {
		return provider.PriceFact{}, err
	}
	stored, err := c.localQEMUFact(ctx, profile)
	if err != nil {
		return provider.PriceFact{}, err
	}
	if err := validateLocalQEMUFact(stored, "local", profile, now); err != nil {
		return provider.PriceFact{}, err
	}
	return stored, nil
}

func validateLocalQEMUFact(fact provider.PriceFact, region, profile string, now time.Time) error {
	if fact.Provider != "local-qemu" || fact.OfferID != "host-local-qemu:"+profile || fact.Region != region || fact.Size != profile {
		return errors.New("local-qemu billing fact does not match the exact host-local profile offer")
	}
	if fact.Currency != "USD" || fact.TaxBasis != "pre-tax" || fact.ServerMinor != 0 || fact.IPv4Minor != 0 {
		return errors.New("local-qemu billing fact is not the declared zero provider compute/IPv4 offer")
	}
	if fact.SourceURL != localQEMUPriceSource || fact.SourceVersion != localQEMUPriceVersion {
		return errors.New("local-qemu billing fact has an unrecognized declaration source or version")
	}
	if fact.ObservedAt.IsZero() || fact.ExpiresAt.IsZero() || fact.ObservedAt.After(now) || !now.Before(fact.ExpiresAt) || !fact.ObservedAt.Before(fact.ExpiresAt) {
		return errors.New("local-qemu billing fact is stale or has invalid freshness timestamps")
	}
	return nil
}

// FetchPriceFact obtains a fresh provider-sourced observation. It deliberately
// supports only contracts for which every requested component is explicit.
func FetchPriceFact(ctx context.Context, client *http.Client, token, providerID, region, size string, includeIPv4 bool, now time.Time) (provider.PriceFact, error) {
	if providerID == "hetzner" {
		return fetchHetznerPrice(ctx, client, token, region, size, includeIPv4, now)
	}
	if providerID == "digitalocean" {
		return fetchDigitalOceanPrice(ctx, client, token, region, size, includeIPv4, now)
	}
	if providerID == "akash" {
		return provider.PriceFact{}, errors.New("Akash bid is not admissible: no current source-backed bid observation is configured; exact chain denomination scale and per-block interval provenance are required; a current source-backed AKT/ACT-to-USD exchange rate and explicit admission-month duration are required before producing the shared USD monthly quote")
	}
	return provider.PriceFact{}, fmt.Errorf("unknown provider %q", providerID)
}

type priceEntry struct {
	Location string `json:"location"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Prices   []struct {
		Location     string `json:"location"`
		PriceMonthly struct {
			Net   string `json:"net"`
			Gross string `json:"gross"`
		} `json:"price_monthly"`
	} `json:"prices"`
}

func fetchHetznerPrice(ctx context.Context, client *http.Client, token, region, size string, includeIPv4 bool, now time.Time) (provider.PriceFact, error) {
	const url = "https://api.hetzner.cloud/v1/pricing"
	body, version, err := getProviderJSON(ctx, client, url, token)
	if err != nil {
		return provider.PriceFact{}, err
	}
	var response struct {
		Pricing struct {
			Currency    string       `json:"currency"`
			VATRate     string       `json:"vat_rate"`
			ServerTypes []priceEntry `json:"server_types"`
			PrimaryIPs  []priceEntry `json:"primary_ips"`
		} `json:"pricing"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return provider.PriceFact{}, errors.New("Hetzner pricing response could not be decoded")
	}
	if response.Pricing.Currency == "" {
		return provider.PriceFact{}, errors.New("Hetzner pricing response omitted currency")
	}
	var serverMinor int64
	found := false
	for _, item := range response.Pricing.ServerTypes {
		if item.Name != size {
			continue
		}
		for _, price := range item.Prices {
			if price.Location != region {
				continue
			}
			value := price.PriceMonthly.Net
			if value == "" {
				value = price.PriceMonthly.Gross
			}
			serverMinor, err = decimalMinor(value)
			if err != nil {
				return provider.PriceFact{}, err
			}
			found = true
			break
		}
	}
	if !found {
		return provider.PriceFact{}, fmt.Errorf("Hetzner returned no price for %s in %s", size, region)
	}
	ipv4Minor := int64(0)
	if includeIPv4 {
		ipv4Found := false
		for _, item := range response.Pricing.PrimaryIPs {
			if item.Type != "ipv4" {
				continue
			}
			for _, price := range item.Prices {
				if price.Location != region {
					continue
				}
				value := price.PriceMonthly.Net
				if value == "" {
					value = price.PriceMonthly.Gross
				}
				ipv4Minor, err = decimalMinor(value)
				if err != nil {
					return provider.PriceFact{}, err
				}
				ipv4Found = true
				break
			}
		}
		if !ipv4Found {
			return provider.PriceFact{}, fmt.Errorf("Hetzner returned no standalone IPv4 price for %s", region)
		}
	}
	now = now.UTC()
	offerID := size
	return provider.PriceFact{Provider: "hetzner", OfferID: offerID, Region: region, Size: size, Currency: response.Pricing.Currency, ServerMinor: serverMinor, IPv4Minor: ipv4Minor, TaxBasis: "pre-tax", SourceURL: url, SourceVersion: version, ObservedAt: now, ExpiresAt: now.Add(5 * time.Minute)}, nil
}

func getProviderJSON(ctx context.Context, client *http.Client, url, token string) ([]byte, string, error) {
	if token == "" {
		return nil, "", errors.New("provider credential is unavailable")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", errors.New("provider price request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("provider price request returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", errors.New("provider price response could not be read")
	}
	version := strings.TrimSpace(resp.Header.Get("ETag"))
	if version == "" {
		version = strings.TrimSpace(resp.Header.Get("Last-Modified"))
	}
	if version == "" {
		sum := sha256.Sum256(body)
		version = "sha256:" + hex.EncodeToString(sum[:])
	}
	return body, version, nil
}

func decimalMinor(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("provider price is empty")
	}
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, errors.New("provider price is not a decimal amount")
	}
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() || !r.Num().IsInt64() {
		return 0, errors.New("provider price cannot be represented exactly in minor currency units")
	}
	return r.Num().Int64(), nil
}
