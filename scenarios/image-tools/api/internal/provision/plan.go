package provision

import (
	"fmt"
	"sort"
	"strings"

	"image-tools/internal/capabilities"
	"image-tools/internal/models"
)

const (
	PlanExplicit = "explicit_models"
	PlanHostFit  = "host_fit_operation_coverage"
)

// ExcludedModel explains why an enabled local candidate is absent from a
// host-fit plan. Selected candidates are separately reported by the job's
// per-model lifecycle results.
type ExcludedModel struct {
	ModelID    string   `json:"model_id"`
	Operations []string `json:"operations,omitempty"`
	Reason     string   `json:"reason"`
}

type UncoveredOperation struct {
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
}

type CandidateGap struct {
	ModelID   string `json:"model_id"`
	Operation string `json:"operation"`
	Reason    string `json:"reason"`
}

type fitCandidate struct {
	model models.Model
	ops   []string
}

// Plan records the exact scope and omissions of a default provisioning run.
// It is retained in both the server-owned job request and its final evidence.
type Plan struct {
	Mode                string               `json:"mode"`
	SelectedModelIDs    []string             `json:"selected_model_ids"`
	CoveredOperations   []string             `json:"covered_operations"`
	UncoveredOperations []UncoveredOperation `json:"uncovered_operations,omitempty"`
	ExcludedModels      []ExcludedModel      `json:"excluded_models,omitempty"`
	CandidateGaps       []CandidateGap       `json:"candidate_gaps,omitempty"`
	CoverageComplete    bool                 `json:"coverage_complete"`
}

// HostFitPlan chooses a deterministic small set of enabled, commercial-clean
// local models that covers as many registered operations as the host can run.
// It evaluates total host capacity (not transient free VRAM) because this is a
// preparation plan; job admission and smoke still enforce live capacity.
func HostFitPlan(candidates []models.Model, operations []string, host capabilities.Host, enabled models.EnabledFunc, operationUnavailable func(models.Model, string) string) Plan {
	plan := Plan{Mode: PlanHostFit}
	ops := append([]string(nil), operations...)
	sort.Strings(ops)
	remaining := make(map[string]bool, len(ops))
	for _, op := range ops {
		remaining[op] = true
	}

	var eligible []fitCandidate
	for _, m := range candidates {
		if enabled != nil && !enabled(m.ID) || enabled == nil && !m.Enabled {
			continue
		}
		reason := hostFitExclusion(m, host)
		if reason != "" {
			plan.ExcludedModels = append(plan.ExcludedModels, ExcludedModel{ModelID: m.ID, Operations: modelOperations(m, ops), Reason: reason})
			continue
		}
		covered := make([]string, 0, len(m.Operations))
		for _, op := range ops {
			if !remaining[op] {
				continue
			}
			effective, serves := modelOperation(m, op)
			if !serves {
				continue
			}
			if !effective.Offerable() {
				plan.CandidateGaps = append(plan.CandidateGaps, CandidateGap{ModelID: m.ID, Operation: op, Reason: "derived_pipeline_unproven:" + effective.Technique})
				continue
			}
			if operationUnavailable != nil {
				if reason := operationUnavailable(m, op); reason != "" {
					plan.CandidateGaps = append(plan.CandidateGaps, CandidateGap{ModelID: m.ID, Operation: op, Reason: reason})
					continue
				}
			}
			covered = append(covered, op)
		}
		if len(covered) == 0 {
			reason := "serves_no_registered_operation"
			if m.RequiresWeights() && !models.HasLoadSmokeContract(m) {
				reason = "unsupported_load_smoke_contract"
			} else {
				for _, gap := range plan.CandidateGaps {
					if gap.ModelID == m.ID {
						if strings.HasPrefix(gap.Reason, "derived_pipeline_unproven:") {
							reason = "derived_pipeline_unproven"
						} else {
							reason = "no_registered_local_backend_for_any_served_operation"
						}
						break
					}
				}
			}
			plan.ExcludedModels = append(plan.ExcludedModels, ExcludedModel{ModelID: m.ID, Operations: modelOperations(m, ops), Reason: reason})
			continue
		}
		eligible = append(eligible, fitCandidate{model: m, ops: covered})
	}

	selected := make(map[string]bool)
	for {
		best := -1
		bestCoverage := 0
		for i := range eligible {
			if selected[eligible[i].model.ID] {
				continue
			}
			coverage := 0
			for _, op := range eligible[i].ops {
				if remaining[op] {
					coverage++
				}
			}
			if coverage == 0 {
				continue
			}
			if best < 0 || betterCoverageCandidate(eligible[i], coverage, eligible[best], bestCoverage) {
				best, bestCoverage = i, coverage
			}
		}
		if best < 0 {
			break
		}
		chosen := eligible[best]
		selected[chosen.model.ID] = true
		plan.SelectedModelIDs = append(plan.SelectedModelIDs, chosen.model.ID)
		for _, op := range chosen.ops {
			if remaining[op] {
				remaining[op] = false
				plan.CoveredOperations = append(plan.CoveredOperations, op)
			}
		}
	}

	sort.Strings(plan.SelectedModelIDs)
	sort.Strings(plan.CoveredOperations)
	for _, op := range ops {
		if remaining[op] {
			plan.UncoveredOperations = append(plan.UncoveredOperations, UncoveredOperation{
				Operation: op,
				Reason:    uncoveredReason(op, plan),
			})
		}
	}
	plan.CoverageComplete = len(plan.UncoveredOperations) == 0
	for _, candidate := range eligible {
		if !selected[candidate.model.ID] {
			plan.ExcludedModels = append(plan.ExcludedModels, ExcludedModel{
				ModelID:    candidate.model.ID,
				Operations: candidate.ops,
				Reason:     "not_selected_by_minimal_coverage_plan",
			})
		}
	}
	sort.Slice(plan.ExcludedModels, func(i, j int) bool { return plan.ExcludedModels[i].ModelID < plan.ExcludedModels[j].ModelID })
	sort.Slice(plan.CandidateGaps, func(i, j int) bool {
		if plan.CandidateGaps[i].Operation != plan.CandidateGaps[j].Operation {
			return plan.CandidateGaps[i].Operation < plan.CandidateGaps[j].Operation
		}
		return plan.CandidateGaps[i].ModelID < plan.CandidateGaps[j].ModelID
	})
	return plan
}

// EnabledOperations returns the registered operation vocabulary intersected
// with the full effective-operation set of every effectively enabled model,
// including unproven architecture derivations. It deliberately includes
// candidates that are unsupported on this host or lack verified commercial
// rights: those remain visible as uncovered gaps instead of disappearing from
// the provisioning contract.
func EnabledOperations(candidates []models.Model, registered []string, enabled models.EnabledFunc) []string {
	known := make(map[string]bool, len(registered))
	for _, op := range registered {
		known[op] = true
	}
	set := make(map[string]bool)
	for _, m := range candidates {
		if enabled != nil && !enabled(m.ID) || enabled == nil && !m.Enabled {
			continue
		}
		for _, effective := range m.EffectiveOps() {
			op := effective.Op
			if known[op] {
				set[op] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for op := range set {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func uncoveredReason(operation string, plan Plan) string {
	var causes []string
	for _, candidate := range plan.CandidateGaps {
		if candidate.Operation == operation {
			causes = append(causes, fmt.Sprintf("%s:%s", candidate.ModelID, candidate.Reason))
		}
	}
	for _, candidate := range plan.ExcludedModels {
		for _, op := range candidate.Operations {
			if op == operation {
				causes = append(causes, fmt.Sprintf("%s:%s", candidate.ModelID, candidate.Reason))
				break
			}
		}
	}
	if len(causes) == 0 {
		return "no_enabled_candidate_declares_operation"
	}
	sort.Strings(causes)
	return "enabled_candidates_unavailable: " + strings.Join(causes, "; ")
}

func betterCoverageCandidate(candidate fitCandidate, coverage int, current fitCandidate, currentCoverage int) bool {
	if coverage != currentCoverage {
		return coverage > currentCoverage
	}
	if candidate.model.SizeMBApprox != current.model.SizeMBApprox {
		return candidate.model.SizeMBApprox < current.model.SizeMBApprox
	}
	candidateTier, currentTier := tierPreference(candidate.model.Tier), tierPreference(current.model.Tier)
	if candidateTier != currentTier {
		return candidateTier > currentTier
	}
	return candidate.model.ID < current.model.ID
}

func tierPreference(tier models.Tier) int {
	switch tier {
	case models.TierQuality:
		return 3
	case models.TierDefaultVariant:
		return 2
	case models.TierDefault:
		return 1
	default:
		return 0
	}
}

func hostFitExclusion(m models.Model, host capabilities.Host) string {
	if m.CapabilityLabels.CommercialUse != models.CommercialUseYes {
		return "commercial_use_not_verified"
	}
	if m.Backend == models.BackendOpenRouter {
		return "remote_model_requires_credentials"
	}
	if m.RequiresWeights() && models.NeedsLoadSmokeContract(m) && !models.HasLoadSmokeContract(m) {
		return "unsupported_load_smoke_contract"
	}
	if !m.SupportsHost(host) {
		return fmt.Sprintf("unsupported_host_%s_%s", host.OS, host.Arch)
	}
	mem := host.AvailableMemoryBytes
	if mem == 0 {
		mem = host.TotalMemoryBytes
	}
	if m.Hardware.MinRAMGB > 0 && uint64(m.Hardware.MinRAMGB) > mem/(1<<30) {
		return fmt.Sprintf("requires_%d_gib_ram_host_has_%d_gib_available", m.Hardware.MinRAMGB, mem/(1<<30))
	}
	if m.Hardware.MinVRAMGB > 0 {
		vram, known := host.MaxVRAMBytes()
		if !known || vram < uint64(m.Hardware.MinVRAMGB)*(1<<30) {
			if !m.Hardware.CPUCapable || m.Hardware.GPURequired {
				return fmt.Sprintf("requires_%d_gib_vram", m.Hardware.MinVRAMGB)
			}
		}
	}
	if m.Hardware.GPURequired && len(host.GPUs) == 0 {
		return "gpu_required_but_unavailable"
	}
	return ""
}

func modelOperations(m models.Model, operations []string) []string {
	var out []string
	for _, op := range operations {
		if _, ok := modelOperation(m, op); ok {
			out = append(out, op)
		}
	}
	return out
}

func modelOperation(m models.Model, operation string) (models.EffectiveOp, bool) {
	for _, effective := range m.EffectiveOps() {
		if effective.Op == operation {
			return effective, true
		}
	}
	return models.EffectiveOp{}, false
}
