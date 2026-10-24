import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import { CheckSquare, GitCompare } from "lucide-react";
import type { RunAttribution } from "../lib/runAttribution";
import type { DiffStats } from "../lib/api";

export function RunSheet({
  runId,
  files,
  attribution,
  onClose,
  onSelectAll,
  stats,
}: {
  runId: string | null;
  files: string[];
  attribution?: RunAttribution;
  onClose: () => void;
  onSelectAll: () => void;
  stats?: Record<string, DiffStats>;
}) {
  const lineTotals = files.reduce((totals, file) => ({
    additions: totals.additions + (stats?.[file]?.additions ?? 0),
    deletions: totals.deletions + (stats?.[file]?.deletions ?? 0),
  }), { additions: 0, deletions: 0 });
  return (
    <BottomSheet isOpen={Boolean(runId)} onClose={onClose} title="Agent run">
      <div data-testid="run-sheet" className="space-y-3 text-sm text-slate-300">
        <div><div className="font-mono text-slate-100">{runId}</div><div>{attribution?.owner || "Unknown owner"}</div><div className="text-xs text-slate-500">Applied {attribution?.appliedAt ? new Date(attribution.appliedAt).toLocaleString() : "time unavailable"}</div></div>
        <p className="rounded border border-amber-500/30 bg-amber-950/20 p-2 text-xs text-amber-200">Auto-applied by the sandbox. Nothing here has been reviewed.</p>
        <div className="text-xs text-slate-400">{files.length} file{files.length === 1 ? "" : "s"} in this run · <span className="text-emerald-300">+{lineTotals.additions}</span> <span className="text-red-300">-{lineTotals.deletions}</span> lines</div>
        <div className="flex gap-2">
          <BottomSheetAction icon={<CheckSquare className="h-5 w-5" />} label={`Select all ${files.length}`} description="Add these files to the selection toolbar" onClick={onSelectAll} />
          <BottomSheetAction icon={<GitCompare className="h-5 w-5" />} label="Combined diff" description="Review the combined changes" onClick={onClose} />
        </div>
        <a
          className="block text-center text-xs text-cyan-300 underline-offset-2 hover:underline"
          href={`/embedded/agent-manager/runs/${encodeURIComponent(runId || "")}`}
          target="_blank"
          rel="noreferrer"
        >
          Open run in Agent Manager
        </a>
      </div>
    </BottomSheet>
  );
}
