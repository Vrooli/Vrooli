// VisualsDiff (Plan B §4.3) — visual snapshot delta for one baseline.
//
// Pixel-level side-by-side comparison stays in the Screenshots tab (which owns
// the existing lightbox + api-visual.ts image paths); this view reports the
// per-page verdict roll-up and links there for the imagery.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function VisualsDiff({
  diff,
  onOpenScreenshots,
}: {
  diff: SurfaceDiff;
  onOpenScreenshots?: () => void;
}) {
  return (
    <DiffSection surfaceId="visuals" diff={diff}>
      <SurfaceDiffBody diff={diff} cleanLabel="Visuals match the baseline." />
      {onOpenScreenshots && (
        <button
          type="button"
          onClick={onOpenScreenshots}
          className="text-xs text-blue-400 hover:text-blue-300"
        >
          Open Screenshots tab for side-by-side
        </button>
      )}
    </DiffSection>
  );
}
