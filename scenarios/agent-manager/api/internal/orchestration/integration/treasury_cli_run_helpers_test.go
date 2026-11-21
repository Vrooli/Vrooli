//go:build linux

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-manager/internal/orchestration"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	fundingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding/funding_v1connect"
)

type cf1TreasuryOperatorVerifier struct{}

func (cf1TreasuryOperatorVerifier) Verify(_ context.Context, token string) (coreidentity.Principal, error) {
	if token != cf1TreasuryOperator {
		return coreidentity.Principal{}, errors.New("invalid local CF1 operator fixture")
	}
	policy, ok := authn.InteractivePolicy("treasury-spending")
	if !ok {
		return coreidentity.Principal{}, errors.New("Treasury spending policy unavailable")
	}
	scopes := []string{"agent-manager:write", "agent-manager:read", "agent-manager:supervise"}
	for _, scope := range policy.Scopes {
		found := false
		for _, existing := range scopes {
			found = found || existing == scope
		}
		if !found {
			scopes = append(scopes, scope)
		}
	}
	readPolicy, ok := authn.InteractivePolicy("treasury-wallet-reading")
	if !ok {
		return coreidentity.Principal{}, errors.New("Treasury wallet-reading policy unavailable")
	}
	for _, scope := range readPolicy.Scopes {
		found := false
		for _, existing := range scopes {
			found = found || existing == scope
		}
		if !found {
			scopes = append(scopes, scope)
		}
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "cf1-local-operator", Scopes: scopes, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type cf1RunSessionKeys struct {
	store *orchestration.CredentialAuthorityTreasurySessionKeys
	path  string
}

func (s *cf1RunSessionKeys) ResolveOrMint(ctx context.Context, id uuid.UUID) (string, error) {
	return s.store.ResolveOrMint(ctx, id)
}

func (s *cf1RunSessionKeys) Acquire(ctx context.Context, id uuid.UUID) (string, func(), error) {
	return s.store.Acquire(ctx, id)
}

func (s *cf1RunSessionKeys) AcquireContinuation(ctx context.Context, id uuid.UUID) (string, func(), error) {
	return s.store.AcquireContinuation(ctx, id)
}

func (s *cf1RunSessionKeys) AcquireRotation(ctx context.Context, id uuid.UUID, requestID string) (string, func(), error) {
	return s.store.AcquireRotation(ctx, id, requestID)
}

func (s *cf1RunSessionKeys) Rotate(ctx context.Context, id uuid.UUID, requestID string) (string, error) {
	return s.store.Rotate(ctx, id, requestID)
}

func (s *cf1RunSessionKeys) RevokeRunCredentialUse(ctx context.Context, id uuid.UUID, reason string) error {
	return s.store.RevokeRunCredentialUse(ctx, id, reason)
}

func (s *cf1RunSessionKeys) WasRevoked(id uuid.UUID) bool {
	identity, err := credentialauthority.ParseIdentity("vrooli/agent-manager-run-" + id.String())
	if err != nil {
		return false
	}
	authority, err := credentialauthority.OpenEncryptedFile(s.path, cf1RunKeyPassphrase)
	if err != nil {
		return false
	}
	value, err := authority.Resolve(identity, cf1RunKeyField)
	if err != nil {
		return false
	}
	var state struct {
		Version int    `json:"version"`
		Key     string `json:"key"`
		Revoked bool   `json:"revoked"`
	}
	if json.Unmarshal([]byte(value), &state) != nil {
		return false
	}
	return state.Version == 1 && state.Revoked && state.Key == ""
}

func cf1LiveStaleSessionKeyProbe(ctx context.Context, apiBase, identityToken, budgetID, mandateID, oldPublicKey string) (string, error) {
	previousToken, hadToken := os.LookupEnv(cliutil.EnvIdentityToken)
	if err := os.Setenv(cliutil.EnvIdentityToken, identityToken); err != nil {
		return "", err
	}
	defer func() {
		if hadToken {
			_ = os.Setenv(cliutil.EnvIdentityToken, previousToken)
		} else {
			_ = os.Unsetenv(cliutil.EnvIdentityToken)
		}
	}()
	credentials, err := cliutil.ResolveAgentCredentialsForOperation(ctx, "treasury.read", "treasury:budget/"+budgetID)
	if err != nil {
		return "", fmt.Errorf("resolve current ordinary-run read grant for stale-key probe: %w", err)
	}
	request := connect.NewRequest(&fundingv1.GetBudgetWalletRequest{BudgetId: budgetID, MandateId: mandateID})
	request.Header().Set(cliutil.HeaderAgentIdentityToken, credentials.BaselineToken)
	request.Header().Set(cliutil.HeaderAgentOperationToken, credentials.OperationToken)
	request.Header().Set("X-Vrooli-Treasury-Session-Public-Key", oldPublicKey)
	_, err = fundingconnect.NewFundingAgentServiceClient(http.DefaultClient, apiBase).GetBudgetWallet(ctx, request)
	if err == nil {
		return "accepted", errors.New("Treasury accepted the retired generation-1 public key after rotation")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		return connect.CodeOf(err).String(), fmt.Errorf("retired generation-1 key was refused with unexpected status %s: %w", connect.CodeOf(err), err)
	}
	return connect.CodeOf(err).String(), nil
}

type cf1TreasuryRunConfig struct {
	APIBaseURL            string                          `json:"api_base_url"`
	CLIBinary             string                          `json:"cli_binary"`
	BudgetID              string                          `json:"budget_id"`
	MandateID             string                          `json:"mandate_id"`
	IsolationBudgetID     string                          `json:"isolation_budget_id"`
	IsolationMandateID    string                          `json:"isolation_mandate_id"`
	ForeignBudgetID       string                          `json:"foreign_budget_id"`
	ForeignMandateID      string                          `json:"foreign_mandate_id"`
	CounterpartIdentity   *cf1TreasuryCounterpartIdentity `json:"counterpart_identity,omitempty"`
	PairedProbeBarrierDir string                          `json:"paired_probe_barrier_dir,omitempty"`
	Counterparty          string                          `json:"counterparty"`
	InCapInstrumentID     string                          `json:"in_cap_instrument_id"`
	InCapAmountMinor      int64                           `json:"in_cap_amount_minor"`
	InCapAuthorization    string                          `json:"in_cap_authorization_id"`
	InCapSettlement       string                          `json:"in_cap_settlement_id"`
	InCapIdempotency      string                          `json:"in_cap_idempotency_key"`
	OverCapInstrumentID   string                          `json:"over_cap_instrument_id"`
	OverCapAmountMinor    int64                           `json:"over_cap_amount_minor"`
	OverCapAuthorization  string                          `json:"over_cap_authorization_id"`
	OverCapSettlement     string                          `json:"over_cap_settlement_id"`
	OverCapIdempotency    string                          `json:"over_cap_idempotency_key"`
	ExpiryMandateID       string                          `json:"expiry_mandate_id"`
	ExpiryAuthorization   string                          `json:"expiry_authorization_id"`
	ExpirySettlement      string                          `json:"expiry_settlement_id"`
	ExpiryInstrumentID    string                          `json:"expiry_instrument_id"`
	ExpiryIdempotency     string                          `json:"expiry_idempotency_key"`
	InCapGrantID          string                          `json:"in_cap_operation_grant_id"`
	OverCapGrantID        string                          `json:"over_cap_operation_grant_id"`
	// CF4 is an additive extension: older CF1 local-fork rehearsals omit these
	// fields and retain their existing flow unchanged.
	X402CatalogURL      string                `json:"x402_catalog_url"`
	X402PriceID         string                `json:"x402_price_id"`
	X402CallID          string                `json:"x402_call_id"`
	X402Counterparty    string                `json:"x402_counterparty"`
	X402InstrumentID    string                `json:"x402_instrument_id"`
	X402AmountMinor     int64                 `json:"x402_expected_amount_minor"`
	X402AuthorizationID string                `json:"x402_authorization_id"`
	X402SettlementID    string                `json:"x402_settlement_id"`
	X402IdempotencyKey  string                `json:"x402_idempotency_key"`
	X402Calls           []cf1TreasuryX402Call `json:"x402_calls"`
}

func (config cf1TreasuryRunConfig) foreignProbeIDs() (string, string, error) {
	if config.ForeignBudgetID != "" || config.ForeignMandateID != "" {
		if config.ForeignBudgetID == "" || config.ForeignMandateID == "" {
			return "", "", errors.New("cross-budget refusal probes require both foreign budget and mandate IDs")
		}
		if config.ForeignBudgetID == config.BudgetID || config.ForeignMandateID == config.MandateID {
			return "", "", errors.New("counterpart budget and mandate IDs must both differ from the current run")
		}
		if config.CounterpartIdentity != nil && (config.ForeignBudgetID != config.CounterpartIdentity.BudgetID || config.ForeignMandateID != config.CounterpartIdentity.MandateID) {
			return "", "", errors.New("foreign probe IDs do not match the attested counterpart wallet tuple")
		}
		return config.ForeignBudgetID, config.ForeignMandateID, nil
	}
	if config.IsolationBudgetID == "" || config.IsolationMandateID == "" {
		return "", "", errors.New("cross-budget refusal probes require a complete counterpart or isolation budget/mandate pair")
	}
	return config.IsolationBudgetID, config.IsolationMandateID, nil
}

type cf1TreasuryCounterpartIdentity struct {
	RunID             string `json:"run_id"`
	BudgetID          string `json:"budget_id"`
	MandateID         string `json:"mandate_id"`
	WalletID          string `json:"wallet_id"`
	WalletAddress     string `json:"wallet_address"`
	WalletRunID       string `json:"wallet_run_id"`
	WalletBudgetID    string `json:"wallet_budget_id"`
	WalletMandateID   string `json:"wallet_mandate_id"`
	MandateStatus     string `json:"mandate_status"`
	SessionGeneration int64  `json:"session_generation"`
}

func validateCF1TreasuryCounterpartIdentity(config cf1TreasuryRunConfig, runID string) error {
	if config.PairedProbeBarrierDir == "" {
		if config.CounterpartIdentity != nil {
			return errors.New("counterpart identity requires a paired-probe barrier directory")
		}
		return nil
	}
	counterpart := config.CounterpartIdentity
	if counterpart == nil || config.ForeignBudgetID == "" || config.ForeignMandateID == "" {
		return errors.New("paired probes require explicit counterpart identity and foreign budget/mandate IDs")
	}
	if counterpart.RunID == "" || counterpart.RunID == runID || counterpart.BudgetID != config.ForeignBudgetID || counterpart.MandateID != config.ForeignMandateID || counterpart.BudgetID == config.BudgetID || counterpart.MandateID == config.MandateID {
		return errors.New("paired probe counterpart run/budget/mandate IDs are missing, self-bound, or inconsistent")
	}
	if counterpart.WalletID == "" || !cf1HexIdentity(counterpart.WalletAddress, 20) || counterpart.WalletRunID != counterpart.RunID || counterpart.WalletBudgetID != counterpart.BudgetID || counterpart.WalletMandateID != counterpart.MandateID {
		return errors.New("paired probe counterpart wallet is not bound to its run/budget/mandate identity")
	}
	if counterpart.MandateStatus != "LIVE" || counterpart.SessionGeneration <= 0 {
		return errors.New("paired probe counterpart mandate/session is not live")
	}
	if strings.TrimSpace(os.Getenv(treasuryCF1CrossBudgetEvidencePathEnv)) == "" {
		return errors.New("paired probes require one shared cross-budget identity evidence path")
	}
	if !filepath.IsAbs(config.PairedProbeBarrierDir) || filepath.Clean(config.PairedProbeBarrierDir) != config.PairedProbeBarrierDir {
		return errors.New("paired probe barrier path must be absolute and clean")
	}
	info, err := os.Lstat(config.PairedProbeBarrierDir)
	if err != nil {
		return fmt.Errorf("inspect paired probe barrier directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return errors.New("paired probe barrier directory must be a private real directory")
	}
	return nil
}

// cf2AgentManagerOrderConfig is the narrow hand-off from the local-fork
// rehearsal into an ordinary Agent Manager run. The payment signature lives
// in a separate mode-0600 file and is never copied into run evidence.
type cf2AgentManagerOrderConfig struct {
	LPBSURL              string `json:"lpbs_url"`
	PaymentSignaturePath string `json:"payment_signature_path"`
	DeliveryDir          string `json:"delivery_dir"`
}

type cf2AgentManagerOrder struct {
	OrderID            string          `json:"order_id"`
	JobID              string          `json:"job_id"`
	AgentManagerRunID  string          `json:"agent_manager_run_id"`
	OfferID            string          `json:"offer_id"`
	OfferVersion       int             `json:"offer_version"`
	JobInput           json.RawMessage `json:"job_input"`
	AmountMinor        int64           `json:"amount_minor"`
	Currency           string          `json:"currency"`
	Mode               string          `json:"mode"`
	Door               string          `json:"door"`
	State              string          `json:"state"`
	AuthorizationID    string          `json:"authorization_id"`
	CaptureID          string          `json:"capture_id"`
	LedgerPostingID    string          `json:"ledger_posting_id"`
	LedgerReceiptID    string          `json:"ledger_receipt_id"`
	TreasuryExternalID string          `json:"treasury_external_id"`
}

// executeCF2AgentManagerOrder is the ordinary-run side of the CF2 rehearsal.
// It asks LPBS for the canonical quote, begins the durable order with the run
// ID stored on it, and lets LPBS verify delivery before capture. The local-fork
// harness owns payment signing and the restart/readback; this run reconciles
// the seller receipt after that independent Treasury evidence is confirmed.
func executeCF2AgentManagerOrder(ctx context.Context, runID string, config cf2AgentManagerOrderConfig, coord string, write func(string, any) error) (cf2AgentManagerOrder, error) {
	var empty cf2AgentManagerOrder
	if !filepath.IsAbs(config.PaymentSignaturePath) || !filepath.IsAbs(config.DeliveryDir) {
		return empty, errors.New("CF2 payment signature and delivery paths must be absolute")
	}
	base, err := url.Parse(strings.TrimSpace(config.LPBSURL))
	if err != nil || base.Scheme != "http" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Path != "" && base.Path != "/" || !isLoopbackHostForIntegration(base.Hostname()) {
		return empty, errors.New("CF2 Agent Manager run requires a loopback LPBS test API origin")
	}
	baseURL := strings.TrimRight(base.String(), "/")
	client := &http.Client{Timeout: 15 * time.Second}
	quoteURL := baseURL + "/api/v1/service-orders/offer?offer_id=company-market-scan&offer_version=1"
	quoteRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, quoteURL, nil)
	if err != nil {
		return empty, err
	}
	quoteResponse, err := client.Do(quoteRequest)
	if err != nil {
		return empty, fmt.Errorf("read canonical LPBS company-market-scan quote: %w", err)
	}
	var quote struct {
		OfferID      string `json:"offer_id"`
		OfferVersion int    `json:"offer_version"`
		AmountMinor  int64  `json:"amount_minor"`
		Currency     string `json:"currency"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(quoteResponse.Body, 1<<20)).Decode(&quote)
	quoteResponse.Body.Close()
	if quoteResponse.StatusCode != http.StatusOK || decodeErr != nil || quote.OfferID != "company-market-scan" || quote.OfferVersion != 1 || quote.AmountMinor != 4900 || quote.Currency != "USD" {
		return empty, fmt.Errorf("LPBS canonical quote did not match company-market-scan v1 at 4900 USD (status=%d decode=%v quote=%+v)", quoteResponse.StatusCode, decodeErr, quote)
	}
	if err := write("agent-manager-lpbs-canonical-quote.json", quote); err != nil {
		return empty, err
	}

	secret, err := os.ReadFile(config.PaymentSignaturePath)
	if err != nil {
		return empty, fmt.Errorf("read one-use synthetic local-fork payment signature: %w", err)
	}
	signatureInfo, err := os.Stat(config.PaymentSignaturePath)
	if err != nil || !signatureInfo.Mode().IsRegular() || signatureInfo.Mode().Perm()&0o077 != 0 {
		return empty, errors.New("CF2 payment signature must be held in a regular owner-only file")
	}
	paymentSignature := strings.TrimSpace(string(secret))
	if paymentSignature == "" {
		return empty, errors.New("CF2 local-fork payment signature is empty")
	}
	jobInput, err := json.Marshal(map[string]string{
		"sector": "payment infrastructure",
	})
	if err != nil {
		return empty, err
	}
	idempotencyKey := "cf2-agent-manager-run-" + runID
	beginBody, err := json.Marshal(map[string]any{
		"offer_id": "company-market-scan", "offer_version": 1, "door": "agent_manager_x402",
		"idempotency_key": idempotencyKey, "agent_manager_run_id": runID, "job_input": json.RawMessage(jobInput),
		"payment_signature": paymentSignature,
	})
	if err != nil {
		return empty, err
	}
	beginStatus, beginBodyRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, baseURL+"/api/v1/service-orders", beginBody)
	if err != nil || beginStatus != http.StatusOK {
		return empty, fmt.Errorf("begin LPBS service order from Agent Manager run (status=%d): %s (%v)", beginStatus, strings.TrimSpace(string(beginBodyRaw)), err)
	}
	var order cf2AgentManagerOrder
	if err := json.Unmarshal(beginBodyRaw, &order); err != nil {
		return empty, fmt.Errorf("decode Agent Manager LPBS order: %w", err)
	}
	if order.OrderID == "" || order.JobID == "" || order.AgentManagerRunID != runID || order.AuthorizationID == "" || order.State != "authorized" || order.Mode != "sandbox" || order.Door != "agent_manager_x402" || order.OfferID != quote.OfferID || order.OfferVersion != quote.OfferVersion || order.AmountMinor != quote.AmountMinor || order.Currency != quote.Currency || order.CaptureID != "" {
		return empty, fmt.Errorf("LPBS did not persist the run-bound authorized-but-not-captured order: %+v", order)
	}
	replayStatus, replayRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, baseURL+"/api/v1/service-orders", beginBody)
	if err != nil || replayStatus != http.StatusOK {
		return empty, fmt.Errorf("LPBS exact order replay failed (status=%d): %s (%v)", replayStatus, strings.TrimSpace(string(replayRaw)), err)
	}
	var replayOrder cf2AgentManagerOrder
	if err := json.Unmarshal(replayRaw, &replayOrder); err != nil || replayOrder.OrderID != order.OrderID || replayOrder.JobID != order.JobID || replayOrder.AuthorizationID != order.AuthorizationID || replayOrder.AgentManagerRunID != runID {
		return empty, fmt.Errorf("LPBS exact replay changed the durable Agent Manager order join: first=%+v replay=%+v decode=%v", order, replayOrder, err)
	}
	var changedRequest map[string]json.RawMessage
	if err := json.Unmarshal(beginBody, &changedRequest); err != nil {
		return empty, err
	}
	changedRequest["job_input"] = json.RawMessage(`{"sector":"changed payment infrastructure"}`)
	changedBody, err := json.Marshal(changedRequest)
	if err != nil {
		return empty, err
	}
	changedStatus, changedRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, baseURL+"/api/v1/service-orders", changedBody)
	if err != nil || changedStatus != http.StatusConflict {
		return empty, fmt.Errorf("LPBS changed-input replay did not conflict before verification or settlement (status=%d): %s (%v)", changedStatus, strings.TrimSpace(string(changedRaw)), err)
	}
	if err := write("agent-manager-lpbs-order-idempotency.json", map[string]any{
		"run_id": runID, "idempotency_key": idempotencyKey, "order_id": order.OrderID, "job_id": order.JobID,
		"authorization_id": order.AuthorizationID, "exact_replay_status": replayStatus,
		"exact_replay_order_id": replayOrder.OrderID, "exact_replay_job_id": replayOrder.JobID,
		"exact_replay_authorization_id": replayOrder.AuthorizationID, "changed_input_status": changedStatus,
		"changed_input_response": json.RawMessage(changedRaw), "capture_before_replay": order.CaptureID,
	}); err != nil {
		return empty, err
	}
	var storedInput struct {
		Sector string `json:"sector"`
	}
	var storedInputFields map[string]json.RawMessage
	if err := json.Unmarshal(order.JobInput, &storedInput); err != nil || storedInput.Sector != "payment infrastructure" || json.Unmarshal(order.JobInput, &storedInputFields) != nil || len(storedInputFields) != 1 {
		return empty, fmt.Errorf("LPBS durable job input is not the exact Company Market Scan sector input (input=%s decode=%v)", order.JobInput, err)
	}
	if err := write("agent-manager-lpbs-order-authorized.json", map[string]any{
		"run_id": runID, "order_id": order.OrderID, "job_id": order.JobID,
		"agent_manager_run_id": order.AgentManagerRunID,
		"authorization_id":     order.AuthorizationID, "state": order.State, "capture_id": order.CaptureID,
		"offer_id": order.OfferID, "offer_version": order.OfferVersion,
		"amount_minor": order.AmountMinor, "currency": order.Currency, "door": order.Door,
		"job_input": json.RawMessage(order.JobInput), "payment_signature_recorded": false,
	}); err != nil {
		return empty, err
	}

	report := "## Scope\nThis is a synthetic local-fork acceptance artifact for the canonical Company Market Scan offer. It does not represent newly completed market research.\n\n" +
		"## Source\nThe offer fixture references the public Apple filing at https://www.sec.gov/Archives/edgar/data/320193/000032019326000026/aapl-20260926.htm.\n\n" +
		"## Not checked\nNo company facts, market estimates, projections, or current vendor claims were independently researched or verified in this payment-lifecycle test.\n"
	delivery := map[string]any{
		"job_id": order.JobID, "offer_id": order.OfferID, "offer_version": order.OfferVersion,
		"job_input": json.RawMessage(order.JobInput), "report_markdown": report,
	}
	deliveryBytes, err := json.Marshal(delivery)
	if err != nil {
		return empty, err
	}
	deliveryPath := filepath.Join(config.DeliveryDir, order.JobID+".json")
	if err := os.MkdirAll(filepath.Dir(deliveryPath), 0o700); err != nil {
		return empty, err
	}
	if err := os.WriteFile(deliveryPath, deliveryBytes, 0o600); err != nil {
		return empty, err
	}
	verifyStatus, verifyRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, baseURL+"/api/v1/service-orders/"+url.PathEscape(order.OrderID)+"/verify-capture", nil)
	if err != nil || verifyStatus != http.StatusOK {
		return empty, fmt.Errorf("LPBS delivery verification/capture failed (status=%d): %s (%v)", verifyStatus, strings.TrimSpace(string(verifyRaw)), err)
	}
	var captured cf2AgentManagerOrder
	if err := json.Unmarshal(verifyRaw, &captured); err != nil {
		return empty, fmt.Errorf("decode LPBS capture response: %w", err)
	}
	if captured.OrderID != order.OrderID || captured.JobID != order.JobID || captured.AgentManagerRunID != runID || captured.AuthorizationID != order.AuthorizationID || captured.CaptureID == "" || captured.TreasuryExternalID == "" {
		return empty, fmt.Errorf("LPBS capture did not preserve the run's authorization and order identity: %+v", captured)
	}
	captureReplayStatus, captureReplayRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, baseURL+"/api/v1/service-orders/"+url.PathEscape(order.OrderID)+"/verify-capture", nil)
	if err != nil || captureReplayStatus != http.StatusOK {
		return empty, fmt.Errorf("LPBS capture replay failed (status=%d): %s (%v)", captureReplayStatus, strings.TrimSpace(string(captureReplayRaw)), err)
	}
	var captureReplay cf2AgentManagerOrder
	if err := json.Unmarshal(captureReplayRaw, &captureReplay); err != nil || captureReplay.CaptureID != captured.CaptureID || captureReplay.TreasuryExternalID != captured.TreasuryExternalID {
		return empty, fmt.Errorf("LPBS capture replay did not return the original Treasury settlement (capture=%s replay=%s decode=%v)", captured.CaptureID, captureReplay.CaptureID, err)
	}
	if err := write("agent-manager-lpbs-captured-before-restart.json", map[string]any{
		"run_id": runID, "agent_manager_run_id": captured.AgentManagerRunID,
		"order_id": captured.OrderID, "job_id": captured.JobID, "authorization_id": captured.AuthorizationID,
		"capture_id": captured.CaptureID, "treasury_external_id": captured.TreasuryExternalID,
		"state_before_restart": captured.State, "delivery_verification_preceded_capture": true,
	}); err != nil {
		return empty, err
	}
	var restartEvidence struct {
		RunID             string `json:"run_id"`
		OrderID           string `json:"order_id"`
		JobID             string `json:"job_id"`
		AuthorizationID   string `json:"authorization_id"`
		TransactionID     string `json:"transaction_id"`
		ExternalID        string `json:"external_id"`
		PostingID         string `json:"posting_id"`
		ReceiptID         string `json:"receipt_id"`
		TreasuryRestarted bool   `json:"treasury_restarted"`
	}
	if err := waitForFile(ctx, filepath.Join(coord, "treasury-restart-confirmed.json"), &restartEvidence); err != nil {
		return empty, fmt.Errorf("wait for Treasury restart and accepted seller receipt readback: %w", err)
	}
	if restartEvidence.RunID != runID || restartEvidence.OrderID != order.OrderID || restartEvidence.JobID != order.JobID || restartEvidence.AuthorizationID != order.AuthorizationID || !restartEvidence.TreasuryRestarted || restartEvidence.TransactionID != captured.CaptureID || restartEvidence.ExternalID != captured.TreasuryExternalID || restartEvidence.PostingID == "" || restartEvidence.ReceiptID == "" {
		return empty, fmt.Errorf("Treasury restart/accepted-receipt readback does not match this Agent Manager order and settlement: %+v", restartEvidence)
	}
	reconcileURL := baseURL + "/api/v1/service-orders/" + url.PathEscape(order.OrderID) + "/reconcile"
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, body, requestErr := cf2AgentManagerJSONRequest(ctx, client, http.MethodPost, reconcileURL, nil)
		if requestErr == nil && status == http.StatusOK {
			if json.Unmarshal(body, &order) == nil && order.State == "reconciled" {
				break
			}
		}
		select {
		case <-ctx.Done():
			return empty, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	if order.State != "reconciled" || order.LedgerPostingID == "" || order.LedgerReceiptID == "" || order.CaptureID != captured.CaptureID || order.LedgerPostingID != restartEvidence.PostingID || order.LedgerReceiptID != restartEvidence.ReceiptID {
		return empty, fmt.Errorf("LPBS did not reconcile the accepted seller receipt to Money Ledger after Treasury restart: %+v", order)
	}
	readStatus, readRaw, err := cf2AgentManagerJSONRequest(ctx, client, http.MethodGet, baseURL+"/api/v1/service-orders/"+url.PathEscape(order.OrderID), nil)
	if err != nil || readStatus != http.StatusOK {
		return empty, fmt.Errorf("read reconciled LPBS service order after Treasury restart (status=%d): %s (%v)", readStatus, strings.TrimSpace(string(readRaw)), err)
	}
	var readback cf2AgentManagerOrder
	if err := json.Unmarshal(readRaw, &readback); err != nil || readback.State != "reconciled" || readback.OrderID != order.OrderID || readback.JobID != order.JobID || readback.AgentManagerRunID != runID || readback.AuthorizationID != order.AuthorizationID || readback.CaptureID != order.CaptureID || readback.LedgerPostingID != order.LedgerPostingID || readback.LedgerReceiptID != order.LedgerReceiptID {
		return empty, fmt.Errorf("LPBS order readback lost the accepted receipt/posting join: order=%+v readback=%+v decode=%v", order, readback, err)
	}
	if err := write("agent-manager-lpbs-order-reconciled.json", map[string]any{
		"run_id": runID, "agent_manager_run_id": order.AgentManagerRunID, "order_id": order.OrderID, "job_id": order.JobID, "authorization_id": order.AuthorizationID,
		"capture_id": order.CaptureID, "treasury_external_id": order.TreasuryExternalID,
		"posting_id": order.LedgerPostingID, "receipt_id": order.LedgerReceiptID,
		"state": order.State, "readback_after_treasury_restart": true,
	}); err != nil {
		return empty, err
	}
	return order, nil
}

func cf2AgentManagerJSONRequest(ctx context.Context, client *http.Client, method, endpoint string, body []byte) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	return response.StatusCode, data, err
}

func isLoopbackHostForIntegration(host string) bool {
	return strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1"
}

// cf1TreasuryX402Call is one exact priced call inside the same ordinary run.
// Every row has its own Treasury authorization, settlement, idempotency key,
// and Agent Manager grant; no call inherits another call's authority.
type cf1TreasuryX402Call struct {
	PriceID         string `json:"price_id"`
	CallID          string `json:"call_id"`
	CatalogURL      string `json:"catalog_url"`
	Counterparty    string `json:"counterparty"`
	InstrumentID    string `json:"instrument_id"`
	AuthorizationID string `json:"authorization_id"`
	SettlementID    string `json:"settlement_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AmountMinor     int64  `json:"amount_minor"`
}

type cf7X402SettlementReceipt struct {
	Settlement struct {
		Id            string `json:"id"`
		Authorization string `json:"authorization_id"`
		AmountMinor   int64  `json:"amount_minor"`
		Currency      string `json:"currency"`
		Outcome       string `json:"outcome"`
		ExternalID    string `json:"external_id"`
		InstrumentID  string `json:"instrument_id"`
		SettledAt     string `json:"settled_at"`
		Receipt       string `json:"receipt_reference"`
		Detail        string `json:"detail"`
	} `json:"settlement"`
}

// TestTreasuryCLIFromOrdinaryRun is invoked by Treasury's opt-in local-fork
// rehearsal. The fixture waits for the Treasury owner to create a run-bound
// mandate and instrument, then exercises the real CLI through the normal
// CreateProfile -> CreateTask -> CreateRun -> RunExecutor path.
func readCF1RunConfig(coord string) (cf1TreasuryRunConfig, error) {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		path := filepath.Join(coord, "treasury-config.json")
		if data, err := os.ReadFile(path); err == nil {
			var config cf1TreasuryRunConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return cf1TreasuryRunConfig{}, err
			}
			return config, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cf1TreasuryRunConfig{}, errors.New("timed out waiting for Treasury fixture configuration")
}

func waitForFile(ctx context.Context, path string, output any) error {
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if output != nil {
				if err := json.Unmarshal(data, output); err != nil {
					return err
				}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitForFileOrRunTerminal(ctx context.Context, service *orchestration.Orchestrator, runID uuid.UUID, path string, output any) error {
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if output != nil {
				if err := json.Unmarshal(data, output); err != nil {
					return err
				}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		run, runErr := service.GetStoredRun(ctx, runID)
		if runErr == nil && run.Status.IsTerminal() {
			return fmt.Errorf("Agent Manager run became terminal before writing %s: status=%s error=%s", filepath.Base(path), run.Status, run.ErrorMsg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitForCF1SessionRevocation(ctx context.Context, keys *cf1RunSessionKeys, runID uuid.UUID) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if keys.WasRevoked(runID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for run-owned session revocation: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForRunReceipt(ctx context.Context, service *orchestration.Orchestrator, runID uuid.UUID, path string) error {
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		run, err := service.GetRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("read Agent Manager run while waiting for runner receipt: %w", err)
		}
		if run.Status.IsTerminal() {
			return fmt.Errorf("Agent Manager run terminated before the runner wrote its receipt: status=%s error=%s", run.Status, run.ErrorMsg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
