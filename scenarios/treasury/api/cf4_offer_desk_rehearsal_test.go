//go:build treasury_rehearsal

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"treasury/internal/rail"
	x402rail "treasury/internal/rail/x402"
	"treasury/internal/testutil/basefork"
)

const (
	cf4OfferDeskTestOfferName   = "cf4-research-inference"
	cf4OfferDeskTestVariantName = "cf4-research-inference-call"
	cf4OfferDeskTestPriceID     = "cf4-research-inference"
)

// TestCF4OfferDeskPinnedFacilitatorLocalFork pays the actual Offer Desk
// resource endpoint using a one-cent Offer Desk graph price, a real Treasury
// admission record, and the lifecycle-managed pinned facilitator. The first
// successful endpoint response is dropped after admission; exact payload
// replay returns the same settled transfer and never creates another charge.
func TestCF4OfferDeskPinnedFacilitatorLocalFork(t *testing.T) {
	if os.Getenv("TREASURY_CF4_OFFER_DESK_REHEARSAL") != "1" {
		t.Skip("set TREASURY_CF4_OFFER_DESK_REHEARSAL=1 and TREASURY_BASE_FORK_TEST=1 for the Offer Desk metered Base-fork rehearsal")
	}
	fixtureRoot := strings.TrimSpace(os.Getenv("TREASURY_CF4_FIXTURE_DIR"))
	require.True(t, filepath.IsAbs(fixtureRoot), "CF4 fixture path must be absolute")
	runDir := filepath.Join(fixtureRoot, "offer-desk-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	const peakBytes int64 = 2 * 1024 * 1024 * 1024
	admitTreasuryRehearsalDisk(t, fixtureRoot, runDir, "crypto-foundations:CF4-offer-desk-pinned-facilitator", "Offer Desk price -> Treasury -> pinned facilitator -> local Base-fork replay", peakBytes)

	basefork.AcquireSharedFacilitatorHarness(t)
	fork := basefork.StartAt(t, cf5PinnedBaseForkPort)
	forkIdentity := fork.Identity()
	require.Equal(t, "0x2105", forkIdentity.ChainID)
	writeCF5JSON(t, filepath.Join(runDir, "fork-identity.json"), forkIdentity)
	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	cf5ProvisionPinnedFacilitatorKey(t, facilitatorKey)
	cf5StartPinnedFacilitator(t, ctx, runDir)

	offerDeskEnv := map[string]string{
		"VROOLI_STORAGE_ROOT":                      filepath.Join(runDir, "storage"),
		"TREASURY_API_URL":                         "",
		"OFFER_DESK_TEST_DROP_FIRST_PAID_RESPONSE": "1",
	}
	offerDeskSource := prepareCF5APIScenarioVariant(t, "offer-desk", "../../offer-desk", runDir, offerDeskEnv)
	// This paid catalog call uses the authoritative SQLite graph. Keep the
	// optional semantic-search backend out of this API-only variant so an absent
	// Qdrant service cannot hold the listener behind its connection timeout.
	disableCF4OfferDeskVectorSearch(t, offerDeskSource)
	offerDeskName := "offer-desk@" + instanceName("cf4-offer-desk")
	offerDeskStart := lifecycleRehearsal(t, ctx, runDir, "offer-desk-start", map[string]string{}, "start", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	offerDeskURL := rehearsalURL(t, offerDeskStart)
	offerDeskStopped := false
	t.Cleanup(func() {
		if !offerDeskStopped {
			stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
			offerDeskStopped = true
		}
	})
	offerID, variantID := seedCF4OfferDeskGraph(t, ctx, offerDeskURL, runDir)
	paidURL := offerDeskURL + "/api/v1/paid/catalog"

	custodyDir := filepath.Join(runDir, "custody")
	initializeCF2Credentials(t, filepath.Join(custodyDir, "synthetic-custody.enc"))
	treasuryEnv := map[string]string{
		"VROOLI_STORAGE_ROOT":     filepath.Join(runDir, "storage"),
		"TREASURY_REHEARSAL_DIR":  custodyDir,
		"TREASURY_BASE_RPC_URL":   fork.URL,
		"TREASURY_BASE_USDC_MODE": "local-fork",
		"MONEY_LEDGER_API_URL":    "http://127.0.0.1:1", "SCENARIO_AUTHENTICATOR_API_URL": "http://127.0.0.1:1",
		"OFFER_DESK_BASE_URL":  offerDeskURL,
		"X402_FACILITATOR_URL": "http://127.0.0.1:14021",
	}
	treasurySource := prepareCF5APIScenarioVariant(t, "treasury", "..", runDir, treasuryEnv)
	treasuryName := "treasury@" + instanceName("cf4-treasury")
	treasuryStart := lifecycleRehearsal(t, ctx, runDir, "treasury-start", map[string]string{
		"GOFLAGS": "-tags=treasury_rehearsal", "API_TOKEN": "", "TREASURY_OPERATOR_TOKEN": "", "TREASURY_MANDATE_SIGNING_KEY": "",
	}, "start", treasuryName, "--path", treasurySource, "--json", "--timeout", "240")
	treasuryURL := rehearsalURL(t, treasuryStart)
	treasuryStopped := false
	t.Cleanup(func() {
		if !treasuryStopped {
			stopCF5Variant(t, runDir, treasuryName, treasuryURL)
			treasuryStopped = true
		}
	})
	requireHTTPStatus(t, treasuryURL+"/health", http.StatusOK)

	// Restart this isolated Offer Desk instance with the known Treasury variant
	// URL so its payment admission cannot resolve to a different global instance.
	setCF5VariantEnvironment(t, offerDeskSource, map[string]string{"TREASURY_API_URL": treasuryURL})
	restartedOfferDesk := lifecycleRehearsal(t, ctx, runDir, "offer-desk-bind-treasury", map[string]string{}, "restart", offerDeskName, "--path", offerDeskSource, "--json", "--timeout", "240")
	restartedOfferDeskURL := rehearsalURL(t, restartedOfferDesk)
	require.Equal(t, offerDeskURL, restartedOfferDeskURL, "scenario restart must retain its bound API address")

	price := map[string]any{
		"id": cf4OfferDeskTestPriceID, "resource_url": paidURL,
		"description": "Offer Desk research catalog lookup", "mime_type": "application/json",
		"network": "eip155:8453", "scheme": "exact", "pay_to": fork.Accounts[2],
		"asset": basefork.USDC, "asset_decimals": 6, "max_timeout_seconds": 300,
		"extra_json": `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
	}
	declaration := map[string]any{"offer_id": offerID, "variant_id": variantID, "price": price}
	declared := cf4DeclareOfferDeskPrice(t, ctx, treasuryURL, declaration)
	require.Equal(t, int64(1), declared.AmountMinor)
	require.Equal(t, "10000", declared.Amount)
	require.Equal(t, "USD", declared.Currency)
	writeCF5JSON(t, filepath.Join(runDir, "offer-desk-price.json"), declared)

	challengeResponse, err := (&http.Client{Timeout: 10 * time.Second}).Get(paidURL)
	require.NoError(t, err)
	defer challengeResponse.Body.Close()
	require.Equal(t, http.StatusPaymentRequired, challengeResponse.StatusCode)
	challenge := challengeResponse.Header.Get("Payment-Required")
	require.NotEmpty(t, challenge)
	challengeBody, err := io.ReadAll(io.LimitReader(challengeResponse.Body, 64<<10))
	require.NoError(t, err)
	require.NotContains(t, string(challengeBody), cf4OfferDeskTestOfferName, "catalog content must remain withheld before admission")

	buyerKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	buyerAddress := gethcrypto.PubkeyToAddress(buyerKey.PublicKey)
	_, err = fork.Mint(ctx, buyerAddress.Hex(), declared.Amount)
	require.NoError(t, err, "the disposable buyer receives only the one-cent local-fork test amount")
	require.NoError(t, fork.Call(ctx, "evm_setNextBlockTimestamp", []any{time.Now().Unix()}, nil), "align only the isolated local fork's authorization clock")
	require.NoError(t, fork.Call(ctx, "evm_mine", []any{}, nil))
	payeeBefore := cf5USDCBalance(t, ctx, fork, fork.Accounts[2])
	signer := httptest.NewServer(cf4TypedDataRPCSigner{key: buyerKey})
	t.Cleanup(signer.Close)
	credential, err := json.Marshal(x402rail.Credential{
		EndpointURL: paidURL, SignerURL: signer.URL, Account: buyerAddress.Hex(),
		Networks: []string{"eip155:8453"},
		Assets:   map[string]x402rail.AssetPolicy{strings.ToLower(basefork.USDC): {Decimals: 6, Currency: "USD"}},
	})
	require.NoError(t, err)
	walletSigner, err := x402rail.NewRPCSigner(&http.Client{Timeout: 20 * time.Second})
	require.NoError(t, err)
	adapter, err := x402rail.New(&http.Client{Timeout: 20 * time.Second}, walletSigner)
	require.NoError(t, err)
	var paymentDigest, paymentPayload string
	command := rail.SettleCommand{
		SettlementID: "cf4-offer-desk-settlement", AuthorizationID: "cf4-local-test-authorization",
		MandateReference: "cf4-local-test-mandate", InstrumentID: "cf4-offer-desk-test-instrument",
		IdempotencyKey: "cf4-offer-desk-one-cent-call", AmountMinor: 1, Currency: "USD",
		Counterparty: "127.0.0.1", Credential: string(credential),
		RecordRecoveryInput: func(_ context.Context, digest, payload string) error {
			paymentDigest, paymentPayload = digest, payload
			return nil
		},
	}
	first, firstErr := adapter.Settle(ctx, command)
	require.Error(t, firstErr, "Offer Desk drops the first successful response only after Treasury admission settles")
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	require.NotEmpty(t, paymentDigest)
	require.NotEmpty(t, paymentPayload)

	replayed, err := adapter.QueryOutcome(ctx, rail.Query{
		SettlementID: command.SettlementID, AuthorizationID: command.AuthorizationID,
		MandateReference: command.MandateReference, InstrumentID: command.InstrumentID,
		ExternalID: paymentDigest, ReceiptReference: paymentPayload, IdempotencyKey: command.IdempotencyKey,
		Counterparty: command.Counterparty, Credential: command.Credential,
	})
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, replayed.Outcome)
	require.Equal(t, paymentDigest, first.ExternalID)
	require.Equal(t, paymentDigest, replayed.ReceiptReference, "replay result keeps the protected payload digest as its receipt reference")
	require.NotEmpty(t, replayed.ExternalID)

	contentRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, paidURL, nil)
	require.NoError(t, err)
	contentRequest.Header.Set("Payment-Signature", paymentPayload)
	contentResponse, err := (&http.Client{Timeout: 20 * time.Second}).Do(contentRequest)
	require.NoError(t, err)
	defer contentResponse.Body.Close()
	require.Equal(t, http.StatusOK, contentResponse.StatusCode)
	responseReceipt := contentResponse.Header.Get("Payment-Response")
	require.NotEmpty(t, responseReceipt, "the paid resource must return the original settlement receipt")
	var responseSettlement struct {
		Transaction string `json:"transaction"`
		Payer       string `json:"payer"`
		Network     string `json:"network"`
		Success     bool   `json:"success"`
	}
	decodedReceipt, err := base64.StdEncoding.DecodeString(responseReceipt)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(decodedReceipt, &responseSettlement))
	require.True(t, responseSettlement.Success)
	require.Equal(t, replayed.ExternalID, responseSettlement.Transaction)
	require.Equal(t, buyerAddress.Hex(), responseSettlement.Payer)
	require.Equal(t, "eip155:8453", responseSettlement.Network)
	content, err := io.ReadAll(io.LimitReader(contentResponse.Body, 256<<10))
	require.NoError(t, err)
	var catalog offerspb.ListNodesResponse
	require.NoError(t, protojson.Unmarshal(content, &catalog))
	var foundOffer, foundVariant bool
	for _, node := range catalog.GetNodes() {
		foundOffer = foundOffer || node.GetName() == cf4OfferDeskTestOfferName
		foundVariant = foundVariant || node.GetName() == cf4OfferDeskTestVariantName
	}
	require.True(t, foundOffer && foundVariant, "catalog should be disclosed after the settled payment is replayed")

	buyerAfter := cf5USDCBalance(t, ctx, fork, buyerAddress.Hex())
	require.Equal(t, "0", buyerAfter.String(), "exactly 10000 atoms should be consumed once from the one-cent test balance")
	payeeAfter := cf5USDCBalance(t, ctx, fork, fork.Accounts[2])
	require.Equal(t, new(big.Int).Add(payeeBefore, big.NewInt(10_000)).String(), payeeAfter.String(), "the one-cent Offer Desk price must settle once to the declared payee")
	forkReceipt := map[string]any{
		"price_id": declared.ID, "offer_id": cf4OfferDeskTestOfferName, "variant_id": cf4OfferDeskTestVariantName,
		"resource_url": paidURL, "amount_minor": declared.AmountMinor, "atomic_amount": declared.Amount,
		"payer": buyerAddress.Hex(), "payee": fork.Accounts[2], "network": "eip155:8453",
		"transaction_hash": replayed.ExternalID, "payment_digest": paymentDigest,
		"first_response_dropped_after_settlement": true, "exact_replay_same_transaction": true,
		"buyer_funded_by_local_test_mint": true, "buyer_balance_atoms_after": buyerAfter.String(),
		"payee_balance_atoms_after": payeeAfter.String(), "fork": forkIdentity,
	}
	writeCF5JSON(t, filepath.Join(runDir, "offer-desk-settled-call.json"), forkReceipt)

	stopCF5Variant(t, runDir, treasuryName, treasuryURL)
	treasuryStopped = true
	stopCF5Variant(t, runDir, offerDeskName, offerDeskURL)
	offerDeskStopped = true
	facilitatorStop := cf5SourceVrooliCommand(ctx, "resource", "stop", "x402-facilitator-local-test", "--json")
	facilitatorStopOutput, err := facilitatorStop.CombinedOutput()
	require.NoErrorf(t, err, "stop the isolated facilitator before stopping its fork: %s", strings.TrimSpace(string(facilitatorStopOutput)))
	require.NoError(t, os.WriteFile(filepath.Join(runDir, "facilitator-stop.json"), facilitatorStopOutput, 0o600))
	stopReceipt := fork.Stop(ctx)
	writeCF5JSON(t, filepath.Join(runDir, "fork-stop.json"), stopReceipt)
	require.True(t, stopReceipt.Clean)
}

func disableCF4OfferDeskVectorSearch(t *testing.T, source string) {
	t.Helper()
	searchPath := filepath.Join(source, ".vrooli", "search.json")
	var config map[string]any
	readRehearsalJSON(t, searchPath, &config)
	config["providers"] = []any{}
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(searchPath, raw, 0o600))
}

func seedCF4OfferDeskGraph(t *testing.T, ctx context.Context, baseURL, runDir string) (string, string) {
	t.Helper()
	catalogClient := offersconnect.NewCatalogServiceClient(&http.Client{Timeout: 15 * time.Second}, baseURL)
	offer, err := catalogClient.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_OFFER, Name: cf4OfferDeskTestOfferName, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	variant, err := catalogClient.CreateNode(ctx, connect.NewRequest(&offerspb.CreateNodeRequest{
		Kind: offerspb.NodeKind_VARIANT, Name: cf4OfferDeskTestVariantName, Status: offerspb.Status_ACTIVE,
	}))
	require.NoError(t, err)
	_, err = catalogClient.CreateEdge(ctx, connect.NewRequest(&offerspb.CreateEdgeRequest{Edge: &offerspb.Edge{
		FromId: offer.Msg.GetNode().GetId(), ToId: variant.Msg.GetNode().GetId(), Kind: "sells_at",
		IntendedPriceMinor: 1, Currency: "USD", IntendedPriceDeclared: true,
	}}))
	require.NoError(t, err)
	writeCF5JSON(t, filepath.Join(runDir, "offer-desk-graph.json"), map[string]any{
		"offer_id": offer.Msg.GetNode().GetId(), "offer_name": cf4OfferDeskTestOfferName,
		"variant_id": variant.Msg.GetNode().GetId(), "variant_name": cf4OfferDeskTestVariantName,
		"edge": "sells_at", "amount_minor": 1, "currency": "USD", "declared": true,
	})
	return offer.Msg.GetNode().GetId(), variant.Msg.GetNode().GetId()
}

func cf4DeclareOfferDeskPrice(t *testing.T, ctx context.Context, treasuryURL string, declaration map[string]any) x402rail.Price {
	t.Helper()
	encoded, err := json.Marshal(declaration)
	require.NoError(t, err)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, treasuryURL+"/api/v1/x402/prices/offer-desk", strings.NewReader(string(encoded)))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vrooli-Operator-Token", "cf2-local-operator-only")
	response, err := (&http.Client{Timeout: 20 * time.Second}).Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, response.StatusCode, "price declaration response: %s", strings.TrimSpace(string(body)))
	var price x402rail.Price
	require.NoError(t, json.Unmarshal(body, &price))
	return price
}

func setCF5VariantEnvironment(t *testing.T, source string, values map[string]string) {
	t.Helper()
	manifestPath := filepath.Join(source, ".vrooli", "service.json")
	var manifest map[string]any
	readRehearsalJSON(t, manifestPath, &manifest)
	components := manifest["components"].(map[string]any)
	api := components["api"].(map[string]any)
	run := api["run"].(map[string]any)
	env, ok := run["env"].(map[string]any)
	if !ok {
		env = make(map[string]any)
		run["env"] = env
	}
	for key, value := range values {
		env[key] = value
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, raw, 0o600))
}
