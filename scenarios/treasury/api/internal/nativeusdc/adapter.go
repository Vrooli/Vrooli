// Package nativeusdc owns custody-signed Base USDC execution. Treasury's
// settlement service supplies authorization, idempotency and protected replay.
package nativeusdc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"treasury/internal/rail"
	"treasury/internal/wallet"
)

const Asset = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Custody interface {
	Get(context.Context, string, string, string) (wallet.Wallet, error)
	SignUSDCTransfer(context.Context, wallet.SessionSignAuthorization, *types.Transaction) (*types.Transaction, error)
}
type Quote struct {
	WalletID                string    `json:"wallet_id"`
	Operator                string    `json:"operator_identity"`
	Network                 string    `json:"network"`
	From                    string    `json:"from"`
	Recipient               string    `json:"recipient"`
	AtomicAmount            string    `json:"atomic_amount"`
	ValueMinor              int64     `json:"value_minor"`
	AuthorizationValueMinor int64     `json:"authorization_value_minor,omitempty"`
	SigningPurpose          string    `json:"signing_purpose,omitempty"`
	Currency                string    `json:"currency"`
	Gas                     uint64    `json:"gas"`
	FeeCap                  string    `json:"fee_cap_wei"`
	Tip                     string    `json:"tip_wei"`
	MaxFee                  string    `json:"max_fee_wei"`
	ExpiresAt               time.Time `json:"expires_at"`
}
type Adapter struct {
	db             DB
	custody        Custody
	endpoint, mode string
	client         *http.Client
}

func New(db DB, custody Custody, endpoint, mode string) (*Adapter, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && u.Hostname() == "127.0.0.1")) {
		return nil, errors.New("native USDC requires an operator-configured HTTPS or loopback RPC")
	}
	if mode != "disabled" && mode != "local-fork" && mode != "mainnet" {
		return nil, errors.New("native USDC execution mode must be explicit")
	}
	if mode == "local-fork" && (u.Scheme != "http" || u.Hostname() != "127.0.0.1") {
		return nil, errors.New("local-fork execution requires loopback")
	}
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("native RPC redirects refused") }}
	return &Adapter{db: db, custody: custody, endpoint: endpoint, mode: mode, client: client}, nil
}
func (a *Adapter) Name() string { return "base-usdc" }
func (a *Adapter) Mode() string { return a.mode }

func (a *Adapter) Call(ctx context.Context, method string, params []any, out any) error {
	switch method {
	case "eth_chainId", "web3_clientVersion", "eth_getTransactionCount", "eth_gasPrice", "eth_maxPriorityFeePerGas", "eth_estimateGas", "eth_call", "eth_getTransactionReceipt", "eth_sendRawTransaction", "eth_getLogs", "eth_blockNumber", "eth_getBlockByNumber":
	default:
		return errors.New("method outside native USDC RPC contract")
	}
	if method == "eth_sendRawTransaction" && a.mode == "disabled" {
		return errors.New("native USDC execution is disabled")
	}
	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return errors.New("native RPC unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("native RPC HTTP %d", resp.StatusCode)
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Error != nil {
		return fmt.Errorf("native RPC %s failed (%d)", method, envelope.Error.Code)
	}
	if out != nil {
		return json.Unmarshal(envelope.Result, out)
	}
	return nil
}

func (a *Adapter) network(ctx context.Context) error {
	var chain string
	if err := a.Call(ctx, "eth_chainId", []any{}, &chain); err != nil {
		return err
	}
	if chain != "0x2105" {
		return errors.New("native USDC refuses a non-Base chain")
	}
	if a.mode == "local-fork" {
		var name string
		if err := a.Call(ctx, "web3_clientVersion", []any{}, &name); err != nil {
			return err
		}
		if !strings.Contains(strings.ToLower(name), "anvil") {
			return errors.New("local-fork mode requires the Anvil test node")
		}
	}
	return nil
}

func TransferData(recipient, atomic string) ([]byte, error) {
	value, ok := new(big.Int).SetString(atomic, 10)
	if !ok || value.Sign() <= 0 || value.BitLen() > 256 || value.String() != atomic || !common.IsHexAddress(recipient) || common.HexToAddress(recipient) == (common.Address{}) {
		return nil, errors.New("positive canonical token atoms and recipient are required")
	}
	data := make([]byte, 68)
	copy(data, []byte{0xa9, 0x05, 0x9c, 0xbb})
	copy(data[16:36], common.HexToAddress(recipient).Bytes())
	value.FillBytes(data[36:])
	return data, nil
}

func quantity(s string) (*big.Int, error) {
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok || !strings.HasPrefix(s, "0x") || n.Sign() < 0 {
		return nil, errors.New("invalid RPC quantity")
	}
	return n, nil
}

func (a *Adapter) Preview(ctx context.Context, w wallet.Wallet, recipient, atoms string, value int64, currency, maxFee string) (Quote, error) {
	if err := a.network(ctx); err != nil {
		return Quote{}, err
	}
	data, err := TransferData(recipient, atoms)
	if err != nil {
		return Quote{}, err
	}
	expectedAtoms, amountErr := wallet.AtomicAmountForUSDCMinor(value, currency)
	if w.Network != "eip155:8453" || amountErr != nil || expectedAtoms != atoms {
		return Quote{}, errors.New("explicit valuation and Base wallet are required")
	}
	var gasHex, priceHex, tipHex string
	if err := a.Call(ctx, "eth_estimateGas", []any{map[string]string{"from": w.Address, "to": Asset, "data": "0x" + hex.EncodeToString(data)}}, &gasHex); err != nil {
		return Quote{}, err
	}
	if err := a.Call(ctx, "eth_gasPrice", []any{}, &priceHex); err != nil {
		return Quote{}, err
	}
	if err := a.Call(ctx, "eth_maxPriorityFeePerGas", []any{}, &tipHex); err != nil {
		return Quote{}, err
	}
	gas, err := quantity(gasHex)
	if err != nil || !gas.IsUint64() {
		return Quote{}, errors.New("invalid gas estimate")
	}
	gasLimit := gas.Uint64() + gas.Uint64()/5 + 1000
	if gasLimit > 1_000_000 {
		return Quote{}, errors.New("gas estimate exceeds transfer bound")
	}
	price, err := quantity(priceHex)
	if err != nil {
		return Quote{}, err
	}
	tip, err := quantity(tipHex)
	if err != nil {
		return Quote{}, err
	}
	cap := new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(2)), tip)
	bound := new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), cap)
	// Base charges an L1 data fee in addition to EVM gas. Query its canonical
	// oracle using a conservative encoded-transaction size, then include it.
	oracleData := append(gethcrypto.Keccak256([]byte("getL1FeeUpperBound(uint256)"))[:4], make([]byte, 32)...)
	big.NewInt(300).FillBytes(oracleData[4:])
	var l1Hex string
	if err := a.Call(ctx, "eth_call", []any{map[string]string{"to": "0x420000000000000000000000000000000000000f", "data": "0x" + hex.EncodeToString(oracleData)}, "latest"}, &l1Hex); err != nil {
		return Quote{}, errors.New("Base L1 fee preview unavailable")
	}
	l1, err := quantity(l1Hex)
	if err != nil {
		return Quote{}, err
	}
	bound.Add(bound, l1)
	if maxFee != "" {
		max, ok := new(big.Int).SetString(maxFee, 10)
		if !ok || max.Sign() <= 0 || bound.Cmp(max) > 0 {
			return Quote{}, errors.New("fee preview exceeds the requested maximum")
		}
	}
	return Quote{WalletID: w.ID, Operator: w.OperatorIdentity, Network: w.Network, From: w.Address, Recipient: strings.ToLower(recipient), AtomicAmount: atoms, ValueMinor: value, Currency: currency, Gas: gasLimit, FeeCap: cap.String(), Tip: tip.String(), MaxFee: bound.String(), ExpiresAt: time.Now().UTC().Add(2 * time.Minute)}, nil
}

type replay struct {
	Quote    Quote  `json:"quote"`
	Unsigned string `json:"unsigned_transaction"`
	Raw      string `json:"raw_transaction,omitempty"`
}

func (a *Adapter) Settle(ctx context.Context, c rail.SettleCommand) (rail.Result, error) {
	if err := rail.ValidateSettle(c); err != nil {
		return rail.Result{}, err
	}
	if a.mode == "disabled" {
		return failed("native USDC execution is disabled"), nil
	}
	var quote Quote
	if json.Unmarshal([]byte(c.Credential), &quote) != nil {
		return failed("native USDC quote is invalid"), nil
	}
	expectedAtoms, amountErr := wallet.AtomicAmountForUSDCMinor(quote.ValueMinor, quote.Currency)
	authorizationValue := quote.AuthorizationValueMinor
	if authorizationValue == 0 {
		authorizationValue = quote.ValueMinor
	}
	if amountErr != nil || expectedAtoms != quote.AtomicAmount || authorizationValue != c.AmountMinor || quote.Currency != c.Currency || quote.Recipient != c.Counterparty || !time.Now().Before(quote.ExpiresAt) {
		return failed("native USDC quote is expired or does not cover this exact charge"), nil
	}
	if err := a.network(ctx); err != nil {
		return failed(err.Error()), nil
	}
	w, err := a.custody.Get(ctx, quote.WalletID, quote.Network, quote.Operator)
	if err != nil || !strings.EqualFold(w.Address, quote.From) {
		return failed("native USDC custody binding changed"), nil
	}
	// A different in-flight settlement keeps the wallet's nonce slot. A stale
	// terminal slot can be atomically replaced by this claim across processes.
	_, err = a.db.ExecContext(ctx, `INSERT INTO native_usdc_send_locks(wallet_id,settlement_id) VALUES(?,?) ON CONFLICT(wallet_id) DO UPDATE SET settlement_id=excluded.settlement_id WHERE native_usdc_send_locks.settlement_id=excluded.settlement_id OR EXISTS(SELECT 1 FROM settlements WHERE id=native_usdc_send_locks.settlement_id AND outcome IN ('failed','settled'))`, w.ID, c.SettlementID)
	if err != nil {
		return rail.Result{}, err
	}
	var holder string
	if err = a.db.QueryRowContext(ctx, `SELECT settlement_id FROM native_usdc_send_locks WHERE wallet_id=?`, w.ID).Scan(&holder); err != nil {
		return rail.Result{}, err
	}
	if holder != c.SettlementID {
		return failed("wallet already has an unresolved transfer"), nil
	}
	var nonceHex string
	if err = a.Call(ctx, "eth_getTransactionCount", []any{w.Address, "pending"}, &nonceHex); err != nil {
		return failed(err.Error()), nil
	}
	nonce, err := quantity(nonceHex)
	if err != nil || !nonce.IsUint64() {
		return failed("nonce unavailable"), nil
	}
	data, err := TransferData(quote.Recipient, quote.AtomicAmount)
	if err != nil {
		return failed(err.Error()), nil
	}
	cap, ok := new(big.Int).SetString(quote.FeeCap, 10)
	if !ok {
		return failed("invalid fee cap"), nil
	}
	tip, ok := new(big.Int).SetString(quote.Tip, 10)
	if !ok {
		return failed("invalid priority fee"), nil
	}
	asset := common.HexToAddress(Asset)
	unsigned := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(8453), Nonce: nonce.Uint64(), To: &asset, Gas: quote.Gas, GasFeeCap: cap, GasTipCap: tip, Data: data})
	unsignedBytes, err := unsigned.MarshalBinary()
	if err != nil {
		return rail.Result{}, err
	}
	replayIntent := replay{Quote: quote, Unsigned: "0x" + hex.EncodeToString(unsignedBytes)}
	intentDigest := sha256.Sum256(unsignedBytes)
	intentHash := "sha256:" + hex.EncodeToString(intentDigest[:])
	if c.RecordRecoveryInput == nil {
		return failed("protected transaction intent storage is required before custody signing"), nil
	}
	intentPayload, err := json.Marshal(replayIntent)
	if err != nil {
		return rail.Result{}, err
	}
	if err := c.RecordRecoveryInput(ctx, intentHash, string(intentPayload)); err != nil {
		return failed("unsigned transaction intent could not be retained before signing"), nil
	}
	signRequest := wallet.SessionSignAuthorization{
		WalletID: w.ID, Network: w.Network, OperatorIdentity: w.OperatorIdentity,
		MandateID: c.MandateReference, AuthorizationID: c.AuthorizationID,
		SigningPurpose: quote.SigningPurpose, AuthorizationValueMinor: c.AmountMinor,
		Recipient: quote.Recipient, AtomicAmount: quote.AtomicAmount,
		Currency: quote.Currency, ValueMinor: quote.ValueMinor,
	}
	signed, err := a.custody.SignUSDCTransfer(ctx, signRequest, unsigned)
	if err != nil {
		return failed("custody refused transfer signing: " + err.Error()), nil
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		return rail.Result{}, err
	}
	replayIntent.Raw = "0x" + hex.EncodeToString(raw)
	payload, err := json.Marshal(replayIntent)
	if err != nil {
		return rail.Result{}, err
	}
	if err := c.RecordRecoveryInput(ctx, intentHash, string(payload)); err != nil {
		return rail.Result{Outcome: rail.OutcomeUnknown, Basis: "base_usdc_signing_recovery_pending", Detail: "signed transaction storage is inconclusive; recover the retained unsigned intent before dispatch"}, nil
	}
	return a.dispatch(ctx, replay{Quote: quote, Raw: "0x" + hex.EncodeToString(raw)}, signed.Hash().Hex())
}

func (a *Adapter) QueryOutcome(ctx context.Context, q rail.Query) (rail.Result, error) {
	var retained replay
	if json.Unmarshal([]byte(q.ReceiptReference), &retained) != nil || retained.Quote.Recipient != q.Counterparty || retained.Quote.ValueMinor <= 0 {
		return rail.Result{}, errors.New("protected transaction replay is invalid")
	}
	unsignedRaw, err := hex.DecodeString(strings.TrimPrefix(retained.Unsigned, "0x"))
	if err != nil {
		return rail.Result{}, err
	}
	digest := sha256.Sum256(unsignedRaw)
	if q.ExternalID != "sha256:"+hex.EncodeToString(digest[:]) {
		return rail.Result{}, errors.New("protected unsigned transaction intent digest mismatch")
	}
	var unsigned types.Transaction
	if err := unsigned.UnmarshalBinary(unsignedRaw); err != nil || unsigned.Type() != types.DynamicFeeTxType || unsigned.To() == nil || *unsigned.To() != common.HexToAddress(Asset) {
		return rail.Result{}, errors.New("protected unsigned transaction intent is invalid")
	}
	if retained.Raw == "" {
		if q.AuthorizationID == "" {
			return rail.Result{}, errors.New("authorization ID is required to recover an unsigned signing intent")
		}
		authorizationValue := retained.Quote.AuthorizationValueMinor
		if authorizationValue == 0 {
			authorizationValue = retained.Quote.ValueMinor
		}
		signed, signErr := a.custody.SignUSDCTransfer(ctx, wallet.SessionSignAuthorization{
			WalletID: retained.Quote.WalletID, Network: retained.Quote.Network, OperatorIdentity: retained.Quote.Operator,
			MandateID: q.MandateReference, AuthorizationID: q.AuthorizationID, SigningPurpose: retained.Quote.SigningPurpose,
			AuthorizationValueMinor: authorizationValue, Recipient: retained.Quote.Recipient,
			AtomicAmount: retained.Quote.AtomicAmount, Currency: retained.Quote.Currency, ValueMinor: retained.Quote.ValueMinor,
		}, &unsigned)
		if signErr != nil {
			return rail.Result{}, fmt.Errorf("recover protected signing intent: %w", signErr)
		}
		raw, marshalErr := signed.MarshalBinary()
		if marshalErr != nil {
			return rail.Result{}, marshalErr
		}
		retained.Raw = "0x" + hex.EncodeToString(raw)
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(retained.Raw, "0x"))
	if err != nil {
		return rail.Result{}, err
	}
	var tx types.Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		return rail.Result{}, err
	}
	if !sameTransferIntent(&unsigned, &tx) {
		return rail.Result{}, errors.New("protected signed transaction does not match its unsigned intent")
	}
	return a.dispatch(ctx, retained, tx.Hash().Hex())
}

func sameTransferIntent(unsigned, signed *types.Transaction) bool {
	if unsigned == nil || signed == nil {
		return false
	}
	_, r, _ := signed.RawSignatureValues()
	return signed.Type() == unsigned.Type() && signed.ChainId().Cmp(unsigned.ChainId()) == 0 &&
		signed.Nonce() == unsigned.Nonce() && signed.To() != nil && unsigned.To() != nil && *signed.To() == *unsigned.To() &&
		signed.Value().Cmp(unsigned.Value()) == 0 && signed.Gas() == unsigned.Gas() && signed.GasFeeCap().Cmp(unsigned.GasFeeCap()) == 0 &&
		signed.GasTipCap().Cmp(unsigned.GasTipCap()) == 0 && bytes.Equal(signed.Data(), unsigned.Data()) && r != nil
}

func (a *Adapter) dispatch(ctx context.Context, r replay, hash string) (rail.Result, error) {
	if err := a.network(ctx); err != nil {
		return rail.Result{}, err
	}
	// Receipt-first reconciliation avoids a new signature or nonce. A retry
	// broadcasts only the same protected raw transaction, with the same hash.
	if result, found, err := a.receipt(ctx, hash); err == nil && found {
		return result, nil
	}
	var sent string
	if err := a.Call(ctx, "eth_sendRawTransaction", []any{r.Raw}, &sent); err != nil {
		return rail.Result{Outcome: rail.OutcomeUnknown, Detail: "signed transfer dispatch is inconclusive"}, nil
	}
	if !strings.EqualFold(sent, hash) {
		return rail.Result{Outcome: rail.OutcomeUnknown, Detail: "RPC returned a different transaction hash"}, nil
	}
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	for {
		if result, found, err := a.receipt(ctx, hash); err == nil && found {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return rail.Result{Outcome: rail.OutcomeUnknown, Detail: "transfer confirmation deadline"}, nil
		case <-deadline.C:
			return rail.Result{Outcome: rail.OutcomeUnknown, Detail: "transfer awaiting inclusion"}, nil
		case <-ticker.C:
		}
	}
}

func (a *Adapter) receipt(ctx context.Context, hash string) (rail.Result, bool, error) {
	var receipt *struct {
		Status          string `json:"status"`
		TransactionHash string `json:"transactionHash"`
		BlockNumber     string `json:"blockNumber"`
	}
	if err := a.Call(ctx, "eth_getTransactionReceipt", []any{hash}, &receipt); err != nil {
		return rail.Result{}, false, err
	}
	if receipt == nil {
		return rail.Result{}, false, nil
	}
	if !strings.EqualFold(receipt.TransactionHash, hash) {
		return rail.Result{}, false, errors.New("transaction receipt mismatch")
	}
	if receipt.Status != "0x1" {
		return rail.Result{Outcome: rail.OutcomeFailed, ExternalID: hash, ReceiptReference: "eip155:8453/tx/" + hash, Basis: "base_usdc_reverted", Detail: "native USDC transfer reverted"}, true, nil
	}
	var block struct {
		Timestamp string `json:"timestamp"`
	}
	if err := a.Call(ctx, "eth_getBlockByNumber", []any{receipt.BlockNumber, false}, &block); err != nil {
		return rail.Result{}, false, err
	}
	at, err := quantity(block.Timestamp)
	if err != nil || !at.IsInt64() {
		return rail.Result{}, false, errors.New("receipt block time unavailable")
	}
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: hash, ReceiptReference: "eip155:8453/tx/" + hash, Basis: "base_usdc_included", OccurredAt: time.Unix(at.Int64(), 0).UTC(), Detail: "custody-signed native USDC transfer included on Base"}, true, nil
}

func failed(detail string) rail.Result {
	return rail.Result{Outcome: rail.OutcomeFailed, Basis: "native_usdc_preflight_refused", Detail: detail}
}
