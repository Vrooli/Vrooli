package soak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type verifiedDemandLease struct {
	Verified  bool   `json:"verified"`
	ReceiptID string `json:"receipt_id"`
	Owner     string `json:"owner"`
	Target    string `json:"target"`
}

func VerifyAudioToolsDemandLease(ctx context.Context, baseURL, receipt string, client *http.Client) error {
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return errors.New("Test Genie API endpoint is unavailable")
	}
	endpoint := base.ResolveReference(&url.URL{Path: "/api/v1/runs/demand-lease-receipts/verify", RawQuery: "target=audio-tools"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return errors.New("could not create Test Genie lease verification request")
	}
	req.Header.Set("X-Test-Genie-Lease-Receipt", strings.TrimSpace(receipt))
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Test Genie lease verifier is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Test Genie lease receipt rejected (HTTP %d)", resp.StatusCode)
	}
	var result verifiedDemandLease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return errors.New("Test Genie lease verifier returned an invalid response")
	}
	if !result.Verified || result.Owner != "test-genie" || result.Target != "audio-tools" || result.ReceiptID != strings.TrimSpace(receipt) {
		return errors.New("Test Genie did not confirm the exact active Audio Tools lease")
	}
	return nil
}
