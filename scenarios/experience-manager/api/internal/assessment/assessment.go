// Package assessment maps experience-manager findings into the shared maturity
// envelope once the experience Test Genie phase descriptor exists.
package assessment

import (
	"fmt"

	"experience-manager/internal/spec"

	maturity "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// DefaultSpec returns the experience phase maturity model embedded in the
// provider descriptor and used by the native/shared validation surfaces.
func DefaultSpec() *maturity.Spec {
	return &maturity.Spec{
		Provider: "experience-manager",
		Phase:    "experience",
		Version:  "1.0.0",
		Capabilities: []maturity.CapabilitySpec{
			{
				ID:          "spec_contract",
				Label:       "Spec Contract",
				Description: "Experience index, page, journey, route, reference, binding, and tier rules are parseable and internally consistent.",
				Levels: []maturity.Level{
					{ID: "L0", Name: "Contract unavailable", StatusLabel: "Unavailable", Description: "experience/ is absent or unreadable.", CapabilitySummary: "Experience intent cannot be evaluated.", NextUnlock: "Add an experience/index.json contract."},
					{ID: "L1", Name: "Contract parseable", StatusLabel: "Foundation", Description: "The contract parses, but schema or index parity findings remain.", CapabilitySummary: "Experience intent exists but is not structurally trustworthy.", NextUnlock: "Resolve schema and index parity findings."},
					{ID: "L2", Name: "References linked", StatusLabel: "Ready", Description: "Documents parse and cross-file references, PRD refs, bindings, and tier semantics are clean.", CapabilitySummary: "The experience contract is internally linked.", NextUnlock: "Ground active pages in structure evidence."},
					{ID: "L3", Name: "Contract clean", StatusLabel: "Complete", Description: "Parser-era experience findings are clean.", CapabilitySummary: "Experience intent is a trustworthy machine-checkable contract."},
				},
			},
			{
				ID:          "structure_reconciliation",
				Label:       "Structure Reconciliation",
				Description: "Machine-tier claims are reconciled against captured accessibility structure with honest skip semantics.",
				Levels: []maturity.Level{
					{ID: "L0", Name: "Capture unavailable", StatusLabel: "Unavailable", Description: "No usable BAS accessibility capture is available.", CapabilitySummary: "Machine-tier experience claims cannot be checked against UI structure.", NextUnlock: "Capture the target page accessibility tree."},
					{ID: "L1", Name: "Bindings joinable", StatusLabel: "Foundation", Description: "Bindings can be compared with captured structure, but unresolved bindings or uncheckable claims remain.", CapabilitySummary: "The spec can start grounding in the live UI.", NextUnlock: "Resolve binding and checker coverage gaps."},
					{ID: "L2", Name: "Claims checked", StatusLabel: "Ready", Description: "Tier 0-1 machine claims are evaluated against captured structure.", CapabilitySummary: "Active machine-tier claims have deterministic evidence.", NextUnlock: "Resolve failed or unproven claims."},
					{ID: "L3", Name: "Structure clean", StatusLabel: "Complete", Description: "Structure reconciliation findings are clean for active pages.", CapabilitySummary: "The built UI structure satisfies the machine-tier experience spec."},
				},
			},
			{
				ID:          "manual_evidence",
				Label:       "Manual Evidence",
				Description: "Manual-tier claims carry explicit attestations with freshness semantics.",
				Levels: []maturity.Level{
					{ID: "L0", Name: "No attestation evidence", StatusLabel: "Unavailable", Description: "Manual-tier claims have no usable attestation evidence.", CapabilitySummary: "Manual experience claims are unproven.", NextUnlock: "Record an attestation with author, rationale, and expiry."},
					{ID: "L1", Name: "Attestations present", StatusLabel: "Foundation", Description: "Manual attestations exist but expired or stale evidence remains.", CapabilitySummary: "Manual claims have reviewable evidence with freshness debt.", NextUnlock: "Refresh expired attestations."},
					{ID: "L2", Name: "Attestations fresh", StatusLabel: "Complete", Description: "Manual-tier attestation findings are clean.", CapabilitySummary: "Manual experience claims are honestly current."},
				},
			},
			{
				ID:          "perception_advisory",
				Label:       "Perception Advisory",
				Description: "Deferred perception-tier checks remain advisory until calibrated.",
				Levels: []maturity.Level{
					{ID: "L0", Name: "Perception unevaluated", StatusLabel: "Deferred", Description: "Perception-tier checks are not part of v1 gating.", CapabilitySummary: "Visual saliency and glance judgment are visible only as future seams.", NextUnlock: "Calibrate deterministic perception checks in the P2 tier."},
					{ID: "L1", Name: "Advisory perception visible", StatusLabel: "Advisory", Description: "Perception findings are reported as non-gating advisory evidence.", CapabilitySummary: "Experience-manager can show perception debt without failing suites."},
				},
			},
		},
		Findings: map[string]maturity.FindingMapping{
			spec.CodeSchemaInvalid:       requiredManual("spec_contract", "L1", spec.SeverityError, "Schema repair needs source-level contract edits."),
			spec.CodeIndexParity:         requiredManual("spec_contract", "L1", spec.SeverityError, "Index parity repair needs source-level contract edits."),
			spec.CodeRefUnresolved:       requiredManual("spec_contract", "L2", spec.SeverityError, "Choosing the correct referenced page, state, or element requires author intent."),
			spec.CodePRDRefUnmatched:     requiredManual("spec_contract", "L2", spec.SeverityError, "Choosing the correct operational target is an intent decision."),
			spec.CodeBindingOrphan:       requiredManual("spec_contract", "L2", spec.SeverityError, "Binding repair depends on the intended UI element contract."),
			spec.CodeTierViolation:       requiredManual("spec_contract", "L2", spec.SeverityError, "Tier changes alter enforcement intent and need author judgment."),
			spec.CodeRouteUnspecced:      requiredManual("spec_contract", "L2", spec.SeverityWarning, "Route coverage decisions require product intent."),
			spec.CodeStateMissing:        requiredManual("spec_contract", "L2", spec.SeverityWarning, "State coverage decisions require design intent."),
			spec.CodeBindingUnresolved:   requiredManual("structure_reconciliation", "L1", spec.SeverityError, "Resolving live binding drift depends on the intended selector or markup."),
			spec.CodeClaimFailed:         requiredManual("structure_reconciliation", "L2", spec.SeverityError, "A failed experience claim may require product, design, or implementation changes."),
			spec.CodeClaimUnverifiable:   requiredManual("structure_reconciliation", "L1", spec.SeverityWarning, "Checker coverage gaps require choosing a supported claim type or adding validator capability."),
			spec.CodeCaptureUnavailable:  requiredManual("structure_reconciliation", "L0", spec.SeverityInfo, "Capture availability is an operational condition, not an in-place source fix."),
			spec.CodeClaimUnproven:       requiredManual("structure_reconciliation", "L2", spec.SeverityWarning, "Evidence must be captured or the claim should be retiered deliberately."),
			spec.CodeAttestationExpired:  requiredManual("manual_evidence", "L1", spec.SeverityWarning, "Refreshing manual evidence requires a human attestation."),
			spec.CodeImportanceMismatch:  advisoryManual("perception_advisory", "L1", "Perception calibration is deferred to the P2 tier."),
			spec.CodeGlanceJudgeMismatch: advisoryManual("perception_advisory", "L1", "Judge-based perception is deferred and non-gating."),
		},
		Fallback: maturity.FallbackPolicy{
			CapabilityID:     "spec_contract",
			LocalLevelImpact: "L0",
			GlobalImpact:     maturity.ImpactUnknown,
			Dimension:        "ui",
			SeverityDefault:  spec.SeverityWarning,
			CleanRequirement: string(maturity.CleanRequirementAdvisory),
		},
	}
}

func requiredManual(capabilityID, level, severity, reason string) maturity.FindingMapping {
	return maturity.FindingMapping{
		CapabilityID:        capabilityID,
		LocalLevelImpact:    level,
		GlobalImpact:        maturity.ImpactCapabilityGap,
		Dimension:           "ui",
		SeverityDefault:     severity,
		CleanRequirement:    string(maturity.CleanRequirementRequired),
		RecommendedSkillIDs: []string{"experience-spec-authoring"},
		FixClass:            maturity.FixClassManual,
		FixReason:           reason,
	}
}

func advisoryManual(capabilityID, level, reason string) maturity.FindingMapping {
	return maturity.FindingMapping{
		CapabilityID:        capabilityID,
		LocalLevelImpact:    level,
		GlobalImpact:        maturity.ImpactAdvisory,
		Dimension:           "ui",
		SeverityDefault:     spec.SeverityWarning,
		CleanRequirement:    string(maturity.CleanRequirementAdvisory),
		RecommendedSkillIDs: []string{"experience-spec-authoring"},
		FixClass:            maturity.FixClassManual,
		FixReason:           reason,
	}
}

// Builder converts parser/check reports into shared maturity assessments.
type Builder struct {
	spec *maturity.Spec
}

// NewBuilder validates and wraps the loaded maturity spec.
func NewBuilder(spec *maturity.Spec) (*Builder, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	if err := maturity.ValidateSpec(*spec); err != nil {
		return nil, fmt.Errorf("maturity spec invalid: %w", err)
	}
	return &Builder{spec: spec}, nil
}

// Spec exposes the loaded spec for registry tests.
func (b *Builder) Spec() *maturity.Spec { return b.spec }

// Build maps neutral experience findings into the shared assessment.
func (b *Builder) Build(scenario string, findings []spec.Finding) (*commonv1.MaturityAssessment, error) {
	in := make([]maturity.Finding, 0, len(findings))
	for _, f := range findings {
		mf := maturity.Finding{
			Code:        f.Code,
			Severity:    severityToken(f.Severity),
			Message:     f.Message,
			Location:    firstLocation(f.Locations),
			Remediation: f.Suggestion,
			Phase:       b.spec.Phase,
		}
		if mapping, ok := b.spec.Findings[f.Code]; ok {
			mf.Maturity = mapping
			mf.HasMaturity = true
		}
		in = append(in, mf)
	}
	return maturity.BuildProtoAssessment(maturity.BuildInput{
		Scenario: scenario,
		Spec:     *b.spec,
		Findings: in,
	})
}

func severityToken(s string) string {
	switch s {
	case "error", "SEVERITY_ERROR":
		return "SEVERITY_ERROR"
	case "warning", "SEVERITY_WARNING":
		return "SEVERITY_WARNING"
	case "info", "SEVERITY_INFO":
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}

func firstLocation(locations []string) string {
	if len(locations) == 0 {
		return ""
	}
	return locations[0]
}
