import { useEffect, useState } from "react";
import { getEpisodeCohort, getEpisodes, getLedger, type Availability, type Episode, type EpisodeSignal, type Ledger } from "../api/episodes";
import { Dialog, DialogContent, DialogHeader } from "../components/ui/dialog";
import { getApiBaseUrl } from "../lib/utils";
import type { TypedInvestigation } from "../types";

function coverageSummary(item: TypedInvestigation): string {
  const coverage = item.result?.coverage ?? [];
  if (coverage.length === 0) return "not yet available";
  return coverage.map((plane) => `${plane.plane ?? "plane"}: ${plane.state ?? "unknown"}`).join(" · ");
}

function AvailabilityBadge({ value }: { value: Availability }) {
  return <span className="rounded bg-muted px-2 py-1 text-xs" data-testid={`availability-${value.state}`}>{value.state}{value.reason ? `: ${value.reason}` : ""}</span>;
}

export function InvestigationsPage() {
  const [signals, setSignals] = useState<EpisodeSignal[]>([]);
  const [availability, setAvailability] = useState<Availability>({ state: "unavailable", reason: "loading" });
  const [selected, setSelected] = useState<{ runId: string; episodes: Episode[]; ledger: Ledger } | null>(null);
  const [typedInvestigations, setTypedInvestigations] = useState<TypedInvestigation[]>([]);
  const [typedError, setTypedError] = useState<string | null>(null);
  useEffect(() => { void getEpisodeCohort().then((data) => { setSignals(data.signals); setAvailability(data.availability); }).catch((error: unknown) => setAvailability({ state: "unavailable", reason: error instanceof Error ? error.message : "request failed" })); }, []);
  useEffect(() => {
    let active = true;
    void fetch(`${getApiBaseUrl()}/investigations?limit=50`, { headers: { "Content-Type": "application/json" } })
      .then(async (response) => {
        if (!response.ok) throw new Error(`lifecycle request failed: ${response.status}`);
        return response.json() as Promise<{ investigations?: TypedInvestigation[] }>;
      })
      .then((data) => { if (active) setTypedInvestigations(data.investigations ?? []); })
      .catch((error: unknown) => { if (active) setTypedError(error instanceof Error ? error.message : "lifecycle request failed"); });
    return () => { active = false; };
  }, []);
  const openSignal = async (signal: EpisodeSignal) => { const runId = signal.representativeRunIds[0]; if (!runId) return; const [episodeData, ledger] = await Promise.all([getEpisodes(runId), getLedger(runId)]); setSelected({ runId, episodes: episodeData.episodes, ledger }); };
  return (
    <section className="p-6 space-y-3" data-testid="investigations-page">
      <h1 className="text-2xl font-semibold">Investigations</h1>
      <section className="rounded border p-4 space-y-2" aria-label="Durable investigation lifecycle">
        <div><h2 className="text-lg font-medium">Durable lifecycle</h2><p className="text-sm text-muted-foreground">A completed subject run is not the same as a completed investigation. Review state, evidence coverage, and current applicability here.</p></div>
        {typedError ? <AvailabilityBadge value={{ state: "unavailable", reason: typedError }} /> : null}
        {!typedError && typedInvestigations.length === 0 ? <p className="text-sm text-muted-foreground">No caller-neutral investigations have been admitted.</p> : null}
        {typedInvestigations.length > 0 ? <div className="overflow-auto"><table className="w-full text-sm"><thead><tr><th className="text-left">Status</th><th className="text-left">Subject</th><th className="text-left">Diagnosis</th><th className="text-left">Coverage</th><th className="text-left">Applicability</th><th className="text-left">Updated</th></tr></thead><tbody>{typedInvestigations.map((item) => <tr key={item.investigationId} data-testid={`typed-investigation-${item.investigationId}`}><td><span className="rounded bg-muted px-2 py-1 text-xs">{item.operationStatus}</span></td><td>{item.request?.subject?.kind ?? "subject"}: {item.request?.subject?.ref ?? item.investigationId}<div className="text-xs text-muted-foreground">{item.request?.subject?.runIds?.length ?? 0} explicit run IDs</div></td><td>{item.result?.diagnosis ? `${item.result.diagnosis.condition ?? "unknown"} · ${item.result.diagnosis.disposition ?? "inconclusive"}` : "pending"}</td><td>{coverageSummary(item)}</td><td>{item.result?.applicability?.state ?? "not yet available"}</td><td>{item.updatedAt ? new Date(item.updatedAt).toLocaleString() : "—"}</td></tr>)}</tbody></table></div> : null}
      </section>
      <AvailabilityBadge value={availability} />
      <table className="w-full text-sm"><thead><tr><th>Fingerprint</th><th>Cost</th><th>Runs</th><th>Confidence</th></tr></thead><tbody>{signals.map((signal) => <tr key={signal.fingerprint}><td><button className="text-primary underline" onClick={() => void openSignal(signal)}>{signal.fingerprint}</button></td><td>{signal.summedCostMs} ms</td><td>{signal.distinctRuns}</td><td>{signal.confidence}</td></tr>)}</tbody></table>
      <Dialog open={selected !== null} onOpenChange={() => setSelected(null)} contentClassName="fixed inset-y-0 right-0 m-0 w-full max-w-3xl"><DialogContent className="h-full max-h-none max-w-none rounded-none"><DialogHeader onClose={() => setSelected(null)}><h2>Episode detail</h2></DialogHeader><div className="space-y-3 overflow-auto p-4">{selected?.episodes.map((episode) => <article key={episode.episodeId} className="rounded border p-3"><p>{episode.pattern} · {episode.causeScope} · {episode.ownerConfidence}</p><p>{episode.turns} turns, {episode.tokens} tokens, {episode.wallClockMs} ms</p><p>{episode.suspectedOwnerScenario} {episode.suspectedOwnerCommand}</p><p>{episode.evidenceEventIds.map((id) => <a key={id} className="mr-2 text-primary underline" href={`/runs/${selected.runId}?event=${encodeURIComponent(id)}`}>{id}</a>)}</p></article>)}{selected && <><AvailabilityBadge value={selected.ledger.ledgerAvailability} /><AvailabilityBadge value={selected.ledger.projectionAvailability} /><table className="w-full text-sm"><tbody>{selected.ledger.ledgerTargetRollups.map((rollup) => <tr key={rollup.targetScenario}><td>{rollup.targetScenario}</td><td>{rollup.calls} calls</td><td>{rollup.failures} failures</td><td>{rollup.medianDurationMs} ms median</td></tr>)}</tbody></table></>}</div></DialogContent></Dialog>
    </section>
  );
}
