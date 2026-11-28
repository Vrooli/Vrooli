package x402

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
)

var ErrOfferDeskUnavailable = errors.New("Offer Desk price source unavailable")

// OfferDeskPrice identifies a declared sells_at edge in Offer Desk's offer
// graph. AmountMinor and Currency are read from the graph, never accepted from
// the caller.
type OfferDeskPrice struct {
	OfferID     string
	VariantID   string
	AmountMinor int64
	Currency    string
}

type OfferDeskPriceReader interface {
	ResolvePrice(context.Context, string, string) (OfferDeskPrice, error)
}

type OfferDeskPriceSource struct {
	client offersconnect.CatalogServiceClient
}

func NewOfferDeskPriceSource(baseURL string, client *http.Client) (*OfferDeskPriceSource, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("Offer Desk URL must be absolute and contain no credentials, query, or fragment")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return nil, errors.New("Offer Desk URL must use HTTPS except on loopback")
	}
	if client == nil {
		return nil, errors.New("Offer Desk HTTP client is required")
	}
	return &OfferDeskPriceSource{client: offersconnect.NewCatalogServiceClient(client, strings.TrimRight(parsed.String(), "/"))}, nil
}

func (s *OfferDeskPriceSource) ResolvePrice(ctx context.Context, offerID, variantID string) (OfferDeskPrice, error) {
	offerID, variantID = strings.TrimSpace(offerID), strings.TrimSpace(variantID)
	if s == nil || s.client == nil || offerID == "" || variantID == "" {
		return OfferDeskPrice{}, errors.New("Offer Desk offer and variant IDs are required")
	}
	response, err := s.client.ListEdges(ctx, connect.NewRequest(&offerspb.ListEdgesRequest{NodeId: offerID}))
	if err != nil {
		return OfferDeskPrice{}, fmt.Errorf("%w: %w", ErrOfferDeskUnavailable, err)
	}
	var match *offerspb.Edge
	for _, edge := range response.Msg.GetEdges() {
		if edge.GetKind() != "sells_at" || edge.GetFromId() != offerID || edge.GetToId() != variantID {
			continue
		}
		if match != nil {
			return OfferDeskPrice{}, errors.New("Offer Desk returned duplicate sells_at prices for the selected variant")
		}
		match = edge
	}
	if match == nil {
		return OfferDeskPrice{}, errors.New("Offer Desk has no declared sells_at price for the selected variant")
	}
	if !match.GetIntendedPriceDeclared() || match.GetIntendedPriceMinor() <= 0 || strings.TrimSpace(match.GetCurrency()) == "" {
		return OfferDeskPrice{}, errors.New("Offer Desk sells_at edge has no positive declared price")
	}
	return OfferDeskPrice{
		OfferID: offerID, VariantID: variantID,
		AmountMinor: match.GetIntendedPriceMinor(),
		Currency:    strings.ToUpper(strings.TrimSpace(match.GetCurrency())),
	}, nil
}

// DeclareFromOfferDesk resolves the current graph price and uses it to create
// the x402 admission price. Network, token, payout address, and resource
// identity remain Treasury-owned fixed configuration.
func (g *Gate) DeclareFromOfferDesk(ctx context.Context, template Price, source OfferDeskPriceReader, offerID, variantID string) (Price, error) {
	if source == nil {
		return Price{}, errors.New("Offer Desk price source is required")
	}
	quote, err := source.ResolvePrice(ctx, offerID, variantID)
	if err != nil {
		return Price{}, err
	}
	if quote.Currency != "USD" {
		return Price{}, fmt.Errorf("x402 native-USDC metering requires USD Offer Desk prices, got %s", quote.Currency)
	}
	template.AmountMinor = quote.AmountMinor
	template.Currency = quote.Currency
	if template.AssetDecimals < 2 || template.AssetDecimals > 30 {
		return Price{}, errors.New("x402 asset decimals are invalid")
	}
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(template.AssetDecimals-2)), nil)
	template.Amount = new(big.Int).Mul(big.NewInt(quote.AmountMinor), multiplier).String()
	return g.Declare(ctx, template)
}
