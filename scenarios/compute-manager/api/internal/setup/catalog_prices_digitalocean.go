package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"compute-manager/internal/provider"
)

const (
	digitalOceanSizesURL       = "https://api.digitalocean.com/v2/sizes?per_page=200"
	digitalOceanPricePolicyURL = "https://docs.digitalocean.com/products/droplets/details/pricing/"
)

// fetchDigitalOceanPrice returns an exact quote for a currently available
// bundled Droplet plan. GET /v2/sizes reports price_monthly in USD and regions;
// it does not model v5 configurations. Those remain fail-closed because their
// IPv4 line item is separately priced and may change. For bundled plans the
// provider's pricing page identifies the separate IPv4 charge only for v5.
// See https://docs.digitalocean.com/reference/api/reference/sizes/ and
// https://docs.digitalocean.com/products/droplets/details/pricing/.
func fetchDigitalOceanPrice(ctx context.Context, client *http.Client, token, region, size string, includeIPv4 bool, now time.Time) (provider.PriceFact, error) {
	if strings.HasPrefix(size, "s5-") {
		return provider.PriceFact{}, errors.New("DigitalOcean v5 pricing requires a separately verified IPv4 price; quote refused")
	}
	body, version, err := getProviderJSON(ctx, client, digitalOceanSizesURL, token)
	if err != nil {
		return provider.PriceFact{}, err
	}
	var response struct {
		Sizes []struct {
			Available    bool        `json:"available"`
			Slug         string      `json:"slug"`
			PriceMonthly json.Number `json:"price_monthly"`
			Regions      []string    `json:"regions"`
		} `json:"sizes"`
		Links struct {
			Pages struct {
				Next string `json:"next"`
			} `json:"pages"`
		} `json:"links"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return provider.PriceFact{}, errors.New("DigitalOcean sizes response could not be decoded")
	}
	if response.Links.Pages.Next != "" {
		return provider.PriceFact{}, errors.New("DigitalOcean sizes response is paginated beyond the requested page; price remains unknown")
	}
	for _, item := range response.Sizes {
		if item.Slug != size || !item.Available || !containsRegion(item.Regions, region) {
			continue
		}
		monthlyMinor, err := decimalMinor(item.PriceMonthly.String())
		if err != nil {
			return provider.PriceFact{}, fmt.Errorf("DigitalOcean monthly price is not exact in USD cents: %w", err)
		}
		if monthlyMinor < 0 {
			return provider.PriceFact{}, errors.New("DigitalOcean returned a negative monthly price")
		}
		ipv4Minor := int64(0)
		if includeIPv4 {
			// The bundled-plan API size has no separate IPv4 line; v5 is rejected
			// above because its separately billed IPv4 amount is dynamic.
			ipv4Minor = 0
		}
		now = now.UTC()
		return provider.PriceFact{Provider: "digitalocean", OfferID: item.Slug, Region: region, Size: item.Slug, Currency: "USD", ServerMinor: monthlyMinor, IPv4Minor: ipv4Minor, TaxBasis: "pre-tax", SourceURL: digitalOceanSizesURL, SourceVersion: version, ObservedAt: now, ExpiresAt: now.Add(5 * time.Minute)}, nil
	}
	return provider.PriceFact{}, fmt.Errorf("DigitalOcean has no available bundled size %s in %s", size, region)
}

func containsRegion(regions []string, expected string) bool {
	for _, region := range regions {
		if region == expected {
			return true
		}
	}
	return false
}
