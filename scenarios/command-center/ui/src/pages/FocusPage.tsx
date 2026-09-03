import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { AmbientShell } from "../components/AmbientShell";
import { InkMark } from "@vrooli/react-component-library/ProvenanceInk/0.1.1";
import { useBoardController } from "../lib/boardContext";
import { fetchFocus, type FocusEntry, type FocusKind } from "../lib/api";

const KIND_LABEL: Record<FocusKind, { label: string; ink: "unavailable" | "solid" | "hollow" | "dotted"; what: string }> = {
  "untrusted-reading": { label: "untrusted reading", ink: "solid", what: "a number arrived and cannot be believed" },
  "source-unavailable": { label: "source not answering", ink: "unavailable", what: "a sensor exists and is silent" },
  "no-pipeline": { label: "no pipeline", ink: "hollow", what: "the substrate exists; one piece of plumbing is missing" },
  "no-instrument": { label: "no instrument", ink: "dotted", what: "the team has no control loop at all" },
  "unregistered-outcome": { label: "unregistered outcome", ink: "dotted", what: "the objective names it; the board has no row" },
};

/** One ranked surface answering what to build next. Surfaces and ranks; never decides. */
export default function FocusPage() {
  const [searchParams] = useSearchParams();
  const board = useBoardController();
  const query = useQuery({ queryKey: ["focus"], queryFn: fetchFocus, refetchInterval: 60_000 });
  const forcedState = searchParams.get("state");
  const forcedError = forcedState === "error";
  const entries = forcedState === "empty" ? [] : query.data?.entries ?? [];
  const failed = forcedError || Boolean(query.error);
  const partial = forcedState === "partial" || (board.board?.sources.some((source) => !source.readable) ?? false);
  const state: ExperienceSurfaceState = query.isLoading ? "loading" : failed ? "error" : partial ? "partial" : entries.length ? "ready" : "empty";
  const surfaceState: ExperienceSurfaceState = state === "error" ? "ready" : state;
  const owners = [...new Set(entries.map((entry) => entry.owner))];
  return (
    <AmbientShell theme="ground-control" title="Focus" position="RANKED SURFACE">
      <main className="cc-read" data-testid="focus-page">
        <ExperienceSurface surfaceId="ranked-list" as="section" data-testid="focus-ranked-list" data-requested-state={forcedState ?? undefined} className="cc-read-main" state={surfaceState} statusMessage={failed ? "The focus ranking is unavailable." : undefined} aria-labelledby="focus-title">
          <h2 id="focus-title" className="cc-read-title">What is worth building next</h2>
          <p className="cc-read-lede">{entries.length} findings across {owners.length} {owners.length === 1 ? "owner" : "owners"}. Sensor integrity ranks above coverage breadth.</p>
          {failed ? <p role="alert" className="cc-degraded">The ranking could not be computed.</p> : null}
          {state === "partial" ? <p role="status" className="cc-degraded cc-degraded-quiet">The ranking is partial: at least one declared source could not be read.</p> : null}
          {!failed && !query.isLoading && entries.length === 0 ? <p data-testid="focus-empty" className="cc-empty">The board found no current findings inside its declared denominator.</p> : null}
          <ol className="cc-rank">{entries.map((entry, i) => <FocusItem key={`${entry.kind}:${entry.metricId ?? entry.owner}`} entry={entry} rank={i + 1} />)}</ol>
        </ExperienceSurface>
        <aside className="cc-read-aside">
          <ExperienceSurface surfaceId="ranking-basis" as="section" data-testid="focus-ranking-basis" className="cc-panel" state="static" aria-label="Ranking basis">
            <h3>Ranking basis</h3>
            <ol className="cc-basis">
              <li><InkMark ink="unavailable">1</InkMark> sources that exist and are not answering</li>
              <li><InkMark ink="hollow">2</InkMark> pipelines missing on a live instrument</li>
              <li><InkMark ink="dotted">3</InkMark> teams with no instrument at all</li>
              <li><InkMark ink="dotted">4</InkMark> outcomes the objective set names and no row covers</li>
            </ol>
            <p className="cc-panel-note">A finding is one owner and one kind. Six missing pipelines on a team with no instrument are one finding, not six.</p>
          </ExperienceSurface>
          <ExperienceSurface surfaceId="source-availability" as="section" data-testid="focus-source-availability" data-requested-state={forcedState ?? undefined} className="cc-panel" state={partial ? "partial" : "ready"} aria-label="Source availability">
            <h3>Declared sources</h3>
            <ul className="cc-source-list">
              {(board.board?.sources ?? []).map((source) => (
                <li key={`${source.team}:${source.name}`} data-readable={source.readable}>
                  <span className="cc-source-dot" />
                  <span className="cc-source-name">{source.name.replace(/^scenario:/, "")}</span>
                  <span className="cc-source-meta">{source.team} · {source.instrumentStatus}{source.state?.status ? ` · ${source.state.status}` : ""}{source.reason ? ` · ${source.reason}` : ""}</span>
                  {source.state?.featureStatus ? <span className="cc-source-meta">features: {Object.entries(source.state.featureStatus).map(([feature, status]) => `${feature}=${status}`).join(", ")}</span> : null}
                </li>
              ))}
            </ul>
          </ExperienceSurface>
        </aside>
      </main>
    </AmbientShell>
  );
}

function FocusItem({ entry, rank }: { entry: FocusEntry; rank: number }) {
  const kind = KIND_LABEL[entry.kind];
  return (
    <li className="cc-rank-item" data-kind={entry.kind}>
      <span className="cc-rank-number" aria-label={`rank ${rank}`}>{String(rank).padStart(2, "0")}</span>
      <div className="cc-rank-body">
        <div className="cc-rank-head">
          <InkMark ink={kind.ink}>{kind.label}</InkMark>
          <span className="cc-rank-owner" data-owner>{entry.owner || "unknown owner"}</span>
          {entry.metricId ? <span className="cc-rank-metric">{entry.metricId}</span> : null}
        </div>
        <p className="cc-rank-reason">{entry.reason}</p>
        <p className="cc-rank-why">{kind.what} · {entry.rankReason}</p>
      </div>
    </li>
  );
}
