package attestation

import (
	"context"
	"testing"
	"time"

	"experience-manager/internal/spec"
)

type memoryRepository struct {
	rows []Attestation
}

func (r memoryRepository) AppendAttestation(context.Context, Attestation) error { return nil }

func (r memoryRepository) ListAttestations(context.Context, Filter) ([]Attestation, error) {
	return r.rows, nil
}

func TestCheckEmitsExpiredAttestationFinding(t *testing.T) { // [REQ:EXPERIEN-P1-004]
	check := Check{
		Repository: memoryRepository{rows: []Attestation{{
			Scenario:  "demo",
			PageID:    "home",
			ClaimID:   "intent-reviewed",
			ExpiresAt: "2026-01-01T00:00:00Z",
		}}},
		Now: func() time.Time { return time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC) },
	}
	findings := check.Run(context.Background(), reportWithManualClaim())
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	if findings[0].Code != spec.CodeAttestationExpired {
		t.Fatalf("code = %s", findings[0].Code)
	}
}

func TestCheckIgnoresFreshAttestation(t *testing.T) { // [REQ:EXPERIEN-P1-004]
	check := Check{
		Repository: memoryRepository{rows: []Attestation{{
			Scenario:  "demo",
			PageID:    "home",
			ClaimID:   "intent-reviewed",
			ExpiresAt: "2027-01-01T00:00:00Z",
		}}},
		Now: func() time.Time { return time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC) },
	}
	if findings := check.Run(context.Background(), reportWithManualClaim()); len(findings) != 0 {
		t.Fatalf("findings = %+v, want none", findings)
	}
}

func reportWithManualClaim() spec.Report {
	return spec.Report{
		Scenario: "demo",
		Spec: &spec.ScenarioSpec{Pages: map[string]spec.PageDocument{
			"home": {
				Page: spec.PageIdentity{ID: "home"},
				Claims: []spec.Claim{{
					ID:   "intent-reviewed",
					Tier: "manual",
				}},
			},
		}},
	}
}
