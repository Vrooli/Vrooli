// Package assessment maps business-health's neutral findings into the
// shared maturity envelope: it owns the loaded maturity spec and the
// conversion from intent.Finding to the assessment the shared
// ScenarioValidationService mount emits for test-genie.
package assessment

import (
	"fmt"
	"strings"

	intent "intent-go"

	maturity "github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Builder converts engine reports into shared maturity assessments using
// the scenario's frozen finding vocabulary (.vrooli/maturity.json).
type Builder struct {
	spec *maturity.Spec
}

// NewBuilder validates and wraps the loaded spec.
func NewBuilder(spec *maturity.Spec) (*Builder, error) {
	if spec == nil {
		return nil, fmt.Errorf("maturity spec is required")
	}
	if err := maturity.ValidateSpec(*spec); err != nil {
		return nil, fmt.Errorf("maturity spec invalid: %w", err)
	}
	return &Builder{spec: spec}, nil
}

// Spec exposes the loaded spec (identity checks, autofix coverage).
func (b *Builder) Spec() *maturity.Spec { return b.spec }

// Build maps neutral findings into the shared MaturityAssessment.
//
// Codes may carry a `:CLAIM-ID` suffix (per-defect afid stability, matching
// the native business phase); the maturity mapping is looked up by the BASE
// code and attached explicitly so suffixed findings land on their declared
// capability ladder instead of the fallback.
func (b *Builder) Build(scenario string, findings []intent.Finding) (*commonv1.MaturityAssessment, error) {
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
		if mapping, ok := b.spec.Findings[baseCode(f.Code)]; ok {
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

// baseCode strips the `:CLAIM-ID` suffix from a finding code.
func baseCode(code string) string {
	if i := strings.IndexByte(code, ':'); i > 0 {
		return code[:i]
	}
	return code
}

// severityToken normalizes the neutral severity vocabulary ("error",
// "warning", "info", or already-tokenized values) into the shared
// SEVERITY_* tokens. The business dimension is advisory: anything above
// ERROR was already capped by the engine.
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
