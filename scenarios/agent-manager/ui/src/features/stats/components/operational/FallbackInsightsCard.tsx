// Surfaces typed-event runner+model fallback patterns.
//
// Honesty contract: when sample size < MinSampleMeaningful (5), the card
// renders InsufficientDataCard rather than fabricated percentages. The
// HistoryBanner above the page already informs users when the event log is
// younger than the 30-day window.

import { Link } from "react-router-dom";
import { InsufficientDataCard } from "../../../../components/stats/InsufficientDataCard";
import type { FallbackInsights, FallbackPair } from "../../api/operationalTypes";
import { useFallbackInsights } from "../../hooks/useOperationalStats";

const MIN_SAMPLE = 5;
const TOP_PAIRS = 5;
const TOP_REASONS = 5;

export function FallbackInsightsCard() {
  const { data, isLoading, error } = useFallbackInsights();

  if (isLoading) {
    return <Section title="Fallback insights"><p className="text-sm text-muted-foreground">Loading…</p></Section>;
  }
  if (error) {
    return (
      <Section title="Fallback insights">
        <p className="text-sm text-destructive">Failed to load fallback insights: {error.message}</p>
      </Section>
    );
  }
  if (!data) {
    return null;
  }

  const totalAttempts = data.runner_attempts + data.model_attempts;
  if (totalAttempts < MIN_SAMPLE) {
    return (
      <Section title="Fallback insights">
        <InsufficientDataCard
          label="Fallback patterns"
          reason="Need at least 5 fallback events to surface patterns honestly."
          have={totalAttempts}
          required={MIN_SAMPLE}
          testId="fallback-insufficient"
        />
      </Section>
    );
  }

  return (
    <Section title="Fallback insights">
      <div className="grid gap-3 lg:grid-cols-2" data-testid="fallback-insights">
        <Subsection title={`Runner fallbacks (${data.runner_attempts} attempted, ${data.runner_exhausted} exhausted)`}>
          <ChainDepthList depth={data.runner_chain_depth} />
          <PairsList pairs={data.runner_by_pair.slice(0, TOP_PAIRS)} testIdPrefix="runner-pair" />
          <ReasonsList counts={data.runner_by_reason} top={TOP_REASONS} testIdPrefix="runner-reason" />
        </Subsection>
        <Subsection title={`Model fallbacks (${data.model_attempts} attempted, ${data.model_exhausted} exhausted)`}>
          <ChainDepthList depth={data.model_chain_depth} />
          <PairsList pairs={data.model_by_pair.slice(0, TOP_PAIRS)} testIdPrefix="model-pair" />
          <ReasonsList counts={data.model_by_reason} top={TOP_REASONS} testIdPrefix="model-reason" />
          <PresetsList counts={data.model_by_preset} />
        </Subsection>
      </div>
      <SummaryFooter data={data} />
    </Section>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-border bg-card/40 p-4" data-testid="fallback-insights-card">
      <h3 className="mb-3 text-sm font-semibold">{title}</h3>
      {children}
    </section>
  );
}

function Subsection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">{title}</h4>
      {children}
    </div>
  );
}

function ChainDepthList({ depth }: { depth: Record<string, number> }) {
  const entries = Object.entries(depth)
    .map(([k, v]) => [Number(k), v] as const)
    .filter(([k]) => Number.isFinite(k))
    .sort((a, b) => a[0] - b[0]);
  if (entries.length === 0) return null;
  return (
    <div>
      <p className="text-xs text-muted-foreground">Chain depth distribution</p>
      <ul className="mt-1 space-y-0.5 text-sm">
        {entries.map(([d, count]) => (
          <li key={d} className="flex justify-between">
            <span>Depth {d}</span>
            <span className="font-mono text-xs">{count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function PairsList({ pairs, testIdPrefix }: { pairs: FallbackPair[]; testIdPrefix: string }) {
  if (pairs.length === 0) return null;
  return (
    <div>
      <p className="text-xs text-muted-foreground">Top transitions</p>
      <ul className="mt-1 space-y-0.5 text-sm">
        {pairs.map((p, i) => (
          <li
            key={`${p.from}|${p.to}|${p.reason}|${i}`}
            className="flex justify-between gap-2"
            data-testid={`${testIdPrefix}-${i}`}
          >
            <span className="truncate font-mono text-xs">
              {p.from} → {p.to}
              {p.reason ? ` (${p.reason})` : ""}
            </span>
            <span className="font-mono text-xs">{p.count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function ReasonsList({ counts, top, testIdPrefix }: { counts: Record<string, number>; top: number; testIdPrefix: string }) {
  const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, top);
  if (entries.length === 0) return null;
  return (
    <div>
      <p className="text-xs text-muted-foreground">Top reasons</p>
      <ul className="mt-1 space-y-0.5 text-sm">
        {entries.map(([reason, count], i) => (
          <li key={reason} className="flex justify-between" data-testid={`${testIdPrefix}-${i}`}>
            <span className="font-mono text-xs">{reason}</span>
            <span className="font-mono text-xs">{count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function PresetsList({ counts }: { counts: Record<string, number> }) {
  const entries = Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 5);
  if (entries.length === 0) return null;
  return (
    <div>
      <p className="text-xs text-muted-foreground">By preset</p>
      <ul className="mt-1 space-y-0.5 text-sm">
        {entries.map(([preset, count]) => (
          <li key={preset} className="flex justify-between">
            <span className="font-mono text-xs">{preset || "(none)"}</span>
            <span className="font-mono text-xs">{count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function SummaryFooter({ data }: { data: FallbackInsights }) {
  const days = Math.max(1, Math.round(data.history.history_days));
  return (
    <p className="mt-3 text-xs text-muted-foreground" data-testid="fallback-summary-footer">
      {data.event_count} fallback events over {days} day{days === 1 ? "" : "s"} · see{" "}
      <Link to="/observability" className="underline">
        /observability
      </Link>{" "}
      for the current snapshot.
    </p>
  );
}
