// SurfaceComparePanel (Plan C Phase 2 / Decision 3) — the single per-tab
// compare UI, generalized from the old BaselineSurfaceSection (which was
// hard-typed to "tests" | "rules"). It owns one useCompareOnDemand handle and
// renders the shared SurfaceBaselineBar above an on-demand diff body for ANY
// baseline surface. Tabs drop this in above their own live content.
//
// The diff body reuses the surface-agnostic DiffSection + SurfaceDiffBody, so
// every surface renders its regressions/new-failures/preexisting/cleared lists
// identically. Surface-specific extras (videos, side-by-side visuals) live in
// the tab's own content, not here.

import type { ReactNode } from "react";
import { Loader2 } from "lucide-react";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useCompareOnDemand } from "../../lib/hooks-baselines";
import { surfaceLabel, type BaselineSurface } from "./model";
import { SurfaceBaselineBar } from "./SurfaceBaselineBar";
import { DiffSection, SurfaceDiffBody } from "./parts";

export function SurfaceComparePanel({
  scenario,
  surface,
  repoId,
  onOpenBaselines,
  onCaptureBaseline,
  viewingLabel,
}: {
  scenario: string;
  surface: BaselineSurface;
  repoId?: string | null;
  onOpenBaselines: () => void;
  onCaptureBaseline?: () => void;
  viewingLabel?: ReactNode;
}) {
  const compare = useCompareOnDemand(scenario, { surface, repoId });
  const label = surfaceLabel(surface);
  const surfaceDiff = compare.diff?.surfaces.find((s) => s.surfaceId === surface);

  return (
    <div className="space-y-2">
      <SurfaceBaselineBar
        scenario={scenario}
        repoId={repoId}
        compare={compare}
        onOpenBaselines={onOpenBaselines}
        onCaptureBaseline={onCaptureBaseline}
        viewingLabel={viewingLabel}
      />

      {compare.comparing &&
        (compare.isRunning ? (
          <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-2 text-xs text-slate-500">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Running {label.toLowerCase()} — this can take a few minutes.
          </div>
        ) : compare.error ? (
          <MutationErrorBanner error={compare.error} />
        ) : surfaceDiff ? (
          <DiffSection surfaceId={surface} diff={surfaceDiff}>
            <SurfaceDiffBody diff={surfaceDiff} cleanLabel={`${label} match the baseline.`} />
          </DiffSection>
        ) : (
          <p className="rounded-lg border border-dashed border-slate-800 px-3 py-2 text-xs text-slate-500">
            This baseline did not capture {label.toLowerCase()}.
          </p>
        ))}
    </div>
  );
}
