import type { SkyState } from "../lib/sky";
import { clipOutsideQuiet, drawGlow, freeBand, mulberry32, rgba, seedFrom, type Frame, type Rect, type Scene, type SceneGroup } from "./engine";

/**
 * Panorama: a star atlas of the whole board. Each room is a constellation and
 * each of its signals a star, drawn in the provenance material of where it
 * stands, so the sky fills in as sensors are built. Positions are seeded by
 * room and metric id: a signal going live changes weight, never place.
 */

interface Point { x: number; y: number }

export interface AtlasFigure {
  id: string;
  title: string;
  stars: Point[];
  edges: Array<[number, number]>;
  /** The widest a label may run before it meets the next constellation's. */
  labelWidth: number;
}

interface Dust { angle: number; distance: number; size: number; alpha: number; phase: number }

/** The sky field turns once in about half an hour. */
const SKY_TURN = 0.0035;
/** Stars and rings are drawn on a slightly flattened sky, as if seen at an angle. */
const TILT = 0.82;

/** The free region the atlas may occupy: beside a landscape hero, or between hero and readings in portrait; never under the room header. */
export function skyField(frame: Pick<Frame, "w" | "h" | "quiet">): Rect {
  const { w, h, quiet } = frame;
  const pad = Math.min(w, h) * 0.04;
  const narrow = quiet.filter((rect) => rect.w < w * 0.7);
  let x = pad;
  let band = freeBand(h, quiet);
  if (narrow.length > 0) {
    const rightEdge = Math.max(...narrow.map((rect) => rect.x + rect.w));
    if (w - rightEdge >= w * 0.34) {
      x = rightEdge + pad;
      band = freeBand(h, quiet.filter((rect) => rect.w >= w * 0.7));
    }
  }
  const top = Math.max(band.top + pad, h * 0.12);
  const bottom = band.bottom - pad;
  return { x, y: top, w: Math.max(1, w - pad - x), h: Math.max(1, bottom - top) };
}

/** Lays the constellations out on a grid that fits the field, centring a short last row. */
export function layoutAtlas(groups: SceneGroup[], field: Rect, labelHeight: number): AtlasFigure[] {
  const count = groups.length;
  if (count === 0) return [];
  const cols = Math.max(1, Math.min(count, Math.round(Math.sqrt((count * field.w) / Math.max(1, field.h)))));
  const rows = Math.ceil(count / cols);
  const cellW = field.w / cols;
  const cellH = field.h / rows;
  return groups.map((group, index) => {
    const row = Math.floor(index / cols);
    const inRow = row === rows - 1 ? count - row * cols : cols;
    const col = index - row * cols;
    const rng = mulberry32(seedFrom(group.id));
    const radius = Math.max(8, Math.min(cellW * 0.38, (cellH - labelHeight) * 0.44));
    const cx = field.x + ((cols - inRow) * cellW) / 2 + cellW * (col + 0.5) + (rng() - 0.5) * cellW * 0.12;
    const cy = field.y + cellH * row + (cellH - labelHeight) / 2 + (rng() - 0.5) * cellH * 0.06;
    const stars = placeStars(group, cx, cy, radius);
    return { id: group.id, title: group.title, stars, edges: spanningEdges(stars), labelWidth: cellW * 0.92 };
  });
}

/** Seeds each star by its own id so adding a signal leaves the others where they were. */
function placeStars(group: SceneGroup, cx: number, cy: number, radius: number): Point[] {
  const placed: Point[] = [];
  const minGap = radius * Math.min(0.42, 1.5 / Math.sqrt(Math.max(1, group.stars.length)));
  for (const star of group.stars) {
    const rng = mulberry32(seedFrom(`${group.id}:${star.id}`));
    let best: Point = { x: cx, y: cy };
    let bestGap = -1;
    for (let attempt = 0; attempt < 16; attempt += 1) {
      const angle = rng() * Math.PI * 2;
      const distance = radius * (0.2 + 0.8 * Math.sqrt(rng()));
      const candidate = { x: cx + Math.cos(angle) * distance, y: cy + Math.sin(angle) * distance * TILT };
      const gap = placed.reduce((nearest, point) => Math.min(nearest, Math.hypot(point.x - candidate.x, point.y - candidate.y)), Number.POSITIVE_INFINITY);
      if (gap > bestGap) {
        best = candidate;
        bestGap = gap;
      }
      if (gap >= minGap) break;
    }
    placed.push(best);
  }
  return placed;
}

/** A minimum spanning tree: the shortest lines that join every star, the way a constellation is drawn. */
export function spanningEdges(points: Point[]): Array<[number, number]> {
  const edges: Array<[number, number]> = [];
  const origin = points[0];
  if (!origin || points.length < 2) return edges;
  const joined = points.map((_, index) => index === 0);
  const nearest = points.map((point) => Math.hypot(point.x - origin.x, point.y - origin.y));
  const from = points.map(() => 0);
  for (let step = 1; step < points.length; step += 1) {
    let next = -1;
    for (let i = 0; i < points.length; i += 1) {
      if (!joined[i] && (next < 0 || (nearest[i] ?? Infinity) < (nearest[next] ?? Infinity))) next = i;
    }
    const joinedPoint = points[next];
    if (next < 0 || !joinedPoint) break;
    joined[next] = true;
    edges.push([from[next] ?? 0, next]);
    for (let i = 0; i < points.length; i += 1) {
      const point = points[i];
      if (joined[i] || !point) continue;
      const distance = Math.hypot(point.x - joinedPoint.x, point.y - joinedPoint.y);
      if (distance < (nearest[i] ?? Infinity)) {
        nearest[i] = distance;
        from[i] = next;
      }
    }
  }
  return edges;
}

const INK_STATE: Record<string, SkyState> = { solid: "measured", dimmed: "measured", unavailable: "failing", hollow: "in-reach" };

/** Without constellations the room's own readings form a single figure, so the scene is never empty. */
function groupsFor(frame: Frame): SceneGroup[] {
  const { data } = frame;
  if (data.groups) return data.groups;
  if (data.order.length === 0) return [];
  return [{ id: "room", title: "", stars: data.order.map((id) => ({ id, state: INK_STATE[data.readings[id]?.ink ?? ""] ?? "missing", cached: data.readings[id]?.ink === "dimmed" })) }];
}

const lit = (state: SkyState) => state === "measured";

function drawStar(frame: Frame, x: number, y: number, size: number, star: SceneGroup["stars"][number], twinkle: number): void {
  const { ctx, palette } = frame;
  if (lit(star.state)) {
    ctx.globalCompositeOperation = "lighter";
    drawGlow(frame, x, y, size * (star.cached ? 3.2 : 5.5), palette.primary, star.cached ? 0.2 : 0.4 + 0.12 * twinkle);
    ctx.globalCompositeOperation = "source-over";
    if (!star.cached && frame.tier === "full") {
      ctx.strokeStyle = rgba(ctx, palette.accent, 0.32);
      ctx.lineWidth = 0.8;
      ctx.beginPath();
      ctx.moveTo(x - size * 3.4, y);
      ctx.lineTo(x + size * 3.4, y);
      ctx.moveTo(x, y - size * 3.4);
      ctx.lineTo(x, y + size * 3.4);
      ctx.stroke();
    }
    ctx.fillStyle = rgba(ctx, palette.foreground, star.cached ? 0.5 : 1);
    ctx.beginPath();
    ctx.arc(x, y, star.cached ? size * 0.8 : size, 0, Math.PI * 2);
    ctx.fill();
    return;
  }
  ctx.lineWidth = 1.3;
  if (star.state === "failing") {
    // A struck-through ring: the sensor exists and did not answer, or answered untrustworthily.
    const ring = size * 1.4;
    const strike = ring * Math.SQRT1_2;
    ctx.strokeStyle = rgba(ctx, palette.warning, 0.85);
    ctx.beginPath();
    ctx.arc(x, y, ring, 0, Math.PI * 2);
    ctx.moveTo(x - strike, y + strike);
    ctx.lineTo(x + strike, y - strike);
    ctx.stroke();
    return;
  }
  ctx.strokeStyle = rgba(ctx, palette.gap, star.state === "in-reach" ? 0.85 : 0.7);
  ctx.setLineDash(star.state === "in-reach" ? [] : [1.6, 2.6]);
  ctx.beginPath();
  ctx.arc(x, y, size * 1.4, 0, Math.PI * 2);
  ctx.stroke();
  ctx.setLineDash([]);
}

function buildNebula(frame: Frame): HTMLCanvasElement {
  const { w, h, palette } = frame;
  const scale = 0.5;
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.floor(w * scale));
  canvas.height = Math.max(1, Math.floor(h * scale));
  const nctx = canvas.getContext("2d");
  if (!nctx) return canvas;
  const rng = mulberry32(seedFrom("panorama-nebula"));
  const clouds: Array<[string, number]> = [[palette.primary, 0.11], [palette.gap, 0.06], [palette.accent, 0.05], [palette.primary, 0.07]];
  for (const [color, alpha] of clouds) {
    const cx = canvas.width * (0.45 + rng() * 0.5);
    const cy = canvas.height * (0.15 + rng() * 0.7);
    const radius = Math.max(canvas.width, canvas.height) * (0.3 + rng() * 0.3);
    const gradient = nctx.createRadialGradient(cx, cy, 0, cx, cy, radius);
    gradient.addColorStop(0, rgba(nctx, color, alpha));
    gradient.addColorStop(0.55, rgba(nctx, color, alpha * 0.35));
    gradient.addColorStop(1, rgba(nctx, color, 0));
    nctx.fillStyle = gradient;
    nctx.fillRect(0, 0, canvas.width, canvas.height);
  }
  return canvas;
}

export function panoramaConstellation(): Scene {
  let dust: Dust[] = [];
  let nebula: HTMLCanvasElement | null = null;
  let nebulaKey = "";
  let figures: AtlasFigure[] = [];
  let layoutKey = "";
  return {
    init(frame) {
      const count = frame.tier === "full" ? 520 : 220;
      dust = Array.from({ length: count }, () => ({
        angle: frame.rng() * Math.PI * 2,
        distance: Math.sqrt(frame.rng()),
        size: 0.5 + frame.rng() ** 3 * 1.3,
        alpha: 0.12 + frame.rng() * 0.45,
        phase: frame.rng() * Math.PI * 2,
      }));
      nebulaKey = "";
      layoutKey = "";
    },
    draw(frame) {
      const { ctx, w, h, t, palette } = frame;
      const groups = groupsFor(frame);
      const field = skyField(frame);
      const titleSize = Math.max(11, Math.min(16, w / 100));
      const countSize = Math.max(10, Math.min(13, w / 130));
      const starSize = Math.max(2, Math.min(4.2, Math.min(w, h) / 300));
      const labelHeight = titleSize + countSize + starSize * 3 + 26;
      const key = `${Math.round(field.x / 4)}:${Math.round(field.y / 4)}:${Math.round(field.w / 4)}:${Math.round(field.h / 4)}:${groups.map((group) => `${group.id}=${group.stars.map((star) => star.id).join(",")}`).join("|")}`;
      if (key !== layoutKey) {
        figures = layoutAtlas(groups, field, labelHeight);
        layoutKey = key;
      }
      const wantNebula = `${w}x${h}:${palette.primary}:${palette.gap}:${palette.accent}`;
      if (wantNebula !== nebulaKey) {
        nebula = buildNebula(frame);
        nebulaKey = wantNebula;
      }

      // Ground: a faint nebula, drifting almost imperceptibly.
      if (nebula) {
        const drift = Math.sin(t * 0.01) * w * 0.01;
        ctx.drawImage(nebula, drift, 0, w, h);
      }

      // The turning sky: a celestial grid and dust, kept off the figures.
      const pole = { x: field.x + field.w / 2, y: field.y + field.h / 2 };
      const reach = Math.hypot(w, h);
      const turn = t * SKY_TURN;
      clipOutsideQuiet(frame);
      ctx.strokeStyle = rgba(ctx, palette.primary, 0.07);
      ctx.lineWidth = 0.7;
      for (let ring = 1; ring <= 5; ring += 1) {
        const radius = ring * reach * 0.13;
        ctx.beginPath();
        ctx.ellipse(pole.x, pole.y, radius, radius * TILT, 0, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.strokeStyle = rgba(ctx, palette.primary, 0.045);
      ctx.beginPath();
      for (let meridian = 0; meridian < 12; meridian += 1) {
        const angle = turn + (meridian / 12) * Math.PI * 2;
        ctx.moveTo(pole.x + Math.cos(angle) * reach * 0.05, pole.y + Math.sin(angle) * reach * 0.05 * TILT);
        ctx.lineTo(pole.x + Math.cos(angle) * reach, pole.y + Math.sin(angle) * reach * TILT);
      }
      ctx.stroke();
      for (const mote of dust) {
        const angle = mote.angle + turn;
        const x = pole.x + Math.cos(angle) * mote.distance * reach * 0.75;
        const y = pole.y + Math.sin(angle) * mote.distance * reach * 0.75 * TILT;
        if (x < 0 || y < 0 || x > w || y > h) continue;
        const twinkle = frame.tier === "full" ? 0.7 + 0.3 * Math.sin(t * 0.9 + mote.phase) : 1;
        ctx.fillStyle = rgba(ctx, palette.foreground, mote.alpha * twinkle);
        ctx.fillRect(x, y, mote.size, mote.size);
      }
      ctx.restore();

      // The atlas: one constellation per room, stars in provenance material.
      figures.forEach((figure, figureIndex) => {
        const group = groups[figureIndex];
        if (!group) return;
        const gapFromStar = starSize * 2.2 + 3;
        ctx.lineWidth = 1;
        for (const [a, b] of figure.edges) {
          const from = figure.stars[a];
          const to = figure.stars[b];
          const fromStar = group.stars[a];
          const toStar = group.stars[b];
          if (!from || !to || !fromStar || !toStar) continue;
          const length = Math.hypot(to.x - from.x, to.y - from.y);
          if (length <= gapFromStar * 2) continue;
          const ux = (to.x - from.x) / length;
          const uy = (to.y - from.y) / length;
          // A line is drawn solid only between two measured stars: a constellation completes as its sensors come online.
          const complete = lit(fromStar.state) && lit(toStar.state);
          ctx.strokeStyle = complete ? rgba(ctx, palette.accent, 0.42) : rgba(ctx, palette.gap, 0.24);
          ctx.setLineDash(complete ? [] : [2, 4]);
          ctx.beginPath();
          ctx.moveTo(from.x + ux * gapFromStar, from.y + uy * gapFromStar);
          ctx.lineTo(to.x - ux * gapFromStar, to.y - uy * gapFromStar);
          ctx.stroke();
        }
        ctx.setLineDash([]);
        figure.stars.forEach((point, starIndex) => {
          const star = group.stars[starIndex];
          if (star) drawStar(frame, point.x, point.y, starSize, star, Math.sin(t * 0.6 + starIndex * 1.7 + figureIndex));
        });
        if (!figure.title) return;
        const lowest = figure.stars.reduce((max, point) => Math.max(max, point.y), Number.NEGATIVE_INFINITY);
        const centre = figure.stars.reduce((sum, point) => sum + point.x, 0) / Math.max(1, figure.stars.length);
        const labelY = lowest + starSize * 3 + titleSize + 8;
        const measured = group.stars.filter((star) => lit(star.state)).length;
        ctx.textAlign = "center";
        ctx.textBaseline = "alphabetic";
        const title = figure.title.toUpperCase();
        const setTitleFont = (size: number) => {
          if ("letterSpacing" in ctx) ctx.letterSpacing = `${(size * 0.22).toFixed(1)}px`;
          ctx.font = `600 ${size.toFixed(1)}px Inter, ui-sans-serif, system-ui, sans-serif`;
        };
        setTitleFont(titleSize);
        // Narrow screens pack constellations close; a title shrinks to its cell rather than run into a neighbour's.
        const titleWidth = ctx.measureText(title).width;
        if (titleWidth > figure.labelWidth) setTitleFont(Math.max(7, (titleSize * figure.labelWidth) / titleWidth));
        ctx.fillStyle = rgba(ctx, palette.foreground, 0.9);
        ctx.fillText(title, centre, labelY);
        if ("letterSpacing" in ctx) ctx.letterSpacing = "0px";
        ctx.font = `500 ${countSize.toFixed(1)}px "JetBrains Mono", ui-monospace, monospace`;
        ctx.fillStyle = rgba(ctx, palette.foreground, 0.58);
        ctx.fillText(`${measured.toString()} of ${group.stars.length.toString()} measured`, centre, labelY + countSize + 7);
      });
    },
  };
}
