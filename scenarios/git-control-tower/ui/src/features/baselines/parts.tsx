// ============================================================================
// Baselines feature — shared presentational primitives (Plan B §4.3)
// ============================================================================
//
// VerdictBadge / EntityList / DiffSection are reused by all five surface-diff
// components and the compare/detail views. SurfaceStatusChips summarizes which
// surfaces a stored baseline pinned.

import type { ReactNode } from "react";
import { Check, Minus, Slash } from "lucide-react";
import { Badge } from "../../components/ui/badge";
import {
  BASELINE_SURFACES,
  SURFACE_META,
  surfaceLabel,
  surfacePresence,
  verdictMeta,
  type SurfacePresence,
} from "./model";
import type {
  BaselineManifest,
  SurfaceDiff,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function VerdictBadge({ verdict }: { verdict: string }) {
  const meta = verdictMeta(verdict);
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}

// EntityList renders a titled, counted set of entity names (tests, workflows,
// rule IDs…). Hidden entirely when empty so diffs stay terse.
export function EntityList({
  title,
  items,
  tone = "default",
}: {
  title: string;
  items: string[];
  tone?: "regression" | "new" | "preexisting" | "cleared" | "changed" | "default";
}) {
  if (!items || items.length === 0) return null;
  const toneClass = {
    regression: "text-red-300",
    new: "text-amber-300",
    preexisting: "text-slate-400",
    cleared: "text-emerald-300",
    changed: "text-blue-300",
    default: "text-slate-300",
  }[tone];
  return (
    <div className="space-y-1">
      <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">
        {title} ({items.length})
      </p>
      <ul className="space-y-0.5">
        {items.map((item) => (
          <li key={item} className={`text-xs font-mono break-all ${toneClass}`}>
            {item}
          </li>
        ))}
      </ul>
    </div>
  );
}

// DiffSection is the uniform frame for one surface's diff: header (label +
// verdict + summary) over a surface-specific body.
export function DiffSection({
  surfaceId,
  diff,
  children,
}: {
  surfaceId: string;
  diff?: SurfaceDiff;
  children?: ReactNode;
}) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium text-slate-200">{surfaceLabel(surfaceId)}</span>
        {diff ? <VerdictBadge verdict={diff.verdict} /> : <Badge variant="default">Not captured</Badge>}
      </div>
      {diff?.summary && <p className="text-xs text-slate-500">{diff.summary}</p>}
      {children}
    </div>
  );
}

// SurfaceDiffBody renders the four verdict-grouped entity lists shared by
// every surface diff. Surface components wrap this in a DiffSection and add
// surface-specific extras (videos, side-by-side visuals, deep links).
export function SurfaceDiffBody({
  diff,
  cleanLabel = "No differences from baseline.",
}: {
  diff: SurfaceDiff;
  cleanLabel?: string;
}) {
  const empty =
    diff.regressions.length === 0 &&
    diff.newFailures.length === 0 &&
    diff.preexisting.length === 0 &&
    diff.cleared.length === 0 &&
    (diff.changed?.length ?? 0) === 0;
  if (empty) {
    return <p className="text-xs text-slate-500">{cleanLabel}</p>;
  }
  return (
    <div className="space-y-2">
      <EntityList title="Regressions" items={diff.regressions} tone="regression" />
      <EntityList title="New failures" items={diff.newFailures} tone="new" />
      <EntityList title="Preexisting" items={diff.preexisting} tone="preexisting" />
      <EntityList title="Cleared" items={diff.cleared} tone="cleared" />
      <EntityList title="Changed — review before/after" items={diff.changed} tone="changed" />
    </div>
  );
}

// SurfaceStatusChips shows, for a stored baseline, which of the five surfaces
// were captured (check), skipped (slash + reason on hover), or absent (dash).
export function SurfaceStatusChips({ manifest }: { manifest: BaselineManifest }) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {BASELINE_SURFACES.map((id) => {
        const presence = surfacePresence(manifest, id);
        return (
          <SurfaceChip
            key={id}
            label={SURFACE_META[id].label}
            presence={presence}
            reason={presence === "skipped" ? manifest.skipped[id] : undefined}
          />
        );
      })}
    </div>
  );
}

function SurfaceChip({
  label,
  presence,
  reason,
}: {
  label: string;
  presence: SurfacePresence;
  reason?: string;
}) {
  const icon =
    presence === "captured" ? (
      <Check className="h-3 w-3 text-emerald-400" />
    ) : presence === "skipped" ? (
      <Slash className="h-3 w-3 text-amber-400" />
    ) : (
      <Minus className="h-3 w-3 text-slate-600" />
    );
  const textClass =
    presence === "captured"
      ? "text-slate-300"
      : presence === "skipped"
        ? "text-amber-400"
        : "text-slate-600";
  return (
    <span
      className="inline-flex items-center gap-1 rounded border border-slate-800 px-1.5 py-0.5 text-[11px]"
      title={presence === "skipped" && reason ? `${label} skipped: ${reason}` : label}
    >
      {icon}
      <span className={textClass}>{label}</span>
    </span>
  );
}
