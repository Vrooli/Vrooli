import { formatRelativeTime } from "../../components/ScenarioReviewPanelShared";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function BaselineDetailView({ baseline }: { baseline: BaselineManifest }) {
  const git = baseline.git;
  const run = baseline.run;
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2 text-xs">
      <div className="flex flex-wrap gap-x-3 gap-y-1 text-slate-500">
        <span>Captured {formatRelativeTime(baseline.createdAt)}</span>
        {baseline.createdBy && <span>by {baseline.createdBy}</span>}
        {git?.sha && <span className="font-mono">sha={git.sha.slice(0, 8)}</span>}
        {baseline.branch && <span className="font-mono">branch={baseline.branch}</span>}
        {git?.dirty && <span className="text-amber-500">dirty</span>}
      </div>
      {run ? (
        <dl className="grid gap-1 sm:grid-cols-[9rem_1fr]">
          <dt className="text-slate-500">Test Genie run</dt>
          <dd className="font-mono text-slate-300 break-all">{run.runId}</dd>
          <dt className="text-slate-500">Tree digest</dt>
          <dd className="font-mono text-slate-300 break-all">{run.treeDigest || "legacy unavailable"}</dd>
          <dt className="text-slate-500">Phase-set digest</dt>
          <dd className="font-mono text-slate-300 break-all">{run.phaseSetDigest || "legacy unavailable"}</dd>
          <dt className="text-slate-500">Descriptor snapshot</dt>
          <dd className="font-mono text-slate-300 break-all">{run.descriptorSnapshotDigest || "legacy unavailable"}</dd>
        </dl>
      ) : (
        <p className="text-amber-400">Run anchor unavailable; recapture this baseline.</p>
      )}
      {baseline.migration?.degradedReasons.map((reason) => (
        <p key={reason} className="text-amber-400">⚠ {reason}</p>
      ))}
    </div>
  );
}
