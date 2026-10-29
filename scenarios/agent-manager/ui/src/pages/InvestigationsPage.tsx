import { useState } from "react";
import { Link } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { getEpisodeCohort, getEpisodes, getLedger, type Availability, type Episode, type EpisodeSignal, type Ledger } from "../api/episodes";
import { DataState } from "../components/feedback/DataState";
import { ErrorNotice } from "../components/feedback/ErrorNotice";
import { Dialog, DialogContent, DialogHeader } from "../components/ui/dialog";
import { useApiQuery } from "../hooks/useApiQuery";
import { useBoundedList } from "../hooks/useBoundedList";
import { ShowMoreButton } from "../components/patterns/ShowMoreButton";
import { apiRequest } from "../hooks/useApiTransport";
import { formatStandardRelativeTime } from "../lib/dateTime";
import { formatCode, formatCount, formatDuration, formatTokens, pluralize, shortenIds } from "../lib/format";
import { statusTone } from "../lib/display";
import type { TypedInvestigation } from "../types";

// Both lists can hold hundreds of rows; show a page at a time.
const LIST_PAGE_SIZE = 20;

/** Short columns keep their words whole; the table scrolls sideways before it breaks "Applicability" mid-word. */
const NO_BREAK = "whitespace-nowrap break-normal";

function coverageSummary(item: TypedInvestigation): string {
  const coverage = item.result?.coverage ?? [];
  if (coverage.length === 0) return "Not yet available";
  return coverage.map((plane) => `${formatCode(plane.plane)}: ${formatCode(plane.state)}`).join(" · ");
}

function diagnosisSummary(item: TypedInvestigation): string {
  const diagnosis = item.result?.diagnosis;
  return diagnosis ? `${formatCode(diagnosis.condition)} · ${formatCode(diagnosis.disposition ?? "inconclusive")}` : "Pending";
}

function AvailabilityNote({ value }: { value: Availability }) {
  if (value.state === "available") return null;
  return <p className="text-sm text-muted-foreground" data-testid={`availability-${value.state}`}>{formatCode(value.state)}{value.reason ? `: ${value.reason}` : ""}</p>;
}

function InvestigationRows({ items: all }: { items: TypedInvestigation[] }) {
  const { visibleItems: items, total, showMore } = useBoundedList(all, LIST_PAGE_SIZE);
  return (
    <>
    <Table>
      <table className="min-w-[56rem]">
        <caption className="sr-only">Durable investigations</caption>
        <thead><tr><th className={NO_BREAK}>Status</th><th className="min-w-[10rem] break-normal">Subject</th><th className="min-w-[10rem] break-normal">Diagnosis</th><th className="min-w-[12rem] break-normal">Coverage</th><th className={NO_BREAK}>Applicability</th><th className={NO_BREAK}>Updated</th></tr></thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.investigationId} data-testid={`typed-investigation-${item.investigationId}`}>
              <td className={NO_BREAK}><StatusBadge tone={statusTone(item.operationStatus)}>{formatCode(item.operationStatus)}</StatusBadge></td>
              <td className="min-w-[10rem] break-normal">
                {formatCode(item.request?.subject?.kind ?? "subject")}: {shortenIds(item.request?.subject?.ref ?? item.investigationId)}
                <div className="text-xs text-muted-foreground">{pluralize(item.request?.subject?.runIds?.length ?? 0, "run")}</div>
              </td>
              <td className="min-w-[10rem] break-normal">{diagnosisSummary(item)}</td>
              <td className="min-w-[12rem] break-normal">{coverageSummary(item)}</td>
              <td className={NO_BREAK}>{item.result?.applicability?.state ? formatCode(item.result.applicability.state) : "Not yet available"}</td>
              <td className={NO_BREAK}>{formatStandardRelativeTime(item.updatedAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Table>
    <ShowMoreButton shown={items.length} total={total} onShowMore={showMore} testId="investigations-show-more" />
    </>
  );
}

function EpisodeDetail({ runId, episodes, ledger }: { runId: string; episodes: Episode[]; ledger: Ledger }) {
  return (
    <div className="space-y-3 overflow-auto p-4">
      {episodes.map((episode) => (
        <article key={episode.episodeId} className="rounded border p-3">
          <p>{formatCode(episode.pattern)} · {formatCode(episode.causeScope)} · {formatCode(episode.ownerConfidence)} confidence</p>
          <p>{formatCount(episode.turns)} requests, {formatTokens(episode.tokens)} tokens, {formatDuration(episode.wallClockMs)}</p>
          <p>{shortenIds(`${episode.suspectedOwnerScenario ?? ""} ${episode.suspectedOwnerCommand ?? ""}`)}</p>
          <p className="flex flex-wrap gap-x-3">{episode.evidenceEventIds.map((id, index) => <Link key={id} className="inline-flex min-h-11 items-center text-primary underline sm:min-h-6" title={id} to={`/runs/${runId}?event=${encodeURIComponent(id)}`}>{`Evidence ${index + 1}`}</Link>)}</p>
        </article>
      ))}
      <AvailabilityNote value={ledger.ledgerAvailability} />
      <AvailabilityNote value={ledger.projectionAvailability} />
      <Table>
        <table className="min-w-[28rem]"><caption className="sr-only">Calls by target</caption><tbody>
          {ledger.ledgerTargetRollups.map((rollup) => (
            <tr key={rollup.targetScenario}>
              <td>{rollup.targetScenario}</td><td>{formatCount(rollup.calls)} calls</td><td>{formatCount(rollup.failures)} failures</td><td>{formatDuration(rollup.medianDurationMs)} median</td>
            </tr>
          ))}
        </tbody></table>
      </Table>
    </div>
  );
}

export function InvestigationsPage() {
  const cohort = useApiQuery(["episode-cohort"], () => getEpisodeCohort());
  const lifecycle = useApiQuery<TypedInvestigation[]>(
    ["investigations"],
    async (signal) => (await apiRequest<{ investigations?: TypedInvestigation[] }>("/investigations?limit=50", { signal })).investigations ?? [],
    { fallback: [] },
  );
  const [selected, setSelected] = useState<{ runId: string; episodes: Episode[]; ledger: Ledger } | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);

  const openSignal = async (signal: EpisodeSignal) => {
    const runId = signal.representativeRunIds[0];
    if (!runId) return;
    setDetailError(null);
    try {
      const [episodeData, ledger] = await Promise.all([getEpisodes(runId), getLedger(runId)]);
      setSelected({ runId, episodes: episodeData.episodes, ledger });
    } catch (error) {
      setDetailError(error instanceof Error ? error.message : "The episode detail request failed.");
    }
  };

  const items = lifecycle.data ?? [];
  const allSignals = cohort.data?.signals ?? [];
  const { visibleItems: signals, total: signalTotal, showMore: showMoreSignals } = useBoundedList(allSignals, LIST_PAGE_SIZE);
  return (
    <section className="h-full overflow-y-auto overflow-x-hidden p-4 sm:p-6" data-testid="investigations-page"><div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Investigations"
        description="A completed subject run is not the same as a completed investigation. Review state, evidence coverage and current applicability here."
        actions={<Link to="/observability" className="inline-flex min-h-11 items-center text-sm font-medium text-primary hover:underline sm:min-h-8">Health →</Link>}
      />
      <section aria-label="Durable investigation lifecycle" className="space-y-2">
        <DataState
          subject="Investigations"
          loading={lifecycle.loading}
          error={lifecycle.error}
          isEmpty={items.length === 0}
          onRetry={lifecycle.refetch}
          empty={<EmptyState density="compact" title="No investigations yet" description="Start one from a run's Investigate action." />}
        >
          <InvestigationRows items={items} />
        </DataState>
      </section>
      <section aria-label="Recurring friction" className="space-y-2">
        <h2 className="text-lg font-medium">Recurring friction</h2>
        {cohort.data ? <AvailabilityNote value={cohort.data.availability} /> : null}
        <DataState
          subject="Recurring friction"
          loading={cohort.loading}
          error={cohort.error}
          isEmpty={allSignals.length === 0}
          onRetry={cohort.refetch}
          empty={<EmptyState density="compact" title="No recurring friction this week" description="Patterns that repeat across runs appear here." />}
        >
          <Table>
            <table className="min-w-[28rem]">
              <caption className="sr-only">Recurring friction patterns</caption>
              <thead><tr><th>Pattern</th><th>Time lost</th><th>Runs</th><th>Confidence</th></tr></thead>
              <tbody>
                {signals.map((signal) => (
                  <tr key={signal.fingerprint}>
                    <td><Button variant="ghost" size="sm" className="min-h-11 px-0 text-primary underline sm:min-h-8" title={signal.fingerprint} onClick={() => void openSignal(signal)}>{shortenIds(signal.fingerprint)}</Button></td>
                    <td className={NO_BREAK}>{formatDuration(signal.summedCostMs)}</td>
                    <td>{formatCount(signal.distinctRuns)}</td>
                    <td className={NO_BREAK}>{formatCode(signal.confidence)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Table>
          <ShowMoreButton shown={signals.length} total={signalTotal} onShowMore={showMoreSignals} testId="friction-signals-show-more" />
        </DataState>
        {detailError ? <ErrorNotice compact error={detailError} subject="Episode detail" /> : null}
      </section>
      <Dialog open={selected !== null} onOpenChange={() => setSelected(null)} contentClassName="fixed inset-y-0 right-0 m-0 w-full max-w-3xl">
        <DialogContent className="h-full max-h-none max-w-none rounded-none">
          <DialogHeader onClose={() => setSelected(null)}><h2>Episode detail</h2></DialogHeader>
          {selected ? <EpisodeDetail {...selected} /> : null}
        </DialogContent>
      </Dialog>
    </div></section>
  );
}
