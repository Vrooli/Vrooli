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
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	"treasury/internal/instrument"
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
