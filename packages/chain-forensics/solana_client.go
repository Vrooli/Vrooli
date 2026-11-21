package chainforensics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultSolanaRPC = "https://solana-rpc.publicnode.com"
const maxSolanaResponseBytes = 1 << 20

var ErrSolanaProviderRateLimited = errors.New("Solana RPC provider rate limited the request")

type SolanaProviderError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *SolanaProviderError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("Solana RPC returned HTTP %d; retry after %s", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("Solana RPC returned HTTP %d", e.StatusCode)
}
func (e *SolanaProviderError) Is(target error) bool {
	return target == ErrSolanaProviderRateLimited && e.StatusCode == http.StatusTooManyRequests
}

type SolanaSignature struct {
	Signature string          `json:"signature"`
	Slot      uint64          `json:"slot"`
	BlockTime *int64          `json:"blockTime"`
	Err       json.RawMessage `json:"err"`
}

type CapturedSolanaTransaction struct {
	Transaction SolanaTransaction
	Source      Source
	Occurrences []Occurrence
	Failed      bool
}

type CapturedSolanaPage struct {
	Address      string
	Before       string
	NextBefore   string
	Signatures   []SolanaSignature
	Transactions []CapturedSolanaTransaction
	Source       Source
}

type SolanaTransaction struct {
	Slot        uint64 `json:"slot"`
	BlockTime   *int64 `json:"blockTime"`
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys []json.RawMessage `json:"accountKeys"`
		} `json:"message"`
	} `json:"transaction"`
	Meta struct {
		Err               json.RawMessage      `json:"err"`
		Fee               uint64               `json:"fee"`
		PreTokenBalances  []SolanaTokenBalance `json:"preTokenBalances"`
		PostTokenBalances []SolanaTokenBalance `json:"postTokenBalances"`
	} `json:"meta"`
}

type SolanaTokenBalance struct {
	AccountIndex uint32 `json:"accountIndex"`
	Mint         string `json:"mint"`
	Owner        string `json:"owner"`
	ProgramID    string `json:"programId"`
	UIAmount     struct {
		Amount   string `json:"amount"`
		Decimals uint32 `json:"decimals"`
	} `json:"uiTokenAmount"`
}

type SolanaRPCClient struct {
	endpoint string
	http     *http.Client
	now      func() time.Time
	interval time.Duration
	gate     sync.Mutex
	last     time.Time
	cacheMu  sync.Mutex
	pages    map[string]solanaPageEntry
	txs      map[string]solanaTxEntry
	request  atomic.Uint64
}

type solanaPageEntry struct {
	page    CapturedSolanaPage
	expires time.Time
}
type solanaTxEntry struct {
	tx      CapturedSolanaTransaction
	expires time.Time
}

func NewSolanaRPCClient(endpoint string, client *http.Client, now func() time.Time) (*SolanaRPCClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname()))) {
		return nil, errors.New("Solana RPC endpoint must be HTTPS (HTTP only on loopback), without credentials, query, or fragment")
	}
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("Solana RPC redirects are disabled") }}
	}
	if now == nil {
		now = time.Now
	}
	return &SolanaRPCClient{endpoint: strings.TrimRight(endpoint, "/"), http: client, now: now, interval: 250 * time.Millisecond, pages: make(map[string]solanaPageEntry), txs: make(map[string]solanaTxEntry)}, nil
}

func (c *SolanaRPCClient) FetchAddressPage(ctx context.Context, address, before string) (CapturedSolanaPage, error) {
	if !validSolanaPubkey(address) {
		return CapturedSolanaPage{}, errors.New("Solana address must be a base58-encoded 32-byte public key")
	}
	if before != "" && !validSolanaSignature(before) {
		return CapturedSolanaPage{}, errors.New("Solana before cursor must be a base58-encoded 64-byte signature")
	}
	key := address + ":" + before
	c.cacheMu.Lock()
	if hit, ok := c.pages[key]; ok && c.now().Before(hit.expires) {
		page := cloneSolanaPage(hit.page)
		c.cacheMu.Unlock()
		return page, nil
	}
	delete(c.pages, key)
	c.cacheMu.Unlock()
	params := []any{address, map[string]any{"commitment": "finalized", "limit": 25}}
	if before != "" {
		params[1].(map[string]any)["before"] = before
	}
	body, responseBody, err := c.callCaptured(ctx, "getSignaturesForAddress", params)
	if err != nil {
		return CapturedSolanaPage{}, err
	}
	var signatures []SolanaSignature
	if err := json.Unmarshal(body, &signatures); err != nil {
		return CapturedSolanaPage{}, fmt.Errorf("decode Solana signature page: %w", err)
	}
	if len(signatures) > 25 {
		return CapturedSolanaPage{}, errors.New("Solana RPC returned more than the requested 25 signatures")
	}
	page := CapturedSolanaPage{Address: address, Before: before, Signatures: signatures, Transactions: make([]CapturedSolanaTransaction, 0, len(signatures))}
	seen := make(map[string]struct{}, len(signatures))
	pageSource := sourceForBody("solana-address-page-"+address+"-"+before, "Solana finalized address signature page", c.endpoint, "getSignaturesForAddress(finalized,limit=25)", c.now().UTC(), body, "One bounded finalized signature page; exact result and HTTP response bytes are retained.", responseBody)
	pageSource.ID += "-" + pageSource.SHA256
	page.Source = pageSource
	for _, sig := range signatures {
		if !validSolanaSignature(sig.Signature) || sig.Slot == 0 {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page contains invalid or failed entry %q", sig.Signature)
		}
		if _, exists := seen[sig.Signature]; exists {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature page repeats %q", sig.Signature)
		}
		seen[sig.Signature] = struct{}{}
		captured, err := c.fetchTransaction(ctx, sig.Signature, pageSource.ID, sig.BlockTime)
		if err != nil {
			return CapturedSolanaPage{}, err
		}
		if captured.Transaction.Slot != sig.Slot {
			return CapturedSolanaPage{}, fmt.Errorf("Solana signature slot %d disagrees with transaction slot %d", sig.Slot, captured.Transaction.Slot)
		}
		page.Transactions = append(page.Transactions, captured)
	}
	if len(signatures) > 0 {
		page.NextBefore = signatures[len(signatures)-1].Signature
	}
	c.cacheMu.Lock()
	if len(c.pages) >= 16 {
		for oldKey, old := range c.pages {
			if !c.now().Before(old.expires) || len(c.pages) >= 16 {
				delete(c.pages, oldKey)
			}
		}
	}
	c.pages[key] = solanaPageEntry{page: cloneSolanaPage(page), expires: c.now().Add(10 * time.Minute)}
	c.cacheMu.Unlock()
	return page, nil
}

func (c *SolanaRPCClient) fetchTransaction(ctx context.Context, signature, pageSourceID string, pageBlockTime *int64) (CapturedSolanaTransaction, error) {
	c.cacheMu.Lock()
	if hit, ok := c.txs[signature]; ok && c.now().Before(hit.expires) {
		tx := cloneCapturedSolanaTx(hit.tx)
		c.cacheMu.Unlock()
		return tx, nil
	}
	delete(c.txs, signature)
	c.cacheMu.Unlock()
	body, responseBody, err := c.callCaptured(ctx, "getTransaction", []any{signature, map[string]any{"commitment": "finalized", "encoding": "jsonParsed", "maxSupportedTransactionVersion": 1}})
	if err != nil {
		return CapturedSolanaTransaction{}, err
	}
	var tx SolanaTransaction
	if err := json.Unmarshal(body, &tx); err != nil {
		return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana transaction: %w", err)
	}
	if tx.Slot == 0 || len(tx.Transaction.Signatures) == 0 || tx.Transaction.Signatures[0] != signature {
		return CapturedSolanaTransaction{}, errors.New("Solana transaction is missing finalized identity")
	}
	blockTimestamp := tx.BlockTime
	if blockTimestamp == nil {
		blockTimestamp = pageBlockTime
	}
	keyByIndex := make(map[uint32]string, len(tx.Transaction.Message.AccountKeys))
	for i, raw := range tx.Transaction.Message.AccountKeys {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			var key struct {
				Pubkey string `json:"pubkey"`
			}
			if err := json.Unmarshal(raw, &key); err != nil {
				return CapturedSolanaTransaction{}, fmt.Errorf("decode Solana account key %d: %w", i, err)
			}
			text = key.Pubkey
		}
		keyByIndex[uint32(i)] = text
	}
	pre := make(map[uint32]SolanaTokenBalance, len(tx.Meta.PreTokenBalances))
	post := make(map[uint32]SolanaTokenBalance, len(tx.Meta.PostTokenBalances))
	for _, balance := range tx.Meta.PreTokenBalances {
		pre[balance.AccountIndex] = balance
	}
	for _, balance := range tx.Meta.PostTokenBalances {
		post[balance.AccountIndex] = balance
	}
	indexes := make(map[uint32]struct{}, len(pre)+len(post))
	for i := range pre {
		indexes[i] = struct{}{}
	}
	for i := range post {
		indexes[i] = struct{}{}
	}
	ordered := make([]int, 0, len(indexes))
	for i := range indexes {
		ordered = append(ordered, int(i))
	}
	sort.Ints(ordered)
	blockTime := ""
	if blockTimestamp != nil {
		blockTime = time.Unix(*blockTimestamp, 0).UTC().Format(time.RFC3339)
	}
	source := sourceForBody("solana-transaction-"+signature, "Solana finalized transaction", c.endpoint, "getTransaction(jsonParsed,finalized)", c.now().UTC(), body, "Exact pre/post SPL token-account balances; exact result and HTTP response bytes are retained. Address-page source: "+pageSourceID, responseBody)
	if blockTimestamp == nil {
		source.Limitations += " Provider omitted block time; slot is retained but occurrence timestamp is unavailable."
	}
	captured := CapturedSolanaTransaction{Transaction: tx, Source: source, Failed: tx.Meta.Err != nil && string(tx.Meta.Err) != "null"}
	if captured.Failed {
		source.Limitations += " Transaction failed on-chain; no token balance deltas are treated as transfers."
		captured.Source = source
	}
	for _, rawIndex := range ordered {
		if captured.Failed {
			break
		}
		index := uint32(rawIndex)
		a, aok := pre[index]
		b, bok := post[index]
		mint, decimals, owner := "", uint32(0), ""
		if aok {
			mint, decimals, owner = a.Mint, a.UIAmount.Decimals, a.Owner
		}
		if bok {
			if mint != "" && mint != b.Mint {
				return CapturedSolanaTransaction{}, errors.New("Solana token mint changed for one account index")
			}
			if mint == "" {
				mint = b.Mint
			}
			if aok && decimals != b.UIAmount.Decimals {
				return CapturedSolanaTransaction{}, errors.New("Solana token decimals changed across one transaction")
			}
			decimals = b.UIAmount.Decimals
			if owner == "" {
				owner = b.Owner
			}
		}
		if mint == "" {
			return CapturedSolanaTransaction{}, errors.New("Solana token balance is missing mint")
		}
		before, after := new(big.Int), new(big.Int)
		if aok {
			if _, ok := before.SetString(a.UIAmount.Amount, 10); !ok || before.Sign() < 0 {
				return CapturedSolanaTransaction{}, errors.New("invalid pre-transaction SPL atomic amount")
			}
		}
		if bok {
			if _, ok := after.SetString(b.UIAmount.Amount, 10); !ok || after.Sign() < 0 {
				return CapturedSolanaTransaction{}, errors.New("invalid post-transaction SPL atomic amount")
			}
		}
		delta := new(big.Int).Sub(after, before)
		if delta.Sign() == 0 {
			continue
		}
		account := owner
		if account == "" {
			account = keyByIndex[index]
		}
		direction, from, to := "output", "", account
		value := new(big.Int).Set(delta)
		if delta.Sign() < 0 {
			direction, from, to = "input", account, ""
			value.Neg(value)
		}
		occurrence := Occurrence{ID: fmt.Sprintf("solana:%s:spl:%d", signature, index), Chain: ChainSolana, Transaction: signature, Block: tx.Slot, Timestamp: blockTime, EventIndex: index, Direction: direction, From: from, To: to, Asset: "SPL:" + mint, Atomic: value.String(), Scale: decimals, Kind: EvidenceObservation, SourceID: source.ID}
		captured.Occurrences = append(captured.Occurrences, occurrence)
	}
	c.cacheMu.Lock()
	if len(c.txs) >= 128 {
		for oldKey, old := range c.txs {
			if !c.now().Before(old.expires) || len(c.txs) >= 128 {
				delete(c.txs, oldKey)
			}
		}
	}
	c.txs[signature] = solanaTxEntry{tx: cloneCapturedSolanaTx(captured), expires: c.now().Add(10 * time.Minute)}
	c.cacheMu.Unlock()
	return captured, nil
}

func (c *SolanaRPCClient) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	result, _, err := c.callCaptured(ctx, method, params)
	return result, err
}

func (c *SolanaRPCClient) callCaptured(ctx context.Context, method string, params []any) (json.RawMessage, []byte, error) {
	c.gate.Lock()
	if !c.last.IsZero() {
		if wait := c.interval - c.now().Sub(c.last); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				c.gate.Unlock()
				return nil, nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	c.last = c.now()
	c.gate.Unlock()
	request := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      uint64 `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}{"2.0", c.request.Add(1), method, params}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	bounded, err := io.ReadAll(io.LimitReader(resp.Body, maxSolanaResponseBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(bounded) > maxSolanaResponseBytes {
		return nil, nil, errors.New("Solana RPC response exceeds 1 MiB")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, &SolanaProviderError{StatusCode: resp.StatusCode, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), c.now())}
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bounded, &envelope); err != nil {
		return nil, nil, err
	}
	if envelope.Error != nil {
		return nil, nil, fmt.Errorf("Solana RPC error %d: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil, nil, errors.New("Solana RPC returned no result")
	}
	return envelope.Result, append([]byte(nil), bounded...), nil
}

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func decodeBase58(raw string) ([]byte, bool) {
	if raw == "" {
		return nil, false
	}
	n := new(big.Int)
	base := big.NewInt(58)
	for _, r := range raw {
		i := strings.IndexRune(base58Alphabet, r)
		if i < 0 {
			return nil, false
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(int64(i)))
	}
	decoded := n.Bytes()
	zeros := 0
	for zeros < len(raw) && raw[zeros] == '1' {
		zeros++
	}
	return append(bytes.Repeat([]byte{0}, zeros), decoded...), true
}
func validSolanaPubkey(raw string) bool    { b, ok := decodeBase58(raw); return ok && len(b) == 32 }
func validSolanaSignature(raw string) bool { b, ok := decodeBase58(raw); return ok && len(b) == 64 }
func cloneSolanaPage(page CapturedSolanaPage) CapturedSolanaPage {
	clone := page
	clone.Source.ResultBody = append([]byte(nil), page.Source.ResultBody...)
	clone.Source.RawResponseBody = append([]byte(nil), page.Source.RawResponseBody...)
	clone.Signatures = append([]SolanaSignature(nil), page.Signatures...)
	clone.Transactions = make([]CapturedSolanaTransaction, len(page.Transactions))
	for i, tx := range page.Transactions {
		clone.Transactions[i] = cloneCapturedSolanaTx(tx)
	}
	return clone
}
func cloneCapturedSolanaTx(tx CapturedSolanaTransaction) CapturedSolanaTransaction {
	clone := tx
	clone.Source = tx.Source
	clone.Source.ResultBody = append([]byte(nil), tx.Source.ResultBody...)
	clone.Source.RawResponseBody = append([]byte(nil), tx.Source.RawResponseBody...)
	clone.Occurrences = append([]Occurrence(nil), tx.Occurrences...)
	clone.Transaction.Transaction.Signatures = append([]string(nil), tx.Transaction.Transaction.Signatures...)
	clone.Transaction.Transaction.Message.AccountKeys = append([]json.RawMessage(nil), tx.Transaction.Transaction.Message.AccountKeys...)
	clone.Transaction.Meta.PreTokenBalances = append([]SolanaTokenBalance(nil), tx.Transaction.Meta.PreTokenBalances...)
	clone.Transaction.Meta.PostTokenBalances = append([]SolanaTokenBalance(nil), tx.Transaction.Meta.PostTokenBalances...)
	return clone
}
