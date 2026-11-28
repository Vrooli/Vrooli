import type { Run } from "../../types";
import { RunUsagePanel } from "../../features/usage/RunUsagePanel";
import { ModelCostComparison } from "../ModelCostComparison";
import { CostBreakdown, type CostTotals } from "../RunDetailParts";

interface UsageTabProps {
  run: Run;
  costTotals: CostTotals;
  eventsLoading: boolean;
}

/** Tokens come from the run's usage measure (RunUsagePanel) alone; cost events add dollars and model context. */
export function UsageTab({ run, costTotals, eventsLoading }: UsageTabProps) {
  return (
    <RunUsagePanel runId={run.id} costLoading={eventsLoading} hasCost={costTotals.events > 0}>
      {costTotals.events > 0 ? (
        <div className="space-y-3 sm:space-y-4">
          <CostBreakdown totals={costTotals} />
          <ModelCostComparison
            costTotals={costTotals}
            actualModel={costTotals.models[0] || run.resolvedConfig?.model || ""}
          />
        </div>
      ) : null}
    </RunUsagePanel>
  );
}
