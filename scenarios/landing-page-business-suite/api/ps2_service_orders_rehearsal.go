//go:build lpbs_rehearsal

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/handlers/serviceorders"
	"landing-page-business-suite-api/internal/commerce"
)

func configurePS2ServiceOrders(s *Server) error {
	if strings.TrimSpace(os.Getenv("LPBS_SERVICE_ORDER_TEST_MODE")) != "1" {
		return nil
	}
	if isProductionEnvironment() {
		return errors.New("PS2 test service-order routes cannot be enabled in a production environment")
	}
	offerURL := strings.TrimSpace(os.Getenv("OFFER_DESK_API_URL"))
	treasuryURL := strings.TrimSpace(os.Getenv("TREASURY_API_URL"))
	authenticatorURL := strings.TrimSpace(os.Getenv("SCENARIO_AUTHENTICATOR_API_URL"))
	moneyLedgerURL := strings.TrimSpace(os.Getenv("MONEY_LEDGER_API_URL"))
	deliveryDir := strings.TrimSpace(os.Getenv("LPBS_REHEARSAL_DELIVERY_DIR"))
	adapterID := strings.TrimSpace(os.Getenv("LPBS_TEST_SERVICE_ORDER_ADAPTER_ID"))
	bookID := strings.TrimSpace(os.Getenv("LPBS_TEST_SERVICE_ORDER_BOOK_ID"))
	revenueAccountID := strings.TrimSpace(os.Getenv("LPBS_TEST_SERVICE_ORDER_REVENUE_ACCOUNT_ID"))
	if offerURL == "" || treasuryURL == "" || authenticatorURL == "" || !filepath.IsAbs(deliveryDir) {
		return errors.New("PS2 rehearsal requires Offer Desk, Treasury, Scenario Authenticator, and absolute delivery-directory settings")
	}
	ledgerConfigured := moneyLedgerURL != "" || adapterID != "" || bookID != "" || revenueAccountID != ""
	if ledgerConfigured && (moneyLedgerURL == "" || adapterID == "" || bookID == "" || revenueAccountID == "") {
		return errors.New("PS2 rehearsal Money Ledger service URL and test routing must be configured together")
	}
	offers, err := commerce.NewOfferDeskServiceOrderSource(offerURL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		return fmt.Errorf("configure canonical Offer Desk source: %w", err)
	}
	orderTokens := commerce.NewTreasuryX402OrderTokens(rehearsalServicePrincipalCredential("treasury-x402-order"), authenticatorURL, nil)
	orderClient, err := commerce.NewTreasuryX402HTTPClient(treasuryURL, orderTokens, nil)
	if err != nil {
		return fmt.Errorf("configure Treasury order client: %w", err)
	}
	provider, err := commerce.NewTreasuryX402ServiceOrderProvider(orderClient)
	if err != nil {
		return fmt.Errorf("configure Treasury test provider: %w", err)
	}
	localProvider := commerce.NewLocalTestServiceOrderProvider()
	providers, err := commerce.NewServiceOrderProviderRouter(localProvider, provider)
	if err != nil {
		return fmt.Errorf("configure service-order provider router: %w", err)
	}
	receiptTokens := commerce.NewTreasuryX402ReceiptTokens(rehearsalServicePrincipalCredential("treasury-x402-receipt"), authenticatorURL, nil)
	receipts, err := commerce.NewTreasuryX402ReceiptReader(treasuryURL, receiptTokens, nil)
	if err != nil {
		return fmt.Errorf("configure Treasury accepted-receipt reader: %w", err)
	}
	var ledger commerce.ServiceOrderLedgerWriter
	var ledgerTarget commerce.ServiceOrderLedgerTarget
	if ledgerConfigured {
		ledgerTokens := commerce.NewMoneyLedgerServiceOrderTokens(rehearsalServicePrincipalCredential("money-ledger"), authenticatorURL, nil)
		ledger, err = commerce.NewMoneyLedgerTestServiceWriter(moneyLedgerURL, ledgerTokens, nil)
		if err != nil {
			return fmt.Errorf("configure Money Ledger test writer: %w", err)
		}
		ledgerTarget = commerce.ServiceOrderLedgerTarget{
			FixtureID: "money-ledger-test-service:ps2-local-fork", AdapterID: adapterID,
			BookID: bookID, RevenueAccountID: revenueAccountID,
		}
	}
	verifier, err := newPS2RehearsalFileVerifier(deliveryDir)
	if err != nil {
		return fmt.Errorf("configure PS2 delivery verifier: %w", err)
	}
	runtime, err := commerce.NewServiceOrderSandbox(commerce.ServiceOrderSandboxOptions{
		DB:               s.routedDB,
		Provider:         providers,
		Verifier:         verifier,
		Ledger:           ledger,
		TreasuryReceipts: receipts,
		LedgerTarget:     ledgerTarget,
	})
	if err != nil {
		return fmt.Errorf("configure durable PS2 service-order runtime: %w", err)
	}
	handler, err := serviceorders.NewHandler(serviceorders.Dependencies{Runtime: runtime, Offers: offers, Results: verifier, TestMode: true})
	if err != nil {
		return fmt.Errorf("configure PS2 service-order transport: %w", err)
	}
	s.serviceOrderHTTP = handler
	return nil
}

func rehearsalServicePrincipalCredential(name string) servicetoken.CredentialSource {
	return func(context.Context) (string, error) {
		authority, err := openLPBSRehearsalAuthority()
		if err != nil {
			return "", fmt.Errorf("open isolated rehearsal credential authority: %w", err)
		}
		identity, err := credentialauthority.ParseIdentity("scenario-authenticator/service-principal/landing-page-business-suite/" + name)
		if err != nil {
			return "", fmt.Errorf("parse rehearsal service-principal identity: %w", err)
		}
		credential, err := authority.Require(identity, "credential")
		if err != nil {
			return "", fmt.Errorf("resolve scoped rehearsal service credential: %w", err)
		}
		return credential, nil
	}
}

type ps2DeliveryArtifact struct {
	JobID        string          `json:"job_id"`
	OfferID      string          `json:"offer_id"`
	OfferVersion int             `json:"offer_version"`
	JobInput     json.RawMessage `json:"job_input"`
	Report       string          `json:"report_markdown"`
}

type ps2RehearsalFileVerifier struct{ root string }

func newPS2RehearsalFileVerifier(root string) (*ps2RehearsalFileVerifier, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return nil, errors.New("delivery artifact directory must be an absolute clean path")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("read delivery artifact directory: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("delivery artifact path is not a directory")
	}
	return &ps2RehearsalFileVerifier{root: root}, nil
}

func (v *ps2RehearsalFileVerifier) Verify(_ context.Context, order commerce.ServiceOrder) (commerce.ServiceOrderVerification, error) {
	_, receiptID, passed, err := v.readArtifact(order)
	if err != nil {
		return commerce.ServiceOrderVerification{}, err
	}
	return commerce.ServiceOrderVerification{ReceiptID: receiptID, Passed: passed}, nil
}

func (v *ps2RehearsalFileVerifier) ReadResult(_ context.Context, order commerce.ServiceOrder) (serviceorders.DeliveryResult, error) {
	if order.State != commerce.ServiceOrderReconciled {
		return serviceorders.DeliveryResult{}, errors.New("PS2 test result is available only after order reconciliation")
	}
	artifact, receiptID, passed, err := v.readArtifact(order)
	if err != nil {
		return serviceorders.DeliveryResult{}, err
	}
	if !passed || order.VerificationID != receiptID {
		return serviceorders.DeliveryResult{}, errors.New("PS2 test result does not match the verified order artifact")
	}
	return serviceorders.DeliveryResult{Format: "markdown", Body: []byte(artifact.Report)}, nil
}

func (v *ps2RehearsalFileVerifier) readArtifact(order commerce.ServiceOrder) (ps2DeliveryArtifact, string, bool, error) {
	if v == nil || v.root == "" || strings.TrimSpace(order.JobID) == "" {
		return ps2DeliveryArtifact{}, "", false, errors.New("PS2 rehearsal delivery verifier is not bound to a durable job")
	}
	path := filepath.Join(v.root, order.JobID+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return ps2DeliveryArtifact{}, "", false, fmt.Errorf("read durable PS2 delivery artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 2<<20 {
		return ps2DeliveryArtifact{}, "", false, errors.New("PS2 delivery artifact must be a nonempty regular file under 2 MiB")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ps2DeliveryArtifact{}, "", false, fmt.Errorf("read PS2 delivery artifact: %w", err)
	}
	var artifact ps2DeliveryArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return ps2DeliveryArtifact{}, "", false, fmt.Errorf("decode PS2 delivery artifact: %w", err)
	}
	inputMatches := compactJSON(artifact.JobInput, order.JobInput)
	passed := artifact.JobID == order.JobID && artifact.OfferID == order.OfferID && artifact.OfferVersion == order.OfferVersion && inputMatches && validPS2MarketScanReport(artifact.Report, order.JobInput)
	digest := sha256.Sum256(append([]byte(order.JobID+":"), raw...))
	return artifact, "ps2-delivery:" + hex.EncodeToString(digest[:]), passed, nil
}

func compactJSON(left, right json.RawMessage) bool {
	var compactLeft, compactRight bytes.Buffer
	if json.Compact(&compactLeft, left) != nil || json.Compact(&compactRight, right) != nil {
		return false
	}
	return bytes.Equal(compactLeft.Bytes(), compactRight.Bytes())
}

func validPS2MarketScanReport(report string, input json.RawMessage) bool {
	if len(report) < 200 || !strings.Contains(report, "## Scope") || !strings.Contains(report, "## Source") || !strings.Contains(report, "## Not checked") || !strings.Contains(report, "https://") {
		return false
	}
	var request map[string]json.RawMessage
	if json.Unmarshal(input, &request) != nil || len(request) == 0 {
		return false
	}
	return true
}
