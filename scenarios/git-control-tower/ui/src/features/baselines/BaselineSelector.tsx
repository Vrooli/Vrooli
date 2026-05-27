// BaselineSelector (Plan C Phase 2) — the shared "which baseline am I comparing
// against" control. Reads/sets the per-scenario default baseline (Decision 4)
// and lists every baseline on the current branch. When none exist it collapses
// to a single "Open Baselines" affordance so empty states stay actionable.

import { Anchor } from "lucide-react";
import { Button } from "../../components/ui/button";
import { useBaselines, useDefaultBaseline } from "../../lib/hooks-baselines";

export function BaselineSelector({
  scenario,
  repoId,
  onOpenBaselines,
}: {
  scenario: string;
  repoId?: string | null;
  onOpenBaselines: () => void;
}) {
  const baselinesQuery = useBaselines(scenario, { repoId });
  const { defaultBaselineName, setDefaultBaseline } = useDefaultBaseline(scenario);
  const baselines = baselinesQuery.data ?? [];

  if (baselinesQuery.isLoading) {
    return <span className="text-xs text-slate-600">loading baselines…</span>;
  }

  if (baselines.length === 0) {
    return (
      <Button
        variant="outline"
        size="sm"
        onClick={onOpenBaselines}
        className="h-7 px-3 gap-1.5 shrink-0"
      >
        <Anchor className="h-3.5 w-3.5" />
        Open Baselines
      </Button>
    );
  }

  // Default to the first baseline if the stored default is gone (e.g. deleted).
  const selected = baselines.some((b) => b.name === defaultBaselineName)
    ? (defaultBaselineName as string)
    : "";

  return (
    <label className="inline-flex items-center gap-1.5 text-xs text-slate-400">
      <span className="sr-only">Baseline</span>
      <Anchor className="h-3.5 w-3.5 text-slate-500" />
      <select
        aria-label="Compare against baseline"
        value={selected}
        onChange={(e) => setDefaultBaseline(e.target.value || null)}
        className="rounded-md border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-200 focus:border-blue-500 focus:outline-none"
      >
        <option value="">none</option>
        {baselines.map((b) => (
          <option key={`${b.branch}/${b.name}`} value={b.name}>
            {b.name}
          </option>
        ))}
      </select>
    </label>
  );
}
