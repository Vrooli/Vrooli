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
  // The file-linked generated package can lag a deployed API. Keep this read
  // additive so old UIs still render the rest of the comparison safely.
  const enriched = diff as PhaseDiff & { behavior?: string; coverage?: string; compatibility?: string; provenance?: string; diagnostics?: Array<{ side?: string; code?: string; detail?: string; remediation?: string }> };
  const [expanded, setExpanded] = useState(diff.verdict !== "clean");
  return (
    <section className="rounded-lg border border-slate-800 bg-slate-900/40">
      <button type="button" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)} className="flex w-full flex-wrap items-center justify-between gap-2 rounded-lg p-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500">
        <div className="flex min-w-0 items-center gap-2">
          {expanded ? <ChevronDown className="h-3.5 w-3.5 shrink-0 text-slate-500" /> : <ChevronRight className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
          <div>
          <h3 className="text-sm font-medium text-slate-200">{title}</h3>
          <p className="text-[11px] font-mono text-slate-500">{diff.phase}{descriptor?.provider ? ` · ${descriptor.provider}` : ""}</p>
          </div>
        </div>
        <VerdictBadge verdict={diff.verdict} />
      </button>
      {expanded && <div className="space-y-2 border-t border-slate-800 px-3 pb-3 pt-2">
        <p className="text-xs text-slate-500">{diff.statusA || "absent"} → {diff.statusB || "absent"}</p>
        {descriptor?.description && <p className="text-xs text-slate-400">{descriptor.description}</p>}
        <EntityList title="Regressions" items={diff.regressions} tone="regression" />
        <EntityList title="New failures" items={diff.newFailures} tone="new" />
        <EntityList title="Preexisting" items={diff.preexistingFailures} tone="preexisting" />
        <EntityList title="Cleared" items={diff.clearedFailures} tone="cleared" />
        <div className="flex flex-wrap gap-1 text-[11px] text-slate-400">
          <span>Behavior: {enriched.behavior || "unknown"}</span><span>·</span>
          <span>Coverage: {enriched.coverage || "legacy"}</span><span>·</span>
          <span>Contract: {enriched.compatibility || "legacy"}</span><span>·</span>
          <span>Provenance: {enriched.provenance || "legacy"}</span>
        </div>
        {diff.reasons.map((reason, index) => <p key={`${reason.code}-${index}`} className="text-xs text-amber-400">{reason.detail || "Comparison metadata unavailable"}</p>)}
        {enriched.diagnostics?.map((diagnostic, index) => <div key={`${diagnostic.side}-${diagnostic.code}-${index}`} className="rounded border border-amber-900/40 bg-amber-950/20 px-2 py-1 text-xs text-amber-200"><span className="font-mono">{diagnostic.side || "comparison"}/{diagnostic.code || "diagnostic"}</span>{diagnostic.detail ? ` — ${diagnostic.detail}` : ""}{diagnostic.remediation ? <p className="mt-1 text-amber-300">Recovery: {diagnostic.remediation}</p> : null}</div>)}
      </div>}
    </section>
  );
}

export function RunAnchorBadge({ manifest }: { manifest: BaselineManifest }) {
  if (!manifest.run?.runId) return <Badge variant="warning">Recapture required</Badge>;
  return (
    <span className="inline-flex items-center gap-1.5 rounded border border-slate-800 px-2 py-1 text-[11px] text-slate-400">
      run <span className="font-mono text-slate-200">{manifest.run.runId}</span>
      {manifest.run.evidenceTier && <span className={manifest.run.evidenceTier === "shared-scoped" ? "text-sky-300" : manifest.run.evidenceTier === "strict" ? "text-emerald-300" : "text-amber-400"}>{manifest.run.evidenceTier}</span>}
      {!manifest.run.sourceStable && <span className="text-amber-400">retry needed</span>}
      {manifest.migration && <span className="text-amber-400">degraded</span>}
    </span>
  );
}
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";
