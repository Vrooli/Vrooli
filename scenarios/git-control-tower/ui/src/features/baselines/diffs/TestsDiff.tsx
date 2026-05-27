// TestsDiff (Plan B §4.3) — test regression detail for one baseline.
//
// The substrate classifies the whole "tests" surface (unit/integration/smoke
// slice) into regressions / new-failures / preexisting / cleared. New failures
// are reported distinctly from regressions so a failure added by the current
// change reads differently from one the change broke.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function TestsDiff({
  diff,
  onOpenTests,
}: {
  diff: SurfaceDiff;
  onOpenTests?: () => void;
}) {
  return (
    <DiffSection surfaceId="tests" diff={diff}>
      <SurfaceDiffBody diff={diff} cleanLabel="All tests match the baseline." />
      {onOpenTests && (
        <button
          type="button"
          onClick={onOpenTests}
          className="text-xs text-blue-400 hover:text-blue-300"
        >
          Open Tests tab for run logs
        </button>
      )}
    </DiffSection>
  );
}
