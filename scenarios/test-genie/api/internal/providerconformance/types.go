// Package providerconformance validates scenarios that declare themselves as
// Test Genie phase providers through .vrooli/test-genie.json. It reuses the
// provider descriptor loader as the single source of descriptor/maturity/policy
// rules and the selfhealth conformance probe for live contract checks, then
// maps failures into stable maturity finding codes for the provider-conformance
// phase.
package providerconformance

import (
	"fmt"
	"strings"
)

const (
	CodeDescriptorMissing            = "PROVIDER_DESCRIPTOR_MISSING"
	CodeDescriptorInvalid            = "PROVIDER_DESCRIPTOR_INVALID"
	CodeIdentityMismatch             = "PROVIDER_IDENTITY_MISMATCH"
	CodeMaturityInvalid              = "PROVIDER_MATURITY_INVALID"
	CodeStaleMaturityFile            = "PROVIDER_STALE_MATURITY_FILE"
	CodePolicyUnsafe                 = "PROVIDER_POLICY_UNSAFE"
	CodeDocsMissing                  = "PROVIDER_DOCS_MISSING"
	CodeDocsSkeletonIncomplete       = "PROVIDER_DOCS_SKELETON_INCOMPLETE"
	CodeNorthStarMissing             = "PROVIDER_NORTH_STAR_MISSING"
	CodeLadderIncomplete             = "PROVIDER_LADDER_INCOMPLETE"
	CodeRungUngated                  = "PROVIDER_RUNG_UNGATED"
	CodeAutofixDeclarationIncomplete = "PROVIDER_AUTOFIX_DECLARATION_INCOMPLETE"
	CodeProviderUnreachable          = "PROVIDER_UNREACHABLE"
	CodeContractInvalid              = "PROVIDER_CONTRACT_INVALID"
	CodeContractIdentityMismatch     = "PROVIDER_CONTRACT_IDENTITY_MISMATCH"
	CodeMetricsMissing               = "PROVIDER_METRICS_MISSING"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type Finding struct {
	Code        string
	Severity    Severity
	Title       string
	Message     string
	Location    string
	Remediation string
}

type Report struct {
	Scenario string
	Path     string
	// Phase is the Test Genie phase the target descriptor declares, when the
	// descriptor parsed far enough to know it.
	Phase string
	// Probed reports whether the live provider contract probe ran. It is false
	// for the test-genie self target (recursion guard) and when the descriptor
	// was too broken to identify the provider contract.
	Probed bool
	// ProbeSkipReason explains a false Probed value for operators.
	ProbeSkipReason string
	Findings        []Finding
	Summary         Summary
}

type Summary struct {
	Errors   int
	Warnings int
}

func (s Summary) Status() string {
	if s.Errors > 0 {
		return "failed"
	}
	return "passed"
}

func (s Summary) String() string {
	return fmt.Sprintf("errors=%d warnings=%d", s.Errors, s.Warnings)
}

func (r *Report) add(f Finding) {
	r.Findings = append(r.Findings, f)
}

func (r *Report) finish() {
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			r.Summary.Errors++
		case SeverityWarning:
			r.Summary.Warnings++
		}
	}
}

func severityToAssessment(severity Severity) string {
	switch severity {
	case SeverityError:
		return "SEVERITY_ERROR"
	case SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_WARNING"
	}
}

func normalizeScenario(value string) string {
	return strings.TrimSpace(value)
}
