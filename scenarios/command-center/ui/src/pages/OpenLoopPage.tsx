import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { DashboardLayout } from "../components/DashboardLayout";
import { fetchOpenLoop } from "../lib/api";

export default function OpenLoopPage() {
  const [searchParams] = useSearchParams();
  const query = useQuery({ queryKey: ["open-loop"], queryFn: fetchOpenLoop });
  const forcedState = searchParams.get("state");
  const forcedError = forcedState === "error" || forcedState === "objective-set-unavailable";
  const data = forcedState === "empty" ? { missing: [], unregistered: [], self: [] } : query.data;
  const state: ExperienceSurfaceState = query.isLoading ? "loading" : forcedError || query.error ? "error" : forcedState === "partial" ? "partial" : "ready";
  const surfaceState: ExperienceSurfaceState = state === "error" ? "ready" : state;
  return <DashboardLayout themeKey="signal-tower" title="Open loop">
    <ExperienceSurface surfaceId="blind-spots" data-requested-state={forcedState ?? undefined} className="cc-read-surface" state={surfaceState} statusMessage={forcedError || query.error ? "The open-loop report is unavailable." : undefined} aria-labelledby="open-loop-title">
      <h2 id="open-loop-title">What the board cannot see</h2>
      {forcedError || query.error ? <p role="alert">The open-loop report is unavailable.</p> : null}
      {state === "partial" ? <p role="status">This report is partial because one or more source declarations could not be read.</p> : null}
      <p>Missing: {data?.missing.length ?? 0} · Unregistered: {data?.unregistered.length ?? 0}</p>
      <ExperienceSurface surfaceId="missing" as="div" data-testid="open-loop-missing" data-requested-state={forcedState ?? undefined} state={surfaceState}><h3>Missing cells</h3><ul>{data?.missing.map((item) => <li key={`missing-${item.id}`}><strong>{item.label}</strong> — {item.owner ?? "unknown owner"}; open {item.gapOpenDays ?? 0} days</li>)}</ul></ExperienceSurface>
      <ExperienceSurface surfaceId="unregistered" as="div" data-testid="open-loop-unregistered" data-requested-state={forcedState ?? undefined} state={surfaceState}><h3>Unregistered outcomes</h3><ul>{data?.unregistered.map((item) => <li key={`unregistered-${item.id}`}><strong>{item.label}</strong> — unregistered outcome</li>)}</ul></ExperienceSurface>
    </ExperienceSurface>
    <ExperienceSurface surfaceId="self" data-testid="open-loop-self" className="cc-read-surface" state="static" aria-label="Instrument blind spot"><p>The board does not yet read its own render telemetry as an outcome.</p></ExperienceSurface>
    <ExperienceSurface surfaceId="denominator" data-testid="open-loop-denominator" className="cc-read-surface" state={state === "partial" ? "partial" : "ready"} aria-label="Denominator"><p>The declared outcome registry is the searched denominator; confidence remains partial until all sources are readable.</p></ExperienceSurface>
  </DashboardLayout>;
}
