package x402

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"treasury/internal/testutil/basefork"
)

func TestDeclareFromOfferDeskUsesSelectedGraphPrice(t *testing.T) {
	const offerID, variantID = "offer-inference", "variant-call"
	service := &offerDeskCatalogFixture{edges: []*offerspb.Edge{
		{FromId: offerID, ToId: "variant-other", Kind: "sells_at", IntendedPriceMinor: 999, Currency: "USD", IntendedPriceDeclared: true},
		{FromId: offerID, ToId: variantID, Kind: "sells_at", IntendedPriceMinor: 37, Currency: "USD", IntendedPriceDeclared: true},
	}}
	path, handler := offersconnect.NewCatalogServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	source, err := NewOfferDeskPriceSource(server.URL, server.Client())
	require.NoError(t, err)
	gate, _, _, _ := newInboundFixture(t)
	price, err := gate.DeclareFromOfferDesk(context.Background(), Price{
		ID: "offerdesk-x402-price", ResourceURL: "http://127.0.0.1:14055/api/paid/inference",
		Description: "Offer Desk priced inference", MIMEType: "application/json",
		Network: "eip155:8453", Scheme: "exact", PayTo: "0x3333333333333333333333333333333333333333",
		Asset: basefork.USDC, AssetDecimals: 6, MaxTimeoutSeconds: 300,
		ExtraJSON: "{\"assetTransferMethod\":\"eip3009\",\"name\":\"USD Coin\",\"version\":\"2\"}",
	}, source, offerID, variantID)
	require.NoError(t, err)
	require.EqualValues(t, 37, price.AmountMinor)
	require.Equal(t, "USD", price.Currency)
	require.Equal(t, "370000", price.Amount, "37 USD minor units must become 370000 six-decimal USDC atoms")
	require.Equal(t, offerID, service.requestedOfferID)
}

type offerDeskCatalogFixture struct {
	offersconnect.UnimplementedCatalogServiceHandler
	edges            []*offerspb.Edge
	requestedOfferID string
}

func (f *offerDeskCatalogFixture) ListEdges(_ context.Context, request *connect.Request[offerspb.ListEdgesRequest]) (*connect.Response[offerspb.ListEdgesResponse], error) {
	f.requestedOfferID = request.Msg.GetNodeId()
	return connect.NewResponse(&offerspb.ListEdgesResponse{Edges: f.edges}), nil
}
