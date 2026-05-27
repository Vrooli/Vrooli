// BaselineRow (Plan B §4.2) — one baseline summary in the list.
//
// Surfaces the git context captured (sha/branch/dirty), per-surface presence
// chips, and the row actions. "Default" is a UI-only marker (Decision 4) the
// other tabs read to know which baseline to diff against.

import { GitCompare, Pencil, Star, Trash2, Loader2 } from "lucide-react";
import { formatRelativeTime } from "../../components/ScenarioReviewPanelShared";
import { SurfaceStatusChips } from "./parts";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

interface BaselineRowProps {
  baseline: BaselineManifest;
  isDefault: boolean;
  isDeleting: boolean;
  onCompare: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onSetDefault: () => void;
}

export function BaselineRow({
  baseline,
  isDefault,
  isDeleting,
  onCompare,
  onEdit,
  onDelete,
  onSetDefault,
}: BaselineRowProps) {
  const git = baseline.git;
  const sha8 = git?.sha ? git.sha.slice(0, 8) : "";

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 space-y-1">
          <div className="flex items-center gap-1.5">
            <button
              type="button"
              onClick={onSetDefault}
              title={isDefault ? "Default baseline" : "Set as default"}
              aria-label={isDefault ? "Default baseline" : "Set as default"}
              className={isDefault ? "text-amber-400" : "text-slate-600 hover:text-slate-400"}
            >
              <Star className={`h-3.5 w-3.5 ${isDefault ? "fill-amber-400" : ""}`} />
            </button>
            <span className="text-sm font-medium text-slate-200 truncate">{baseline.name}</span>
          </div>
          <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
            <span>{formatRelativeTime(baseline.createdAt)}</span>
            {sha8 && <span className="font-mono">sha={sha8}</span>}
            {baseline.branch && <span className="font-mono">branch={baseline.branch}</span>}
          </div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <RowAction icon={<GitCompare className="h-3.5 w-3.5" />} label="Compare" onClick={onCompare} />
          <RowAction icon={<Pencil className="h-3.5 w-3.5" />} label="Edit" onClick={onEdit} />
          <RowAction
            icon={isDeleting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
            label="Delete"
            onClick={onDelete}
            disabled={isDeleting}
            danger
          />
        </div>
      </div>

      {git?.dirty && (
        <p className="text-[11px] text-amber-500">
          Captured against dirty tree{git.dirtySummary ? ` (${git.dirtySummary})` : ""}
        </p>
      )}

      <SurfaceStatusChips manifest={baseline} />
    </div>
  );
}

function RowAction({
  icon,
  label,
  onClick,
  disabled,
  danger,
}: {
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={label}
      aria-label={label}
      className={`h-7 w-7 inline-flex items-center justify-center rounded transition-colors disabled:opacity-50 ${
        danger
          ? "text-slate-400 hover:text-red-300 hover:bg-red-950/40"
          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
      }`}
    >
      {icon}
    </button>
  );
}
