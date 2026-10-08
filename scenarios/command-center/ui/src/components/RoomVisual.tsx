import { useEffect, useMemo, useRef } from "react";
import { ExperienceSurface, type ExperienceSurfaceState } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";
import { resolveReading } from "@vrooli/react-component-library/ProvenanceInk/0.1.2";
import { AmbientCanvas } from "./AmbientCanvas";
import { HeroReadout } from "./HeroReadout";
import { ReadingTile } from "./ReadingTile";
import { PanelReadout } from "./PanelReadout";
import { NextRungReadout } from "./NextRungReadout";
import { ReachMapReadout } from "./ReachMapReadout";
import { LadderTile } from "./LadderTile";
import { PanelTile } from "./PanelTile";
import { SkyReadout } from "./SkyReadout";
import { SupportingStrip } from "./SupportingStrip";
import { FunnelReadout } from "./FunnelReadout";
import { LeaderboardReadout } from "./LeaderboardReadout";
import { PostureReadout } from "./PostureReadout";
import { TileQualifier } from "./TileQualifier";
import { useFitProbe } from "../lib/useFitProbe";
import { roomReadingSeconds } from "../lib/readingTime";
import { readingsForBeat } from "../lib/beat";
import { hasValue, type BoardRoom, type Constellation, type Reading, type SourceMetadata } from "../lib/api";
import type { SamplesMode } from "../lib/boardContext";
import { pickHero } from "../lib/hero";
import { resolveSlotBindings } from "../lib/catalogs";

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

export interface RoomVisualProps {
  roomId: string;
  room: BoardRoom;
  beatIndex: number;
  readings: Reading[];
  samples: SamplesMode;
  constellations?: Constellation[];
  sources?: Record<string, SourceMetadata>;
  isLoading?: boolean;
  error?: unknown;
  forcedTier?: string | null;
  seed?: string;
  dataUpdatedAt?: number;
  onReadingSeconds?: (seconds: number[]) => void;
  logicalViewport?: { width: number; height: number };
}

/** The canonical room content renderer. RoomPage and settings preview both use this. */
export function RoomVisual({ roomId, room, beatIndex, readings, samples, constellations, sources = {}, isLoading = false, error, forcedTier = null, seed = "", dataUpdatedAt = 0, onReadingSeconds, logicalViewport }: RoomVisualProps) {
  const roomRef = useRef<HTMLElement>(null);
  const heroRef = useRef<HTMLDivElement>(null);
  const supportingRef = useRef<HTMLUListElement>(null);
  const beats = useMemo(() => room.beats ?? [], [room.beats]);
  const beat = beats[beatIndex];
  const beatReadings = useMemo(() => readingsForBeat(beat, readings), [beat, readings]);
  const visible = useMemo(() => samples === "hide" ? beatReadings.filter(hasValue) : beatReadings, [beatReadings, samples]);
  const hero = constellations ? null : pickHero(visible, beat?.hero);
  const composition = beat?.composition ?? room.composition ?? "orbital-field";
  const slotBindings = useMemo(() => resolveSlotBindings(composition, visible.map((reading) => reading.id), room.bind), [composition, room.bind, visible]);
  const layout = beat?.layout === "wide" ? "wide" : "standard";
  const stripless = hero?.kind === "ladder";
  const supporting = useMemo(() => stripless ? [] : visible.filter((reading) => reading.id !== hero?.id), [hero, stripless, visible]);
  const supportingTrendIDs = useMemo(() => new Set(supporting.filter((reading) => reading.trend?.state === "meaningful" || reading.trend?.state === "neutral").slice(0, 2).map((reading) => reading.id)), [supporting]);
  const measured = visible.filter(hasValue).length;
  const allIllustrative = visible.length > 0 && measured === 0;
  const origins = new Set(visible.map((reading) => reading.origin_env || "local"));
  const showOrigin = origins.size > 1;
  const heroState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : "ready";
  const supportingState: ExperienceSurfaceState = isLoading ? "loading" : error ? "error" : supporting.length === 0 ? "empty" : Object.values(sources).some((meta) => meta.staleness_ts) ? "partial" : "ready";
  const readingSeconds = useMemo(() => constellations || !readings.length ? [] : roomReadingSeconds(beats, visible), [beats, constellations, readings.length, visible]);

  useFitProbe(roomRef, `${roomId}:${beatIndex}:${samples}`, String(dataUpdatedAt));
  useEffect(() => { if (readingSeconds.length) onReadingSeconds?.(readingSeconds); }, [onReadingSeconds, readingSeconds]);

  return <main ref={roomRef} className={`cc-room cc-room-${composition}`} data-testid="room-composition" data-composition={composition} data-layout={layout} data-room={roomId} data-beat={beatIndex} data-all-illustrative={allIllustrative || undefined}>
    <ExperienceSurface surfaceId="scene" as="div" data-testid="room-scene" className="cc-scene-layer" state={isLoading ? "loading" : "ready"}>
      {!isLoading ? <AmbientCanvas composition={composition} readings={visible} focus={hero?.id} forcedTier={forcedTier} quietRefs={[heroRef, supportingRef]} seed={`${roomId}:${beatIndex}:${seed}`} constellations={constellations} slotBindings={slotBindings} logicalViewport={logicalViewport} /> : null}
    </ExperienceSurface>
    <div key={`${roomId}:${beatIndex}`} className="cc-figure-layer">
      {error ? <p className="cc-degraded" role="status" data-testid="error-banner">The room could not be read. Showing nothing rather than a stale composition.</p> : null}
      {allIllustrative ? <p className="cc-room-stamp" data-testid="room-all-illustrative">Entire room illustrative · nothing here has been measured</p> : null}
      <ExperienceSurface surfaceId="hero" as="section" data-testid="room-hero" className="cc-hero-region" state={heroState} statusMessage={error ? "Unable to read this room." : undefined} data-provenance={hero ? resolveReading(hero).figure : constellations ? "measured" : "none"}>
        {isLoading ? <div className="cc-loading" data-testid="loading"><span className="cc-loading-figure" aria-hidden="true">––</span><span>Reading {room.title}…</span></div> : constellations ? <SkyReadout ref={heroRef} constellations={constellations} /> : hero?.kind === "ladder" ? (layout === "wide" ? <ReachMapReadout ref={heroRef} reading={hero} /> : <NextRungReadout ref={heroRef} reading={hero} />) : hero?.kind === "panel" ? <PanelReadout reading={hero} /> : hero?.kind === "funnel" ? <FunnelReadout reading={hero} /> : hero?.kind === "leaderboard" ? <LeaderboardReadout reading={hero} /> : hero?.kind === "posture" ? <PostureReadout reading={hero} /> : <HeroReadout ref={heroRef} reading={hero} emptyReason={samples === "hide" ? "Illustrative figures are hidden. Nothing in this room is measured yet." : undefined} />}
        {constellations ? null : <span className="cc-hero-count" data-testid="room-measured-count">{measured} measured · {visible.length} {visible.length === 1 ? "signal" : "signals"}</span>}
      </ExperienceSurface>
      <ExperienceSurface surfaceId="supporting" as="section" data-testid="room-supporting" className="cc-supporting-region" state={supportingState} statusMessage={error ? "Unable to load supporting readings." : undefined} aria-label="Supporting readings" data-omitted={stripless || undefined}>
        {supporting.length > 0 ? <SupportingStrip ref={supportingRef}>{supporting.map((reading) => <SupportingTile key={reading.id} reading={reading} showTrend={supportingTrendIDs.has(reading.id)} showOrigin={showOrigin} />)}</SupportingStrip> : !isLoading && !stripless ? <p className="cc-empty" data-testid="metric-list-empty">No further readings in this room.</p> : null}
      </ExperienceSurface>
    </div>
  </main>;
}
