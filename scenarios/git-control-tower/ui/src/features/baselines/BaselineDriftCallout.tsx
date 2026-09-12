// BaselineDriftCallout (Plan B §4.7) — a compact Overview callout pointing at
// the active (default) baseline. It deliberately does NOT auto-run a diff
// (that re-executes surfaces server-side and costs minutes); it surfaces the
// active baseline and a one-click jump to the Baselines tab to compare there.

import { formatRelativeTime } from "../../components/ScenarioReviewPanelShared";
import { useBaselines, useDefaultBaseline } from "../../lib/hooks-baselines";
import { RunAnchorBadge } from "./parts";
import { BaselineSelector } from "./BaselineSelector";

export function BaselineDriftCallout({
  scenario,
  repoId,
  onOpenBaselines,
}: {
  scenario: string;
  repoId?: string | null;
  onOpenBaselines: () => void;
}) {
  const baselinesQuery = useBaselines(scenario, { repoId });
  const { defaultBaselineName } = useDefaultBaseline(scenario);
  const baselines = baselinesQuery.data ?? [];

  // Nothing to show until at least one baseline exists.
  if (baselinesQuery.isLoading || baselines.length === 0) return null;

  const active = defaultBaselineName
    ? baselines.find((b) => b.name === defaultBaselineName)
    : undefined;

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 flex items-center justify-between gap-3">
      <div className="min-w-0 space-y-1">
        {active ? (
          <>
            <p className="text-xs text-slate-300">
              Active baseline <span className="font-medium">{active.name}</span>
              <span className="text-slate-500"> · captured {formatRelativeTime(active.createdAt)}</span>
              {active.git?.dirty && <span className="text-amber-500"> · dirty</span>}
            </p>
            <RunAnchorBadge manifest={active} />
          </>
        ) : (
          <p className="text-xs text-slate-400">
            {baselines.length} baseline{baselines.length !== 1 ? "s" : ""} available. Mark a default to
            track drift here.
          </p>
        )}
      </div>
      <BaselineSelector scenario={scenario} repoId={repoId} onOpenBaselines={onOpenBaselines} />
    </div>
  );
}
