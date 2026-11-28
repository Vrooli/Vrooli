package hygiene

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tidinessprovider"
	"github.com/vrooli/vrooli/internal/tuning"
)

const tidinessProviderID = "tidiness-manager"

var tidinessProviderBudget = tuning.TidinessProviderBudget()

type tidinessProvider struct {
	root   string
	client tidinessprovider.Client
}

// NewTidinessProvider adapts the owner client to Hygiene's provider registry.
// It lets callers compose the real status and finding contract around a
// controlled client, while keeping interpretation in the Hygiene owner.
func NewTidinessProvider(client tidinessprovider.Client) Provider {
	return tidinessProvider{client: client}
}

func (p tidinessProvider) ID() string { return tidinessProviderID }

func (p tidinessProvider) Budget() time.Duration { return tidinessProviderBudget }

func (p tidinessProvider) Run(ctx context.Context, req Request, report *Report) error {
	if p.client == nil {
		p.reportUnavailable(report, req.RequireTidinessProvider, "client is not configured")
		return nil
	}
	result, err := p.client.Validate(ctx, p.root)
	if err != nil {
		p.reportUnavailable(report, req.RequireTidinessProvider, err.Error())
		return nil
	}
	passed := strings.EqualFold(result.Status, "VALIDATION_STATUS_PASSED")
	statusSeverity := severityForTidinessStatus(result.Status)
	if !passed && req.RequireTidinessProvider {
		statusSeverity = SeverityError
	}
	report.addCheck(tidinessProviderID, passed, statusSeverity, result.Status)
	hasBlockingFinding := false
	for _, finding := range result.Findings {
		if strings.EqualFold(finding.Severity, "error") || strings.EqualFold(finding.Severity, "critical") || strings.EqualFold(finding.Code, "TIDINESS_BUDGET_EXCEEDED") {
			hasBlockingFinding = true
			break
		}
	}
	if !passed && statusSeverity == SeverityError && !hasBlockingFinding {
		report.addFinding(Finding{
			Severity:   SeverityError,
			Code:       "tidiness_validation_failed",
			Message:    fmt.Sprintf("tidiness provider returned status %s without blocking findings", result.Status),
			Why:        "A failed provider response must remain visible to the control-plane gate.",
			Fixability: FixabilityGuided,
		})
	} else if !passed && statusSeverity == SeverityWarning && len(result.Findings) == 0 {
		report.addFinding(Finding{
			Severity:   SeverityWarning,
			Code:       "tidiness_validation_incomplete",
			Message:    fmt.Sprintf("tidiness provider returned non-passed status %s without findings", result.Status),
			Why:        "A non-passed provider status must remain visible even when the owner returned no findings.",
			Fixability: FixabilityGuided,
		})
	}
	for _, finding := range result.Findings {
		severity := SeverityWarning
		if strings.EqualFold(finding.Code, "TIDINESS_BUDGET_EXCEEDED") || strings.EqualFold(finding.Severity, "error") || strings.EqualFold(finding.Severity, "critical") {
			severity = SeverityError
		} else if strings.EqualFold(finding.Severity, "info") {
			severity = SeverityInfo
		}
		report.addFinding(Finding{
			Severity:   severity,
			Code:       finding.Code,
			Locations:  nonEmptyLocations(finding.Location),
			Message:    finding.Message,
			Why:        finding.Remediation,
			Fixability: FixabilityGuided,
		})
	}
	return nil
}

func (p tidinessProvider) reportUnavailable(report *Report, required bool, reason string) {
	severity := SeverityWarning
	if required {
		severity = SeverityError
	}
	message := fmt.Sprintf("tidiness provider unavailable: %s", reason)
	action := Action{Code: "inspect_tidiness_manager", Message: "Start tidiness-manager and rerun the control-plane tidiness gate.", Command: "vrooli scenario start tidiness-manager --timeout 120", Fixability: FixabilityGuided}
	report.addCheck(tidinessProviderID, false, severity, message)
	report.addFinding(Finding{Severity: severity, Code: "tidiness_provider_unavailable", Message: message, Why: "The CI tidiness lane must not pass without a live provider; ordinary hygiene runs keep provider outages visible without blocking unrelated checks.", Fixability: FixabilityGuided, NextActions: []Action{action}})
	report.Actions = append(report.Actions, action)
}

func severityForTidinessStatus(status string) Severity {
	if strings.EqualFold(status, "VALIDATION_STATUS_FAILED") || strings.EqualFold(status, "VALIDATION_STATUS_ERROR") {
		return SeverityError
	}
	if strings.EqualFold(status, "VALIDATION_STATUS_PASSED") {
		return SeverityInfo
	}
	return SeverityWarning
}

func nonEmptyLocations(location string) []string {
	if strings.TrimSpace(location) == "" {
		return nil
	}
	return []string{location}
}
