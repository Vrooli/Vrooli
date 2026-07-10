import { Badge } from "../../components/ui/badge";
import { verdictMeta } from "./model";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import type { PhaseDiff } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";

export function VerdictBadge({ verdict }: { verdict: string }) {
  const meta = verdictMeta(verdict);
  return <Badge variant={meta.variant}>{meta.label}</Badge>;
}

export function EntityList({
  title,
  items,
  tone = "default",
}: {
  title: string;
  items: string[];
  tone?: "regression" | "new" | "preexisting" | "cleared" | "default";
}) {
  if (items.length === 0) return null;
  const toneClass = {
    regression: "text-red-300",
    new: "text-amber-300",
    preexisting: "text-slate-400",
    cleared: "text-emerald-300",
    default: "text-slate-300",
  }[tone];
  return (
    <div className="space-y-1">
      <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">{title} ({items.length})</p>
      <ul className="space-y-0.5">
        {items.map((item) => <li key={item} className={`text-xs font-mono break-all ${toneClass}`}>{item}</li>)}
      </ul>
    </div>
  );
}

export function PhaseDiffCard({ diff }: { diff: PhaseDiff }) {
  const descriptor = diff.descriptorB ?? diff.descriptorA;
  const title = descriptor?.displayName || diff.phase;
  return (
    <section className="rounded-lg border border-slate-800 bg-slate-900/40 p-3 space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-medium text-slate-200">{title}</h3>
          <p className="text-[11px] font-mono text-slate-500">{diff.phase}{descriptor?.provider ? ` · ${descriptor.provider}` : ""}</p>
        </div>
        <VerdictBadge verdict={diff.verdict} />
      </div>
      <p className="text-xs text-slate-500">{diff.statusA || "absent"} → {diff.statusB || "absent"}</p>
      <EntityList title="Regressions" items={diff.regressions} tone="regression" />
      <EntityList title="New failures" items={diff.newFailures} tone="new" />
      <EntityList title="Preexisting" items={diff.preexistingFailures} tone="preexisting" />
      <EntityList title="Cleared" items={diff.clearedFailures} tone="cleared" />
      {diff.reasons.map((reason, index) => (
        <p key={`${reason.code}-${index}`} className="text-xs text-amber-400">{reason.detail || "Comparison metadata unavailable"}</p>
      ))}
    </section>
  );
}

export function RunAnchorBadge({ manifest }: { manifest: BaselineManifest }) {
  if (!manifest.run?.runId) return <Badge variant="warning">Recapture required</Badge>;
  return (
    <span className="inline-flex items-center gap-1.5 rounded border border-slate-800 px-2 py-1 text-[11px] text-slate-400">
      run <span className="font-mono text-slate-200">{manifest.run.runId}</span>
      {manifest.migration && <span className="text-amber-400">degraded</span>}
    </span>
  );
}
