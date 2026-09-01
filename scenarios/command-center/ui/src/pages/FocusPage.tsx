import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { DashboardLayout } from "../components/DashboardLayout";
import { fetchFocus, type FocusEntry } from "../lib/api";

export default function FocusPage() {
  const [searchParams] = useSearchParams();
  const query = useQuery({ queryKey: ["focus"], queryFn: fetchFocus });
  const forcedState = searchParams.get("state");
  const forcedError = forcedState === "error";
  const entries = forcedState === "empty" ? [] : query.data?.entries ?? [];
  const state: ExperienceSurfaceState = query.isLoading ? "loading" : forcedError || query.error ? "error" : forcedState === "partial" ? "partial" : entries.length ? "ready" : "empty";
  const surfaceState: ExperienceSurfaceState = state === "error" ? "ready" : state;
  return <DashboardLayout themeKey="ground-control" title="Focus">
    <ExperienceSurface surfaceId="ranked-list" data-testid="focus-ranked-list" data-requested-state={forcedState ?? undefined} className="cc-read-surface" state={surfaceState} statusMessage={forcedError || query.error ? "The focus ranking is unavailable." : undefined} aria-labelledby="focus-title">
      <h2 id="focus-title">What is worth building next</h2>
      {forcedError || query.error ? <p role="alert">The ranking could not be computed.</p> : null}
      {state === "partial" ? <p role="status">The ranking is incomplete because one or more declared sources are unavailable.</p> : null}
      {!forcedError && !query.error && !query.isLoading && entries.length === 0 ? <p data-testid="focus-empty">The board found no current findings in its declared denominator.</p> : null}
      <ol>{entries.map((entry) => <FocusItem key={`${entry.kind}:${entry.metricId ?? entry.owner}`} entry={entry} />)}</ol>
    </ExperienceSurface>
    <ExperienceSurface surfaceId="ranking-basis" data-testid="focus-ranking-basis" className="cc-read-surface" state="static" aria-label="Ranking basis"><p>Source integrity outranks coverage breadth; unmeasurable predictions and unavailable sources stay visible.</p></ExperienceSurface>
    <ExperienceSurface surfaceId="source-availability" data-testid="focus-source-availability" data-requested-state={forcedState ?? undefined} className="cc-read-surface" state={state === "partial" ? "partial" : "ready"} aria-label="Source availability"><p>{state === "partial" ? "Some declared sources are unavailable; the ranking is explicitly partial." : "Ranking is computed from the sources declared by the outcome registry."}</p></ExperienceSurface>
  </DashboardLayout>;
}

function FocusItem({ entry }: { entry: FocusEntry }) {
  return <li className="cc-focus-item"><strong>{entry.kind}</strong><span data-owner>{entry.owner || "unknown owner"}</span><p>{entry.reason}</p><small>{entry.rankReason}</small></li>;
}
