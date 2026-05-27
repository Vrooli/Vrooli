// BaselineDetailView (Plan B §4.2) — static manifest detail.
//
// Renders what a stored baseline actually pinned: per-surface pointer (kind +
// ref + captured-at), the git state at capture, and any skipped surfaces.
// Shown above the live comparison in the compare view.

import { formatRelativeTime } from "../../components/ScenarioReviewPanelShared";
import { BASELINE_SURFACES, SURFACE_META, surfacePresence } from "./model";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function BaselineDetailView({ baseline }: { baseline: BaselineManifest }) {
  const git = baseline.git;
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2 text-xs">
      <div className="flex flex-wrap gap-x-3 gap-y-1 text-slate-500">
        <span>Captured {formatRelativeTime(baseline.createdAt)}</span>
        {baseline.createdBy && <span>by {baseline.createdBy}</span>}
        {git?.sha && <span className="font-mono">sha={git.sha.slice(0, 8)}</span>}
        {baseline.branch && <span className="font-mono">branch={baseline.branch}</span>}
        {git?.dirty && <span className="text-amber-500">dirty</span>}
      </div>
      <div className="space-y-1">
        {BASELINE_SURFACES.map((id) => {
          const presence = surfacePresence(baseline, id);
          const pointer = baseline.surfaces[id];
          return (
            <div key={id} className="flex items-baseline justify-between gap-2">
              <span className="text-slate-400">{SURFACE_META[id].label}</span>
              {presence === "captured" && pointer ? (
                <span className="font-mono text-slate-300 truncate" title={pointer.ref}>
                  {pointer.kind}:{pointer.ref}
                </span>
              ) : presence === "skipped" ? (
                <span className="text-amber-400 truncate" title={baseline.skipped[id]}>
                  skipped — {baseline.skipped[id]}
                </span>
              ) : (
                <span className="text-slate-600">not captured</span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
