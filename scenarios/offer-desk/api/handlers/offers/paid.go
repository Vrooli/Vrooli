package offers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/encoding/protojson"
	"offer-desk/internal/module"
)

var paidCatalogDropOnce sync.Once

const (
	paidOffersPath          = "/api/v1/paid/catalog"
	paidOffersPriceID       = "cf4-research-inference"
	cf7OverCapOffersPath    = "/api/v1/paid/catalog/over-cap"
	cf7OverCapOffersPriceID = "cf7-research-inference-overcap"
)

// paidOffers provides a real Offer Desk resource behind Treasury's x402
// admission route. The price is declared in Treasury from Offer Desk's
// sells_at edge; this endpoint forwards the challenge/payment headers and only
// releases the catalog response after Treasury confirms facilitator settlement.
func (s *Service) paidOffers(writer http.ResponseWriter, request *http.Request) {
	s.servePaidOffers(writer, request, paidOffersPriceID)
}

// paidOffersOverCap is an opt-in local-rehearsal endpoint for the fourth CF7
// quote. It is absent unless the isolated rehearsal explicitly enables it.
func (s *Service) paidOffersOverCap(writer http.ResponseWriter, request *http.Request) {
	if s.getenv("OFFER_DESK_CF7_TEST_SECOND_PRICE") != "1" {
		http.NotFound(writer, request)
		return
	}
	s.servePaidOffers(writer, request, cf7OverCapOffersPriceID)
}

func (s *Service) servePaidOffers(writer http.ResponseWriter, request *http.Request, priceID string) {
	resolveTreasuryURL := s.resolveTreasuryURL
	if resolveTreasuryURL == nil {
		writePaidError(writer, http.StatusServiceUnavailable, "Treasury x402 admission is unavailable")
		return
	}
	base, err := resolveTreasuryURL(request.Context())
	if err != nil {
		writePaidError(writer, http.StatusServiceUnavailable, "Treasury x402 admission is unavailable")
		return
	}
	admissionURL := strings.TrimRight(base, "/") + "/api/v1/x402/prices/" + url.PathEscape(priceID) + "/admit"
	admissionRequest, err := http.NewRequestWithContext(request.Context(), http.MethodGet, admissionURL, nil)
	if err != nil {
		writePaidError(writer, http.StatusInternalServerError, "create Treasury admission request")
		return
	}
	if payment := strings.TrimSpace(request.Header.Get("Payment-Signature")); payment != "" {
		admissionRequest.Header.Set("Payment-Signature", payment)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(admissionRequest)
	if err != nil {
		writePaidError(writer, http.StatusServiceUnavailable, "Treasury x402 admission request failed")
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusPaymentRequired {
		if challenge := response.Header.Get("Payment-Required"); challenge != "" {
			writer.Header().Set("Payment-Required", challenge)
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusPaymentRequired)
		_, _ = io.Copy(writer, io.LimitReader(response.Body, 64<<10))
		return
	}
	if response.StatusCode != http.StatusOK {
		writePaidError(writer, http.StatusServiceUnavailable, fmt.Sprintf("Treasury admission returned HTTP %d", response.StatusCode))
		return
	}
	paymentResponse := strings.TrimSpace(response.Header.Get("Payment-Response"))
	if paymentResponse == "" || !validSettlementReceipt(paymentResponse) {
		writePaidError(writer, http.StatusServiceUnavailable, "Treasury admission omitted a complete settlement receipt")
		return
	}
	if s.getenv("OFFER_DESK_TEST_DROP_FIRST_PAID_RESPONSE") == "1" {
		drop := false
		paidCatalogDropOnce.Do(func() { drop = true })
		if drop {
			if hijacker, ok := writer.(http.Hijacker); ok {
				connection, _, hijackErr := hijacker.Hijack()
				if hijackErr == nil {
					_ = connection.Close()
					return
				}
			}
			writePaidError(writer, http.StatusBadGateway, "test fixture could not drop the post-admission response")
			return
		}
	}
	nodes, err := s.store.ListNodes(request.Context(), offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		writePaidError(writer, http.StatusInternalServerError, "load Offer Desk catalog")
		return
	}
	body, err := protojson.Marshal(&offerspb.ListNodesResponse{Nodes: nodes})
	if err != nil {
		writePaidError(writer, http.StatusInternalServerError, "encode Offer Desk catalog")
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Payment-Response", paymentResponse)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(body)
}

func writePaidError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": message})
}

var paidOffersEndpoint = module.EndpointDescriptor{
	ID: "paid_offer_catalog", Path: paidOffersPath, Method: http.MethodGet,
	Summary: "Serve the Offer Desk catalog after Treasury x402 settlement", Category: "x402",
	RESTException: &module.RESTException{Reason: module.RESTReasonThirdPartyShape, Note: "x402 challenge and receipt headers follow the external HTTP protocol."},
}

var cf7OverCapPaidOffersEndpoint = module.EndpointDescriptor{
	ID: "cf7_over_cap_paid_offer_catalog", Path: cf7OverCapOffersPath, Method: http.MethodGet,
	Summary: "Serve the local CF7 over-cap catalog quote after a separate Treasury x402 settlement", Category: "x402",
	RESTException: &module.RESTException{Reason: module.RESTReasonThirdPartyShape, Note: "enabled only for the isolated local-fork rehearsal."},
}

func validSettlementReceipt(encoded string) bool {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	var receipt struct {
		Success     bool   `json:"success"`
		Payer       string `json:"payer"`
		Transaction string `json:"transaction"`
		Network     string `json:"network"`
	}
	if json.Unmarshal(decoded, &receipt) != nil {
		return false
	}
	return receipt.Success && strings.TrimSpace(receipt.Payer) != "" && strings.TrimSpace(receipt.Transaction) != "" && strings.TrimSpace(receipt.Network) != ""
}
