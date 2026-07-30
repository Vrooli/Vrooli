import { useEffect, useState } from "react";
import { getEpisodeCohort, getEpisodes, getLedger, type Availability, type Episode, type EpisodeSignal, type Ledger } from "../api/episodes";
import { Dialog, DialogContent, DialogHeader } from "../components/ui/dialog";

function AvailabilityBadge({ value }: { value: Availability }) {
  return <span className="rounded bg-muted px-2 py-1 text-xs" data-testid={`availability-${value.state}`}>{value.state}{value.detail ? `: ${value.detail}` : ""}</span>;
}

export function InvestigationsPage() {
  const [signals, setSignals] = useState<EpisodeSignal[]>([]);
  const [availability, setAvailability] = useState<Availability>({ state: "unavailable", detail: "loading" });
  const [selected, setSelected] = useState<{ runId: string; episodes: Episode[]; ledger: Ledger } | null>(null);
  useEffect(() => { void getEpisodeCohort().then((data) => { setSignals(data.signals); setAvailability(data.availability); }).catch((error: unknown) => setAvailability({ state: "unavailable", detail: error instanceof Error ? error.message : "request failed" })); }, []);
  const openSignal = async (signal: EpisodeSignal) => { const runId = signal.representativeRunIds[0]; if (!runId) return; const [episodeData, ledger] = await Promise.all([getEpisodes(runId), getLedger(runId)]); setSelected({ runId, episodes: episodeData.episodes, ledger }); };
  return (
    <section className="p-6 space-y-3" data-testid="investigations-page">
      <h1 className="text-2xl font-semibold">Investigations</h1>
      <AvailabilityBadge value={availability} />
      <table className="w-full text-sm"><thead><tr><th>Fingerprint</th><th>Cost</th><th>Runs</th><th>Confidence</th></tr></thead><tbody>{signals.map((signal) => <tr key={signal.fingerprint}><td><button className="text-primary underline" onClick={() => void openSignal(signal)}>{signal.fingerprint}</button></td><td>{signal.summedCostMs} ms</td><td>{signal.distinctRuns}</td><td>{signal.confidence}</td></tr>)}</tbody></table>
      <Dialog open={selected !== null} onOpenChange={() => setSelected(null)} contentClassName="fixed inset-y-0 right-0 m-0 w-full max-w-3xl"><DialogContent className="h-full max-h-none max-w-none rounded-none"><DialogHeader onClose={() => setSelected(null)}><h2>Episode detail</h2></DialogHeader><div className="space-y-3 overflow-auto p-4">{selected?.episodes.map((episode) => <article key={episode.episodeId} className="rounded border p-3"><p>{episode.pattern} · {episode.causeScope} · {episode.ownerConfidence}</p><p>{episode.turns} turns, {episode.tokens} tokens, {episode.wallClockMs} ms</p><p>{episode.suspectedOwnerScenario} {episode.suspectedOwnerCommand}</p><p>{episode.evidenceEventIds.map((id) => <a key={id} className="mr-2 text-primary underline" href={`/runs/${selected.runId}?event=${encodeURIComponent(id)}`}>{id}</a>)}</p></article>)}{selected && <><AvailabilityBadge value={selected.ledger.ledgerAvailability} /><AvailabilityBadge value={selected.ledger.projectionAvailability} /><table className="w-full text-sm"><tbody>{selected.ledger.ledgerTargetRollups.map((rollup) => <tr key={rollup.targetScenario}><td>{rollup.targetScenario}</td><td>{rollup.calls} calls</td><td>{rollup.failures} failures</td><td>{rollup.medianDurationMs} ms median</td></tr>)}</tbody></table></>}</div></DialogContent></Dialog>
    </section>
  );
}
