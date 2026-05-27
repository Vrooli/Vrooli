// BaselineSurfaceSection (Plan B §4.5/§4.6) — the "vs. active baseline" block
// embedded in the Tests and Rules tabs. Those tabs can READ a baseline diff but
// not create baselines (Decision 1); creation stays in the Baselines tab.
//
// The default baseline is a per-device UI convenience (Decision 4). Running a
// diff re-executes the surface server-side, so it is an explicit action.

import { useState } from "react";
import { Anchor, GitCompare, Loader2 } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useDefaultBaseline, useBaselineDiff } from "../../lib/hooks-baselines";
import { surfaceLabel } from "./model";
import { SurfaceDiffBody, DiffSection } from "./parts";

export function BaselineSurfaceSection({
  scenario,
  surface,
  repoId,
  onOpenBaselines,
}: {
  scenario: string;
  surface: "tests" | "rules";
  repoId?: string | null;
  onOpenBaselines: () => void;
}) {
  const { defaultBaselineName } = useDefaultBaseline(scenario);
  const [started, setStarted] = useState(false);
  const diffQuery = useBaselineDiff(scenario, defaultBaselineName ?? "", "", {
    surface,
    enabled: started && Boolean(defaultBaselineName),
    repoId,
  });

  if (!defaultBaselineName) {
    return (
      <div className="rounded-lg border border-dashed border-slate-800 p-3 flex items-center justify-between gap-2">
        <p className="text-xs text-slate-500">
          No baseline set. Mark a default baseline to see {surfaceLabel(surface).toLowerCase()}{" "}
          regressions here.
        </p>
        <Button variant="outline" size="sm" onClick={onOpenBaselines} className="h-7 px-3 gap-1.5 shrink-0">
          <Anchor className="h-3.5 w-3.5" />
          Baselines
        </Button>
      </div>
    );
  }

  const surfaceDiff = diffQuery.data?.surfaces.find((s) => s.surfaceId === surface);

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-slate-400">
          vs. baseline <span className="text-slate-200 font-medium">{defaultBaselineName}</span>
        </span>
        {!started && (
          <Button variant="outline" size="sm" onClick={() => setStarted(true)} className="h-7 px-3 gap-1.5">
            <GitCompare className="h-3.5 w-3.5" />
            Compare
          </Button>
        )}
      </div>

      {started &&
        (diffQuery.isLoading || diffQuery.isFetching ? (
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Running {surfaceLabel(surface).toLowerCase()} — this can take a few minutes.
          </div>
        ) : diffQuery.error ? (
          <MutationErrorBanner error={diffQuery.error} />
        ) : surfaceDiff ? (
          <DiffSection surfaceId={surface} diff={surfaceDiff}>
            <SurfaceDiffBody diff={surfaceDiff} cleanLabel={`${surfaceLabel(surface)} match the baseline.`} />
          </DiffSection>
        ) : (
          <p className="text-xs text-slate-500">This baseline did not capture {surfaceLabel(surface).toLowerCase()}.</p>
        ))}
    </div>
  );
}
