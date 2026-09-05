package orchestrator

import (
	"encoding/json"
	"fmt"
	"strings"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerdescriptor"
	sharedruns "test-genie/internal/shared/runs"
)

// buildRunDescriptorSnapshot freezes the effective catalog and target-specific
// applicability decisions from the already-built phase plan. It never consults
// the live registry after planning, so execution-time catalog changes cannot
// rewrite the run's historical presentation or policy.
func buildRunDescriptorSnapshot(plan *phasePlan) (sharedruns.DescriptorSnapshot, error) {
	if plan == nil || len(plan.Definitions) == 0 {
		return sharedruns.DescriptorSnapshot{}, fmt.Errorf("cannot snapshot an empty phase plan")
	}
	planned := make(map[string]struct{}, len(plan.Selected))
	for _, def := range plan.Selected {
		planned[def.Name.Key()] = struct{}{}
	}

	entries := make([]sharedruns.PhaseDescriptorSnapshot, 0, len(plan.Definitions))
	for index, def := range plan.Definitions {
		notice := plan.Applicability[def.Name.Key()]
		descriptor := notice.Descriptor
		entry := phaseDescriptorSnapshot(def, descriptor, index)
		_, entry.Applicability.Planned = planned[def.Name.Key()]
		entry.Applicability.Status = string(notice.Result.Status)
		if entry.Applicability.Status == "" {
			entry.Applicability.Status = "applies"
		}
		for _, reason := range notice.Result.Reasons {
			entry.Applicability.Reasons = append(entry.Applicability.Reasons, sharedruns.ApplicabilityReasonSnapshot{
				Code: reason.Code, Message: reason.Message,
			})
		}
		entries = append(entries, entry)
	}
	return sharedruns.NewDescriptorSnapshot(entries)
}

func phaseDescriptorSnapshot(def phases.Definition, descriptor providerdescriptor.Descriptor, fallbackOrder int) sharedruns.PhaseDescriptorSnapshot {
	policy := def.Policy
	if !descriptor.Policy.Policy.IsZero() {
		policy = descriptor.Policy.Policy
	}
	order := descriptor.OrderHint
	if descriptor.Phase == "" {
		order = fallbackOrder
	}
	provider := strings.TrimSpace(descriptor.Scenario)
	if provider == "" {
		provider = strings.TrimSpace(def.ProviderScenario)
	}
	displayName := strings.TrimSpace(descriptor.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(def.DisplayName)
	}
	if displayName == "" {
		displayName = def.Name.String()
	}
	findingSource := strings.TrimSpace(descriptor.FindingSource)

	entry := sharedruns.PhaseDescriptorSnapshot{
		Phase:         def.Name.String(),
		DisplayName:   displayName,
		Description:   strings.TrimSpace(descriptor.Description),
		Provider:      provider,
		OrderHint:     order,
		PhaseClass:    firstSnapshotValue(descriptor.PhaseClass, def.PhaseClass),
		RuntimeClass:  firstSnapshotValue(descriptor.RuntimeClass, def.RuntimeClass),
		Dimensions:    cloneStrings(firstNonEmptySlice(descriptor.Dimensions, def.Dimensions)),
		FindingSource: findingSource,
		Policy: sharedruns.DescriptorPolicySnapshot{
			Selection:         string(policy.Selection),
			ProviderReadiness: string(policy.ProviderReadiness),
			ProviderLifecycle: string(policy.ProviderLifecycle),
			Freshness:         string(policy.Freshness),
			ResultGating:      string(policy.ResultGating),
			Unavailable:       string(policy.Unavailable),
		},
		DocsPath:               strings.TrimSpace(descriptor.Docs.Path),
		MaturityReference:      maturityReference(descriptor),
		ApplicabilityDefault:   strings.TrimSpace(descriptor.Applicability.Default),
		EvidenceKinds:          cloneStrings(descriptor.EvidenceKinds),
		Aliases:                cloneStrings(descriptor.Aliases),
		Supersedes:             cloneStrings(descriptor.Supersedes),
		ComparisonMode:         strings.TrimSpace(descriptor.Comparison.Mode),
		ValidationContract:     strings.TrimSpace(descriptor.Validation.Contract),
		ValidationDeliveryMode: strings.TrimSpace(descriptor.Validation.DeliveryMode),
		ValidationExecution:    descriptor.Validation.Execution,
		ValidationRunService:   strings.TrimSpace(descriptor.Validation.RunService),
		Determinism: sharedruns.DeterminismSnapshot{
			Default:      descriptor.Determinism.Default,
			Inputs:       append([]string(nil), descriptor.Determinism.Inputs...),
			Reason:       descriptor.Determinism.Reason,
			Capabilities: snapshotDeterminismCapabilities(descriptor.Determinism.Capabilities),
		},
	}
	fingerprint, err := sharedruns.PhaseComparisonFingerprint(entry)
	if err == nil {
		entry.ComparisonFingerprint = fingerprint
	}
	return entry
}

func snapshotDeterminismCapabilities(values map[string]providerdescriptor.DeterminismOverride) map[string]sharedruns.DeterminismOverride {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]sharedruns.DeterminismOverride, len(values))
	for name, value := range values {
		out[name] = sharedruns.DeterminismOverride{Mode: value.Mode, Inputs: append([]string(nil), value.Inputs...), Reason: value.Reason}
	}
	return out
}

func maturityReference(descriptor providerdescriptor.Descriptor) string {
	if len(descriptor.Maturity) == 0 {
		return ""
	}
	var header struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(descriptor.Maturity, &header); err != nil {
		return ""
	}
	if strings.TrimSpace(header.Version) == "" {
		return ""
	}
	return descriptor.Scenario + ":" + descriptor.Phase + "@" + strings.TrimSpace(header.Version)
}

func firstSnapshotValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptySlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}
