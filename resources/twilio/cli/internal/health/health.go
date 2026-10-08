// Package health owns provider-safe Twilio probe logic that should not live in
// CLI wiring.
package health

import (
	"context"
	"fmt"
	"net/http"
)

// Probe verifies credentials against the read-only Accounts endpoint. It does
// not send messages or mutate provider state.
func Probe(ctx context.Context, client *http.Client, endpoint, accountSID, authToken string) (int, error) {
	if endpoint == "" || accountSID == "" || authToken == "" {
		return 0, fmt.Errorf("TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("create Twilio probe request: %w", err)
	}
	req.SetBasicAuth(accountSID, authToken)
	response, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call Twilio Accounts endpoint: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return response.StatusCode, fmt.Errorf("Twilio credentials rejected (HTTP %d)", response.StatusCode)
	}
	if response.StatusCode != http.StatusOK {
		return response.StatusCode, fmt.Errorf("Twilio Accounts endpoint returned HTTP %d", response.StatusCode)
	}
	return response.StatusCode, nil
}
