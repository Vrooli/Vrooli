package chainforensics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	mempoolAddressPageLimit  = 25
	mempoolAddressCacheLimit = 16
	// Full Esplora transactions can be large when a confirmed history page
	// includes high-input consolidations. Keep this path bounded while allowing
	// the 1HQ3 1BBq address page (about 2.2 MiB) to be retained intact.
	mempoolAddressPageMaxBytes = 4 << 20
)

var (
	base58AddressPattern = regexp.MustCompile(`^[13][1-9A-HJ-NP-Za-km-z]{25,34}$`)
	bech32AddressPattern = regexp.MustCompile(`(?i)^bc1[023456789ac-hj-np-z]{11,71}$`)
)

type CapturedBitcoinAddressPage struct {
	Address       string
	AfterTxID     string
	NextAfterTxID string
	Transactions  []MempoolTransaction
	Source        Source
}

// FetchAddressPage fetches one bounded page of confirmed address history from
// the public mempool.space Esplora endpoint. Each page contains at most 25
// transactions; callers may explicitly request a later page with NextAfterTxID.
func (c *MempoolClient) FetchAddressPage(ctx context.Context, address, afterTxID string) (CapturedBitcoinAddressPage, error) {
	address = strings.TrimSpace(address)
	if !base58AddressPattern.MatchString(address) && !bech32AddressPattern.MatchString(address) {
		return CapturedBitcoinAddressPage{}, errors.New("address must be a valid-looking Bitcoin mainnet address")
	}
	afterTxID = strings.ToLower(strings.TrimSpace(afterTxID))
	if afterTxID != "" && !validBitcoinTxID(afterTxID) {
		return CapturedBitcoinAddressPage{}, errors.New("after transaction id must be 64 hexadecimal characters")
	}
	endpoint := c.baseURL + "/address/" + url.PathEscape(address) + "/txs/chain"
	if afterTxID != "" {
		endpoint += "/" + afterTxID
	}
	if cached, ok := c.addressPageCache.Get(endpoint, c.now()); ok {
		return cached, nil
	}

	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	if cached, ok := c.addressPageCache.Get(endpoint, c.now()); ok {
		return cached, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("create mempool address-history request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vrooli-Chain-Forensics/0.1 (+https://github.com/Vrooli/Vrooli)")
	release, err := sharedEndpointRateLimiter.Acquire(ctx, endpoint, c.minInterval)
	if err != nil {
		return CapturedBitcoinAddressPage{}, err
	}
	defer release()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("fetch public Bitcoin address history: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
		if retryAfter > 0 {
			sharedEndpointRateLimiter.DeferUntil(endpoint, c.now().Add(retryAfter))
		}
		return CapturedBitcoinAddressPage{}, &ProviderError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, mempoolAddressPageMaxBytes+1))
	if err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("read public Bitcoin address history: %w", err)
	}
	if len(body) > mempoolAddressPageMaxBytes {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("address page response exceeds configured %d-byte limit", mempoolAddressPageMaxBytes)
	}
	var transactions []MempoolTransaction
	if err := json.Unmarshal(body, &transactions); err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("decode public Bitcoin address history: %w", err)
	}
	if len(transactions) > mempoolAddressPageLimit {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("provider returned %d transactions; page limit is %d", len(transactions), mempoolAddressPageLimit)
	}
	seen := make(map[string]struct{}, len(transactions))
	for i := range transactions {
		txid := strings.ToLower(transactions[i].TxID)
		if !validBitcoinTxID(txid) {
			return CapturedBitcoinAddressPage{}, fmt.Errorf("provider returned invalid transaction id %q", transactions[i].TxID)
		}
		if _, exists := seen[txid]; exists {
			return CapturedBitcoinAddressPage{}, fmt.Errorf("provider returned duplicate transaction %s", txid)
		}
		seen[txid] = struct{}{}
		if !transactions[i].Status.Confirmed || transactions[i].Status.BlockHeight == 0 || transactions[i].Status.BlockTime <= 0 {
			return CapturedBitcoinAddressPage{}, fmt.Errorf("provider returned unconfirmed transaction %s in confirmed history", txid)
		}
		transactions[i].TxID = txid
	}
	hash := sha256.Sum256(body)
	pageDigest := sha256.Sum256([]byte(endpoint))
	page := CapturedBitcoinAddressPage{Address: address, AfterTxID: afterTxID, Transactions: transactions, Source: Source{
		ID: "mempool-address-page-" + hex.EncodeToString(pageDigest[:]) + "-" + hex.EncodeToString(hash[:]), Name: "mempool.space REST",
		URL: endpoint, Retrieved: c.now().UTC().Format(time.RFC3339), SHA256: hex.EncodeToString(hash[:]),
		Citation:    fmt.Sprintf("Confirmed Bitcoin address-history page for %s", address),
		Limitations: "Keyless public read; shared endpoint with no SLA. Page is limited to 25 confirmed transactions; exact bounded HTTP response body is retained.", ResultBody: append([]byte(nil), body...),
	}}
	if len(transactions) == mempoolAddressPageLimit {
		page.NextAfterTxID = transactions[len(transactions)-1].TxID
	}
	c.addressPageCache.Put(endpoint, page, c.now().Add(mempoolCacheTTL), c.now())
	return cloneAddressPage(page), nil
}

func cloneAddressPage(page CapturedBitcoinAddressPage) CapturedBitcoinAddressPage {
	page.Source.ResultBody = append([]byte(nil), page.Source.ResultBody...)
	page.Transactions = append([]MempoolTransaction(nil), page.Transactions...)
	for i := range page.Transactions {
		page.Transactions[i] = cloneMempoolTransaction(page.Transactions[i])
	}
	return page
}

func addressPageCacheWeight(page CapturedBitcoinAddressPage) int64 {
	weight := int64(len(page.Source.ResultBody)+len(page.Source.RawResponseBody)) + int64(len(page.Transactions))*512
	for _, tx := range page.Transactions {
		weight += int64(len(tx.Inputs)+len(tx.Outputs)) * 512
	}
	return weight
}

func validBitcoinTxID(txid string) bool {
	if len(txid) != 64 {
		return false
	}
	_, err := hex.DecodeString(txid)
	return err == nil
}
