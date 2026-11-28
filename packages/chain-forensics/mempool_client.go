package chainforensics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultMempoolBaseURL = "https://mempool.space/api"

const (
	mempoolCacheLimit = 128
	mempoolCacheTTL   = 10 * time.Minute
)

var (
	ErrProviderRateLimited = errors.New("mempool.space rate limited the request")
	ErrResponseTooLarge    = errors.New("mempool.space response exceeded the configured limit")
)

var (
	sharedMempoolTransactionCache = NewWeightedCaptureCache(mempoolCacheLimit, defaultCaptureCacheMaxBytes, cloneCapturedTransaction, capturedTransactionCacheWeight)
	sharedMempoolAddressCache     = NewWeightedCaptureCache(mempoolAddressCacheLimit, defaultCaptureCacheMaxBytes, cloneAddressPage, addressPageCacheWeight)
)

// ProviderError preserves the retry delay from a rate-limited public endpoint.
// It does not retry automatically; callers decide whether to wait or serve
// their cached snapshot.
type ProviderError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *ProviderError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("mempool.space returned HTTP %d; retry after %s", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("mempool.space returned HTTP %d", e.StatusCode)
}

func (e *ProviderError) Is(target error) bool {
	return target == ErrProviderRateLimited && e.StatusCode == http.StatusTooManyRequests
}

type CapturedTransaction struct {
	Transaction MempoolTransaction
	Source      Source
}

type MempoolClient struct {
	baseURL          string
	httpClient       *http.Client
	now              func() time.Time
	maxBytes         int64
	minInterval      time.Duration
	cache            *CaptureCache[CapturedTransaction]
	addressPageCache *CaptureCache[CapturedBitcoinAddressPage]
	requestMu        sync.Mutex
}

// NewMempoolClient creates a read-only REST client. A non-HTTPS base URL is
// accepted only for loopback test servers. The default client has a bounded
// timeout and refuses redirects to prevent silently leaving the public source.
func NewMempoolClient(baseURL string, client *http.Client, now func() time.Time) (*MempoolClient, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultMempoolBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.RawQuery != "" || u.User != nil {
		return nil, errors.New("mempool base URL must be an absolute URL without credentials or query")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && isLoopbackHost(u.Hostname())) {
		return nil, errors.New("mempool base URL must use HTTPS outside loopback tests")
	}
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("mempool redirects are disabled")
			},
		}
	}
	if now == nil {
		now = time.Now
	}
	return &MempoolClient{
		baseURL: strings.TrimRight(baseURL, "/"), httpClient: client,
		now: now, maxBytes: 1 << 20, minInterval: 250 * time.Millisecond,
		cache:            sharedMempoolTransactionCache,
		addressPageCache: sharedMempoolAddressCache,
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// FetchTransaction performs a bounded GET for an immutable confirmed Bitcoin
// transaction. Successful confirmed responses are cached by txid in memory;
// 429 responses preserve Retry-After instead of spinning or flooding a shared
// endpoint. This client exposes no transaction submission methods.
func (c *MempoolClient) FetchTransaction(ctx context.Context, txid string) (CapturedTransaction, error) {
	if len(txid) != 64 {
		return CapturedTransaction{}, errors.New("transaction id must be 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(txid); err != nil {
		return CapturedTransaction{}, errors.New("transaction id must be 64 hexadecimal characters")
	}
	txid = strings.ToLower(txid)
	cacheKey := c.baseURL + "|tx|" + txid
	if cached, ok := c.cache.Get(cacheKey, c.now()); ok {
		return cached, nil
	}

	// Serialize uncached reads and keep a small gap between requests. Check the
	// cache again after acquiring the gate so concurrent identical calls share
	// one provider request.
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	if cached, ok := c.cache.Get(cacheKey, c.now()); ok {
		return cached, nil
	}
	endpoint := c.baseURL + "/tx/" + txid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CapturedTransaction{}, fmt.Errorf("create mempool request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vrooli-Chain-Forensics/0.1 (+https://github.com/Vrooli/Vrooli)")
	release, err := sharedEndpointRateLimiter.Acquire(ctx, endpoint, c.minInterval)
	if err != nil {
		return CapturedTransaction{}, err
	}
	defer release()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CapturedTransaction{}, fmt.Errorf("fetch public Bitcoin transaction %s: %w", txid, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if retryAfter > 0 {
			sharedEndpointRateLimiter.DeferUntil(endpoint, c.now().Add(retryAfter))
		}
		return CapturedTransaction{}, &ProviderError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if retryAfter > 0 {
			sharedEndpointRateLimiter.DeferUntil(endpoint, c.now().Add(retryAfter))
		}
		return CapturedTransaction{}, &ProviderError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		return CapturedTransaction{}, fmt.Errorf("read public Bitcoin transaction %s: %w", txid, err)
	}
	if int64(len(body)) > c.maxBytes {
		return CapturedTransaction{}, ErrResponseTooLarge
	}
	var tx MempoolTransaction
	if err := json.Unmarshal(body, &tx); err != nil {
		return CapturedTransaction{}, fmt.Errorf("decode public Bitcoin transaction %s: %w", txid, err)
	}
	if strings.ToLower(tx.TxID) != txid {
		return CapturedTransaction{}, fmt.Errorf("provider returned transaction %q for requested txid %q", tx.TxID, txid)
	}
	if !tx.Status.Confirmed || tx.Status.BlockHeight == 0 || tx.Status.BlockTime <= 0 {
		return CapturedTransaction{}, errors.New("transaction is not confirmed; unconfirmed records are not cached")
	}
	hash := sha256.Sum256(body)
	captured := CapturedTransaction{
		Transaction: tx,
		Source: Source{
			ID: "mempool-tx-" + txid, Name: "mempool.space REST",
			URL: endpoint, Retrieved: c.now().UTC().Format(time.RFC3339), SHA256: hex.EncodeToString(hash[:]),
			Limitations: "Keyless public read; shared endpoint with no SLA. Exact bounded HTTP response body is retained for independent normalization replay.", ResultBody: append([]byte(nil), body...),
		},
	}
	c.cache.Put(cacheKey, captured, c.now().Add(mempoolCacheTTL), c.now())
	captured = cloneCapturedTransaction(captured)
	return captured, nil
}

func cloneCapturedTransaction(captured CapturedTransaction) CapturedTransaction {
	captured.Transaction = cloneMempoolTransaction(captured.Transaction)
	captured.Source.ResultBody = append([]byte(nil), captured.Source.ResultBody...)
	return captured
}

func capturedTransactionCacheWeight(captured CapturedTransaction) int64 {
	return int64(len(captured.Source.ResultBody)+len(captured.Source.RawResponseBody)) + int64(len(captured.Transaction.Inputs)+len(captured.Transaction.Outputs))*512
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}

func cloneMempoolTransaction(tx MempoolTransaction) MempoolTransaction {
	tx.Inputs = append([]struct {
		Prevout struct {
			Address string `json:"scriptpubkey_address"`
			Value   uint64 `json:"value"`
		} `json:"prevout"`
		PreviousTx    string `json:"txid"`
		PreviousIndex uint32 `json:"vout"`
	}(nil), tx.Inputs...)
	tx.Outputs = append([]struct {
		Address string `json:"scriptpubkey_address"`
		Value   uint64 `json:"value"`
	}(nil), tx.Outputs...)
	return tx
}
