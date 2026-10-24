package capacity

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	FitVerdictFits           = "fits"
	FitVerdictOverSubscribed = "over_subscribed"
)

type FitMachine struct {
	VRAMBytes   int64    `json:"vram_bytes"`
	UsableBytes int64    `json:"usable_bytes"`
	Backends    []string `json:"backends"`
	Compute     string   `json:"compute,omitempty"`
}

type FitResourceChoice struct {
	Enabled  *bool
	Rung     string
	Tunables map[string]any
	Priority string
	GPUIndex *int
}

type FitState struct {
	Posture      string
	ReserveBytes int64
	Resources    map[string]FitResourceChoice
}

type FitOffender struct {
	Resource        string         `json:"resource"`
	Rung            string         `json:"rung"`
	Bytes           int64          `json:"bytes"`
	FootprintSource string         `json:"footprint_source"`
	Tunables        map[string]any `json:"tunables,omitempty"`
}

type FitUnrunnable struct {
	Resource string `json:"resource"`
	Reason   string `json:"reason"`
}

type FitProposal struct {
	Resource   string         `json:"resource"`
	Change     string         `json:"change"`
	From       string         `json:"from,omitempty"`
	To         string         `json:"to,omitempty"`
	Tunables   map[string]any `json:"tunables,omitempty"`
	DeltaBytes int64          `json:"delta_bytes"`
	Source     string         `json:"source"`
}

type FitVerdict struct {
	Verdict            string          `json:"verdict"`
	HostSource         string          `json:"host_source"`
	Machine            FitMachine      `json:"machine"`
	Posture            string          `json:"posture"`
	StaticBytes        int64           `json:"static_bytes"`
	ReserveBytes       int64           `json:"reserve_bytes"`
	AvailableBytes     int64           `json:"available_bytes"`
	Offenders          []FitOffender   `json:"offenders"`
	Unrunnable         []FitUnrunnable `json:"unrunnable"`
	Proposal           []FitProposal   `json:"proposal"`
	ProposedTotalBytes int64           `json:"proposed_total_bytes"`
}

type fitSelection struct {
	resource DeclaredResource
	choice   FitResourceChoice
	rung     string
	bytes    int64
	source   string
	tunables map[string]any
	priority int
	step     int
}

// Fit evaluates the complete enabled resource set against one machine. It is
// pure: callers supply the host snapshot projection, operator choices,
// declarations, and durable footprints.
func Fit(machine FitMachine, state FitState, manifests []DeclaredResource, footprints []Footprint) FitVerdict {
	if machine.UsableBytes < 0 {
		machine.UsableBytes = 0
	}
	if machine.VRAMBytes < 0 {
		machine.VRAMBytes = 0
	}
	if machine.UsableBytes == 0 && machine.VRAMBytes > 0 {
		machine.UsableBytes = machine.VRAMBytes
	}
	if state.Posture == "" {
		state.Posture = "balanced"
	}
	if state.ReserveBytes < 0 {
		state.ReserveBytes = 0
	}
	out := FitVerdict{
		Verdict: FitVerdictFits, Machine: machine, Posture: state.Posture,
		ReserveBytes: state.ReserveBytes, AvailableBytes: machine.UsableBytes,
		Offenders: []FitOffender{}, Unrunnable: []FitUnrunnable{}, Proposal: []FitProposal{},
	}

	selections := make([]fitSelection, 0, len(manifests))
	for _, resource := range manifests {
		choice := state.Resources[resource.Name]
		enabled := resource.Enabled
		if choice.Enabled != nil {
			enabled = *choice.Enabled
		}
		if !enabled || resource.Claim == nil || resource.Claim.ResourceKind != ResourceKindVRAM {
			continue
		}
		if reason := resourceUnrunnable(resource, machine); reason != "" {
			out.Unrunnable = append(out.Unrunnable, FitUnrunnable{Resource: resource.Name, Reason: reason})
			continue
		}
		tunables := effectiveTunables(resource.Tunables, choice.Tunables)
		rung, amount, step := chosenRung(resource.Claim, choice.Rung)
		amount, source := resolveFootprint(footprints, resource.Name, rung, tunablesKey(tunables), choiceGPUIndex(choice), amount)
		priority := ParsePriorityTier(resource.Claim.Priority)
		if choice.Priority != "" {
			priority = ParsePriorityTier(choice.Priority)
		}
		selections = append(selections, fitSelection{resource: resource, choice: choice, rung: rung, bytes: amount, source: source, tunables: tunables, priority: priority, step: step})
		out.StaticBytes += amount
	}

	total := out.StaticBytes + out.ReserveBytes
	out.ProposedTotalBytes = total
	if total <= machine.UsableBytes && len(out.Unrunnable) == 0 {
		return out
	}
	out.Verdict = FitVerdictOverSubscribed

	sort.SliceStable(selections, func(i, j int) bool {
		if selections[i].priority != selections[j].priority {
			return selections[i].priority < selections[j].priority
		}
		if selections[i].bytes != selections[j].bytes {
			return selections[i].bytes > selections[j].bytes
		}
		return selections[i].resource.Name < selections[j].resource.Name
	})
	for i := range selections {
		selection := &selections[i]
		if total <= machine.UsableBytes {
			break
		}
		steps := claimSteps(selection.resource.Claim)
		if selection.step >= len(steps)-1 {
			continue
		}
		oldRung, oldBytes := selection.rung, selection.bytes
		chosenIndex := len(steps) - 1
		for candidate := selection.step + 1; candidate < len(steps); candidate++ {
			candidateBytes, _ := resolveFootprint(footprints, selection.resource.Name, steps[candidate].Label, tunablesKey(selection.tunables), choiceGPUIndex(selection.choice), steps[candidate].AmountBytes)
			if total-oldBytes+candidateBytes <= machine.UsableBytes {
				chosenIndex = candidate
				break
			}
		}
		newStep := steps[chosenIndex]
		newBytes, source := resolveFootprint(footprints, selection.resource.Name, newStep.Label, tunablesKey(selection.tunables), choiceGPUIndex(selection.choice), newStep.AmountBytes)
		if newBytes >= oldBytes {
			continue
		}
		out.Offenders = append(out.Offenders, FitOffender{Resource: selection.resource.Name, Rung: oldRung, Bytes: oldBytes, FootprintSource: selection.source, Tunables: selection.tunables})
		out.Proposal = append(out.Proposal, FitProposal{Resource: selection.resource.Name, Change: "rung", From: oldRung, To: newStep.Label, Tunables: selection.tunables, DeltaBytes: newBytes - oldBytes, Source: source})
		total += newBytes - oldBytes
		selection.rung, selection.bytes, selection.source, selection.step = newStep.Label, newBytes, source, chosenIndex
	}
	out.ProposedTotalBytes = total
	return out
}

func resourceUnrunnable(resource DeclaredResource, machine FitMachine) string {
	if resource.Require != "required" {
		return ""
	}
	available := make(map[string]bool, len(machine.Backends))
	for _, backend := range machine.Backends {
		available[strings.ToLower(strings.TrimSpace(backend))] = true
	}
	for _, backend := range resource.Backends {
		backend = strings.ToLower(strings.TrimSpace(backend))
		if backend == ResourceKindCPU || !available[backend] || machine.VRAMBytes == 0 {
			continue
		}
		if backend == "cuda" && !computeAtLeast(machine.Compute, resource.CUDAMinCompute) {
			continue
		}
		return ""
	}
	return fmt.Sprintf("requires one of [%s] with compatible compute and non-zero accelerator memory", strings.Join(resource.Backends, ","))
}

func computeAtLeast(actual, minimum string) bool {
	if strings.TrimSpace(minimum) == "" {
		return true
	}
	a, errA := strconv.ParseFloat(strings.TrimSpace(actual), 64)
	m, errM := strconv.ParseFloat(strings.TrimSpace(minimum), 64)
	return errA == nil && errM == nil && a >= m
}

func chosenRung(claim *ResourceClaimSpec, requested string) (string, int64, int) {
	steps := claimSteps(claim)
	if requested != "" {
		for i, step := range steps {
			if step.Label == requested {
				return step.Label, step.AmountBytes, i
			}
		}
	}
	return steps[0].Label, steps[0].AmountBytes, 0
}

func claimSteps(claim *ResourceClaimSpec) []DegradeStep {
	if claim != nil && claim.Profile != nil && len(claim.Profile.Steps) > 0 {
		return claim.Profile.Steps
	}
	if claim == nil {
		return []DegradeStep{{Label: "none", AmountBytes: 0}}
	}
	return []DegradeStep{{Label: "preferred", AmountBytes: claim.PreferredBytes}}
}

func effectiveTunables(declared []DeclaredCapacityTunable, overrides map[string]any) map[string]any {
	out := make(map[string]any)
	for _, tunable := range declared {
		if !tunable.scalesFootprint() {
			continue
		}
		value := tunable.Default
		if override, ok := overrides[tunable.Name]; ok {
			value = override
		}
		out[tunable.Name] = value
	}
	return out
}

func tunablesKey(values map[string]any) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return strings.Join(parts, ";")
}

func choiceGPUIndex(choice FitResourceChoice) int {
	if choice.GPUIndex != nil {
		return *choice.GPUIndex
	}
	return 0
}

func resolveFootprint(rows []Footprint, resource, rung, key string, gpu int, fallback int64) (int64, string) {
	for _, row := range rows {
		if row.Resource == resource && row.Rung == rung && row.TunablesKey == key && row.GPUIndex == gpu {
			return row.PeakBytes, row.Source
		}
	}
	return fallback, FootprintSourceManifestDefault
}
