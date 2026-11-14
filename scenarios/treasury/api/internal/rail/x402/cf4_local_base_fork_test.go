package x402

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"treasury/internal/instrument"
	"treasury/internal/rail"
	"treasury/internal/testutil/basefork"
)

// TestCF4LocalBaseForkReconcilesExactPayment exercises the outbound rail,
// the one-minor-unit authorization amount passed to it, and ambiguous-result replay against the fork's
// real native-USDC EIP-3009 contract. The HTTP service below is an explicit
// cryptographic test facilitator: it recovers the payer from EIP-712 and asks
// Anvil to submit transferWithAuthorization. It is not the pinned upstream
// x402-facilitator and does not establish that resource's settlement support.
func TestCF4LocalBaseForkReconcilesExactPayment(t *testing.T) {
	fork := basefork.Start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	clientKey, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	clientAddress := gethcrypto.PubkeyToAddress(clientKey.PublicKey)
	payTo := common.HexToAddress(fork.Accounts[2])
	_, err = fork.Mint(ctx, clientAddress.Hex(), "50000")
	require.NoError(t, err)

	offerCatalog := &offerDeskCatalogFixture{edges: []*offerspb.Edge{{
		FromId: "cf4-inference-offer", ToId: "cf4-inference-call", Kind: "sells_at",
		IntendedPriceMinor: 1, Currency: "USD", IntendedPriceDeclared: true,
	}}}
	offerPath, offerHandler := offersconnect.NewCatalogServiceHandler(offerCatalog)
	offerMux := http.NewServeMux()
	offerMux.Handle(offerPath, offerHandler)
	offerServer := httptest.NewServer(offerMux)
	defer offerServer.Close()
	offerSource, err := NewOfferDeskPriceSource(offerServer.URL, offerServer.Client())
	require.NoError(t, err)
	offerQuote, err := offerSource.ResolvePrice(ctx, "cf4-inference-offer", "cf4-inference-call")
	require.NoError(t, err)
	amount := new(big.Int).Mul(big.NewInt(offerQuote.AmountMinor), big.NewInt(10_000)).String()
	require.Equal(t, int64(1), offerQuote.AmountMinor)
	require.Equal(t, "USD", offerQuote.Currency)
	require.Equal(t, "10000", amount)
	var latestBlock map[string]json.RawMessage
	require.NoError(t, fork.Call(ctx, "eth_getBlockByNumber", []any{"latest", false}, &latestBlock))
	var timestampHex string
	require.NoError(t, json.Unmarshal(latestBlock["timestamp"], &timestampHex))
	blockTimestamp, ok := new(big.Int).SetString(strings.TrimPrefix(timestampHex, "0x"), 16)
	require.True(t, ok)
	chainClock := time.Unix(blockTimestamp.Int64()+1, 0).UTC()
	var endpoint string
	var mu sync.Mutex
	transactions := map[string]string{}
	firstDispatch := true
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payment := request.Header.Get(paymentSignatureHeader)
		if payment == "" {
			writer.Header().Set(paymentRequiredHeader, cf4PaymentRequired(t, endpoint, amount, payTo.Hex()))
			writer.WriteHeader(http.StatusPaymentRequired)
			return
		}
		encoded, decodeErr := base64.StdEncoding.DecodeString(payment)
		if decodeErr != nil {
			http.Error(writer, "invalid payment encoding", http.StatusBadRequest)
			return
		}
		var body map[string]any
		if json.Unmarshal(encoded, &body) != nil {
			http.Error(writer, "invalid payment payload", http.StatusBadRequest)
			return
		}
		inner, _ := body["payload"].(map[string]any)
		authorization, _ := inner["authorization"].(map[string]any)
		signature, _ := inner["signature"].(string)
		typedData, nonce, sig, verifyErr := cf4Authorization(authorization, signature)
		if verifyErr != nil {
			http.Error(writer, "invalid authorization", http.StatusBadRequest)
			return
		}
		digest, _, hashErr := apitypes.TypedDataAndHash(typedData)
		if hashErr != nil {
			http.Error(writer, "invalid typed data", http.StatusBadRequest)
			return
		}
		publicKey, recoverErr := gethcrypto.SigToPub(digest, sig)
		if recoverErr != nil || gethcrypto.PubkeyToAddress(*publicKey) != clientAddress {
			http.Error(writer, "signature does not match payer", http.StatusUnauthorized)
			return
		}

		mu.Lock()
		txHash := transactions[nonce]
		if txHash == "" {
			data, packErr := cf4TransferWithAuthorization(authorization, nonce, sig)
			if packErr == nil {
				var simulation string
				call := map[string]string{"from": fork.Accounts[1], "to": basefork.USDC, "data": data, "gas": "0x100000"}
				if simErr := fork.Call(ctx, "eth_call", []any{call, "latest"}, &simulation); simErr != nil {
					t.Logf("CF4 test facilitator simulation failed: %v", simErr)
				}
				txHash, packErr = fork.Send(ctx, fork.Accounts[1], basefork.USDC, data)
			}
			if packErr != nil {
				mu.Unlock()
				t.Logf("CF4 test facilitator local settlement failed: %v", packErr)
				http.Error(writer, "local settlement failed", http.StatusBadGateway)
				return
			}
			transactions[nonce] = txHash
		}
		dropReceipt := firstDispatch
		firstDispatch = false
		mu.Unlock()
		if dropReceipt {
			// Fail only after local-chain settlement. The buyer must retain and
			// replay the exact signed payload to reconcile the payment.
			http.Error(writer, "simulated dropped settlement receipt", http.StatusBadGateway)
			return
		}
		writePaymentResponse(writer, txHash, clientAddress.Hex())
	}))
	defer server.Close()
	endpoint = server.URL + "/priced/cf4-inference"

	command := cf4Command(t, endpoint, clientAddress.Hex())
	command.AmountMinor = offerQuote.AmountMinor
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(t.TempDir(), "credentials.enc"), "cf4-local-test-passphrase")
	require.NoError(t, err)
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	require.NoError(t, err)
	recoveryVault, err := instrument.NewCredentialClientResolver(client)
	require.NoError(t, err)
	var recoveryDigest, recoveryPayload string
	var recoveryReference string
	command.RecordRecoveryInput = func(recordCtx context.Context, digest, payment string) error {
		var storeErr error
		recoveryReference, storeErr = recoveryVault.StoreSettlementRecoveryInput(recordCtx, "cf4-local-fork-test", payment)
		if storeErr != nil {
			return storeErr
		}
		recoveryDigest, recoveryPayload = digest, payment
		return nil
	}
	adapter, err := New(server.Client(), cf4MemorySigner{key: clientKey})
	require.NoError(t, err)
	adapter.now = func() time.Time { return chainClock }
	first, err := adapter.Settle(ctx, command)
	require.Error(t, err, "the response is intentionally lost after chain settlement")
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	require.Equal(t, paymentDigest(recoveryPayload), recoveryDigest)
	require.Equal(t, recoveryDigest, first.ExternalID)
	require.Equal(t, recoveryPayload, first.ReceiptReference)
	require.NotEmpty(t, recoveryPayload)
	protectedPayload, err := recoveryVault.ResolveSettlementRecoveryInput(ctx, recoveryReference)
	require.NoError(t, err)
	require.Equal(t, recoveryPayload, protectedPayload, "Credential Authority must retain the exact signed bytes")

	var payload map[string]any
	encodedPayload, err := base64.StdEncoding.DecodeString(recoveryPayload)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encodedPayload, &payload))
	payloadBody := payload["payload"].(map[string]any)
	authorization := payloadBody["authorization"].(map[string]any)
	nonce := strings.TrimPrefix(authorization["nonce"].(string), "0x")
	require.Len(t, nonce, 64)

	// Recreate the rail adapter and reconcile using the exact signed bytes.
	restartedAdapter, err := New(server.Client(), cf4MemorySigner{key: clientKey})
	require.NoError(t, err)
	restartedAdapter.now = func() time.Time { return chainClock }
	reconciled, err := restartedAdapter.QueryOutcome(ctx, rail.Query{
		ExternalID: recoveryDigest, ReceiptReference: protectedPayload,
		IdempotencyKey: command.IdempotencyKey, Counterparty: command.Counterparty,
		Credential: command.Credential,
	})
	require.NoError(t, err)
	t.Logf("CF4 reconciliation: outcome=%s basis=%s detail=%s", reconciled.Outcome, reconciled.Basis, reconciled.Detail)
	require.Equal(t, rail.OutcomeSettled, reconciled.Outcome)
	require.Equal(t, "x402_facilitator_replay_confirmation", reconciled.Basis)
	require.Equal(t, recoveryDigest, paymentDigest(recoveryPayload))
	require.NotEmpty(t, reconciled.ExternalID)

	authorizationStateCall := "0x" + cf4Selector("authorizationState(address,bytes32)") + cf4AddressWord(clientAddress.Hex()) + nonce
	var authorizationState string
	require.NoError(t, fork.Call(ctx, "eth_call", []any{map[string]string{"to": basefork.USDC, "data": authorizationStateCall}, "latest"}, &authorizationState))
	state, ok := new(big.Int).SetString(strings.TrimPrefix(authorizationState, "0x"), 16)
	require.True(t, ok)
	require.Equal(t, big.NewInt(1), state, "the exact authorization nonce is consumed on-chain once")
	forkIdentity := fork.Identity()
	t.Logf("CF4 local Base fork: block=%s block_hash=%s per_call_cap_minor=1 amount_atoms=%s payload_digest=%s nonce=0x%s nonce_consumed=true tx=%s ambiguous_receipt_reconciled=true", forkIdentity.BlockNumber, forkIdentity.BlockHash, amount, recoveryDigest, nonce, reconciled.ExternalID)
}

type cf4MemorySigner struct{ key *ecdsa.PrivateKey }

func (s cf4MemorySigner) SignTypedData(_ context.Context, _ Credential, value map[string]any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(encoded, &typedData); err != nil {
		return "", err
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return "", err
	}
	signature, err := gethcrypto.Sign(digest, s.key)
	if err != nil {
		return "", err
	}
	signature[64] += 27
	return "0x" + hex.EncodeToString(signature), nil
}

func cf4Command(t *testing.T, endpoint, account string) rail.SettleCommand {
	t.Helper()
	credential, err := json.Marshal(Credential{
		EndpointURL: endpoint, SignerURL: "http://127.0.0.1:1", Account: account,
		Networks: []string{"eip155:8453"},
		Assets:   map[string]AssetPolicy{strings.ToLower(basefork.USDC): {Decimals: 6, Currency: "USD"}},
	})
	require.NoError(t, err)
	return rail.SettleCommand{
		SettlementID: "cf4-settlement", AuthorizationID: "cf4-mandate-auth",
		MandateReference: "cf4-test-mandate", InstrumentID: "cf4-test-wallet",
		IdempotencyKey: "cf4-idempotency-key", AmountMinor: 1, Currency: "USD",
		Counterparty: "127.0.0.1", Credential: string(credential),
	}
}

func cf4PaymentRequired(t *testing.T, endpoint, amount, payTo string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"x402Version": 2,
		"resource":    map[string]any{"url": endpoint, "description": "CF4 local-fork inference", "mimeType": "application/json"},
		"accepts": []any{map[string]any{
			"scheme": "exact", "network": "eip155:8453", "amount": amount,
			"payTo": payTo, "maxTimeoutSeconds": 300, "asset": basefork.USDC,
			"extra": map[string]any{"assetTransferMethod": "eip3009", "name": "USD Coin", "version": "2"},
		}},
		"extensions": map[string]any{},
	})
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(payload)
}

func cf4Authorization(authorization map[string]any, signature string) (apitypes.TypedData, string, []byte, error) {
	if authorization == nil {
		return apitypes.TypedData{}, "", nil, errors.New("authorization is missing")
	}
	encoded, err := json.Marshal(map[string]any{
		"types": map[string]any{
			"EIP712Domain": []map[string]string{
				{"name": "name", "type": "string"}, {"name": "version", "type": "string"},
				{"name": "chainId", "type": "uint256"}, {"name": "verifyingContract", "type": "address"},
			},
			"TransferWithAuthorization": []map[string]string{
				{"name": "from", "type": "address"}, {"name": "to", "type": "address"},
				{"name": "value", "type": "uint256"}, {"name": "validAfter", "type": "uint256"},
				{"name": "validBefore", "type": "uint256"}, {"name": "nonce", "type": "bytes32"},
			},
		},
		"primaryType": "TransferWithAuthorization",
		"domain":      map[string]any{"name": "USD Coin", "version": "2", "chainId": "8453", "verifyingContract": basefork.USDC},
		"message":     authorization,
	})
	if err != nil {
		return apitypes.TypedData{}, "", nil, err
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(encoded, &typedData); err != nil {
		return apitypes.TypedData{}, "", nil, err
	}
	nonce, ok := authorization["nonce"].(string)
	if !ok || len(strings.TrimPrefix(nonce, "0x")) != 64 {
		return apitypes.TypedData{}, "", nil, errors.New("authorization nonce is invalid")
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	if err != nil || len(sig) != 65 {
		return apitypes.TypedData{}, "", nil, errors.New("authorization signature is invalid")
	}
	sig[64] -= 27
	return typedData, strings.TrimPrefix(nonce, "0x"), sig, nil
}

func cf4TransferWithAuthorization(authorization map[string]any, nonce string, signature []byte) (string, error) {
	if authorization == nil || len(signature) != 65 {
		return "", errors.New("authorization or signature is invalid")
	}
	contractABI, err := abi.JSON(strings.NewReader("[{\"type\":\"function\",\"name\":\"transferWithAuthorization\",\"stateMutability\":\"nonpayable\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"validAfter\",\"type\":\"uint256\"},{\"name\":\"validBefore\",\"type\":\"uint256\"},{\"name\":\"nonce\",\"type\":\"bytes32\"},{\"name\":\"v\",\"type\":\"uint8\"},{\"name\":\"r\",\"type\":\"bytes32\"},{\"name\":\"s\",\"type\":\"bytes32\"}]}]"))
	if err != nil {
		return "", err
	}
	fromValue, fromOK := authorization["from"].(string)
	toValue, toOK := authorization["to"].(string)
	amountValue, amountOK := authorization["value"].(string)
	validAfterValue, validAfterOK := authorization["validAfter"].(string)
	validBeforeValue, validBeforeOK := authorization["validBefore"].(string)
	if !fromOK || !toOK || !amountOK || !validAfterOK || !validBeforeOK {
		return "", errors.New("authorization fields are malformed")
	}
	from, to := common.HexToAddress(fromValue), common.HexToAddress(toValue)
	value, ok := new(big.Int).SetString(amountValue, 10)
	if !ok {
		return "", errors.New("authorization amount is invalid")
	}
	validAfter, ok := new(big.Int).SetString(validAfterValue, 10)
	if !ok {
		return "", errors.New("authorization validAfter is invalid")
	}
	validBefore, ok := new(big.Int).SetString(validBeforeValue, 10)
	if !ok {
		return "", errors.New("authorization validBefore is invalid")
	}
	nonceBytes, err := hex.DecodeString(nonce)
	if err != nil || len(nonceBytes) != 32 {
		return "", errors.New("authorization nonce is invalid")
	}
	var nonceHash, r, s [32]byte
	copy(nonceHash[:], nonceBytes)
	copy(r[:], signature[:32])
	copy(s[:], signature[32:64])
	data, err := contractABI.Pack("transferWithAuthorization", from, to, value, validAfter, validBefore, nonceHash, uint8(signature[64]+27), r, s)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(data), nil
}

func writePaymentResponse(writer http.ResponseWriter, txHash, payer string) {
	response, _ := json.Marshal(map[string]any{
		"success": true, "payer": payer,
		"transaction": txHash, "network": "eip155:8453",
	})
	writer.Header().Set(paymentResponseHeader, base64.StdEncoding.EncodeToString(response))
	writer.WriteHeader(http.StatusOK)
}

func cf4Selector(signature string) string {
	return hex.EncodeToString(gethcrypto.Keccak256([]byte(signature))[:4])
}

func cf4AddressWord(address string) string {
	return strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(address), "0x")
}
