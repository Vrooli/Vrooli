package recommend

import (
	"fmt"

	"resource-whisper/cli/internal/hwprobe"
)

// DefaultBudgetPct is the percentage of detected VRAM/RAM the
// recommender is allowed to spend. 50 leaves headroom for the OS,
// browser tabs, and any other co-tenant workloads.
const DefaultBudgetPct = 50

// Pick returns the recommended Whisper model for the host. The reason
// string is human-readable and intended for the CLI's explain output.
// Pure function: no I/O, no time, no env. Tested via table.
func Pick(caps hwprobe.HostCapabilities, budgetPct int) (Model, string, error) {
	if budgetPct <= 0 || budgetPct > 100 {
		return "", "", fmt.Errorf("recommend: budget_pct must be in (0,100], got %d", budgetPct)
	}

	// GPU path: pick by the strongest GPU's budgeted VRAM.
	bestGPU, bestBudget := bestGPUBudget(caps.GPUs, budgetPct)
	if bestGPU != "" {
		switch {
		case bestBudget >= VRAMRequirement[ModelLargeV3]:
			return ModelLargeV3, fmt.Sprintf("GPU %s, vram_budget=%s → large-v3", bestGPU, fmtGB(bestBudget)), nil
		case bestBudget >= VRAMRequirement[ModelMedium]:
			return ModelMedium, fmt.Sprintf("GPU %s, vram_budget=%s → medium", bestGPU, fmtGB(bestBudget)), nil
		case bestBudget >= VRAMRequirement[ModelSmall]:
			return ModelSmall, fmt.Sprintf("GPU %s, vram_budget=%s → small", bestGPU, fmtGB(bestBudget)), nil
		case bestBudget >= VRAMRequirement[ModelBase]:
			return ModelBase, fmt.Sprintf("GPU %s, vram_budget=%s → base", bestGPU, fmtGB(bestBudget)), nil
		}
		// VRAM budget too small for even base → fall through to CPU/tiny.
	}

	// CPU path: gated on both RAM budget AND core count.
	ramBudget := caps.TotalRAMBytes * uint64(budgetPct) / 100
	tiers := []Model{ModelMedium, ModelSmall, ModelBase}
	for _, m := range tiers {
		if ramBudget >= CPURAMRequirement[m] && caps.CPUCores >= CPUCoreRequirement[m] {
			return m, fmt.Sprintf("no GPU, ram_budget=%s, cpu_cores=%d → %s",
				fmtGB(ramBudget), caps.CPUCores, m), nil
		}
	}
	return ModelTiny, fmt.Sprintf("no GPU, ram_budget=%s, cpu_cores=%d → tiny (fallback)",
		fmtGB(ramBudget), caps.CPUCores), nil
}

func bestGPUBudget(gpus []hwprobe.GPU, budgetPct int) (string, uint64) {
	var name string
	var best uint64
	for _, g := range gpus {
		budget := g.VRAMBytes * uint64(budgetPct) / 100
		if budget > best {
			best = budget
			name = fmt.Sprintf("%s (%s)", g.Name, fmtGB(g.VRAMBytes))
		}
	}
	return name, best
}

func fmtGB(b uint64) string {
	gb := float64(b) / float64(1<<30)
	return fmt.Sprintf("%.1f GB", gb)
}
