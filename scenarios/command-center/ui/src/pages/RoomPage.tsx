import { useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams, useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { AmbientCanvas } from "../components/AmbientCanvas";
import { AmbientShell } from "../components/AmbientShell";
import { useBoardController } from "../lib/boardContext";
import { HeroReadout } from "../components/HeroReadout";
import { ReadingTile } from "../components/ReadingTile";
import { fetchRoom, hasValue, type RoomResponse } from "../lib/api";
import { pickHero } from "../lib/hero";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { PanelReadout } from "../components/PanelReadout";

const THEMES: Record<string, string> = { "mission-control": "ground-control", hive: "bioluminescent", forge: "foundry", ledger: "vault", broadcast: "signal-tower", panorama: "cosmos" };

export default function RoomPage() {
  const { roomId = "mission-control" } = useParams();
  const [searchParams] = useSearchParams();
  const board = useBoardController();
  const heroRef = useRef<HTMLDivElement>(null);
  const supportingRef = useRef<HTMLUListElement>(null);
  const quietRefs = useMemo(() => [heroRef, supportingRef], []);
  const { data, isLoading, error, isFetching } = useQuery<RoomResponse>({
    queryKey: ["room", roomId, board.samples],
    queryFn: () => fetchRoom(roomId, board.samples),
    refetchInterval: (query) => {
      const ttls = query.state.data?.readings.map((reading) => reading.ttlSeconds).filter((ttl) => ttl > 0) ?? [];
      return ttls.length ? Math.max(5, Math.min(...ttls)) * 1000 : 30_000;
    },
  });
  const room = data?.room ?? board.rooms.find((entry) => entry.id === roomId) ?? { id: roomId, title: roomId.replace(/-/g, " "), theme: THEMES[roomId], composition: "orbital-field" };
  const beats = room.beats ?? [];
  const [beatIndex, setBeatIndex] = useState(0);
  const [beatProgress, setBeatProgress] = useState(0);
  const beatElapsed = useRef(0);
  const beatLastAt = useRef(Date.now());
  const beat = beats.length ? beats[Math.min(beatIndex, beats.length - 1)] : undefined;
  const beatDwell = Math.max(1, (beat?.dwellSeconds ?? (board.cycleSeconds / Math.max(1, beats.length))) * (beats.length ? board.cycleSeconds / 60 : 1));
  useEffect(() => { setBeatIndex(0); setBeatProgress(0); beatElapsed.current = 0; beatLastAt.current = Date.now(); }, [roomId, beats.length]);
  useEffect(() => { setBeatProgress(0); beatElapsed.current = 0; beatLastAt.current = Date.now(); }, [beatIndex]);
  useEffect(() => {
    if (!beats.length) return;
    beatLastAt.current = Date.now();
    const timer = window.setInterval(() => {
      const now = Date.now();
      if (board.paused || document.hidden) { beatLastAt.current = now; return; }
      beatElapsed.current += now - beatLastAt.current;
      beatLastAt.current = now;
      setBeatProgress(Math.min(1, beatElapsed.current / (beatDwell * 1000)));
      if (beatElapsed.current >= beatDwell * 1000) {
        beatElapsed.current = 0;
        setBeatProgress(0);
        setBeatIndex((current) => {
          if (current + 1 >= beats.length) { board.dispatch("page-next"); return 0; }
          return current + 1;
        });
      }
    }, 250);
    return () => window.clearInterval(timer);
  }, [beatDwell, beatIndex, beats, board.paused, board.dispatch]);
  const theme = room.theme ?? THEMES[roomId] ?? "ground-control";
  const readings = useMemo(() => data?.readings ?? [], [data]);
  const visible = useMemo(() => (board.samples === "hide" ? readings.filter(hasValue) : readings), [board.samples, readings]);
  const hero = pickHero(visible, beat?.hero);
  const composition = beat?.composition ?? room.composition ?? "orbital-field";
  useEffect(() => {
    if (board.samples !== "hide" || !beats.length || !beat || visible.length === 0) return;
    if (!visible.some((reading) => reading.id === beat.hero)) setBeatIndex((current) => (current + 1) % beats.length);
  }, [beat, beats, board.samples, visible]);
  const supporting = visible.filter((reading) => reading.id !== hero?.id);
  const measured = visible.filter(hasValue).length;
  const allIllustrative = visible.length > 0 && measured === 0;
  const hasSamples = visible.some((reading) => resolveReading(reading).figure === "sample");
  const index = board.rooms.findIndex((entry) => entry.id === roomId);
  const position = index >= 0 ? `ROOM ${index + 1} OF ${board.rooms.length}` : "ROOM";
  const sources = Object.entries(data?.sources ?? {});
  const heroState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : "ready";
  const supportingState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : supporting.length === 0 ? "empty" : sources.some(([, meta]) => meta.staleness_ts) ? "partial" : "ready";
  const sourceState: ExperienceSurfaceState = sources.some(([, meta]) => meta.staleness_ts) ? "partial" : "ready";

  return (
    <AmbientShell
      theme={theme}
      title={room.title}
      position={position}
      beatIndex={beatIndex}
      beatCount={beats.length}
      beatProgress={beatProgress}
      legend={hasSamples}
      status={
        <ExperienceSurface surfaceId="sources" as="div" data-testid="room-sources" className="cc-sources" state={sourceState} aria-label="Source availability">
          {sources.length === 0 ? <span className="cc-source" data-answering="none">{isLoading ? "reading sources" : "no source read"}</span> : null}
          {sources.map(([name, meta]) => (
            <span key={name} className="cc-source" data-answering={meta.staleness_ts ? "stale" : meta.integration_status ?? "yes"} title={meta.staleness_ts ? `last fetch failed ${meta.staleness_ts}` : `${meta.integration_id ?? name} ${meta.integration_status ?? "unknown"}`}>
              <span className="cc-source-dot" />{name}{meta.integration_status ? ` · ${meta.integration_status}` : ""}
            </span>
          ))}
          {isFetching ? <span className="cc-source cc-source-fetching" aria-label="refreshing">·</span> : null}
        </ExperienceSurface>
      }
    >
      <main className={`cc-room cc-room-${composition}`} data-testid="room-composition" data-composition={composition} data-room={roomId} data-beat={beatIndex} data-all-illustrative={allIllustrative || undefined}>
        <ExperienceSurface surfaceId="scene" as="div" data-testid="room-scene" className="cc-scene-layer" state={isLoading ? "loading" : "ready"}>
          {!isLoading ? <AmbientCanvas composition={composition} readings={visible} focus={hero?.id} forcedTier={searchParams.get("tier")} quietRefs={quietRefs} seed={`${roomId}:${beatIndex}:${searchParams.get("seed") ?? ""}`} /> : null}
        </ExperienceSurface>
        <div className="cc-figure-layer">
          {error ? <p className="cc-degraded" role="status" data-testid="error-banner">The room could not be read. Showing nothing rather than a stale composition.</p> : null}
          {allIllustrative ? <p className="cc-room-stamp" data-testid="room-all-illustrative">Entire room illustrative · nothing here has been measured</p> : null}
          <ExperienceSurface surfaceId="hero" as="section" data-testid="room-hero" className="cc-hero-region" state={heroState} statusMessage={error ? "Unable to read this room." : undefined} data-provenance={hero ? resolveReading(hero).figure : "none"}>
            {isLoading ? <div className="cc-loading" data-testid="loading"><span className="cc-loading-figure" aria-hidden="true">––</span><span>Reading {room.title}…</span></div> : hero?.kind === "panel" ? <PanelReadout reading={hero} /> : <HeroReadout ref={heroRef} reading={hero} emptyReason={board.samples === "hide" ? "Illustrative figures are hidden. Nothing in this room is measured yet." : undefined} />}
            <span className="cc-hero-count" data-testid="room-measured-count">{measured} measured · {visible.length} {visible.length === 1 ? "signal" : "signals"}</span>
          </ExperienceSurface>
          <ExperienceSurface surfaceId="supporting" as="section" data-testid="room-supporting" className="cc-supporting-region" state={supportingState} statusMessage={error ? "Unable to load supporting readings." : undefined} aria-label="Supporting readings">
            {supporting.length > 0 ? <ul ref={supportingRef} className="cc-readings" data-testid="metric-list">{supporting.map((reading) => <ReadingTile key={reading.id} reading={reading} />)}</ul> : !isLoading ? <p className="cc-empty" data-testid="metric-list-empty">No further readings in this room.</p> : null}
          </ExperienceSurface>
        </div>
      </main>
    </AmbientShell>
  );
}
