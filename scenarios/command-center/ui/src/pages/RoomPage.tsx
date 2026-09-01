import { useQuery } from "@tanstack/react-query";
import { useParams, useSearchParams } from "react-router-dom";
import { useEffect, useRef, useState } from "react";
import { DashboardLayout } from "../components/DashboardLayout";
import { MetricList } from "../components/MetricList";
import { fetchRoom, type RoomResponse } from "../lib/api";
import type { ThemeKey } from "../components/ThemeProvider";
import { useBoardController } from "../components/BoardController";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1.0.3";

const themes: Record<string, ThemeKey> = { "mission-control":"ground-control", hive:"bioluminescent", forge:"foundry", ledger:"vault", broadcast:"signal-tower", panorama:"cosmos" };
function themeFor(value?: string): ThemeKey { return (value && themes[value]) || "ground-control"; }

export default function RoomPage({ roomIdOverride }: { roomIdOverride?: string } = {}) {
  const { roomId: routeRoomId = "mission-control" } = useParams();
  const roomId = roomIdOverride ?? routeRoomId;
  const [searchParams] = useSearchParams();
  const { samples } = useBoardController();
  const { data, isLoading, error } = useQuery<RoomResponse>({ queryKey:["room",roomId,samples], queryFn:()=>fetchRoom(roomId, samples) });
  const room = data?.room ?? { id: roomId, title: roomId };
  const readings = data?.readings ?? [];
  const visibleReadings = samples === "hide" ? readings.filter((reading) => reading.value !== null && reading.value !== undefined) : readings;
  const hero = visibleReadings.find((reading) => reading.value !== null && reading.value !== undefined) ?? visibleReadings[0];
  const measured = visibleReadings.filter((reading) => reading.value !== null && reading.value !== undefined).length;
  const surfaceState = isLoading ? "loading" : error ? "error" : "ready";
  return <DashboardLayout themeKey={themeFor(room.theme)} title={room.title} aside={<ExperienceSurface surfaceId="supporting" data-testid="room-supporting" state={surfaceState} statusMessage={error ? "Unable to load supporting readings." : undefined}><MetricList metrics={visibleReadings} /></ExperienceSurface>}>
    {error ? <div className="cc-error" data-testid="error-banner">Unable to load this room.</div> : null}
    {isLoading ? <div className="cc-loading" data-testid="loading">Loading {room.title}…</div> : <div className={`cc-composition cc-composition-${room.composition ?? "board"}`} data-testid="room-composition" data-composition={room.composition}>
      <ExperienceSurface surfaceId="scene" data-testid="room-scene" state="ready"><AmbientCanvas composition={room.composition ?? "board"} forcedTier={searchParams.get("tier")} /></ExperienceSurface>
      <ExperienceSurface surfaceId="hero" data-testid="room-hero" className="cc-hero-readout" state="ready" data-provenance={hero ? provenanceFor(hero) : "absent"}><div data-reading><span>{measured} measured · {visibleReadings.length} signals</span><strong data-figure>{hero?.value ?? (samples === "hide" ? "No live reading" : hero?.sample?.value ?? "—")}{hero?.unit ? ` ${hero.unit}` : ""}</strong><small data-qualifier>{hero?.label ?? room.composition ?? "derived instrument surface"}</small><ExperienceSurface surfaceId="legend" as="div" data-testid="room-legend" className="cc-legend" state="static">{samples === "mark" && visibleReadings.some((reading) => reading.sample) ? "Hollow and dotted figures are authored samples" : "Figures are qualified by their source and freshness"}</ExperienceSurface></div></ExperienceSurface>
      <ExperienceSurface surfaceId="sources" data-testid="room-sources" className="cc-room-sources" state="ready">{Object.keys(data?.sources ?? {}).length} sources reported</ExperienceSurface>
    </div>}
  </DashboardLayout>;
}

function provenanceFor(reading: { coverage?: string; trust?: string }): "measured" | "cached" | "sample" | "absent" {
  if (reading.coverage === "MISSING" || reading.coverage === "UNREGISTERED") return "absent";
  if (reading.coverage === "IN-REACH") return "sample";
  return reading.trust === "CACHED" ? "cached" : "measured";
}

type SceneTier = "full" | "reduced" | "still";

function sceneTier(forcedTier: string | null): SceneTier {
  if (forcedTier === "still" || forcedTier === "reduced") return forcedTier;
  if (forcedTier === "full") return "full";
  try {
    const probe = document.createElement("canvas");
    return probe.getContext("webgl2") || probe.getContext("webgl") ? "full" : "still";
  } catch {
    return "still";
  }
}

function AmbientCanvas({ composition, forcedTier }: { composition: string; forcedTier: string | null }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [state, setState] = useState<"ready" | "still">("still");
  const [renderMs, setRenderMs] = useState<number | null>(null);
  const tier = sceneTier(forcedTier);
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const startedAt = performance.now();
    const rect = canvas.getBoundingClientRect();
    const ratio = Math.min(2, Math.max(1, window.devicePixelRatio || 1));
    const width = Math.max(1, Math.floor(rect.width * ratio));
    const height = Math.max(1, Math.floor(rect.height * ratio));
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) return;
    context.scale(ratio, ratio);
    const w = rect.width || 640;
    const h = rect.height || 420;
    context.clearRect(0, 0, w, h);
    context.strokeStyle = "rgba(255,255,255,.22)";
    context.fillStyle = "rgba(255,255,255,.12)";
    context.lineWidth = tier === "full" ? 1.5 : 1;
    const seed = [...composition].reduce((sum, char) => sum + char.charCodeAt(0), 0);
    const count = tier === "full" ? 24 : tier === "reduced" ? 12 : 6;
    if (composition === "hive-lattice") drawHive(context, w, h, count);
    else if (composition === "flow-current") drawFlow(context, w, h, count, seed);
    else if (composition === "signal-constellation") drawBroadcast(context, w, h, count);
    else if (composition === "panorama-constellation") drawPanorama(context, w, h, count, seed);
    else if (composition === "ledger-river") drawLedger(context, w, h, count);
    else drawBoard(context, w, h, count, seed);
    const pixel = context.getImageData(Math.floor(w / 2), Math.floor(h / 2), 1, 1).data;
    if (pixel[3] === 0) {
      context.fillStyle = "rgba(255,255,255,.08)";
      context.fillRect(w / 2 - 1, h / 2 - 1, 2, 2);
      setState("still");
    } else {
      setState("ready");
    }
    setRenderMs(Number((performance.now() - startedAt).toFixed(3)));
  }, [composition, tier]);
  return <div className={`cc-scene-canvas cc-scene-tier-${tier}`} data-testid="scene-canvas" data-scene-state={state} data-scene-tier={tier} data-postprocessing={tier === "full" ? "scene-only" : "disabled"} data-render-ms={renderMs ?? undefined} aria-label={`${composition} ambient composition`}><canvas ref={canvasRef} aria-hidden="true" /></div>;
}

function drawHive(context: CanvasRenderingContext2D, w: number, h: number, count: number) {
  const columns = Math.max(3, Math.ceil(Math.sqrt(count * (w / Math.max(h, 1)))));
  for (let i = 0; i < count; i += 1) {
    const x = ((i % columns) + 0.5) * (w / columns);
    const y = (Math.floor(i / columns) + 0.5) * (h / Math.ceil(count / columns));
    context.beginPath();
    for (let side = 0; side < 6; side += 1) {
      const angle = (Math.PI / 3) * side;
      const pointX = x + Math.cos(angle) * Math.min(w, h) * 0.035;
      const pointY = y + Math.sin(angle) * Math.min(w, h) * 0.035;
      side === 0 ? context.moveTo(pointX, pointY) : context.lineTo(pointX, pointY);
    }
    context.closePath(); context.stroke();
  }
}

function drawFlow(context: CanvasRenderingContext2D, w: number, h: number, count: number, seed: number) {
  context.beginPath();
  for (let i = 0; i <= count; i += 1) {
    const x = (i / count) * w;
    const y = h * 0.5 + Math.sin(i * 0.9 + seed) * h * 0.18;
    i === 0 ? context.moveTo(x, y) : context.lineTo(x, y);
  }
  context.stroke();
  for (let i = 0; i < count; i += 1) { const x = ((i + 0.5) / count) * w; context.beginPath(); context.arc(x, h * 0.5 + Math.sin(i * 0.9 + seed) * h * 0.18, 5 + (i % 3) * 2, 0, Math.PI * 2); context.fill(); }
}

function drawBroadcast(context: CanvasRenderingContext2D, w: number, h: number, count: number) {
  const cx = w * 0.5, cy = h * 0.5;
  context.beginPath(); context.arc(cx, cy, 10, 0, Math.PI * 2); context.fill();
  for (let i = 1; i <= count; i += 1) { context.beginPath(); context.arc(cx, cy, i * Math.min(w, h) / (count * 1.8), -Math.PI * 0.85, Math.PI * 0.85); context.stroke(); }
}

function drawPanorama(context: CanvasRenderingContext2D, w: number, h: number, count: number, seed: number) {
  const cx = w * 0.5, cy = h * 0.5, radius = Math.min(w, h) * 0.28;
  for (let i = 0; i < Math.min(6, count); i += 1) { const angle = (i / 6) * Math.PI * 2 + seed; const x = cx + Math.cos(angle) * radius; const y = cy + Math.sin(angle) * radius; context.beginPath(); context.moveTo(cx, cy); context.lineTo(x, y); context.stroke(); context.beginPath(); context.arc(x, y, 9, 0, Math.PI * 2); context.fill(); }
  context.beginPath(); context.arc(cx, cy, 16, 0, Math.PI * 2); context.stroke();
}

function drawLedger(context: CanvasRenderingContext2D, w: number, h: number, count: number) {
  context.setLineDash([5, 7]);
  for (let i = 0; i < count; i += 1) { const y = (i + 1) * h / (count + 1); context.beginPath(); context.moveTo(w * 0.12, y); context.lineTo(w * 0.88, y + Math.sin(i) * 8); context.stroke(); }
  context.setLineDash([]);
}

function drawBoard(context: CanvasRenderingContext2D, w: number, h: number, count: number, seed: number) {
  for (let i = 0; i < count; i += 1) { const x = (seed * (i + 3) * 17) % Math.max(1, w); const y = (seed * (i + 5) * 11) % Math.max(1, h); context.beginPath(); context.arc(x, y, 8 + ((i * 7) % 26), 0, Math.PI * 2); context.stroke(); }
}
