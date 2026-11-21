package assessment

import (
	"fmt"
	"strings"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// RequireIdentity validates the shared maturity assessment and asserts that its
// provider and phase match the expected catalog identity. It is the single
// identity check shared by Test Genie's consumer seam, the provider-contract
// CLI, and the in-process provider_contract helper (which previously each
// hand-rolled the same provider/phase comparison).
func RequireIdentity(provider, phase string, a *commonv1.MaturityAssessment) error {
	if err := ValidateAssessment(a); err != nil {
		return err
	}
	provider = strings.TrimSpace(provider)
	phase = strings.TrimSpace(phase)
	if got := strings.TrimSpace(a.GetProvider()); provider != "" && got != provider {
		return fmt.Errorf("assessment.provider=%q, want %q", got, provider)
	}
	if got := strings.TrimSpace(a.GetPhase()); phase != "" && got != phase {
		return fmt.Errorf("assessment.phase=%q, want %q", got, phase)
	}
	return nil
}
