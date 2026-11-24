package chainforensics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ExpandBitcoinAddressHistoryPage validates and expands one retained
// mempool.space confirmed address-history page. Every returned occurrence is
// tied to the page's source ID. The function records transaction observations
// only; it does not infer address ownership or the origin of an input's value.
func ExpandBitcoinAddressHistoryPage(source Source, address string) ([]Occurrence, error) {
	address = strings.TrimSpace(address)
	if source.ID == "" || source.Name == "" || source.Retrieved == "" || !sha256Pattern.MatchString(source.SHA256) {
		return nil, fmt.Errorf("Bitcoin address-history page has incomplete source provenance")
	}
	if len(source.ResultBody) == 0 {
		return nil, fmt.Errorf("Bitcoin address-history page source %s has no retained response body", source.ID)
	}
	if len(source.ResultBody) > mempoolAddressPageMaxBytes {
		return nil, fmt.Errorf("Bitcoin address-history page source %s exceeds the retained page size limit", source.ID)
	}
	digest := sha256.Sum256(source.ResultBody)
	if hex.EncodeToString(digest[:]) != strings.ToLower(source.SHA256) {
		return nil, fmt.Errorf("Bitcoin address-history page source %s body does not match its SHA-256", source.ID)
	}
	if err := validateBitcoinHistoryEndpoint(source.URL, address); err != nil {
		return nil, err
	}
	var txs []MempoolTransaction
	if err := json.Unmarshal(source.ResultBody, &txs); err != nil {
		return nil, fmt.Errorf("decode retained Bitcoin address-history page: %w", err)
	}
	if len(txs) > mempoolAddressPageLimit {
		return nil, fmt.Errorf("retained Bitcoin address-history page has %d transactions; limit is %d", len(txs), mempoolAddressPageLimit)
	}
	seen := make(map[string]struct{}, len(txs))
	var occurrences []Occurrence
	for i := range txs {
		tx := txs[i]
		tx.TxID = strings.ToLower(tx.TxID)
		if !validBitcoinTxID(tx.TxID) {
			return nil, fmt.Errorf("retained address-history page has invalid transaction id %q", txs[i].TxID)
		}
		if _, exists := seen[tx.TxID]; exists {
			return nil, fmt.Errorf("retained address-history page has duplicate transaction %s", tx.TxID)
		}
		seen[tx.TxID] = struct{}{}
		if !tx.Status.Confirmed || tx.Status.BlockHeight == 0 || tx.Status.BlockTime <= 0 {
			return nil, fmt.Errorf("retained address-history page transaction %s has no confirmed chain position", tx.TxID)
		}
		events, err := NormalizeBitcoinTransaction(source, tx)
		if err != nil {
			return nil, err
		}
		occurrences = append(occurrences, events...)
	}
	return occurrences, nil
}

func validateBitcoinHistoryEndpoint(rawURL, address string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host != "mempool.space" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("source URL is not a canonical mempool.space Bitcoin address-history endpoint")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return fmt.Errorf("source URL is not a mempool.space Bitcoin address-history page")
	}
	if parts[0] != "api" || parts[1] != "address" || parts[2] != address || parts[3] != "txs" || parts[4] != "chain" || (len(parts) == 6 && !validBitcoinTxID(parts[5])) {
		return fmt.Errorf("source URL does not identify the requested Bitcoin address-history page for %s", address)
	}
	return nil
}
