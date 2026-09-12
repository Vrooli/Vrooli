package attestation

import (
	"context"
	"fmt"
	"sort"
	"time"

	"experience-manager/internal/spec"
)

// Check emits freshness findings for recorded manual attestations.
type Check struct {
	Repository Repository
	Now        func() time.Time
}

func (c Check) Name() string { return "attestation.manual_freshness" }

func (c Check) Run(ctx context.Context, report spec.Report) []spec.Finding {
	if c.Repository == nil || report.Spec == nil {
		return nil
	}
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	rows, err := c.Repository.ListAttestations(ctx, Filter{Scenario: report.Scenario})
	if err != nil {
		return []spec.Finding{{
			Code:       spec.CodeClaimUnproven,
			Severity:   spec.SeverityInfo,
			Message:    fmt.Sprintf("manual attestation ledger unavailable: %v", err),
			Locations:  []string{"experience/index.json"},
			Suggestion: "Retry after the experience-manager database is reachable.",
		}}
	}
	pageByID := report.Spec.Pages
	var findings []spec.Finding
	for _, a := range rows {
		page, ok := pageByID[a.PageID]
		if !ok || !manualClaim(page, a.ClaimID) {
			continue
		}
		expiry, err := time.Parse(time.RFC3339, a.ExpiresAt)
		if err != nil {
			expiry, err = time.Parse(time.RFC3339Nano, a.ExpiresAt)
		}
		if err != nil || !expiry.After(now()) {
			findings = append(findings, spec.Finding{
				Code:       spec.CodeAttestationExpired,
				Severity:   spec.SeverityWarning,
				Message:    fmt.Sprintf("manual attestation for %s/%s is expired or invalid", a.PageID, a.ClaimID),
				Locations:  []string{"experience/pages/" + a.PageID + ".json"},
				Suggestion: "Append a fresh attestation with author, rationale, and future expiry.",
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Locations[0] == findings[j].Locations[0] {
			return findings[i].Message < findings[j].Message
		}
		return findings[i].Locations[0] < findings[j].Locations[0]
	})
	return findings
}

func manualClaim(page spec.PageDocument, claimID string) bool {
	for _, claim := range page.Claims {
		if claim.ID == claimID && claim.Tier == "manual" {
			return true
		}
	}
	return false
}
