// WorkflowsDiff (Plan B §4.3) — workflow regression detail for one baseline.
//
// Lists regressed / new-failed / preexisting / cleared workflows. Video
// drilldown (baseline-vs-current playback) lives in the Workflows tab, which
// owns the run-level data the SurfaceDiff does not carry; this view links there
// rather than rendering dead buttons.

import { DiffSection, SurfaceDiffBody } from "../parts";
import type { SurfaceDiff } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function WorkflowsDiff({
  diff,
  onOpenWorkflows,
}: {
  diff: SurfaceDiff;
  onOpenWorkflows?: () => void;
}) {
  return (
    <DiffSection surfaceId="workflows" diff={diff}>
      <SurfaceDiffBody diff={diff} cleanLabel="All workflows match the baseline." />
      {onOpenWorkflows && (
        <button
          type="button"
          onClick={onOpenWorkflows}
          className="text-xs text-blue-400 hover:text-blue-300"
        >
          Open Workflows tab for run videos
        </button>
      )}
    </DiffSection>
  );
}
