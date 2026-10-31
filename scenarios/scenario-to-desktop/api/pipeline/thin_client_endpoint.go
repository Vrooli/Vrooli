package pipeline

import (
	"fmt"
	"net/url"
	"strings"
)

// validateThinClientProxyURL admits only an HTTP(S) scenario endpoint for a
// thin desktop client. A desktop bundle must never point directly at Vault:
// Vault's default listener is 8200 and its public API lives below /v1/.
//
// This is an admission check, not a reachability check. The probe endpoint
// remains responsible for testing a valid scenario URL before generation.
func validateThinClientProxyURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return fmt.Errorf("thin-client deployment requires an absolute HTTP(S) scenario proxy URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("thin-client deployment requires an HTTP(S) scenario proxy URL")
	}
	if parsed.Port() == "8200" || strings.HasPrefix(strings.ToLower(parsed.EscapedPath()), "/v1/") {
		return fmt.Errorf("thin-client deployment cannot connect directly to a Vault endpoint; provide the scenario's Vrooli proxy URL instead")
	}
	return nil
}
