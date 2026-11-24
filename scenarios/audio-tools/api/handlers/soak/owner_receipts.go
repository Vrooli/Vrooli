//go:build audio_lab

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
	base, err := validatedLeaseVerifierURL(baseURL)
	if err != nil {
		return err
	}
	endpoint := base.ResolveReference(&url.URL{Path: "/api/v1/runs/demand-lease-receipts/verify", RawQuery: "target=audio-tools"})
	req, err := newLeaseVerificationRequest(ctx, endpoint, receipt)
	if err != nil {
		return err
	}
	resp, err := sendLeaseVerificationRequest(req, client)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return validateLeaseVerificationResponse(resp, receipt)
}

func validatedLeaseVerifierURL(baseURL string) (*url.URL, error) {
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || base == nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, errors.New("Test Genie API endpoint is unavailable")
	}
	return base, nil
}

func newLeaseVerificationRequest(ctx context.Context, endpoint *url.URL, receipt string) (*http.Request, error) {
	// #nosec G704 -- baseURL is operator-configured; scheme/authority are validated and the target path is fixed.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return nil, errors.New("could not create Test Genie lease verification request")
	}
	req.Header.Set("X-Test-Genie-Lease-Receipt", strings.TrimSpace(receipt))
	return req, nil
}

// sendLeaseVerificationRequest transfers response-body ownership to its caller.
// VerifyAudioToolsDemandLease closes the body immediately after this returns.
func sendLeaseVerificationRequest(req *http.Request, client *http.Client) (*http.Response, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	// Lease receipts are credential-like bearer tokens. Do not allow a remote
	// verifier to redirect this request and forward the custom receipt header
	// to another host. Treat redirects as an unsuccessful verification.
	noRedirectClient := *client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	// #nosec G704 -- the endpoint was operator-configured and validated; the path is fixed and redirects cannot forward the receipt.
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Test Genie lease verifier is unavailable: %w", err)
	}
	return resp, nil
}

func validateLeaseVerificationResponse(resp *http.Response, receipt string) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Test Genie lease receipt rejected (HTTP %d)", resp.StatusCode)
	}
	var result verifiedDemandLease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&result); err != nil {
		return errors.New("Test Genie lease verifier returned an invalid response")
	}
	if !leaseResponseMatches(result, receipt) {
		return errors.New("Test Genie did not confirm the exact active Audio Tools lease")
	}
	return nil
}

func leaseResponseMatches(result verifiedDemandLease, receipt string) bool {
	return result.Verified && result.Owner == "test-genie" && result.Target == "audio-tools" && result.ReceiptID == strings.TrimSpace(receipt)
}
