// Package health owns AdGuard Home control-plane probes that go beyond
// declarative container liveness.
package health

import (
	"context"
	"fmt"
	"github.com/vrooli/vrooli/internal/tuning"
	"net/http"
	"strings"

	"github.com/vrooli/vrooli/resources/adguard-home/cli/internal/adguard"
)

const (
	StatusHealthy       = "healthy"
	StatusDegraded      = "degraded"
	StatusSetupRequired = "setup_required"
	StatusAuthFailed    = "auth_failed"
	StatusUnreachable   = "unreachable"
)

type Credentials struct {
	Username string
	Password string
}

type Report struct {
	Status             string   `json:"status"`
	BaseURL            string   `json:"base_url"`
	Authenticated      bool     `json:"authenticated"`
	SetupRequired      bool     `json:"setup_required"`
	ProtectionEnabled  *bool    `json:"protection_enabled,omitempty"`
	FilteringKnown     bool     `json:"filtering_known"`
	QueryLogEnabled    *bool    `json:"query_log_enabled,omitempty"`
	Version            string   `json:"version,omitempty"`
	Upstreams          []string `json:"upstreams,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
	Checks             []string `json:"checks"`
	PrivacyPosture     string   `json:"privacy_posture"`
	ResourceConfigured bool     `json:"resource_configured"`
}

func Probe(ctx context.Context, httpClient adguard.HTTPClient, baseURL string, creds Credentials) (Report, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return Report{}, fmt.Errorf("base URL is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: tuning.ControlPlaneClientTimeout(),
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	report := Report{
		BaseURL:            baseURL,
		ResourceConfigured: true,
		PrivacyPosture:     "unknown",
	}

	client, err := adguard.NewClient(baseURL, adguard.Credentials(creds), adguard.WithHTTPClient(httpClient))
	if err != nil {
		return Report{}, err
	}

	status, code, err := client.Status(ctx)
	if err != nil {
		report.Status = StatusUnreachable
		report.Warnings = append(report.Warnings, err.Error())
		report.Checks = append(report.Checks, "AdGuard Home control API was not reachable.")
		return report, nil
	}
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		report.Status = StatusAuthFailed
		report.Warnings = append(report.Warnings, "AdGuard Home rejected the configured credentials.")
		report.Checks = append(report.Checks, "Control API responded but authentication failed.")
		return report, nil
	case http.StatusNotFound, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		report.Status = StatusSetupRequired
		report.SetupRequired = true
		report.Warnings = append(report.Warnings, "AdGuard Home setup appears incomplete or the control API is not mounted yet.")
		report.Checks = append(report.Checks, "Admin endpoint responded, but control status is not available.")
		return report, nil
	}
	if code < 200 || code >= 300 {
		report.Status = StatusDegraded
		report.Warnings = append(report.Warnings, fmt.Sprintf("AdGuard Home control status returned HTTP %d.", code))
		report.Checks = append(report.Checks, "Control API returned a non-success status.")
		return report, nil
	}

	report.Authenticated = creds.Username != "" || creds.Password != ""
	report.Version = strings.TrimSpace(status.Version)
	if status.ProtectionStatus != nil {
		report.ProtectionEnabled = status.ProtectionStatus
	} else if status.Protection != nil {
		report.ProtectionEnabled = status.Protection
	} else if status.Running != nil {
		report.ProtectionEnabled = status.Running
	}
	report.FilteringKnown = report.ProtectionEnabled != nil
	report.Checks = append(report.Checks, "Control status endpoint returned successfully.")

	if dnsInfo, dnsCode, dnsErr := client.DNSInfo(ctx); dnsErr == nil && dnsCode >= 200 && dnsCode < 300 {
		report.Upstreams = cleanStrings(dnsInfo.UpstreamDNS)
		report.Checks = append(report.Checks, "DNS info endpoint returned successfully.")
	} else if dnsErr != nil {
		report.Warnings = append(report.Warnings, "DNS info unavailable: "+dnsErr.Error())
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("DNS info returned HTTP %d.", dnsCode))
	}

	if queryLog, queryEndpoint, queryCode, queryErr := client.QueryLogConfig(ctx); queryErr == nil && queryCode >= 200 && queryCode < 300 {
		report.QueryLogEnabled = queryLog.Enabled
		if queryLog.Enabled != nil && !*queryLog.Enabled {
			report.PrivacyPosture = "minimal"
			report.Checks = append(report.Checks, fmt.Sprintf("Query log is disabled according to %s.", queryEndpoint))
		} else if queryLog.Enabled != nil && *queryLog.Enabled {
			report.PrivacyPosture = "query_log_enabled"
			report.Warnings = append(report.Warnings, "Query log is enabled; Network Manager should avoid query-level data surfaces.")
		}
	} else if queryErr != nil {
		report.Warnings = append(report.Warnings, "Query log posture unavailable: "+queryErr.Error())
	} else {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Query log posture returned HTTP %d.", queryCode))
	}

	report.Status = classify(report)
	return report, nil
}

func classify(report Report) string {
	if report.ProtectionEnabled == nil {
		return StatusDegraded
	}
	if !*report.ProtectionEnabled {
		return StatusDegraded
	}
	if report.QueryLogEnabled != nil && *report.QueryLogEnabled {
		return StatusDegraded
	}
	return StatusHealthy
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
