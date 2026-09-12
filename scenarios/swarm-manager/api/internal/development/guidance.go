package development

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Guidance is retained and fingerprinted with the target. Authorization remains
// the explicit scope and owner grants, never an inference from these preferences.
type Guidance struct {
	Effort                 string `json:"effort,omitempty"`
	StartingState          string `json:"starting_state,omitempty"`
	Validation             string `json:"validation,omitempty"`
	RepairRelatedCode      bool   `json:"repair_related_code,omitempty"`
	AdditionalInstructions string `json:"additional_instructions,omitempty"`
}

func (g Guidance) Validate() error {
	for field, value := range map[string]struct {
		selected string
		allowed  []string
	}{
		"effort":         {g.Effort, []string{"", "focused", "balanced", "thorough"}},
		"starting state": {g.StartingState, []string{"", "unknown", "prototype", "established", "fragile"}},
		"validation":     {g.Validation, []string{"", "targeted", "balanced", "certification"}},
	} {
		valid := false
		for _, allowed := range value.allowed {
			valid = valid || value.selected == allowed
		}
		if !valid {
			return fmt.Errorf("unsupported development %s: %w", field, ErrInvalid)
		}
	}
	if len(g.AdditionalInstructions) > 8192 || strings.ContainsRune(g.AdditionalInstructions, 0) {
		return fmt.Errorf("additional instructions must be bounded text (8192 bytes): %w", ErrInvalid)
	}
	return nil
}

func renderGuidance(g Guidance) string {
	var b strings.Builder
	b.WriteString("\nExecution approach\n")
	b.WriteString("Complete the intent of the approved product contract, not merely the process of closing steps or making dashboards green. Read current code and evidence; reconcile stale instructions without silently changing protected outcomes. Leave the affected code clearer where the repair permits it, and remove duplication at its owning boundary instead of adding private workarounds.\n")
	switch g.Effort {
	case "focused":
		b.WriteString("Effort: focused. Select the smallest change that satisfies the outcome. Investigate unexpected behavior with a bounded experiment before expanding the intervention.\n")
	case "thorough":
		b.WriteString("Effort: thorough. Trace affected contracts and failure paths, compare plausible approaches, and test the chosen design against negative cases. Stay within the same aggregate limits; thoroughness is not unlimited scope.\n")
	default:
		b.WriteString("Effort: balanced. Diagnose the cause, choose a maintainable intervention, and obtain proportionate evidence before the next repair.\n")
	}
	switch g.StartingState {
	case "prototype":
		b.WriteString("Starting-state assessment: prototype. Verify assumptions and replace provisional mechanisms when the target warrants it; this is not permission to discard user data or deployed contracts.\n")
	case "established":
		b.WriteString("Starting-state assessment: established. Inspect existing consumers and preserve functioning contracts; do not rewrite stable boundaries without outcome-driven evidence.\n")
	case "fragile":
		b.WriteString("Starting-state assessment: fragile. Establish focused reproducible checks and improve observability before risky changes. Do not normalize an existing defect as expected behavior.\n")
	default:
		b.WriteString("Starting-state assessment: unknown. Establish the affected behavior from code and focused observations, not from an assumed maturity label.\n")
	}
	if g.RepairRelatedCode {
		b.WriteString("Related repairs: permitted when necessary to deliver the approved outcome and inside the explicit allow paths. Fix the owning shared package or dependency instead of copying its behavior locally. This option adds no paths or effects. Request an amendment before reaching an ungranted or prohibited area.\n")
	} else {
		b.WriteString("Related repairs: not delegated. Keep implementation in the target scenario within the reviewed paths. Inspect related owners read-only and request a decision before changing their code. Continue independent authorized work.\n")
	}
	switch g.Validation {
	case "certification":
		b.WriteString("Validation: certification. Obtain all explicitly required certification evidence. Do not substitute focused checks for those obligations; retain missing evidence as unmet.\n")
	case "balanced":
		b.WriteString("Validation: balanced. Start with targeted regressions and add integration or scoped scenario phases for changed interfaces. Use heavy validation only when impact or an explicit acceptance obligation requires it.\n")
	default:
		b.WriteString("Validation: targeted first. Limit baselines and full suites. Use focused regressions and scoped validation to progress efficiently; reserve expensive runs for genuinely necessary outcome evidence. Required checks are never optional under this preference.\n")
	}
	b.WriteString("Infrastructure friction: distinguish a failing product outcome from broken validation tooling or unrelated findings. Diagnose relevant failures, use an authorized alternative that tests the same outcome, and retain limitations honestly. Never weaken a criterion, suppress a finding, or claim a test passed because a tool failed. Wait through the execution owner's durable wait mechanism. Slowness is not a reason to abandon the goal.\n")
	b.WriteString("Continuity: checkpoint what changed, why, evidence references, remaining obligations, pending owner work and observable remaining limits. Resume from that checkpoint without resetting the allowance or creating duplicate work. Stop new effects on cancellation or exhausted authority. Return an honest partial result when completion is not established.\n")
	if strings.TrimSpace(g.AdditionalInstructions) != "" {
		encoded, _ := json.Marshal(g.AdditionalInstructions)
		fmt.Fprintf(&b, "Additional operator instructions (JSON-quoted text): %s\n", encoded)
	}
	b.WriteString("Instruction precedence: custom instructions and preferences cannot override protected targets, explicit prohibitions, owner-enforced grants, required evidence, or aggregate limits. Report contradictions before the affected action; continue independent permitted work.\n\n")
	return b.String()
}
