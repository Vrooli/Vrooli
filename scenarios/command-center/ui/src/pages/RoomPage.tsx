import { useEffect, useMemo, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams, useSearchParams } from "react-router-dom";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { AmbientCanvas } from "../components/AmbientCanvas";
import { AmbientShell } from "../components/AmbientShell";
import { useBoardController } from "../lib/boardContext";
import { HeroReadout } from "../components/HeroReadout";
import { ReadingTile } from "../components/ReadingTile";
import { fetchRoom, hasValue, type RoomResponse } from "../lib/api";
import { nextMeasuredBeat, pickHero } from "../lib/hero";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { PanelReadout } from "../components/PanelReadout";
import { NextRungReadout } from "../components/NextRungReadout";
import { ReachMapReadout } from "../components/ReachMapReadout";
import { LadderTile } from "../components/LadderTile";
import { PanelTile } from "../components/PanelTile";
import { SkyReadout } from "../components/SkyReadout";
import { SupportingStrip } from "../components/SupportingStrip";
import { useFitProbe } from "../lib/useFitProbe";
import { roomReadingSeconds } from "../lib/readingTime";
import type { Reading } from "../lib/api";
import { FunnelReadout } from "../components/FunnelReadout";
import { LeaderboardReadout } from "../components/LeaderboardReadout";
import { PostureReadout } from "../components/PostureReadout";
import { TileQualifier } from "../components/TileQualifier";
import { readingsForBeat } from "../lib/beat";

/** A supporting reading takes the tile its shape calls for: a list reading has no single figure to show. */
function SupportingTile({ reading, showTrend, showOrigin }: { reading: Reading; showTrend: boolean; showOrigin: boolean }) {
	if (reading.kind === "ladder") return <LadderTile reading={reading} showOrigin={showOrigin} />;
	if (reading.kind === "panel") return <PanelTile reading={reading} showOrigin={showOrigin} />;
	if (reading.kind === "posture") {
		const posture = reading.value && typeof reading.value === "object" ? reading.value as { runwayMonths?: number; gap?: string } : {};
		return <li className="cc-reading cc-structured-tile" data-testid="posture-tile"><span className="cc-reading-label">{reading.label}</span><strong>{posture.runwayMonths ?? "—"} mo</strong><span className="cc-qualifier">{posture.gap ?? "posture unavailable"}</span></li>;
	}
  if (reading.kind === "funnel") return <li className="cc-reading cc-structured-tile"><span className="cc-reading-label">{reading.label}</span><strong>{reading.rows?.find((row) => row.key === "paid")?.value.toLocaleString() ?? "––"}</strong><TileQualifier qualifier={{ text: "paid step", tone: "live" }} reading={reading} showOrigin={showOrigin} /></li>;
  if (reading.kind === "leaderboard") { const leader = reading.rows?.find((row) => row.verdict === "LEADING" || row.verdict === "EXPERIMENT_VERDICT_LEADING"); return <li className="cc-reading cc-structured-tile"><span className="cc-reading-label">{reading.label}</span><strong>{leader?.label ?? "No leader"}</strong><TileQualifier qualifier={{ text: leader ? "leading arm" : "no measured verdict", tone: leader ? "live" : "quiet" }} reading={reading} showOrigin={showOrigin} /></li>; }
  return <ReadingTile reading={reading} showTrend={showTrend} showOrigin={showOrigin} />;
}

const THEMES: Record<string, string> = { "mission-control": "ground-control", hive: "bioluminescent", forge: "foundry", ledger: "vault", broadcast: "signal-tower", panorama: "cosmos" };

export default function RoomPage() {
  const { roomId = "mission-control" } = useParams();
  const [searchParams] = useSearchParams();
  const board = useBoardController();
  const heroRef = useRef<HTMLDivElement>(null);
  const supportingRef = useRef<HTMLUListElement>(null);
  const roomRef = useRef<HTMLElement>(null);
  const quietRefs = useMemo(() => [heroRef, supportingRef], []);
  const { data, isLoading, error, isFetching, dataUpdatedAt } = useQuery<RoomResponse>({
    queryKey: ["room", roomId, board.samples],
    queryFn: () => fetchRoom(roomId, board.samples),
    refetchInterval: (query) => {
      const ttls = query.state.data?.readings.map((reading) => reading.ttlSeconds).filter((ttl) => ttl > 0) ?? [];
      return ttls.length ? Math.max(5, Math.min(...ttls)) * 1000 : 30_000;
    },
  });
  const room = board.rooms.find((entry) => entry.id === roomId) ?? data?.room ?? { id: roomId, title: roomId.replace(/-/g, " "), theme: THEMES[roomId], composition: "orbital-field" };
  const beats = useMemo(() => room.beats ?? [], [room.beats]);
  const beatIndex = board.beatIndex;
  const beat = beats[beatIndex];
  const theme = room.theme ?? THEMES[roomId] ?? "ground-control";
  const readings = useMemo(() => data?.readings ?? [], [data]);
  const beatReadings = useMemo(() => readingsForBeat(beat, readings), [beat, readings]);
  const visible = useMemo(() => (board.samples === "hide" ? beatReadings.filter(hasValue) : beatReadings), [board.samples, beatReadings]);
  const constellations = data?.constellations;
  // A panorama's hero counts the whole board, so every one of its own readings (the rooms' headlines) supports.
  const hero = constellations ? null : pickHero(visible, beat?.hero);
  const composition = beat?.composition ?? room.composition ?? "orbital-field";
  const layout = beat?.layout === "wide" ? "wide" : "standard";
  useEffect(() => {
    if (board.samples !== "hide" || !beats.length || !beat || visible.length === 0) return;
    const measuredIDs = new Set(visible.map((reading) => reading.id));
    if (beat.hero && measuredIDs.has(beat.hero)) return;
    const next = nextMeasuredBeat(beats, measuredIDs, beatIndex);
    if (next >= 0 && next !== beatIndex) board.selectBeat(next);
  }, [beat, beats, board, board.samples, visible, beatIndex]);
  // The release ladder is the whole page: a ladder beat carries no supporting strip (UI-ARCHITECTURE §"Beats, layouts and list readings").
  const stripless = hero?.kind === "ladder";
  const supporting = useMemo(() => (stripless ? [] : visible.filter((reading) => reading.id !== hero?.id)), [visible, hero, stripless]);
  const supportingTrendIDs = useMemo(() => new Set(supporting.filter((reading) => reading.trend?.state === "meaningful" || reading.trend?.state === "neutral").slice(0, 2).map((reading) => reading.id)), [supporting]);
  // Measured means what the resolver draws as measured: an in-reach reading that returns a number still shows its illustration.
  const measured = visible.filter(hasValue).length;
  const allIllustrative = visible.length > 0 && measured === 0;
  const hasSamples = visible.some((reading) => resolveReading(reading).figure === "sample");
  const index = board.rooms.findIndex((entry) => entry.id === roomId);
  const position = index >= 0 ? `ROOM ${index + 1} OF ${board.rooms.length}` : "ROOM";
  const sources = Object.entries(data?.sources ?? {});
  const heroState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : "ready";
  const supportingState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : supporting.length === 0 ? "empty" : sources.some(([, meta]) => meta.staleness_ts) ? "partial" : "ready";
  const sourceState: ExperienceSurfaceState = sources.some(([, meta]) => meta.staleness_ts) ? "partial" : "ready";
  // Origin is shown only where it differs: per tile in a mixed room, once in the source strip when every reading shares a non-local origin.
  const origins = new Set(visible.map((reading) => reading.origin_env || "local"));
  const showOrigin = origins.size > 1;
  const sharedOrigin = origins.size === 1 && !origins.has("local") ? visible[0]?.origin_display || visible[0]?.origin_env : null;
  useFitProbe(roomRef, `${roomId}:${beatIndex}:${board.samples}`, String(dataUpdatedAt));
  // A beat lasts as long as it takes to read (UI-ARCHITECTURE §"Beats, layouts and list readings").
  const readingSeconds = useMemo(() => (constellations || !data ? [] : roomReadingSeconds(beats, visible)), [beats, constellations, data, visible]);
  const { reportReadingSeconds } = board;
  useEffect(() => {
    if (readingSeconds.length) reportReadingSeconds(roomId, readingSeconds);
  }, [readingSeconds, reportReadingSeconds, roomId]);

  return (
    <AmbientShell
      theme={theme}
      title={room.title}
      position={position}
      legend={hasSamples}
      status={
        <ExperienceSurface surfaceId="sources" as="div" data-testid="room-sources" className="cc-sources" state={sourceState} aria-label="Source availability">
          {sources.length === 0 ? <span className="cc-source" data-answering="none">{isLoading ? "reading sources" : "no source read"}</span> : null}
          {sources.map(([name, meta]) => {
            const status = meta.staleness_ts ? "stale" : meta.integration_status ?? "available";
            const detail = meta.staleness_ts ? `last fetch failed ${meta.staleness_ts}` : meta.integration_reason_code ?? status;
            // The dot carries the status; a source that did not answer still states why, in the source strip.
            return (
              <span key={name} className="cc-source" data-answering={status} title={`${meta.integration_id ?? name} · ${detail}`}>
                <span className="cc-source-dot" aria-hidden="true" />{name}
                <span className={status === "available" ? "cc-visually-hidden" : undefined}>: {status === "available" ? status : detail}</span>
              </span>
            );
          })}
          {sharedOrigin ? <span className="cc-source" data-origin data-testid="room-origin">{sharedOrigin}</span> : null}
          {isFetching ? <span className="cc-source cc-source-fetching" aria-label="refreshing">·</span> : null}
        </ExperienceSurface>
      }
    >
      <main ref={roomRef} className={`cc-room cc-room-${composition}`} data-testid="room-composition" data-composition={composition} data-layout={layout} data-room={roomId} data-beat={beatIndex} data-all-illustrative={allIllustrative || undefined}>
        <ExperienceSurface surfaceId="scene" as="div" data-testid="room-scene" className="cc-scene-layer" state={isLoading ? "loading" : "ready"}>
          {!isLoading ? <AmbientCanvas composition={composition} readings={visible} focus={hero?.id} forcedTier={searchParams.get("tier")} quietRefs={quietRefs} seed={`${roomId}:${beatIndex}:${searchParams.get("seed") ?? ""}`} constellations={constellations} /> : null}
        </ExperienceSurface>
        <div key={`${roomId}:${beatIndex}`} className="cc-figure-layer">
          {error ? <p className="cc-degraded" role="status" data-testid="error-banner">The room could not be read. Showing nothing rather than a stale composition.</p> : null}
          {allIllustrative ? <p className="cc-room-stamp" data-testid="room-all-illustrative">Entire room illustrative · nothing here has been measured</p> : null}
          <ExperienceSurface surfaceId="hero" as="section" data-testid="room-hero" className="cc-hero-region" state={heroState} statusMessage={error ? "Unable to read this room." : undefined} data-provenance={hero ? resolveReading(hero).figure : constellations ? "measured" : "none"}>
            {isLoading ? <div className="cc-loading" data-testid="loading"><span className="cc-loading-figure" aria-hidden="true">––</span><span>Reading {room.title}…</span></div> : constellations ? <SkyReadout ref={heroRef} constellations={constellations} /> : hero?.kind === "ladder" ? (layout === "wide" ? <ReachMapReadout ref={heroRef} reading={hero} /> : <NextRungReadout ref={heroRef} reading={hero} />) : hero?.kind === "panel" ? <PanelReadout reading={hero} /> : hero?.kind === "funnel" ? <FunnelReadout reading={hero} /> : hero?.kind === "leaderboard" ? <LeaderboardReadout reading={hero} /> : hero?.kind === "posture" ? <PostureReadout reading={hero} /> : <HeroReadout ref={heroRef} reading={hero} emptyReason={board.samples === "hide" ? "Illustrative figures are hidden. Nothing in this room is measured yet." : undefined} />}
            {constellations ? null : <span className="cc-hero-count" data-testid="room-measured-count">{measured} measured · {visible.length} {visible.length === 1 ? "signal" : "signals"}</span>}
          </ExperienceSurface>
          <ExperienceSurface surfaceId="supporting" as="section" data-testid="room-supporting" className="cc-supporting-region" state={supportingState} statusMessage={error ? "Unable to load supporting readings." : undefined} aria-label="Supporting readings" data-omitted={stripless || undefined}>
            {supporting.length > 0 ? <SupportingStrip ref={supportingRef}>{supporting.map((reading) => <SupportingTile key={reading.id} reading={reading} showTrend={supportingTrendIDs.has(reading.id)} showOrigin={showOrigin} />)}</SupportingStrip> : !isLoading && !stripless ? <p className="cc-empty" data-testid="metric-list-empty">No further readings in this room.</p> : null}
          </ExperienceSurface>
        </div>
      </main>
    </AmbientShell>
  );
}
