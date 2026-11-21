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

type addressPageCacheEntry struct {
	page    CapturedBitcoinAddressPage
	expires time.Time
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
	c.mu.Lock()
	if cached, ok := c.cachedAddressPageLocked(endpoint); ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()

	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.mu.Lock()
	if cached, ok := c.cachedAddressPageLocked(endpoint); ok {
		c.mu.Unlock()
		return cached, nil
	}
	c.mu.Unlock()
	if !c.lastRequest.IsZero() {
		remaining := time.Until(c.lastRequest.Add(c.minInterval))
		if remaining > 0 {
			timer := time.NewTimer(remaining)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return CapturedBitcoinAddressPage{}, ctx.Err()
			case <-timer.C:
			}
		}
	}
	c.lastRequest = time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("create mempool address-history request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Vrooli-Chain-Forensics/0.1 (+https://github.com/Vrooli/Vrooli)")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CapturedBitcoinAddressPage{}, fmt.Errorf("fetch public Bitcoin address history: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CapturedBitcoinAddressPage{}, &ProviderError{StatusCode: resp.StatusCode, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), c.now())}
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
	c.mu.Lock()
	c.evictAddressPagesLocked()
	for len(c.addressPageCache) >= mempoolAddressCacheLimit {
		oldest := c.addressPageOrder[0]
		c.addressPageOrder = c.addressPageOrder[1:]
		delete(c.addressPageCache, oldest)
	}
	c.addressPageCache[endpoint] = addressPageCacheEntry{page: cloneAddressPage(page), expires: c.now().Add(mempoolCacheTTL)}
	c.addressPageOrder = append(c.addressPageOrder, endpoint)
	c.mu.Unlock()
	return cloneAddressPage(page), nil
}

func (c *MempoolClient) cachedAddressPageLocked(endpoint string) (CapturedBitcoinAddressPage, bool) {
	entry, ok := c.addressPageCache[endpoint]
	if !ok {
		return CapturedBitcoinAddressPage{}, false
	}
	if !c.now().Before(entry.expires) {
		delete(c.addressPageCache, endpoint)
		return CapturedBitcoinAddressPage{}, false
	}
	return cloneAddressPage(entry.page), true
}

func (c *MempoolClient) evictAddressPagesLocked() {
	kept := c.addressPageOrder[:0]
	for _, key := range c.addressPageOrder {
		entry, ok := c.addressPageCache[key]
		if !ok || !c.now().Before(entry.expires) {
			delete(c.addressPageCache, key)
			continue
		}
		kept = append(kept, key)
	}
	c.addressPageOrder = kept
}

func cloneAddressPage(page CapturedBitcoinAddressPage) CapturedBitcoinAddressPage {
	page.Source.ResultBody = append([]byte(nil), page.Source.ResultBody...)
	page.Transactions = append([]MempoolTransaction(nil), page.Transactions...)
	for i := range page.Transactions {
		page.Transactions[i] = cloneMempoolTransaction(page.Transactions[i])
	}
	return page
}

func validBitcoinTxID(txid string) bool {
	if len(txid) != 64 {
		return false
	}
	_, err := hex.DecodeString(txid)
	return err == nil
}
