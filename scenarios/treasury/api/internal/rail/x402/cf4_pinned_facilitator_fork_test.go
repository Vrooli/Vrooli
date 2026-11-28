package x402

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"treasury/internal/instrument"
	"treasury/internal/ledger"
	"treasury/internal/rail"
	"treasury/internal/testutil/basefork"
)

const localFacilitatorIdentity = "vrooli/x402-facilitator-local-test"
const localFacilitatorField = "facilitator-private-key"
const localFacilitatorPort = 18545

// TestCF4PinnedFacilitatorLocalBaseFork runs a bounded useful payment through
// the pinned resource. Both signer gas and the buyer's USDC exist only inside
// the owned Anvil fork. The first receipt is deliberately dropped after the
// real facilitator settles; exact-payload reconciliation must return the
// retained transaction rather than consume another authorization.
func TestCF4PinnedFacilitatorLocalBaseFork(t *testing.T) {
	if os.Getenv("TREASURY_CF4_PINNED_FACILITATOR_TEST") != "1" {
		t.Skip("set TREASURY_CF4_PINNED_FACILITATOR_TEST=1 for the pinned local-fork facilitator showcase")
	}
	basefork.AcquireSharedFacilitatorHarness(t)
	fork := basefork.StartAt(t, localFacilitatorPort)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	provisionLocalTestSigner(t, facilitatorKey)
	startLocalTestFacilitator(t)

	clientKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	clientAddress := gethcrypto.PubkeyToAddress(clientKey.PublicKey)
	payTo := common.HexToAddress(fork.Accounts[2])
	_, err = fork.Mint(ctx, clientAddress.Hex(), "50000")
	require.NoError(t, err)

	chainTimestamp := time.Now().Unix()
	require.NoError(t, fork.Call(ctx, "evm_setNextBlockTimestamp", []any{chainTimestamp}, nil))
	require.NoError(t, fork.Call(ctx, "evm_mine", []any{}, nil))
	chainClock := time.Unix(chainTimestamp, 0).UTC()

	facilitator, err := NewHTTPFacilitator("http://127.0.0.1:14021", &http.Client{Timeout: 20 * time.Second})
	require.NoError(t, err)
	var endpoint string
	var mu sync.Mutex
	settledByPayload := map[string]settleResponse{}
	firstDispatch := true
	target := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		signed := request.Header.Get(paymentSignatureHeader)
		if signed == "" {
			writer.Header().Set(paymentRequiredHeader, cf4PaymentRequired(t, endpoint, "10000", payTo.Hex()))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		digest := paymentDigest(signed)
		mu.Lock()
		settled, cached := settledByPayload[digest]
		mu.Unlock()
		if !cached {
			decoded, decodeErr := base64.StdEncoding.DecodeString(signed)
			if decodeErr != nil {
				http.Error(writer, "invalid signed payment", http.StatusBadRequest)
				return
			}
			var payload struct {
				Accepted json.RawMessage `json:"accepted"`
			}
			if json.Unmarshal(decoded, &payload) != nil || len(payload.Accepted) == 0 {
				http.Error(writer, "invalid payment requirements", http.StatusBadRequest)
				return
			}
			verified, verifyErr := facilitator.Verify(request.Context(), decoded, payload.Accepted)
			if verifyErr != nil || !verified.Valid || !strings.EqualFold(verified.Payer, clientAddress.Hex()) {
				http.Error(writer, fmt.Sprintf("facilitator verification failed: %v %s", verifyErr, verified.Reason), http.StatusPaymentRequired)
				return
			}
			settledResult, settleErr := facilitator.Settle(request.Context(), decoded, payload.Accepted)
			if settleErr != nil || !settledResult.Success || settledResult.Transaction == "" || !strings.EqualFold(settledResult.Payer, clientAddress.Hex()) {
				http.Error(writer, fmt.Sprintf("facilitator settlement failed: %v %s", settleErr, settledResult.Reason), http.StatusBadGateway)
				return
			}
			settled = settleResponse{Success: true, Payer: settledResult.Payer, Transaction: settledResult.Transaction, Network: settledResult.Network}
			mu.Lock()
			settledByPayload[digest] = settled
			mu.Unlock()
		}
		mu.Lock()
		dropReceipt := firstDispatch
		firstDispatch = false
		mu.Unlock()
		if dropReceipt {
			http.Error(writer, "response lost after pinned-facilitator settlement", http.StatusBadGateway)
			return
		}
		writePaymentResponse(writer, settled.Transaction, settled.Payer)
	})
	targetServer := &http.Server{Addr: "127.0.0.1:0", Handler: target}
	listener, err := netListenLoopback()
	require.NoError(t, err)
	endpoint = "http://" + listener.Addr().String() + "/priced/cf4-inference"
	go func() { _ = targetServer.Serve(listener) }()
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = targetServer.Shutdown(shutdownCtx)
	})

	command := cf4Command(t, endpoint, clientAddress.Hex())
	command.AmountMinor = 1
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "cf4-pinned-local-test-passphrase")
	require.NoError(t, err)
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	require.NoError(t, err)
	recoveryVault, err := instrument.NewCredentialClientResolver(client)
	require.NoError(t, err)
	var recoveryDigest, recoveryPayload string
	var recoveryReference string
	command.RecordRecoveryInput = func(recordCtx context.Context, digest, payment string) error {
		var storeErr error
		recoveryReference, storeErr = recoveryVault.StoreSettlementRecoveryInput(recordCtx, "cf4-pinned-local-fork-test", payment)
		if storeErr != nil {
			return storeErr
		}
		recoveryDigest, recoveryPayload = digest, payment
		return nil
	}
	adapter, err := New(&http.Client{Timeout: 20 * time.Second}, cf4MemorySigner{key: clientKey})
	require.NoError(t, err)
	adapter.now = func() time.Time { return chainClock }
	first, err := adapter.Settle(ctx, command)
	require.Error(t, err, "the endpoint deliberately loses the post-settlement receipt")
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	require.NotEmpty(t, recoveryPayload)
	protectedPayload, err := recoveryVault.ResolveSettlementRecoveryInput(ctx, recoveryReference)
	require.NoError(t, err)
	require.Equal(t, recoveryPayload, protectedPayload)

	restartedAdapter, err := New(&http.Client{Timeout: 20 * time.Second}, cf4MemorySigner{key: clientKey})
	require.NoError(t, err)
	restartedAdapter.now = func() time.Time { return chainClock }
	reconciled, err := restartedAdapter.QueryOutcome(ctx, rail.Query{
		ExternalID: recoveryDigest, ReceiptReference: protectedPayload,
		IdempotencyKey: command.IdempotencyKey, Counterparty: command.Counterparty,
		Credential: command.Credential,
	})
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, reconciled.Outcome)
	require.Equal(t, "x402_facilitator_replay_confirmation", reconciled.Basis)
	require.NotEmpty(t, reconciled.ExternalID)

	var balanceHex string
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": basefork.USDC, "data": "0x70a08231" + cf4AddressWord(clientAddress.Hex())}, "latest"}, &balanceHex))
	balance, ok := new(big.Int).SetString(strings.TrimPrefix(balanceHex, "0x"), 16)
	require.True(t, ok)
	require.Equal(t, big.NewInt(40000), balance, "the 10000-atom accepted offer settles exactly once")
	forkIdentity := fork.Identity()
	t.Logf("CF4 pinned local-facilitator showcase: block=%s hash=%s rpc=%s amount_atoms=10000 buyer=%s payee=%s facilitator=%s tx=%s replay=%s buyer_balance_atoms=%s", forkIdentity.BlockNumber, forkIdentity.BlockHash, forkIdentity.URL, clientAddress.Hex(), payTo.Hex(), facilitatorAddress.Hex(), reconciled.ExternalID, reconciled.Basis, balance.String())
}

// TestX402DeferredResourceSettlementOnPinnedLocalBaseFork exercises Treasury's
// verify-before-settle resource admission against the pinned facilitator and
// an isolated Base fork. It proves that Treasury queues exactly one seller-side
// ledger emission after settlement; Money Ledger acceptance is a separate
// owner integration proof.
func TestX402DeferredResourceSettlementOnPinnedLocalBaseFork(t *testing.T) {
	if os.Getenv("TREASURY_CF4_PINNED_FACILITATOR_TEST") != "1" {
		t.Skip("set TREASURY_CF4_PINNED_FACILITATOR_TEST=1 for the deferred-resource local-fork proof")
	}
	basefork.AcquireSharedFacilitatorHarness(t)
	fork := basefork.StartAt(t, localFacilitatorPort)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	facilitatorKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	facilitatorAddress := gethcrypto.PubkeyToAddress(facilitatorKey.PublicKey)
	require.NoError(t, fork.Call(ctx, "anvil_setBalance", []any{facilitatorAddress.Hex(), "0x56bc75e2d63100000"}, nil))
	provisionLocalTestSigner(t, facilitatorKey)
	startLocalTestFacilitator(t)

	clientKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	clientAddress := gethcrypto.PubkeyToAddress(clientKey.PublicKey)
	payTo := common.HexToAddress(fork.Accounts[2])
	_, err = fork.Mint(ctx, clientAddress.Hex(), "50000")
	require.NoError(t, err)

	var challenge string
	var endpoint string
	var gate *Gate
	var price Price
	var gateFailure string
	var admittedID string
	var signedPayment string
	target := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payment := request.Header.Get(paymentSignatureHeader)
		if payment == "" {
			writer.Header().Set(paymentRequiredHeader, challenge)
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		verified, verifyErr := gate.VerifyForResource(request.Context(), price.ID, "job-deferred-fork", "resource-inference", "order-deferred-fork", payment)
		if verifyErr != nil {
			gateFailure = verifyErr.Error()
			http.Error(writer, verifyErr.Error(), http.StatusPaymentRequired)
			return
		}
		if verified.Status != "verified" {
			http.Error(writer, "payment is not durably authorized", http.StatusConflict)
			return
		}
		admittedID = verified.ID
		// This handler is the test's resource-verifier boundary: only after
		// verify returned a durable bound admission do we request settlement.
		settled, settleErr := gate.SettleAuthorization(request.Context(), verified.ID, verified.JobID, verified.ResourceID, verified.OrderID)
		if settleErr != nil {
			http.Error(writer, settleErr.Error(), http.StatusBadGateway)
			return
		}
		writePaymentResponse(writer, settled.TransactionID, settled.Payer)
	})
	targetServer := httptest.NewServer(target)
	defer targetServer.Close()
	endpoint = targetServer.URL + "/priced/deferred-inference"

	credentialAuthority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "deferred-x402-credentials.enc"), "deferred-x402-local-test-passphrase")
	require.NoError(t, err)
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: credentialAuthority})
	require.NoError(t, err)
	recoveryVault, err := instrument.NewCredentialClientResolver(client)
	require.NoError(t, err)
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(ledger.Schema), database.SchemaProviderFunc(Schema)))
	facilitator, err := NewHTTPFacilitator("http://127.0.0.1:14021", &http.Client{Timeout: 20 * time.Second})
	require.NoError(t, err)
	gate, err = NewGate(NewSQLiteInboundRepository(db), facilitator)
	require.NoError(t, err)
	gate.SetRecoveryVault(recoveryVault)
	require.NoError(t, gate.SetRuntimeMode("local-fork"))
	price, err = gate.Declare(ctx, Price{
		ID: "deferred-local-fork-price", ResourceURL: endpoint, Description: "deferred local-fork inference",
		MIMEType: "application/json", Network: "eip155:8453", Scheme: "exact", Amount: "10000", AmountMinor: 1,
		Currency: "USD", PayTo: payTo.Hex(), Asset: basefork.USDC, AssetDecimals: 6, MaxTimeoutSeconds: 300,
		ExtraJSON: `{"assetTransferMethod":"eip3009","name":"USD Coin","version":"2"}`,
	})
	require.NoError(t, err)
	challenge, err = gate.PaymentRequired(ctx, price.ID)
	require.NoError(t, err)

	command := cf4Command(t, endpoint, clientAddress.Hex())
	command.AmountMinor = 1
	command.SettlementID = "deferred-local-fork-settlement"
	command.AuthorizationID = "deferred-local-fork-authorization"
	command.MandateReference = "deferred-local-fork-mandate"
	command.InstrumentID = "deferred-local-fork-instrument"
	command.IdempotencyKey = "deferred-local-fork-idempotency"
	command.RecordRecoveryInput = func(recordCtx context.Context, _, payment string) error {
		signedPayment = payment
		_, storeErr := recoveryVault.StoreSettlementRecoveryInput(recordCtx, "deferred-x402-buyer-test", payment)
		return storeErr
	}
	adapter, err := New(targetServer.Client(), cf4MemorySigner{key: clientKey})
	require.NoError(t, err)
	chainTimestamp := time.Now().Unix() + 1
	require.NoError(t, fork.Call(ctx, "evm_setNextBlockTimestamp", []any{chainTimestamp}, nil))
	require.NoError(t, fork.Call(ctx, "evm_mine", []any{}, nil))
	chainClock := time.Unix(chainTimestamp, 0).UTC()
	firstForkBlock := fork.Identity()
	adapter.now = func() time.Time { return chainClock }
	settled, err := adapter.Settle(ctx, command)
	require.NoError(t, err)
	require.Equalf(t, rail.OutcomeSettled, settled.Outcome, "deferred facilitator path: %+v; treasury_error=%s", settled, gateFailure)
	admission, err := gate.repository.(ResourceAdmissionRepository).GetResourceAdmission(ctx, admittedID)
	require.NoError(t, err)
	require.Equal(t, "settled", admission.Status)
	require.Equal(t, "local-fork", admission.Mode)
	require.Equal(t, "job-deferred-fork", admission.JobID)
	require.Empty(t, admission.RecoveryReference)
	var emissions int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_emissions WHERE settlement_id=?`, admission.ID).Scan(&emissions))
	require.Equal(t, 1, emissions, "Treasury queues exactly one seller-side emission for the bound authorization")

	var balanceHex string
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": basefork.USDC, "data": "0x70a08231" + cf4AddressWord(clientAddress.Hex())}, "latest"}, &balanceHex))
	balance, ok := new(big.Int).SetString(strings.TrimPrefix(balanceHex, "0x"), 16)
	require.True(t, ok)
	require.Equal(t, big.NewInt(40000), balance, "the deferred admission charges the local-fork buyer once")
	var transactionReceipt struct {
		Status string `json:"status"`
		Logs   []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	require.NoError(t, fork.Call(ctx, "eth_getTransactionReceipt", []any{settled.ExternalID}, &transactionReceipt))
	require.Equal(t, "0x1", transactionReceipt.Status)
	transferTopic := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	expectedAmountData := "0x" + fmt.Sprintf("%064x", 10_000)
	transferFound := false
	for _, event := range transactionReceipt.Logs {
		if strings.EqualFold(event.Address, basefork.USDC) && len(event.Topics) == 3 && len(event.Topics[1]) == 66 && len(event.Topics[2]) == 66 && strings.EqualFold(event.Topics[0], transferTopic) && strings.EqualFold(event.Topics[1][26:], strings.TrimPrefix(clientAddress.Hex(), "0x")) && strings.EqualFold(event.Topics[2][26:], strings.TrimPrefix(payTo.Hex(), "0x")) && strings.EqualFold(event.Data, expectedAmountData) {
			transferFound = true
		}
	}
	require.True(t, transferFound, "the exact 10000-atom transfer must be present in the accepted fork transaction")
	decodedPayment, err := base64.StdEncoding.DecodeString(signedPayment)
	require.NoError(t, err)
	var signedEnvelope struct {
		Payload struct {
			Authorization struct {
				Nonce string `json:"nonce"`
			} `json:"authorization"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(decodedPayment, &signedEnvelope))
	var authorizationState string
	stateCall := "0x" + cf4Selector("authorizationState(address,bytes32)") + cf4AddressWord(clientAddress.Hex()) + strings.TrimPrefix(signedEnvelope.Payload.Authorization.Nonce, "0x")
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": basefork.USDC, "data": stateCall}, "latest"}, &authorizationState))
	consumed, ok := new(big.Int).SetString(strings.TrimPrefix(authorizationState, "0x"), 16)
	require.True(t, ok)
	require.Equal(t, big.NewInt(1), consumed, "the exact signed nonce is consumed once on the fork")
	t.Logf("Deferred x402 resource local-fork: start_block=%s latest_block=%s buyer=%s payee=%s atoms=10000 admission=%s tx=%s treasury_emissions=%d", firstForkBlock.BlockNumber, fork.Identity().BlockNumber, clientAddress.Hex(), payTo.Hex(), admittedID, settled.ExternalID, emissions)
}

func provisionLocalTestSigner(t *testing.T, key *ecdsa.PrivateKey) {
	t.Helper()
	command := exec.Command("vrooli", "credentials", "provision", "--identity", localFacilitatorIdentity, "--field", localFacilitatorField)
	command.Stdin = strings.NewReader(fmt.Sprintf("%x", gethcrypto.FromECDSA(key)))
	output, err := command.CombinedOutput()
	require.NoError(t, err, "provision local-only facilitator test signer: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		delete := exec.Command("vrooli", "credentials", "delete", "--identity", localFacilitatorIdentity, "--field", localFacilitatorField, "--yes")
		if output, err := delete.CombinedOutput(); err != nil {
			t.Errorf("remove local-only facilitator test signer: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

func startLocalTestFacilitator(t *testing.T) {
	t.Helper()
	command := localVrooliCommand("resource", "start", "x402-facilitator-local-test")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "start isolated pinned facilitator through resource lifecycle: %s", strings.TrimSpace(string(output)))
	t.Cleanup(func() {
		stop := localVrooliCommand("resource", "stop", "x402-facilitator-local-test")
		if output, err := stop.CombinedOutput(); err != nil {
			t.Errorf("stop isolated pinned facilitator through resource lifecycle: %v: %s", err, strings.TrimSpace(string(output)))
		}
	})
}

func localVrooliCommand(args ...string) *exec.Cmd {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../../"))
	cliArgs := append([]string{"run", "./cmd/vrooli"}, args...)
	command := exec.Command("go", cliArgs...)
	command.Dir = root
	return command
}

func netListenLoopback() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
