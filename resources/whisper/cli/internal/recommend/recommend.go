package recommend

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

// DefaultBudgetPct is the percentage of detected VRAM/RAM the
// recommender is allowed to spend. 50 leaves headroom for the OS,
// browser tabs, and any other co-tenant workloads.
const DefaultBudgetPct = 50

// Pick returns the recommended Whisper model for the host. The reason
// string is human-readable and intended for the CLI's explain output.
// Pure function: no I/O, no time, no env. Tested via table.
func Pick(caps hostinventory.Snapshot, budgetPct int) (Model, string, error) {
	if budgetPct <= 0 || budgetPct > 100 {
		return "", "", fmt.Errorf("recommend: budget_pct must be in (0,100], got %d", budgetPct)
	}

	// GPU path: pick by the strongest GPU's budgeted *available* VRAM.
	// Available = total − currently-used (contention-aware): sizing off total
	// is blind to co-tenant model servers already resident on a shared GPU and
	// over-picks (plan §3). Used-aware sizing means a freshly-(re)started
	// whisper claims only what the host can actually spare right now. The
	// capacity broker handles priority-aware reclaim of *lower*-priority idle
	// claims; this initial pick is the cooperative, never-OOM floor.
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
	ramBudget := caps.Memory.TotalBytes * uint64(budgetPct) / 100
	tiers := []Model{ModelMedium, ModelSmall, ModelBase}
	for _, m := range tiers {
		if ramBudget >= CPURAMRequirement[m] && caps.CPU.Cores >= CPUCoreRequirement[m] {
			return m, fmt.Sprintf("no GPU, ram_budget=%s, cpu_cores=%d → %s",
				fmtGB(ramBudget), caps.CPU.Cores, m), nil
		}
	}
	return ModelTiny, fmt.Sprintf("no GPU, ram_budget=%s, cpu_cores=%d → tiny (fallback)",
		fmtGB(ramBudget), caps.CPU.Cores), nil
}

// bestGPUBudget returns the strongest GPU's budgeted *available* VRAM. The
// budget is a percentage of what is currently free (total − used), not of the
// card's total capacity, so the pick respects whatever co-tenant workloads are
// already resident (contention-aware, plan §3). VRAMUsedBytes==0 (no sensing /
// idle card) degrades cleanly to the historical total-based behavior.
func bestGPUBudget(gpus []hostinventory.GPU, budgetPct int) (string, uint64) {
	var name string
	var best uint64
	for _, g := range gpus {
		var available uint64
		if g.VRAMUsedBytes < g.VRAMBytes {
			available = g.VRAMBytes - g.VRAMUsedBytes
		}
		// used >= total (fully consumed or sensor over-report) leaves available
		// at its zero value: nothing to spare.
		budget := available * uint64(budgetPct) / 100
		if budget > best {
			best = budget
			name = fmt.Sprintf("%s (%s total, %s free)", g.Name, fmtGB(g.VRAMBytes), fmtGB(available))
		}
	}
	return name, best
}

func fmtGB(b uint64) string {
	gb := float64(b) / float64(1<<30)
	return fmt.Sprintf("%.1f GB", gb)
}
