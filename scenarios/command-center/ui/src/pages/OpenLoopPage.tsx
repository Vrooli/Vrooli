import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { AmbientShell } from "../components/AmbientShell";
import { useBoardController } from "../lib/boardContext";
import { Figure } from "../components/Figure";
import { fetchOpenLoop, type Reading } from "../lib/api";

const MAX_AGE_DAYS = 90;

/** Every hole, dated and ageing: missing cells, unregistered outcomes, and this board's own blind spots. */
export default function OpenLoopPage() {
  const [searchParams] = useSearchParams();
  const board = useBoardController();
  const query = useQuery({ queryKey: ["open-loop"], queryFn: fetchOpenLoop, refetchInterval: 60_000 });
  const forcedState = searchParams.get("state");
  const forcedError = forcedState === "error" || forcedState === "objective-set-unavailable";
  const data = forcedState === "empty" ? { missing: [], unregistered: [], self: [] } : query.data;
  const failed = forcedError || Boolean(query.error);
  const state: ExperienceSurfaceState = query.isLoading ? "loading" : failed ? "error" : forcedState === "partial" ? "partial" : "ready";
  const surfaceState: ExperienceSurfaceState = state === "error" ? "ready" : state;
  const missing = data?.missing ?? [];
  const unregistered = data?.unregistered ?? [];
  const self = data?.self ?? [];
  const oldest = Math.max(0, ...missing.map((item) => item.gapOpenDays ?? 0));
  const denominator = board.board?.denominator;
  return (
    <AmbientShell theme="signal-tower" title="Open loop" position="SELF-REPORT">
      <main className="cc-read" data-testid="open-loop-page">
        <ExperienceSurface surfaceId="blind-spots" as="section" data-testid="open-loop-blind-spots" data-requested-state={forcedState ?? undefined} className="cc-read-main" state={surfaceState} statusMessage={failed ? "The open-loop report is unavailable." : undefined} aria-labelledby="open-loop-title">
          <h2 id="open-loop-title" className="cc-read-title">What the board cannot see</h2>
          <div className="cc-tally" role="group" aria-label="Open loop tally">
            <div className="cc-tally-item"><Figure value={missing.length} ink="dotted" scale="display" /><span>missing cells</span></div>
            <div className="cc-tally-item"><Figure value={unregistered.length} ink="dotted" scale="display" /><span>unregistered outcomes</span></div>
            <div className="cc-tally-item"><Figure value={self.length} ink="dotted" scale="display" /><span>own blind spots</span></div>
            <div className="cc-tally-item"><Figure value={oldest} ink="solid" scale="display" /><span>days, oldest hole</span></div>
          </div>
          {failed ? <p role="alert" className="cc-degraded">The open-loop report is unavailable.</p> : null}
          {state === "partial" ? <p role="status" className="cc-degraded cc-degraded-quiet">This report is partial: one or more source declarations could not be read.</p> : null}
          <ExperienceSurface surfaceId="missing" as="div" data-testid="open-loop-missing" data-requested-state={forcedState ?? undefined} state={surfaceState} className="cc-holes">
            <h3>Missing cells</h3>
            {missing.length === 0 && !query.isLoading ? <p className="cc-empty">No registered metric is without substrate.</p> : null}
            <ul className="cc-hole-list">{missing.map((item) => <Hole key={`missing-${item.id}`} item={item} />)}</ul>
          </ExperienceSurface>
          <ExperienceSurface surfaceId="unregistered" as="div" data-testid="open-loop-unregistered" data-requested-state={forcedState ?? undefined} state={surfaceState} className="cc-holes">
            <h3>Unregistered outcomes</h3>
            {unregistered.length === 0 && !query.isLoading ? <p className="cc-empty">Every outcome the objective set names has a registry row.</p> : null}
            <ul className="cc-hole-list">{unregistered.map((item) => <Hole key={`unregistered-${item.id}`} item={item} unregistered />)}</ul>
          </ExperienceSurface>
        </ExperienceSurface>
        <aside className="cc-read-aside">
          <ExperienceSurface surfaceId="self" as="section" data-testid="open-loop-self" className="cc-panel" state="static" aria-label="Instrument blind spots">
            <h3>This instrument's own blind spots</h3>
            <ul className="cc-hole-list">
              {self.map((item) => (
                <li key={item.id} className="cc-hole" data-ink="dotted">
                  <span className="cc-hole-label">{item.reason}</span>
                  <span className="cc-hole-meta">since {item.firstObservedMissing} · open {item.gapOpenDays} days</span>
                  <AgeBar days={item.gapOpenDays} />
                </li>
              ))}
            </ul>
          </ExperienceSurface>
          <ExperienceSurface surfaceId="denominator" as="section" data-testid="open-loop-denominator" className="cc-panel" state={state === "partial" ? "partial" : "ready"} aria-label="Denominator">
            <h3>Denominator</h3>
            <p className="cc-denominator"><Figure value={denominator?.outcomeCategories ?? board.rooms.length} ink="solid" scale="display" /> outcome categories · <span className="cc-ink-mark" data-ink="hollow">{denominator?.confidence ?? "partial"} confidence</span></p>
            <p className="cc-panel-note">{denominator?.rationale ?? "The declared outcome registry is the searched denominator."}</p>
          </ExperienceSurface>
        </aside>
      </main>
    </AmbientShell>
  );
}

function Hole({ item, unregistered = false }: { item: Reading; unregistered?: boolean }) {
  const days = item.gapOpenDays ?? 0;
  return (
    <li className="cc-hole" data-ink="dotted" data-metric-id={item.id}>
      <span className="cc-hole-label">{item.label}</span>
      <span className="cc-hole-meta">{unregistered ? "no registry row" : item.owner ?? "owner unknown"} · since {item.firstObservedMissing ?? "unknown"} · open {days} {days === 1 ? "day" : "days"}</span>
      {item.whatIsNeeded ? <span className="cc-hole-need">{item.whatIsNeeded}</span> : null}
      <AgeBar days={days} />
    </li>
  );
}

function AgeBar({ days }: { days: number }) {
  return <span className="cc-agebar" aria-hidden="true"><span style={{ width: `${Math.min(100, (days / MAX_AGE_DAYS) * 100).toFixed(1)}%` }} /></span>;
}
