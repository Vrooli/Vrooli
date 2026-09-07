import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, RefreshCw, X } from "lucide-react";
import { fetchSourceDistribution, fetchSourceDistributions } from "../lib/api-source-distribution";
import type { SourceDistribution, SourceDistributionDetailResponse } from "../lib/api-types-source-distribution";

interface Props { repoId?: string | null; initialDistributionId?: string | null; onClose: () => void; onSelectDistribution?: (id: string | null) => void; }
const short = (value: string) => value ? `${value.slice(0, 12)}…` : "—";
const statusLabel = (value: string) => value.replace(/_/g, " ") || "unknown";

export function SourceDistributionPanel({ repoId, initialDistributionId, onClose, onSelectDistribution }: Props) {
  const [selected, setSelected] = useState<SourceDistribution | null>(null);
  const list = useQuery({ queryKey: ["source-distributions", repoId], queryFn: () => fetchSourceDistributions(repoId ?? undefined), staleTime: 15_000 });
  const detail = useQuery({ queryKey: ["source-distribution", selected?.distribution_id], queryFn: () => fetchSourceDistribution(selected!.distribution_id), enabled: Boolean(selected) });

  useEffect(() => {
    if (!initialDistributionId || !list.data) return;
    const match = list.data.distributions.find((item) => item.distribution_id === initialDistributionId);
    if (match) setSelected(match);
  }, [initialDistributionId, list.data]);

  return <div data-testid="source-distributions-panel" className="fixed inset-0 z-50 flex items-start justify-center bg-slate-950/80 p-3 sm:p-8" role="dialog" aria-modal="true" aria-label="Source distributions">
    <section className="flex max-h-[calc(100vh-1.5rem)] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border border-slate-700 bg-slate-900 shadow-2xl sm:max-h-[calc(100vh-4rem)]">
      <header className="flex items-center justify-between border-b border-slate-800 px-5 py-4"><div><p className="text-xs uppercase tracking-widest text-cyan-400">Repository workspace</p><h2 className="text-xl font-semibold text-slate-100">Source distributions</h2><p className="mt-1 text-sm text-slate-400">Read-only projection owned by scenario-to-repository</p></div><button className="rounded p-2 text-slate-400 hover:bg-slate-800 hover:text-white" onClick={onClose} aria-label="Close source distributions"><X className="h-5 w-5" /></button></header>
      <div className="grid min-h-0 flex-1 md:grid-cols-[19rem_1fr]">
        <aside className="min-h-0 overflow-y-auto border-b border-slate-800 p-3 md:border-b-0 md:border-r"><div className="mb-3 flex items-center justify-between"><span className="text-xs font-medium uppercase tracking-wide text-slate-500">Candidates</span><button onClick={() => list.refetch()} className="rounded p-1 text-slate-400 hover:bg-slate-800" aria-label="Refresh distributions"><RefreshCw className={`h-4 w-4 ${list.isFetching ? "animate-spin" : ""}`} /></button></div>
          {list.isLoading && <p className="p-3 text-sm text-slate-400">Loading authoritative records…</p>}
          {list.isError && <div className="rounded-lg border border-amber-800 bg-amber-950/40 p-3 text-sm text-amber-200">Source-ramp unavailable. Try again when the owner is online.</div>}
          {!list.isLoading && !list.isError && (list.data?.distributions.length ?? 0) === 0 && <div className="rounded-lg border border-slate-800 p-4 text-sm text-slate-400"><p className="font-medium text-slate-200">No distribution exists</p><p className="mt-1">Start Share source in the authoritative source-ramp workflow.</p></div>}
          <div className="space-y-2">{list.data?.distributions.map((item) => <button key={item.distribution_id} onClick={() => { setSelected(item); onSelectDistribution?.(item.distribution_id); }} className={`w-full rounded-lg border p-3 text-left transition ${selected?.distribution_id === item.distribution_id ? "border-cyan-500 bg-cyan-950/30" : "border-slate-800 bg-slate-950/40 hover:border-slate-600"}`}><div className="flex items-center justify-between gap-2"><span className="truncate font-medium text-slate-100">{item.scenario || "Unnamed scenario"}</span><span className="text-[10px] uppercase text-slate-400">{statusLabel(item.publication_status)}</span></div><p className="mt-2 font-mono text-xs text-slate-500">{item.distribution_id}</p><p className="mt-1 text-xs text-slate-400">artifact {short(item.artifact_digest)} · {statusLabel(item.drift_state)}</p></button>)}</div>
        </aside>
        <main className="min-h-0 overflow-y-auto p-5">{!selected && <div className="flex h-full min-h-56 items-center justify-center text-center text-sm text-slate-400"><div><p className="text-base text-slate-200">Select an exact candidate</p><p className="mt-1">Approval, publication, and destination read-back remain separate.</p></div></div>}
          {selected && detail.isLoading && <p className="text-sm text-slate-400">Reading evidence…</p>}
          {selected && detail.isError && <div className="rounded-lg border border-amber-800 bg-amber-950/40 p-4 text-sm text-amber-200">This distribution is temporarily unavailable. The list record is not treated as current.</div>}
          {detail.data && <DistributionDetail data={detail.data} />}
        </main>
      </div>
    </section>
  </div>;
}

function DistributionDetail({ data }: { data: SourceDistributionDetailResponse }) {
  const d = data;
  const item = d.distribution;
  const fields = [["source", item.source_digest], ["closure", item.closure_digest], ["recipe", item.recipe_digest], ["policy", item.policy_digest], ["artifact", item.artifact_digest], ["verification receipt", item.verification_receipt], ["Deployment Manager decision", item.deployment_manager_decision]];
  return <div className="space-y-5"><div className="flex flex-wrap items-start justify-between gap-3"><div><h3 className="text-lg font-semibold text-slate-100">{item.scenario}</h3><p className="font-mono text-xs text-slate-500">{item.distribution_id}</p></div><div className="flex gap-2 text-xs"><span className="rounded-full bg-emerald-950 px-3 py-1 text-emerald-300">verification: {statusLabel(item.verification_status)}</span><span className="rounded-full bg-amber-950 px-3 py-1 text-amber-300">publication: {statusLabel(item.publication_status)}</span></div></div>
    <section><h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">Exact candidate tuple</h4><div className="grid gap-2 sm:grid-cols-2">{fields.map(([label, value]) => <div key={label} className="rounded-lg border border-slate-800 bg-slate-950/40 p-3"><p className="text-xs text-slate-500">{label}</p><p className="mt-1 break-all font-mono text-xs text-slate-200">{value || "not recorded"}</p></div>)}</div></section>
    <section><h4 className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">Contents & obligations</h4><div className="space-y-2 text-sm text-slate-300"><p>{d.contents?.length ?? 0} admitted items · {d.exclusions?.length ?? 0} excluded items</p>{d.contents?.length > 0 && <div className="grid gap-1 sm:grid-cols-2">{d.contents.slice(0, 40).map((x: any) => <div key={x.path} className="rounded border border-slate-800 px-2 py-1 text-xs"><span className="text-cyan-300">{x.category}</span> <span className="text-slate-400">{x.path}</span></div>)}</div>}{d.exclusions?.map((x: any) => <div key={x.path} className="rounded border border-amber-900/60 bg-amber-950/20 px-2 py-1 text-xs text-amber-200">excluded · {x.category} · {x.path} · {x.safe_reason}</div>)}{d.runtime_requirements?.length > 0 && <p className="text-xs text-slate-400">Runtime requirements: {d.runtime_requirements.join(", ")}</p>}{d.unresolved_obligations?.length > 0 && <p className="text-xs text-amber-300">Unresolved: {d.unresolved_obligations.join(", ")}</p>}</div></section>
    <section className="grid gap-3 sm:grid-cols-2"><div className="rounded-lg border border-slate-800 p-4"><h4 className="text-sm font-medium text-slate-200">Publication handoff</h4><p className="mt-2 text-xs text-slate-400">{d.handoff?.human_action ?? "Handoff unavailable"}</p><p className="mt-2 text-xs text-slate-400">Status: {statusLabel(d.handoff?.status ?? item.publication_status)}</p><p className="mt-1 text-xs text-slate-400">Read-back: {d.handoff?.readback_oracle ?? "Exact destination evidence required"}</p><p className="mt-2 text-xs font-medium text-amber-300">Approval is not publication.</p></div><div className="rounded-lg border border-slate-800 p-4"><h4 className="text-sm font-medium text-slate-200">Drift</h4><p className="mt-2 text-sm text-slate-300">{statusLabel(d.drift?.state ?? item.drift_state)}</p><p className="mt-1 text-xs text-slate-400">Source changed: {d.drift?.source_changed ? "yes" : "no"} · Destination changed: {d.drift?.destination_changed ? "yes" : "no"}</p>{d.drift?.actions?.map((a: string) => <p key={a} className="mt-1 text-xs text-amber-300">{a}</p>)}</div></section>
    <footer className="flex flex-wrap items-center gap-3 border-t border-slate-800 pt-4 text-xs text-slate-500"><span>authority: {item.source_of_truth || "scenario-to-repository"}</span><span>freshness: {item.freshness || "unknown"}</span>{item.workflow_url && <a href={item.workflow_url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-cyan-300 hover:text-cyan-200">Open authoritative workflow <ExternalLink className="h-3 w-3" /></a>}</footer>
  </div>;
}
