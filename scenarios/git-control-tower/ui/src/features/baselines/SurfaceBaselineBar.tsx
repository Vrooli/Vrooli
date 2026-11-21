// SurfaceBaselineBar (Plan C Phase 2) — the shared header shown above a surface
// tab's content once it has data. It states what you're viewing (a loose live
// capture vs. a baseline-pinned artifact), lets you switch the comparison
// baseline (BaselineSelector), run/exit an on-demand compare, capture a new
// baseline scoped to this surface, and jump to the Baselines tab.
//
// It is purely presentational over a CompareOnDemand handle owned by the
// caller, so the same compare state drives both this bar and the diff body
// rendered below it (see SurfaceComparePanel).

import type { ReactNode } from "react";
import { Anchor, GitCompare, X } from "lucide-react";
import { Button } from "../../components/ui/button";
import { BaselineSelector } from "./BaselineSelector";
import type { CompareOnDemand } from "../../lib/hooks-baselines";

export function SurfaceBaselineBar({
  scenario,
  repoId,
  compare,
  onOpenBaselines,
  onCaptureBaseline,
  viewingLabel,
}: {
  scenario: string;
  repoId?: string | null;
  compare: CompareOnDemand;
  onOpenBaselines: () => void;
  /** Opens SetBaselineModal scoped to this surface. */
  onCaptureBaseline?: () => void;
  /** What the tab is currently showing, e.g. "latest run · 2m ago". */
  viewingLabel?: ReactNode;
}) {
  const { comparing, baselineName, start, exit } = compare;

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2">
      <div className="min-w-0 text-xs text-slate-400">
        {comparing && baselineName ? (
          <span>
            Comparing vs. baseline{" "}
            <span className="font-medium text-slate-200">{baselineName}</span>
          </span>
        ) : (
          <span>Viewing: {viewingLabel ?? "live capture"}</span>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-2 shrink-0">
        <BaselineSelector scenario={scenario} repoId={repoId} onOpenBaselines={onOpenBaselines} />
        {comparing ? (
          <Button variant="outline" size="sm" onClick={exit} className="h-7 px-3 gap-1.5">
            <X className="h-3.5 w-3.5" />
            Exit compare
          </Button>
        ) : (
          <Button
            variant="outline"
            size="sm"
            onClick={start}
            disabled={!baselineName}
            className="h-7 px-3 gap-1.5"
            title={baselineName ? "Compare what's shown against the selected baseline" : "Select a baseline first"}
          >
            <GitCompare className="h-3.5 w-3.5" />
            Compare
          </Button>
        )}
        {onCaptureBaseline && (
          <Button size="sm" onClick={onCaptureBaseline} className="h-7 px-3 gap-1.5">
            <Anchor className="h-3.5 w-3.5" />
            Capture baseline
          </Button>
        )}
      </div>
    </div>
  );
}
