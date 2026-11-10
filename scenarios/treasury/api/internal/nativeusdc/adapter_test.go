package nativeusdc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"

	"treasury/internal/rail"
	"treasury/internal/settlement"
	"treasury/internal/wallet"
)

type replayCustody struct {
	key     []byte
	wallet  wallet.Wallet
	mu      sync.Mutex
	signs   int
	request wallet.SessionSignAuthorization
}

func (c *replayCustody) Get(_ context.Context, id, network, operator string) (wallet.Wallet, error) {
	if id != c.wallet.ID || network != c.wallet.Network || operator != c.wallet.OperatorIdentity {
		return wallet.Wallet{}, fmt.Errorf("wrong wallet binding")
	}
	return c.wallet, nil
}

func (c *replayCustody) SignUSDCTransfer(_ context.Context, request wallet.SessionSignAuthorization, tx *types.Transaction) (*types.Transaction, error) {
	c.mu.Lock()
	c.signs++
	c.request = request
	c.mu.Unlock()
	key, err := gethcrypto.ToECDSA(c.key)
	if err != nil {
		return nil, err
	}
	return types.SignTx(tx, types.NewLondonSigner(big.NewInt(8453)), key)
}

func TestUnsignedSigningIntentRecoversIdenticalTransactionWithoutDoubleDispatch(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(settlement.Schema), database.SchemaProviderFunc(Schema)))
	db.SetMaxOpenConns(1)
	_, err := db.ExecContext(ctx, `PRAGMA foreign_keys=OFF`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO settlements(id,authorization_id,mandate_id,instrument_id,rail,idempotency_key,amount_minor,currency,counterparty,outcome,external_id,receipt_reference,basis,detail,occurred_at,created_at,updated_at,retain_until) VALUES(?,?,?,?,?,?,?,?,?,'calling','','','base_usdc_preflight_only','','',?,?,?)`,
		"settlement-recovery", "auth-recovery", "mandate-recovery", "instrument-recovery", "base-usdc", "settlement-recovery-key", 100, "USD", "0x0000000000000000000000000000000000000001",
		time.Now().Format(time.RFC3339Nano), time.Now().Format(time.RFC3339Nano), time.Now().Add(time.Hour).Format(time.RFC3339Nano))
	require.NoError(t, err)
	key, err := gethcrypto.GenerateKey()
	require.NoError(t, err)
	keyBytes := gethcrypto.FromECDSA(key)
	address := gethcrypto.PubkeyToAddress(key.PublicKey).Hex()
	custody := &replayCustody{key: keyBytes, wallet: wallet.Wallet{ID: "session-wallet", Network: "eip155:8453", OperatorIdentity: "operator:1", Address: address}}
	var txHash string
	dispatches := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var input struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
			ID     int               `json:"id"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&input))
		var result any
		switch input.Method {
		case "eth_chainId":
			result = "0x2105"
		case "web3_clientVersion":
			result = "anvil/v1.0"
		case "eth_estimateGas":
			result = "0x5208"
		case "eth_gasPrice":
			result = "0x3b9aca00"
		case "eth_maxPriorityFeePerGas":
			result = "0x1"
		case "eth_call":
			result = "0x1"
		case "eth_getTransactionCount":
			result = "0x7"
		case "eth_getTransactionReceipt":
			var hash string
			require.NoError(t, json.Unmarshal(input.Params[0], &hash))
			if strings.EqualFold(hash, txHash) {
				result = map[string]string{"status": "0x1", "transactionHash": txHash, "blockNumber": "0x1"}
			}
		case "eth_getBlockByNumber":
			result = map[string]string{"timestamp": "0x672e0000"}
		case "eth_sendRawTransaction":
			var raw string
			require.NoError(t, json.Unmarshal(input.Params[0], &raw))
			bytes, decodeErr := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
			require.NoError(t, decodeErr)
			var tx types.Transaction
			require.NoError(t, tx.UnmarshalBinary(bytes))
			txHash = tx.Hash().Hex()
			dispatches++
			result = txHash
		default:
			t.Errorf("unexpected RPC method %s", input.Method)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"jsonrpc": "2.0", "id": input.ID, "result": result})
	}))
	defer server.Close()
	adapter, err := New(db, custody, server.URL, "local-fork")
	require.NoError(t, err)
	quote := Quote{
		WalletID: custody.wallet.ID, Operator: custody.wallet.OperatorIdentity, Network: custody.wallet.Network,
		From: address, Recipient: "0x0000000000000000000000000000000000000001", AtomicAmount: "1000000",
		ValueMinor: 100, Currency: "USD", Gas: 100_000, FeeCap: "2000000000", Tip: "1", ExpiresAt: time.Now().Add(time.Minute),
	}
	credential, err := json.Marshal(quote)
	require.NoError(t, err)
	var retained string
	callbacks := 0
	result, err := adapter.Settle(ctx, rail.SettleCommand{
		SettlementID: "settlement-recovery", AuthorizationID: "auth-recovery", MandateReference: "mandate-recovery",
		InstrumentID: "instrument-recovery", IdempotencyKey: "settlement-recovery-key", AmountMinor: 100,
		Currency: "USD", Counterparty: quote.Recipient, Credential: string(credential),
		RecordRecoveryInput: func(_ context.Context, digest, payload string) error {
			callbacks++
			if callbacks == 2 {
				return fmt.Errorf("simulated crash while persisting signed bytes")
			}
			var value replay
			require.NoError(t, json.Unmarshal([]byte(payload), &value))
			unsigned, decodeErr := hex.DecodeString(strings.TrimPrefix(value.Unsigned, "0x"))
			require.NoError(t, decodeErr)
			fingerprint := sha256.Sum256(unsigned)
			require.Equal(t, "sha256:"+hex.EncodeToString(fingerprint[:]), digest)
			retained = payload
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeUnknown, result.Outcome)
	require.Equal(t, 2, callbacks)
	require.Equal(t, 1, custody.signs, "first call created only one deterministic signature")
	require.Zero(t, dispatches, "dispatch is forbidden until signed replay bytes have been retained")

	var unsignedIntent replay
	require.NoError(t, json.Unmarshal([]byte(retained), &unsignedIntent))
	unsignedBytes, err := hex.DecodeString(strings.TrimPrefix(unsignedIntent.Unsigned, "0x"))
	require.NoError(t, err)
	digest := sha256.Sum256(unsignedBytes)
	query := rail.Query{
		SettlementID: "settlement-recovery", AuthorizationID: "auth-recovery", MandateReference: "mandate-recovery",
		InstrumentID: "instrument-recovery", ExternalID: "sha256:" + hex.EncodeToString(digest[:]),
		ReceiptReference: retained, IdempotencyKey: "settlement-recovery-key", Counterparty: quote.Recipient,
	}
	recovered, err := adapter.QueryOutcome(ctx, query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, recovered.Outcome)
	recoveredHash := recovered.ExternalID
	require.NotEmpty(t, recoveredHash)
	require.Equal(t, 1, dispatches)

	retried, err := adapter.QueryOutcome(ctx, query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, retried.Outcome)
	require.Equal(t, recoveredHash, retried.ExternalID)
	require.Equal(t, 1, dispatches, "reconciliation must reuse the exact transaction and receipt")
}
