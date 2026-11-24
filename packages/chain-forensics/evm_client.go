package chainforensics

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DefaultEthereumRPC = "https://ethereum-rpc.publicnode.com"
	DefaultBaseRPC     = "https://mainnet.base.org"
	DefaultArbitrumRPC = "https://arb1.arbitrum.io/rpc"
)

type CapturedEVMTransaction struct {
	Chain           Chain
	TransactionHash string
	BlockHash       string
	BlockNumber     uint64
	BlockTimestamp  string
	Logs            []EVMLog
	Occurrences     []Occurrence
	Sources         []Source
	RetrievedAt     time.Time
}

var sharedEVMTransactionCache = NewWeightedCaptureCache(256, defaultCaptureCacheMaxBytes, cloneCapturedEVM, evmTransactionCacheWeight)

func evmTransactionCacheWeight(captured CapturedEVMTransaction) int64 {
	weight := int64(len(captured.Logs)+len(captured.Occurrences)) * 512
	for _, source := range captured.Sources {
		weight += int64(len(source.ResultBody) + len(source.RawResponseBody))
	}
	return weight
}

// EVMRPCClient reads one transaction receipt and its containing block from a
// caller-configured public JSON-RPC endpoint. It never submits a transaction.
// Cache entries expire quickly because a recent receipt can be reorganized.
type EVMRPCClient struct {
	endpoint  string
	http      *http.Client
	now       func() time.Time
	interval  time.Duration
	cacheTTL  time.Duration
	cache     *CaptureCache[CapturedEVMTransaction]
	requestID atomic.Uint64
}

func NewEVMRPCClient(endpoint string, client *http.Client, now func() time.Time) (*EVMRPCClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname()))) {
		return nil, errors.New("EVM RPC endpoint must be HTTPS (HTTP is allowed only on loopback) and have no credentials, query, or fragment")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("EVM RPC redirects are disabled") }}
	}
	if now == nil {
		now = time.Now
	}
	return &EVMRPCClient{endpoint: strings.TrimRight(endpoint, "/"), http: client, now: now, interval: 250 * time.Millisecond, cacheTTL: 30 * time.Second, cache: sharedEVMTransactionCache}, nil
}

func (c *EVMRPCClient) FetchERC20Transfers(ctx context.Context, chain Chain, txHash string) (CapturedEVMTransaction, error) {
	if chain != ChainEthereum && chain != ChainBase && chain != ChainArbitrum {
		return CapturedEVMTransaction{}, fmt.Errorf("unsupported EVM chain %q", chain)
	}
	if !evmHashPattern.MatchString(txHash) {
		return CapturedEVMTransaction{}, errors.New("transaction hash must be a 32-byte 0x-prefixed hash")
	}
	txHash = strings.ToLower(txHash)
	key := c.endpoint + "|" + string(chain) + "|" + txHash
	if cached, ok := c.cache.Get(key, c.now()); ok {
		return cached, nil
	}
	receiptBody, receiptResponse, err := c.callCaptured(ctx, "eth_getTransactionReceipt", []any{txHash})
	if err != nil {
		return CapturedEVMTransaction{}, err
	}
	var receipt struct {
		TransactionHash string   `json:"transactionHash"`
		BlockHash       string   `json:"blockHash"`
		BlockNumber     string   `json:"blockNumber"`
		Logs            []EVMLog `json:"logs"`
	}
	if err = json.Unmarshal(receiptBody, &receipt); err != nil {
		return CapturedEVMTransaction{}, fmt.Errorf("decode EVM receipt: %w", err)
	}
	if !strings.EqualFold(receipt.TransactionHash, txHash) || !evmHashPattern.MatchString(receipt.BlockHash) {
		return CapturedEVMTransaction{}, errors.New("receipt transaction or block hash does not match the requested transaction")
	}
	blockNum, err := parseEVMIndex(receipt.BlockNumber)
	if err != nil || blockNum == 0 {
		return CapturedEVMTransaction{}, errors.New("receipt is missing a confirmed block number")
	}
	blockBody, blockResponse, err := c.callCaptured(ctx, "eth_getBlockByHash", []any{receipt.BlockHash, false})
	if err != nil {
		return CapturedEVMTransaction{}, err
	}
	var block struct {
		Hash      string `json:"hash"`
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	}
	if err = json.Unmarshal(blockBody, &block); err != nil {
		return CapturedEVMTransaction{}, fmt.Errorf("decode EVM block: %w", err)
	}
	if !strings.EqualFold(block.Hash, receipt.BlockHash) {
		return CapturedEVMTransaction{}, errors.New("block hash changed while capturing transaction; retry after chain settles")
	}
	parsedBlock, err := parseEVMIndex(block.Number)
	if err != nil || parsedBlock != blockNum {
		return CapturedEVMTransaction{}, errors.New("receipt and containing block number disagree")
	}
	seconds, err := parseEVMIndex(block.Timestamp)
	if err != nil || seconds == 0 {
		return CapturedEVMTransaction{}, errors.New("block is missing a valid timestamp")
	}
	timestamp := time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
	retrieved := c.now().UTC()
	receiptSource := sourceForBody("evm-receipt-"+string(chain)+"-"+txHash, "Public EVM JSON-RPC receipt", c.endpoint, "eth_getTransactionReceipt", retrieved, receiptBody, "Keyless read-only JSON-RPC; shared public endpoint; receipt is reorg-sensitive and briefly cached. SHA-256 covers the JSON-RPC result object; exact HTTP response bytes are separately hashed and retained.", receiptResponse)
	blockSource := sourceForBody("evm-block-"+string(chain)+"-"+strings.ToLower(receipt.BlockHash), "Public EVM JSON-RPC block", c.endpoint, "eth_getBlockByHash", retrieved, blockBody, "Keyless read-only JSON-RPC; block hash was checked against the receipt; recent-chain reorganizations remain possible. Exact result and HTTP response bytes are retained with separate hashes.", blockResponse)
	txBody, txResponse, txErr := c.callCaptured(ctx, "eth_getTransactionByHash", []any{txHash})
	if txErr != nil && !strings.Contains(txErr.Error(), "eth_getTransactionByHash returned no result") {
		return CapturedEVMTransaction{}, txErr
	}
	var tx struct {
		Hash        string `json:"hash"`
		BlockNumber string `json:"blockNumber"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
	}
	var txSource *Source
	if txErr == nil {
		if err = json.Unmarshal(txBody, &tx); err != nil || !strings.EqualFold(tx.Hash, txHash) {
			return CapturedEVMTransaction{}, errors.New("transaction hash does not match the requested EVM transaction")
		}
		txBlock, err := parseEVMIndex(tx.BlockNumber)
		if err != nil || txBlock != blockNum {
			return CapturedEVMTransaction{}, errors.New("transaction and receipt block numbers disagree")
		}
		source := sourceForBody("evm-transaction-"+string(chain)+"-"+txHash, "Public EVM JSON-RPC transaction", c.endpoint, "eth_getTransactionByHash", retrieved, txBody, "Keyless read-only JSON-RPC; transaction fields are checked against the confirmed receipt block. Exact result and HTTP response bytes are retained with separate hashes.", txResponse)
		txSource = &source
	}
	occ := make([]Occurrence, 0, len(receipt.Logs)+1)
	for _, log := range receipt.Logs {
		if len(log.Topics) == 0 || !strings.EqualFold(log.Topics[0], ERC20TransferTopic) {
			continue
		}
		normalized, e := NormalizeERC20TransferLog(chain, receiptSource, log, timestamp)
		if e != nil {
			return CapturedEVMTransaction{}, e
		}
		occ = append(occ, normalized)
	}
	if txSource != nil {
		if value, ok := decimalHex(tx.Value); ok && value != "0" && evmAddressPattern.MatchString(tx.From) && evmAddressPattern.MatchString(tx.To) {
			occ = append(occ, Occurrence{ID: string(chain) + ":" + txHash + ":native", Chain: chain, Transaction: txHash, Block: blockNum, Timestamp: timestamp, EventIndex: 0, Direction: "transfer", From: strings.ToLower(tx.From), To: strings.ToLower(tx.To), Asset: "ETH", Atomic: value, Scale: 18, Kind: EvidenceObservation, SourceID: txSource.ID})
		}
	}
	sources := []Source{receiptSource, blockSource}
	if txSource != nil {
		sources = append(sources, *txSource)
	}
	result := CapturedEVMTransaction{Chain: chain, TransactionHash: txHash, BlockHash: strings.ToLower(receipt.BlockHash), BlockNumber: blockNum, BlockTimestamp: timestamp, Logs: receipt.Logs, Occurrences: occ, Sources: sources, RetrievedAt: retrieved}
	now := c.now()
	c.cache.Put(key, result, now.Add(c.cacheTTL), now)
	return result, nil
}

func (c *EVMRPCClient) call(ctx context.Context, method string, params []any) ([]byte, error) {
	result, _, err := c.callCaptured(ctx, method, params)
	return result, err
}

func (c *EVMRPCClient) callCaptured(ctx context.Context, method string, params []any) ([]byte, []byte, error) {
	id := c.requestID.Add(1)
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vrooli-Chain-Forensics/0.1")
	release, err := sharedEndpointRateLimiter.Acquire(ctx, c.endpoint, c.interval)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("EVM RPC %s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if retryAfter > 0 {
			sharedEndpointRateLimiter.DeferUntil(c.endpoint, c.now().Add(retryAfter))
		}
		return nil, nil, &ProviderError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, nil, err
	}
	if len(raw) >= 2<<20 {
		return nil, nil, ErrResponseTooLarge
	}
	var envelope struct {
		ID     uint64          `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, nil, fmt.Errorf("decode EVM RPC envelope: %w", err)
	}
	if envelope.ID != id {
		return nil, nil, errors.New("EVM RPC response id does not match request")
	}
	if envelope.Error != nil {
		return nil, nil, fmt.Errorf("EVM RPC %s error %d: %s", method, envelope.Error.Code, envelope.Error.Message)
	}
	if len(envelope.Result) == 0 || bytes.Equal(envelope.Result, []byte("null")) {
		return nil, nil, fmt.Errorf("EVM RPC %s returned no result", method)
	}
	return envelope.Result, append([]byte(nil), raw...), nil
}

func sourceForBody(id, name, endpoint, method string, at time.Time, body []byte, limitation string, rawResponse ...[]byte) Source {
	hash := sha256.Sum256(body)
	source := Source{ID: id, Name: name, URL: endpoint, Retrieved: at.Format(time.RFC3339), SHA256: hex.EncodeToString(hash[:]), Citation: method + " result object (exact result bytes hashed).", Limitations: limitation, ResultBody: append([]byte(nil), body...)}
	if len(rawResponse) > 0 && len(rawResponse[0]) > 0 {
		responseHash := sha256.Sum256(rawResponse[0])
		source.RawResponseSHA256 = hex.EncodeToString(responseHash[:])
		source.RawResponseBody = append([]byte(nil), rawResponse[0]...)
	}
	return source
}

func cloneCapturedEVM(in CapturedEVMTransaction) CapturedEVMTransaction {
	in.Logs = append([]EVMLog(nil), in.Logs...)
	in.Occurrences = append([]Occurrence(nil), in.Occurrences...)
	in.Sources = append([]Source(nil), in.Sources...)
	for i := range in.Sources {
		in.Sources[i].ResultBody = append([]byte(nil), in.Sources[i].ResultBody...)
	}
	for i := range in.Sources {
		in.Sources[i].RawResponseBody = append([]byte(nil), in.Sources[i].RawResponseBody...)
	}
	return in
}
