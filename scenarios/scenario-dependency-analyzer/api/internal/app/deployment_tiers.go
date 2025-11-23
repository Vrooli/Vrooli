package app

import (
	"sort"

	types "scenario-dependency-analyzer/internal/types"
)

// tierAccumulator holds running totals for a single deployment tier
type tierAccumulator struct {
	RAM        float64
	Disk       float64
	CPU        float64
	FitnessSum float64
	Count      int
	Blockers   map[string]struct{}
}

// requirementNumbers holds numeric requirement values for easier arithmetic
type requirementNumbers struct {
	RAMMB    float64
	DiskMB   float64
	CPUCores float64
}

func (r requirementNumbers) hasValues() bool {
	return r.RAMMB != 0 || r.DiskMB != 0 || r.CPUCores != 0
}

// computeTierAggregates walks the dependency tree and computes aggregate
// requirements, fitness scores, and blocking dependencies for each deployment tier.
func computeTierAggregates(nodes []types.DeploymentDependencyNode) map[string]types.DeploymentTierAggregate {
	accum := map[string]*tierAccumulator{}
	var walk func(types.DeploymentDependencyNode)
	walk = func(node types.DeploymentDependencyNode) {
		for tier, support := range node.TierSupport {
			acc := accum[tier]
			if acc == nil {
				acc = &tierAccumulator{Blockers: map[string]struct{}{}}
				accum[tier] = acc
			}
			acc.Count++
			req := selectRequirements(node.Requirements, support.Requirements)
			acc.RAM += req.RAMMB
			acc.Disk += req.DiskMB
			acc.CPU += req.CPUCores
			if support.FitnessScore != nil {
				acc.FitnessSum += *support.FitnessScore
			}
			if (support.Supported != nil && !*support.Supported) || (support.FitnessScore != nil && *support.FitnessScore < tierBlockerThreshold) {
				acc.Blockers[node.Name] = struct{}{}
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}

	result := make(map[string]types.DeploymentTierAggregate, len(accum))
	for tier, acc := range accum {
		if acc == nil || acc.Count == 0 {
			continue
		}
		aggregate := types.DeploymentTierAggregate{
			DependencyCount: acc.Count,
			EstimatedRequirements: types.AggregatedRequirements{
				RAMMB:    acc.RAM,
				DiskMB:   acc.Disk,
				CPUCores: acc.CPU,
			},
		}
		aggregate.FitnessScore = acc.FitnessSum / float64(acc.Count)
		aggregate.BlockingDependencies = mapKeys(acc.Blockers)
		sort.Strings(aggregate.BlockingDependencies)
		result[tier] = aggregate
	}
	return result
}

// selectRequirements chooses between base and override requirements,
// preferring override if it has values.
func selectRequirements(base, override *types.DeploymentRequirements) requirementNumbers {
	if override != nil {
		if numbers := requirementsToNumbers(override); numbers.hasValues() {
			return numbers
		}
	}
	return requirementsToNumbers(base)
}

// requirementsToNumbers converts DeploymentRequirements pointers to concrete numbers
func requirementsToNumbers(req *types.DeploymentRequirements) requirementNumbers {
	var numbers requirementNumbers
	if req == nil {
		return numbers
	}
	if req.RAMMB != nil {
		numbers.RAMMB = *req.RAMMB
	}
	if req.DiskMB != nil {
		numbers.DiskMB = *req.DiskMB
	}
	if req.CPUCores != nil {
		numbers.CPUCores = *req.CPUCores
	}
	return numbers
}
