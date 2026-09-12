import { useEffect, useMemo, useState } from "react";
import { Activity, Bot, ChevronRight, FilePenLine, Filter, History, ShieldCheck } from "lucide-react";
import { Drawer } from "../../ui/drawer";
import { Button } from "../../ui/button";
import { defaultApiClient } from "../../../lib/api-client";

export type WorkFeedEntry = {
  id: string;
  kind: "execution" | "workflow" | "review" | "workshop" | "event";
  title: string;
  outcome?: string;
  actor?: string;
  started_at: string;
  ended_at?: string;
  cost_estimate?: number;
  correlation?: Record<string, string>;
  detail_ref: string;
  detail_api_ref?: string;
};

type EpisodeDetail = Record<string, unknown>;

const kindLabel: Record<WorkFeedEntry["kind"], string> = {
  execution: "Execution",
  workflow: "Agent workflow",
  review: "Review",
  workshop: "Plan workshop",
  event: "Item update",
};

function feedIcon(kind: WorkFeedEntry["kind"]) {
  const props = { className: "h-4 w-4" };
  if (kind === "review") return <ShieldCheck {...props} />;
  if (kind === "workshop") return <FilePenLine {...props} />;
  if (kind === "workflow") return <Bot {...props} />;
  if (kind === "event") return <History {...props} />;
  return <Activity {...props} />;
}

function outcomeClass(outcome?: string) {
  if (["completed", "complete", "accepted"].includes(outcome ?? "")) return "border-emerald-400/30 bg-emerald-400/10 text-emerald-200";
  if (["failed", "cancelled", "changes-requested"].includes(outcome ?? "")) return "border-rose-400/30 bg-rose-400/10 text-rose-200";
  return "border-slate-600 bg-slate-800 text-slate-300";
}

function durationLabel(entry: WorkFeedEntry) {
  if (!entry.ended_at) return "In progress";
  const milliseconds = new Date(entry.ended_at).getTime() - new Date(entry.started_at).getTime();
  if (!Number.isFinite(milliseconds) || milliseconds < 0) return "";
  const minutes = Math.floor(milliseconds / 60000);
  return minutes < 1 ? "<1m" : `${minutes}m`;
}

function DetailSummary({ entry, detail }: { entry: WorkFeedEntry; detail: EpisodeDetail }) {
  const status = typeof detail.status === "string" ? detail.status : entry.outcome;
  return <div className="rounded-lg border border-white/10 bg-slate-950/30 p-3"><p className="text-xs font-medium text-slate-300">Source detail</p><dl className="mt-2 grid gap-2 text-xs text-slate-400">{status ? <div><dt className="text-slate-500">Status</dt><dd className="mt-0.5 capitalize text-slate-200">{status}</dd></div> : null}{entry.kind === "review" && typeof detail.agent_assessment === "string" ? <div><dt className="text-slate-500">Assessment</dt><dd className="mt-0.5 text-slate-200">{detail.agent_assessment}</dd></div> : null}{entry.kind === "execution" && typeof detail.strategy === "string" ? <div><dt className="text-slate-500">Strategy</dt><dd className="mt-0.5 text-slate-200">{detail.strategy}</dd></div> : null}</dl></div>;
}

export function WorkFeedList({ entries, loading, error }: { entries: WorkFeedEntry[]; loading: boolean; error: Error | null }) {
  const [selected, setSelected] = useState<WorkFeedEntry | null>(null);
  const [kind, setKind] = useState<"all" | WorkFeedEntry["kind"]>("all");
  const [outcome, setOutcome] = useState<"all" | "active" | "terminal">("all");
  const [detail, setDetail] = useState<EpisodeDetail | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  useEffect(() => {
    if (!selected?.detail_api_ref) { setDetail(null); setDetailError(null); return; }
    let cancelled = false;
    setDetail(null); setDetailError(null);
    defaultApiClient.get<EpisodeDetail>(selected.detail_api_ref).then(
      (value) => { if (!cancelled) setDetail(value); },
      () => { if (!cancelled) setDetailError("Source detail is temporarily unavailable."); },
    );
    return () => { cancelled = true; };
  }, [selected]);
  const visible = useMemo(() => entries.filter((entry) => (kind === "all" || entry.kind === kind) && (outcome === "all" || (outcome === "active" ? !entry.ended_at : Boolean(entry.ended_at)))), [entries, kind, outcome]);
  if (loading) return <div className="p-5 text-sm text-slate-400">Loading work history…</div>;
  if (error) return <div className="rounded-lg border border-rose-400/30 bg-rose-400/10 p-4 text-sm text-rose-200">Work history could not be loaded. Try again shortly.</div>;
  return <section className="mt-4 rounded-xl border border-white/10 bg-slate-950/30" aria-label="Work history">
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
      <div><h2 className="text-sm font-semibold text-white">Work history</h2><p className="mt-0.5 text-xs text-slate-400">Every execution, workshop, review, workflow, and item update.</p></div>
      <div className="flex flex-wrap items-center gap-2 text-xs text-slate-300"><Filter className="h-3.5 w-3.5" /><label><span className="sr-only">Filter work history</span><select aria-label="Filter work history" value={kind} onChange={(event) => setKind(event.target.value as typeof kind)} className="rounded border border-white/10 bg-slate-900 px-2 py-1 text-xs text-slate-100"><option value="all">All activity</option><option value="execution">Executions</option><option value="workflow">Workflows</option><option value="review">Reviews</option><option value="workshop">Plan workshops</option><option value="event">Item updates</option></select></label><label><span className="sr-only">Filter work outcome</span><select aria-label="Filter work outcome" value={outcome} onChange={(event) => setOutcome(event.target.value as typeof outcome)} className="rounded border border-white/10 bg-slate-900 px-2 py-1 text-xs text-slate-100"><option value="all">Any outcome</option><option value="active">In progress</option><option value="terminal">Finished</option></select></label></div>
    </div>
    {visible.length === 0 ? <div className="p-6 text-center text-sm text-slate-400">No matching work episodes yet.</div> : <ul className="divide-y divide-white/5">{visible.map((entry) => <li key={entry.id}><button type="button" onClick={() => setSelected(entry)} className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-white/[0.04]"><span className="rounded-md bg-slate-800 p-2 text-cyan-300">{feedIcon(entry.kind)}</span><span className="min-w-0 flex-1"><span className="flex flex-wrap items-center gap-2"><span className="truncate text-sm font-medium text-slate-100">{entry.title}</span>{entry.outcome ? <span className={`rounded-full border px-2 py-0.5 text-[11px] ${outcomeClass(entry.outcome)}`}>{entry.outcome}</span> : null}</span><span className="mt-1 block text-xs text-slate-400">{kindLabel[entry.kind]} · {new Date(entry.started_at).toLocaleString()} · {durationLabel(entry)}{entry.cost_estimate ? ` · ≈ $${entry.cost_estimate.toFixed(2)}` : ""}{entry.actor ? ` · ${entry.actor}` : ""}</span></span><ChevronRight className="h-4 w-4 text-slate-500" /></button></li>)}</ul>}
    <Drawer isOpen={selected !== null} onClose={() => setSelected(null)} title={selected?.title ?? "Work episode"} description={selected ? `${kindLabel[selected.kind]} · ${selected.outcome ?? "in progress"}` : undefined} footer={selected ? <Button asChild><a href={selected.detail_ref}>Open source detail</a></Button> : undefined}><div className="space-y-4 p-4 text-sm"><dl className="grid gap-3 text-slate-300"><div><dt className="text-xs text-slate-500">Started</dt><dd>{selected ? new Date(selected.started_at).toLocaleString() : ""}</dd></div>{selected?.ended_at ? <div><dt className="text-xs text-slate-500">Finished</dt><dd>{new Date(selected.ended_at).toLocaleString()}</dd></div> : null}{selected?.actor ? <div><dt className="text-xs text-slate-500">Actor</dt><dd>{selected.actor}</dd></div> : null}{selected?.correlation && Object.keys(selected.correlation).length > 0 ? <div><dt className="text-xs text-slate-500">Correlation</dt><dd className="mt-1 space-y-1 font-mono text-xs">{Object.entries(selected.correlation).map(([key, value]) => <div key={key}>{key}: {value}</div>)}</dd></div> : null}</dl>{selected?.detail_api_ref && !detail && !detailError ? <p className="text-xs text-slate-400">Loading source detail…</p> : null}{detail && selected ? <DetailSummary entry={selected} detail={detail} /> : null}{detailError ? <p className="text-xs text-amber-200">{detailError}</p> : null}<p className="text-xs leading-5 text-slate-400">Open the source detail for the full evidence and controls. The Activity view keeps this timeline scannable.</p></div></Drawer>
  </section>;
}
