package checks

import (
	"context"
	"fmt"
	"strings"

	"experience-manager/internal/spec"
)

// OracleCoverageCheck ensures every governed regression case remains represented
// by a claim, and that artifact-backed cases retain machine-tier proof.
type OracleCoverageCheck struct{}

func (OracleCoverageCheck) Name() string { return "oracle.claim_coverage" }

func (OracleCoverageCheck) Run(_ context.Context, report spec.Report) []spec.Finding {
	if report.Spec == nil || report.DegradedReason != "" || len(report.Spec.Oracles) == 0 {
		return nil
	}
	var claims []spec.Claim
	for _, page := range report.Spec.Pages {
		claims = append(claims, page.Claims...)
	}
	var findings []spec.Finding
	for fixturePath, oracle := range report.Spec.Oracles {
		for _, oracleCase := range oracle.Cases {
			caseID := strings.ToLower(oracleCase.ID)
			var covering []spec.Claim
			for _, claim := range claims {
				if strings.Contains(strings.ToLower(claim.ID), caseID) {
					covering = append(covering, claim)
				}
			}
			if len(covering) == 0 {
				findings = append(findings, oracleFinding(fixturePath, oracleCase, "a claim id containing the oracle case id"))
				continue
			}
			needsMachine := false
			for _, evidence := range oracleCase.Evidence {
				if evidence == "screenshot_png" || evidence == "layout_json" {
					needsMachine = true
				}
			}
			if needsMachine {
				machine := false
				for _, claim := range covering {
					if claim.Tier == "machine" {
						machine = true
						break
					}
				}
				if !machine {
					findings = append(findings, oracleFinding(fixturePath, oracleCase, "a machine-tier claim for screenshot/layout evidence"))
				}
			}
		}
	}
	sortFindings(findings)
	return findings
}

func oracleFinding(path string, oracleCase spec.OracleCase, requirement string) spec.Finding {
	return spec.Finding{
		Code:       spec.CodeOracleCaseUncovered,
		Severity:   spec.SeverityError,
		Message:    fmt.Sprintf("oracle case %s has no covering %s", oracleCase.ID, requirement),
		Locations:  []string{path},
		Suggestion: fmt.Sprintf("Add a claim whose id contains %s and satisfies %s.", oracleCase.ID, requirement),
	}
}
