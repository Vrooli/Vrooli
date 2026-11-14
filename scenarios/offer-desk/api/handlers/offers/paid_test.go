package offers

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

func TestPaidCatalogForwards402AndOnlyReleasesContentAfterSettlement(t *testing.T) { // [REQ:CF4-OFFER-DESK-METERING]
	clock := schedule.NewFake(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	_, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Metered research inference", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)

	var settled bool
	var seenPayment string
	treasury := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/x402/prices/cf4-research-inference/admit", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		seenPayment = r.Header.Get("Payment-Signature")
		if !settled {
			w.Header().Set("Payment-Required", "eyJ4NDAyVmVyc2lvbiI6Mn0=")
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"error":"payment signature is required"}`))
			return
		}
		w.Header().Set("Payment-Response", base64.StdEncoding.EncodeToString([]byte(`{"success":true,"payer":"0xabc","transaction":"0x123","network":"eip155:8453"}`)))
		w.WriteHeader(http.StatusOK)
	}))
	defer treasury.Close()
	t.Setenv("TREASURY_API_URL", treasury.URL)

	serve := func(payment string) *httptest.ResponseRecorder {
		router := mux.NewRouter()
		router.HandleFunc(paidOffersPath, s.paidOffers).Methods(http.MethodGet)
		req := httptest.NewRequest(http.MethodGet, paidOffersPath, nil)
		if payment != "" {
			req.Header.Set("Payment-Signature", payment)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	challenge := serve("")
	require.Equal(t, http.StatusPaymentRequired, challenge.Code)
	require.Equal(t, "eyJ4NDAyVmVyc2lvbiI6Mn0=", challenge.Header().Get("Payment-Required"))
	require.NotContains(t, challenge.Body.String(), "Metered research inference", "content must stay withheld before admission")

	settled = true
	paid := serve("signed-x402-v2-payload")
	require.Equal(t, "signed-x402-v2-payload", seenPayment)
	require.Equal(t, http.StatusOK, paid.Code)
	require.NotEmpty(t, paid.Header().Get("Payment-Response"))
	require.True(t, strings.Contains(paid.Body.String(), "Metered research inference"))
}

func TestPaidCatalogFailsClosedWhenTreasuryDoesNotConfirmSettlement(t *testing.T) { // [REQ:CF4-OFFER-DESK-METERING]
	clock := schedule.NewFake(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	s, _ := testOfferService(t, clock)
	treasury := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer treasury.Close()
	t.Setenv("TREASURY_API_URL", treasury.URL)

	router := mux.NewRouter()
	router.HandleFunc(paidOffersPath, s.paidOffers).Methods(http.MethodGet)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, paidOffersPath, nil)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.NotContains(t, response.Body.String(), "nodes")
}

func TestPaidCatalogRejectsIncompleteTreasuryReceipt(t *testing.T) { // [REQ:CF4-OFFER-DESK-METERING]
	clock := schedule.NewFake(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	s, ctx := testOfferService(t, clock)
	_, err := s.store.CreateNode(ctx, offerspb.NodeKind_OFFER, "Must remain gated", offerspb.Status_ACTIVE, "", "")
	require.NoError(t, err)
	treasury := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Payment-Response", base64.StdEncoding.EncodeToString([]byte(`{"success":false}`)))
		w.WriteHeader(http.StatusOK)
	}))
	defer treasury.Close()
	t.Setenv("TREASURY_API_URL", treasury.URL)
	router := mux.NewRouter()
	router.HandleFunc(paidOffersPath, s.paidOffers).Methods(http.MethodGet)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, paidOffersPath, nil))
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.NotContains(t, response.Body.String(), "Must remain gated")
}

func TestCF7OverCapPaidCatalogIsOptInAndUsesItsOwnTreasuryPrice(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	s, _ := testOfferService(t, clock)
	t.Setenv("OFFER_DESK_CF7_TEST_SECOND_PRICE", "0")
	router := mux.NewRouter()
	router.HandleFunc(cf7OverCapOffersPath, s.paidOffersOverCap).Methods(http.MethodGet)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, cf7OverCapOffersPath, nil))
	require.Equal(t, http.StatusNotFound, response.Code, "the rehearsal-only route is absent unless explicitly enabled")

	var admissionPath string
	treasury := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		admissionPath = r.URL.Path
		w.Header().Set("Payment-Required", "eyJ4NDAyVmVyc2lvbiI6Mn0=")
		w.WriteHeader(http.StatusPaymentRequired)
	}))
	defer treasury.Close()
	t.Setenv("TREASURY_API_URL", treasury.URL)
	t.Setenv("OFFER_DESK_CF7_TEST_SECOND_PRICE", "1")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, cf7OverCapOffersPath, nil))
	require.Equal(t, http.StatusPaymentRequired, response.Code)
	require.Equal(t, "/api/v1/x402/prices/"+cf7OverCapOffersPriceID+"/admit", admissionPath)
	require.NotContains(t, response.Body.String(), "nodes", "the catalog remains withheld until Treasury confirms settlement")
}
