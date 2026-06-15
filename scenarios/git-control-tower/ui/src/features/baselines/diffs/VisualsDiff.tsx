// VisualsDiff (Plan B §4.3) — visual snapshot delta for one baseline.
//
// The visuals surface is advisory: test-genie compares the two runs' captures at
// the pixel level (internal/visualcheck) and reports neutral per-page deltas. A
// difference here is a "review before/after" signal, never a failure — a
// clearly-broken render fails earlier, at smoke time, on the test surface. This
// view renders the three states (clean / changed-review / not-comparable) and
// links to the Screenshots tab for the side-by-side imagery.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function VisualsDiff({
  diff,
  onOpenScreenshots,
}: {
  diff: SurfaceDiff;
  onOpenScreenshots?: () => void;
}) {
  const hasChanges = (diff.changed?.length ?? 0) > 0;
  return (
    <DiffSection surfaceId="visuals" diff={diff}>
      <SurfaceDiffBody
        diff={diff}
        cleanLabel="Visuals match the baseline — nothing to review."
      />
      {hasChanges && (
        <p className="text-[11px] text-slate-500">
          These are differences to look at, not failures. Confirm the new render is
          correct in the side-by-side.
        </p>
      )}
      {onOpenScreenshots && (
        <button
          type="button"
          onClick={onOpenScreenshots}
          className="text-xs text-blue-400 hover:text-blue-300"
        >
          Open Screenshots tab for side-by-side before/after
        </button>
      )}
    </DiffSection>
  );
}
