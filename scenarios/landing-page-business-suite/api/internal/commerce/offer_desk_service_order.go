package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const offerDeskServiceOfferPath = "/api/v1/proposal-preview/service-offer"

type OfferDeskServiceOrderSource struct {
	endpoint string
	client   *http.Client
}

type offerDeskServiceOfferResponse struct {
	Offer struct {
		ID            string `json:"id"`
		Version       int    `json:"version"`
		X402PriceID   string `json:"x402_price_id"`
		ProposedPrice struct {
			AmountMinor int64  `json:"amount_minor"`
			Currency    string `json:"currency"`
		} `json:"proposed_price"`
	} `json:"offer"`
}

// NewOfferDeskServiceOrderSource creates a read-only quote source for an
// isolated Offer Desk preview. Test-mode commerce only accepts this loopback
// source; caller-supplied amounts are never treated as authoritative.
func NewOfferDeskServiceOrderSource(baseURL string, client *http.Client) (*OfferDeskServiceOrderSource, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Offer Desk URL must be an origin without credentials, path, query, or fragment")
	}
	if parsed.Scheme != "http" || !isLoopbackOfferDeskHost(parsed.Hostname()) {
		return nil, errors.New("service-order Offer Desk source must use loopback HTTP in test mode")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	clone := *client
	clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client = &clone
	return &OfferDeskServiceOrderSource{endpoint: strings.TrimRight(parsed.String(), "/") + offerDeskServiceOfferPath, client: client}, nil
}

func (s *OfferDeskServiceOrderSource) ResolveServiceOrderQuote(ctx context.Context, offerID string, version int) (ServiceOrderQuote, error) {
	if s == nil || s.client == nil || s.endpoint == "" || strings.TrimSpace(offerID) == "" || version < 1 {
		return ServiceOrderQuote{}, errors.New("Offer Desk source, offer ID, and positive version are required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return ServiceOrderQuote{}, fmt.Errorf("create Offer Desk quote request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return ServiceOrderQuote{}, fmt.Errorf("read canonical Offer Desk service offer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ServiceOrderQuote{}, fmt.Errorf("Offer Desk service offer returned HTTP %d", response.StatusCode)
	}
	var payload offerDeskServiceOfferResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return ServiceOrderQuote{}, fmt.Errorf("decode canonical Offer Desk service offer: %w", err)
	}
	offer := payload.Offer
	if offer.ID != strings.TrimSpace(offerID) || offer.Version != version {
		return ServiceOrderQuote{}, errors.New("Offer Desk returned a different offer identity or version")
	}
	currency := strings.ToUpper(strings.TrimSpace(offer.ProposedPrice.Currency))
	if offer.ProposedPrice.AmountMinor <= 0 || currency != "USD" || strings.TrimSpace(offer.X402PriceID) == "" {
		return ServiceOrderQuote{}, errors.New("Offer Desk offer must declare a positive USD price and x402 price ID")
	}
	return ServiceOrderQuote{
		OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor,
		Currency: currency, X402PriceID: strings.TrimSpace(offer.X402PriceID),
	}, nil
}

func isLoopbackOfferDeskHost(host string) bool {
	return host == "127.0.0.1" || host == "::1" || strings.EqualFold(host, "localhost")
}
