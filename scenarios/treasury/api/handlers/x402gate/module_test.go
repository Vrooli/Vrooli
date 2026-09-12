package x402gate

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"treasury/internal/operatorauth"
	x402rail "treasury/internal/rail/x402"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// [REQ:TRS-P1-002] Price mutation is operator-only, while the interoperable
// admission route returns an x402 challenge to an unpaid public caller.
func TestPriceDeclarationAndUnpaidChallenge(t *testing.T) {
	price := fixturePrice()
	repository := &fixtureRepository{price: price}
	gate, err := x402rail.NewGate(repository, fixtureFacilitator{})
	require.NoError(t, err)
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	router := mux.NewRouter()
	Module(gate, authorizer).Mount(router)

	body, err := json.Marshal(price)
	require.NoError(t, err)
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/v1/x402/prices", bytes.NewReader(body))
	unauthorizedResult := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedResult, unauthorized)
	require.Equal(t, http.StatusUnauthorized, unauthorizedResult.Code)

	authorized := httptest.NewRequest(http.MethodPost, "/api/v1/x402/prices", bytes.NewReader(body))
	authorized.Header.Set(operatorauth.HeaderOperatorToken, "operator-secret")
	authorizedResult := httptest.NewRecorder()
	router.ServeHTTP(authorizedResult, authorized)
	require.Equal(t, http.StatusCreated, authorizedResult.Code)

	unpaid := httptest.NewRequest(http.MethodPost, "/api/v1/x402/prices/price-1/admit", nil)
	unpaidResult := httptest.NewRecorder()
	router.ServeHTTP(unpaidResult, unpaid)
	require.Equal(t, http.StatusPaymentRequired, unpaidResult.Code)
	require.NotEmpty(t, unpaidResult.Header().Get("Payment-Required"))
}

func fixturePrice() x402rail.Price {
	return x402rail.Price{
		ID: "price-1", ResourceURL: "https://service.example/paid", Description: "fixture", MIMEType: "application/json",
		Network: "eip155:8453", Scheme: "exact", Amount: "10000", AmountMinor: 1, Currency: "USD",
		PayTo: "0x3333333333333333333333333333333333333333", Asset: "0x2222222222222222222222222222222222222222",
		AssetDecimals: 6, MaxTimeoutSeconds: 300,
		ExtraJSON: `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
		CreatedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
}

type fixtureRepository struct{ price x402rail.Price }

func (r *fixtureRepository) CreatePrice(_ context.Context, price x402rail.Price) (x402rail.Price, error) {
	r.price = price
	return price, nil
}

func (r *fixtureRepository) GetPrice(context.Context, string) (x402rail.Price, error) {
	return r.price, nil
}

func (*fixtureRepository) GetAdmissionByDigest(context.Context, string) (x402rail.Admission, error) {
	return x402rail.Admission{}, x402rail.ErrNotFound
}

func (*fixtureRepository) ClaimAdmission(context.Context, x402rail.Admission) (x402rail.Admission, bool, error) {
	panic("not used")
}

func (*fixtureRepository) CompleteAdmission(context.Context, string, string, x402rail.SettleResult, time.Time, x402rail.Price) (x402rail.Admission, error) {
	panic("not used")
}

type fixtureFacilitator struct{}

func (fixtureFacilitator) Verify(context.Context, json.RawMessage, json.RawMessage) (x402rail.VerifyResult, error) {
	panic("not used")
}

func (fixtureFacilitator) Settle(context.Context, json.RawMessage, json.RawMessage) (x402rail.SettleResult, error) {
	panic("not used")
}
