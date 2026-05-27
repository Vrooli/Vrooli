// StructureDiff (Plan B §4.3) — file-tree + structure-scan delta for one
// baseline. Items are structure violations added / cleared since capture.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function StructureDiff({ diff }: { diff: SurfaceDiff }) {
  return (
    <DiffSection surfaceId="structure" diff={diff}>
      <SurfaceDiffBody diff={diff} cleanLabel="Structure matches the baseline." />
    </DiffSection>
  );
}
