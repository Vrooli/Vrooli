import type { Reading } from "./api";
import { figureValue, resolveReading, type Ink } from "@vrooli/react-component-library/ProvenanceInk/0";

export type SceneTier = "full" | "reduced";

export interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface Palette {
  primary: string;
  accent: string;
  foreground: string;
  glow: string;
  gap: string;
  warning: string;
  background: string;
}

/** One reading, reduced to what a scene needs: a number and its honesty. */
export interface SceneReading {
	value: number | null;
	ink: Ink;
	rows?: Array<{ share: number; value: number }>;
}

export interface SceneData {
  /** Keyed by metric id. */
  readings: Record<string, SceneReading>;
  /** Room metric ids in registry order. */
  order: string[];
  /** Metric currently featured by the room beat, when one is authored. */
  focus?: string;
}

export interface Frame {
  ctx: CanvasRenderingContext2D;
  w: number;
  h: number;
  /** Seconds since the scene mounted. */
  t: number;
  dt: number;
  /** Regions the scene must keep bright bodies out of: the figure layer. Canvas coordinates. */
  quiet: Rect[];
  tier: SceneTier;
  palette: Palette;
  data: SceneData;
  rng: () => number;
}

export interface Scene {
  /** Called once with the first frame; build particle pools here. */
  init(frame: Frame): void;
  draw(frame: Frame): void;
}

export const sceneData = (readings: Reading[], focus?: string): SceneData => ({
  readings: Object.fromEntries(
    readings.map((reading) => {
      const resolution = resolveReading(reading);
		return [reading.id, { value: figureValue(reading, resolution), ink: resolution.ink, rows: reading.rows?.map((row) => ({ share: row.share, value: row.value })) }];
    }),
  ),
  order: readings.map((reading) => reading.id),
  focus,
});

export const read = (data: SceneData, id: string, fallback: number): number => {
  const entry = data.readings[id];
  return entry && entry.value !== null ? entry.value : fallback;
};

export function mulberry32(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export const seedFrom = (text: string): number => Array.from(text).reduce((sum, char) => (sum * 31 + char.charCodeAt(0)) >>> 0, 7);

const colorCache = new Map<string, [number, number, number]>();

/** Normalises any CSS colour to rgb triplets using the canvas parser. */
export function rgb(ctx: CanvasRenderingContext2D, color: string): [number, number, number] {
  const cached = colorCache.get(color);
  if (cached) return cached;
  ctx.fillStyle = "#000";
  ctx.fillStyle = color;
  const normalised = typeof ctx.fillStyle === "string" ? ctx.fillStyle : "";
  let out: [number, number, number] = [255, 255, 255];
  if (normalised.startsWith("#") && normalised.length >= 7) {
    out = [parseInt(normalised.slice(1, 3), 16), parseInt(normalised.slice(3, 5), 16), parseInt(normalised.slice(5, 7), 16)];
  } else {
    const match = /rgba?\((\d+),\s*(\d+),\s*(\d+)/.exec(normalised);
    if (match) out = [Number(match[1]), Number(match[2]), Number(match[3])];
  }
  colorCache.set(color, out);
  return out;
}

export const rgba = (ctx: CanvasRenderingContext2D, color: string, alpha: number): string => {
  const [r, g, b] = rgb(ctx, color);
  return `rgba(${r},${g},${b},${Math.max(0, Math.min(1, alpha))})`;
};

const spriteCache = new Map<string, HTMLCanvasElement>();

/** A soft radial glow, pre-rendered once per colour. Drawn with "lighter" it reads as emission. */
export function glowSprite(ctx: CanvasRenderingContext2D, color: string, size = 64): HTMLCanvasElement {
  const key = `${color}:${size}`;
  const cached = spriteCache.get(key);
  if (cached) return cached;
  const sprite = document.createElement("canvas");
  sprite.width = size;
  sprite.height = size;
  const sctx = sprite.getContext("2d");
  if (sctx) {
    const gradient = sctx.createRadialGradient(size / 2, size / 2, 0, size / 2, size / 2, size / 2);
    gradient.addColorStop(0, rgba(ctx, color, 1));
    gradient.addColorStop(0.25, rgba(ctx, color, 0.55));
    gradient.addColorStop(0.6, rgba(ctx, color, 0.12));
    gradient.addColorStop(1, rgba(ctx, color, 0));
    sctx.fillStyle = gradient;
    sctx.fillRect(0, 0, size, size);
  }
  spriteCache.set(key, sprite);
  return sprite;
}

export function drawGlow(frame: Frame, x: number, y: number, radius: number, color: string, alpha = 1): void {
  const { ctx } = frame;
  ctx.globalAlpha = alpha;
  ctx.drawImage(glowSprite(ctx, color), x - radius, y - radius, radius * 2, radius * 2);
  ctx.globalAlpha = 1;
}

export const inQuiet = (quiet: Rect[], x: number, y: number, pad = 0): boolean =>
  quiet.some((rect) => x >= rect.x - pad && x <= rect.x + rect.w + pad && y >= rect.y - pad && y <= rect.y + rect.h + pad);

/**
 * Where a composition should sit: the centre of the largest band the quiet
 * zones leave free. Landscape leaves a band to the right of the hero;
 * portrait leaves a band between the hero and the supporting readings.
 */
export function focalPoint(frame: Frame): { x: number; y: number } {
  const { w, h, quiet } = frame;
  if (quiet.length === 0) return { x: w * 0.5, y: h * 0.5 };
  const wide = quiet.filter((rect) => rect.w >= w * 0.7);
  const narrow = quiet.filter((rect) => rect.w < w * 0.7);
  const band = freeBand(h, wide);
  if (narrow.length > 0) {
    const rightEdge = Math.max(...narrow.map((rect) => rect.x + rect.w));
    const rightBand = w - rightEdge;
    if (rightBand >= w * 0.34) return { x: rightEdge + rightBand * 0.5, y: band.y };
  }
  return { x: w * 0.5, y: freeBand(h, quiet).y };
}

/** The tallest horizontal band the given rects leave free, as its centre and height. */
export function freeBand(h: number, rects: Rect[]): { y: number; size: number; top: number; bottom: number } {
  const edges = rects.map((rect) => [rect.y, rect.y + rect.h] as const).sort((a, b) => a[0] - b[0]);
  let best = { y: h * 0.5, size: 0, top: 0, bottom: h };
  let cursor = 0;
  for (const [top, bottom] of edges) {
    if (top - cursor > best.size) best = { y: cursor + (top - cursor) / 2, size: top - cursor, top: cursor, bottom: top };
    cursor = Math.max(cursor, bottom);
  }
  if (h - cursor > best.size) best = { y: cursor + (h - cursor) / 2, size: h - cursor, top: cursor, bottom: h };
  return best;
}

/**
 * Clips subsequent drawing to everything outside the quiet zones. Strokes that
 * sweep the whole canvas (rings, beams, rules) use this so they never cross a
 * figure; restore the context when done.
 */
export function clipOutsideQuiet(frame: Frame): void {
  const { ctx, w, h, quiet } = frame;
  ctx.save();
  ctx.beginPath();
  ctx.rect(0, 0, w, h);
  for (const rect of quiet) ctx.rect(rect.x - 8, rect.y - 8, rect.w + 16, rect.h + 16);
  ctx.clip("evenodd");
}

export const ease = (v: number): number => (v < 0 ? 0 : v > 1 ? 1 : v * v * (3 - 2 * v));