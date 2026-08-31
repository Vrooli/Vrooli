/**
 * @libraryId react-component-library:IconGeometry
 * @displayName Icon Geometry
 * @version 1.0.0
 * @tags ["foundation","icons","geometry","motion"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource foundations.icon-geometry */

/**
 * Why this exists, and why it does its own path math.
 *
 * The predecessor (`MorphingIcon`'s private `geometry.ts`) parsed path data
 * with a regex that understood only `M`, `L`, `T`, `H`, `V`, and `Z`. Every
 * curve command — `C`, `S`, `Q`, `A` — was silently dropped, and any subpath
 * left with fewer than two points was discarded outright. That is not a
 * cosmetic gap: the registry's own `search` glyph is
 * `"M11 4a7 7 0 1 0 0 14a7 7 0 1 0 0-14M16 16l4 4"`, whose two arcs *are* the
 * magnifying glass. Both were dropped, the surviving one-point subpath was
 * discarded, and the shipped component rendered a bare diagonal line. The
 * eleven-glyph vocabulary was never a design decision — it was the largest set
 * of icons that could be drawn with straight lines.
 *
 * The obvious fix is to let the browser parse: `SVGGeometryElement` exposes
 * `getTotalLength()` and `getPointAtLength()`, which are correct by
 * construction for every command and every shape element. That is not
 * available to us. Story contracts execute under jsdom (see
 * `api/handlers/preview/assets/story-evaluator.js`, run server-side by the
 * preview handler), and jsdom implements neither method. Depending on them
 * would make morph geometry permanently untestable in the harness that governs
 * this library, and would fail closed in SSR.
 *
 * So the split is: **the DOM supplies structure, this module supplies math.**
 * Reading `d`, `cx`, `points`, and friends is plain attribute access that jsdom
 * models faithfully; flattening, resampling, alignment, and interpolation are
 * pure functions here. One implementation, identical results in jsdom and the
 * browser, unit-testable without a renderer.
 */

export interface Point {
  x: number;
  y: number;
}

/**
 * One continuous stroke of an icon, resampled to a fixed point count so two
 * icons can be interpolated pointwise. `opacity` carries ghost subpaths — see
 * `alignGeometry` — which exist so an icon with three strokes can morph into
 * one with two without the extra stroke snapping out of existence.
 */
export interface Subpath {
  points: Point[];
  closed: boolean;
  opacity: number;
}

export interface IconGeometry {
  viewBox: string;
  subpaths: Subpath[];
}

/** A single SVG child element reduced to the parts that describe its shape. */
export interface SvgNode {
  tag: string;
  attrs: Record<string, string>;
}

export interface MorphCompatibility {
  /** 0..1; higher means path interpolation is more likely to read as a morph. */
  score: number;
  /** Subpath counts before ghost padding. */
  fromSubpaths: number;
  toSubpaths: number;
  /** Mean per-point travel after alignment, as a fraction of the viewBox diagonal. */
  travel: number;
  /** Populated when the pair is rejected outright rather than merely scored low. */
  reason?: string;
}

/** Points sampled per subpath. Even powers keep closed-loop rotation cheap. */
export const SAMPLE_COUNT = 64;

/**
 * Curve flattening density, in viewBox units. Icons are conventionally drawn on
 * a 24x24 grid, so a quarter unit is roughly a tenth of a stroke width — finer
 * than the resample that follows, which is what matters.
 */
const FLATTEN_STEP = 0.25;
const MIN_SEGMENTS = 4;
const MAX_SEGMENTS = 64;

const TAU = Math.PI * 2;

// ---------------------------------------------------------------------------
// Number + command scanning
// ---------------------------------------------------------------------------

/**
 * SVG path data is not comma-separated numbers; it is a grammar where `-`
 * doubles as a separator and `.` may chain. `10-5` is two numbers, `.5.5` is
 * two numbers, and `1e-5` is one. A naive split on `/[\s,]+/` gets all three
 * wrong, so scan with a sticky pattern instead.
 */
const NUMBER = /[+-]?(?:\d*\.\d+|\d+\.?)(?:[eE][+-]?\d+)?/y;
const SEPARATOR = /[\s,]+/y;

function scanNumbers(input: string, start: number): { values: number[]; end: number } {
  const values: number[] = [];
  let index = start;
  for (;;) {
    SEPARATOR.lastIndex = index;
    if (SEPARATOR.test(input)) index = SEPARATOR.lastIndex;
    NUMBER.lastIndex = index;
    const match = NUMBER.exec(input);
    if (!match || match[0] === "" || match[0] === "." || match[0] === "+" || match[0] === "-") break;
    values.push(Number(match[0]));
    index = NUMBER.lastIndex;
  }
  return { values, end: index };
}

interface PathCommand {
  code: string;
  args: number[];
}

/** Arity per command; `Z` takes none and `A` takes seven. */
const ARITY: Record<string, number> = {
  M: 2, L: 2, T: 2, H: 1, V: 1, C: 6, S: 4, Q: 4, A: 7, Z: 0,
};

function tokenizePath(d: string): PathCommand[] {
  const commands: PathCommand[] = [];
  let index = 0;
  while (index < d.length) {
    const char = d[index]!;
    if (/[\s,]/.test(char)) {
      index += 1;
      continue;
    }
    if (!/[a-zA-Z]/.test(char)) {
      // A stray number with no command in front of it is malformed; skip it
      // rather than throwing, so one bad glyph degrades to a crossfade instead
      // of taking down the render.
      index += 1;
      continue;
    }
    const code = char;
    const upper = code.toUpperCase();
    const arity = ARITY[upper];
    index += 1;
    if (arity === undefined) continue;
    if (arity === 0) {
      commands.push({ code, args: [] });
      continue;
    }
    const { values, end } = scanNumbers(d, index);
    index = end;
    if (values.length === 0) continue;
    // A command may carry repeated argument groups: `L1 2 3 4` is two lineTos.
    // A repeated `M` group continues as `L`, per the grammar.
    for (let offset = 0; offset + arity <= values.length; offset += arity) {
      const group = values.slice(offset, offset + arity);
      const isRepeat = offset > 0;
      let effective = code;
      if (isRepeat && upper === "M") effective = code === "m" ? "l" : "L";
      commands.push({ code: effective, args: group });
    }
  }
  return commands;
}

// ---------------------------------------------------------------------------
// Curve flattening
// ---------------------------------------------------------------------------

function distance(a: Point, b: Point): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function segmentCount(approximateLength: number): number {
  if (!Number.isFinite(approximateLength) || approximateLength <= 0) return MIN_SEGMENTS;
  return Math.max(MIN_SEGMENTS, Math.min(MAX_SEGMENTS, Math.ceil(approximateLength / FLATTEN_STEP)));
}

function flattenCubic(p0: Point, p1: Point, p2: Point, p3: Point): Point[] {
  // The control polygon bounds the true arc length from above, which is the
  // right side to err on when picking a segment count.
  const polygon = distance(p0, p1) + distance(p1, p2) + distance(p2, p3);
  const steps = segmentCount(polygon);
  const points: Point[] = [];
  for (let step = 1; step <= steps; step += 1) {
    const t = step / steps;
    const u = 1 - t;
    points.push({
      x: u * u * u * p0.x + 3 * u * u * t * p1.x + 3 * u * t * t * p2.x + t * t * t * p3.x,
      y: u * u * u * p0.y + 3 * u * u * t * p1.y + 3 * u * t * t * p2.y + t * t * t * p3.y,
    });
  }
  return points;
}

function quadraticToCubic(p0: Point, q: Point, p2: Point): [Point, Point] {
  return [
    { x: p0.x + (2 / 3) * (q.x - p0.x), y: p0.y + (2 / 3) * (q.y - p0.y) },
    { x: p2.x + (2 / 3) * (q.x - p2.x), y: p2.y + (2 / 3) * (q.y - p2.y) },
  ];
}

/**
 * Endpoint-parameterized elliptical arc, per SVG 1.1 appendix F.6. Returns the
 * flattened points *after* the start point. Degenerate radii collapse to a
 * straight line, which is what the specification requires.
 */
function flattenArc(
  from: Point,
  rxInput: number,
  ryInput: number,
  rotationDegrees: number,
  largeArc: boolean,
  sweep: boolean,
  to: Point,
): Point[] {
  let rx = Math.abs(rxInput);
  let ry = Math.abs(ryInput);
  if (rx === 0 || ry === 0) return [to];
  if (from.x === to.x && from.y === to.y) return [to];

  const phi = (rotationDegrees * Math.PI) / 180;
  const cosPhi = Math.cos(phi);
  const sinPhi = Math.sin(phi);

  const dx2 = (from.x - to.x) / 2;
  const dy2 = (from.y - to.y) / 2;
  const x1p = cosPhi * dx2 + sinPhi * dy2;
  const y1p = -sinPhi * dx2 + cosPhi * dy2;

  // F.6.6: scale the radii up if they are too small to span the chord.
  const lambda = (x1p * x1p) / (rx * rx) + (y1p * y1p) / (ry * ry);
  if (lambda > 1) {
    const scale = Math.sqrt(lambda);
    rx *= scale;
    ry *= scale;
  }

  const sign = largeArc === sweep ? -1 : 1;
  const numerator = rx * rx * ry * ry - rx * rx * y1p * y1p - ry * ry * x1p * x1p;
  const denominator = rx * rx * y1p * y1p + ry * ry * x1p * x1p;
  const coefficient = sign * Math.sqrt(Math.max(0, numerator / denominator));
  const cxp = (coefficient * (rx * y1p)) / ry;
  const cyp = (coefficient * -(ry * x1p)) / rx;

  const cx = cosPhi * cxp - sinPhi * cyp + (from.x + to.x) / 2;
  const cy = sinPhi * cxp + cosPhi * cyp + (from.y + to.y) / 2;

  const angleOf = (x: number, y: number) => Math.atan2((y - cyp) / ry, (x - cxp) / rx);
  const startAngle = angleOf(x1p, y1p);
  const endAngleRaw = angleOf(-x1p, -y1p);
  let sweepAngle = endAngleRaw - startAngle;
  if (!sweep && sweepAngle > 0) sweepAngle -= TAU;
  if (sweep && sweepAngle < 0) sweepAngle += TAU;

  const arcLength = Math.abs(sweepAngle) * Math.max(rx, ry);
  const steps = segmentCount(arcLength);
  const points: Point[] = [];
  for (let step = 1; step <= steps; step += 1) {
    const angle = startAngle + (sweepAngle * step) / steps;
    const x = rx * Math.cos(angle);
    const y = ry * Math.sin(angle);
    points.push({
      x: cosPhi * x - sinPhi * y + cx,
      y: sinPhi * x + cosPhi * y + cy,
    });
  }
  return points;
}

// ---------------------------------------------------------------------------
// Path -> polylines
// ---------------------------------------------------------------------------

interface RawSubpath {
  points: Point[];
  closed: boolean;
}

/**
 * Flatten path data into polylines. Every command in the grammar is handled;
 * unknown commands are skipped rather than silently truncating the subpath.
 */
export function flattenPath(d: string): RawSubpath[] {
  const subpaths: RawSubpath[] = [];
  let active: Point[] = [];
  let closed = false;
  let current: Point = { x: 0, y: 0 };
  let subpathStart: Point = { x: 0, y: 0 };
  // Reflected control points for the smooth variants `S` and `T`.
  let lastCubicControl: Point | null = null;
  let lastQuadraticControl: Point | null = null;

  const flush = () => {
    // A lone `M` with no drawing after it contributes nothing visible, but a
    // *closed* single point is a legitimate dot. Keep anything with geometry.
    if (active.length > 1) subpaths.push({ points: active, closed });
    active = [];
    closed = false;
  };

  for (const { code, args } of tokenizePath(d)) {
    const upper = code.toUpperCase();
    const relative = code !== upper;
    const originX = relative ? current.x : 0;
    const originY = relative ? current.y : 0;

    switch (upper) {
      case "M": {
        flush();
        current = { x: originX + args[0]!, y: originY + args[1]! };
        subpathStart = current;
        active = [current];
        lastCubicControl = null;
        lastQuadraticControl = null;
        break;
      }
      case "L": {
        current = { x: originX + args[0]!, y: originY + args[1]! };
        active.push(current);
        lastCubicControl = null;
        lastQuadraticControl = null;
        break;
      }
      case "H": {
        current = { x: originX + args[0]!, y: current.y };
        active.push(current);
        lastCubicControl = null;
        lastQuadraticControl = null;
        break;
      }
      case "V": {
        current = { x: current.x, y: originY + args[0]! };
        active.push(current);
        lastCubicControl = null;
        lastQuadraticControl = null;
        break;
      }
      case "C": {
        const c1 = { x: originX + args[0]!, y: originY + args[1]! };
        const c2 = { x: originX + args[2]!, y: originY + args[3]! };
        const end = { x: originX + args[4]!, y: originY + args[5]! };
        active.push(...flattenCubic(current, c1, c2, end));
        lastCubicControl = c2;
        lastQuadraticControl = null;
        current = end;
        break;
      }
      case "S": {
        const c1: Point = lastCubicControl
          ? { x: 2 * current.x - lastCubicControl.x, y: 2 * current.y - lastCubicControl.y }
          : current;
        const c2 = { x: originX + args[0]!, y: originY + args[1]! };
        const end = { x: originX + args[2]!, y: originY + args[3]! };
        active.push(...flattenCubic(current, c1, c2, end));
        lastCubicControl = c2;
        lastQuadraticControl = null;
        current = end;
        break;
      }
      case "Q": {
        const q = { x: originX + args[0]!, y: originY + args[1]! };
        const end = { x: originX + args[2]!, y: originY + args[3]! };
        const [c1, c2] = quadraticToCubic(current, q, end);
        active.push(...flattenCubic(current, c1, c2, end));
        lastQuadraticControl = q;
        lastCubicControl = null;
        current = end;
        break;
      }
      case "T": {
        const q: Point = lastQuadraticControl
          ? { x: 2 * current.x - lastQuadraticControl.x, y: 2 * current.y - lastQuadraticControl.y }
          : current;
        const end = { x: originX + args[0]!, y: originY + args[1]! };
        const [c1, c2] = quadraticToCubic(current, q, end);
        active.push(...flattenCubic(current, c1, c2, end));
        lastQuadraticControl = q;
        lastCubicControl = null;
        current = end;
        break;
      }
      case "A": {
        const end = { x: originX + args[5]!, y: originY + args[6]! };
        active.push(
          ...flattenArc(current, args[0]!, args[1]!, args[2]!, args[3]! !== 0, args[4]! !== 0, end),
        );
        lastCubicControl = null;
        lastQuadraticControl = null;
        current = end;
        break;
      }
      case "Z": {
        closed = true;
        flush();
        current = subpathStart;
        // A subsequent drawing command without an `M` resumes from the close
        // point, which starts a fresh subpath.
        active = [current];
        lastCubicControl = null;
        lastQuadraticControl = null;
        break;
      }
      default:
        break;
    }
  }
  flush();
  return subpaths;
}

// ---------------------------------------------------------------------------
// Shape elements -> path data
// ---------------------------------------------------------------------------

function numberAttr(attrs: Record<string, string>, name: string, fallback = 0): number {
  const raw = attrs[name];
  if (raw === undefined) return fallback;
  const value = Number.parseFloat(raw);
  return Number.isFinite(value) ? value : fallback;
}

function pointsAttr(raw: string | undefined): Point[] {
  if (!raw) return [];
  const { values } = scanNumbers(raw, 0);
  const points: Point[] = [];
  for (let index = 0; index + 1 < values.length; index += 2) {
    points.push({ x: values[index]!, y: values[index + 1]! });
  }
  return points;
}

/**
 * Reduce any SVG shape element to path data. Lucide alone reaches for `rect`
 * (rounded, via `rx`/`ry`), `circle`, `line`, and `polyline` — a path-only
 * reader misses a large fraction of real icons, including `SquareTerminal`,
 * whose frame is a `<rect>`.
 */
export function shapeToPathData(node: SvgNode): string | null {
  const { attrs } = node;
  switch (node.tag.toLowerCase()) {
    case "path":
      return attrs.d ?? null;
    case "line": {
      const x1 = numberAttr(attrs, "x1");
      const y1 = numberAttr(attrs, "y1");
      const x2 = numberAttr(attrs, "x2");
      const y2 = numberAttr(attrs, "y2");
      return `M${x1} ${y1}L${x2} ${y2}`;
    }
    case "circle":
    case "ellipse": {
      const cx = numberAttr(attrs, "cx");
      const cy = numberAttr(attrs, "cy");
      const isCircle = node.tag.toLowerCase() === "circle";
      const rx = isCircle ? numberAttr(attrs, "r") : numberAttr(attrs, "rx");
      const ry = isCircle ? numberAttr(attrs, "r") : numberAttr(attrs, "ry");
      if (rx <= 0 || ry <= 0) return null;
      // Two half arcs; a single 360° arc is undefined because its endpoints
      // coincide and the spec discards zero-length arcs.
      return (
        `M${cx - rx} ${cy}` +
        `A${rx} ${ry} 0 1 0 ${cx + rx} ${cy}` +
        `A${rx} ${ry} 0 1 0 ${cx - rx} ${cy}Z`
      );
    }
    case "rect": {
      const x = numberAttr(attrs, "x");
      const y = numberAttr(attrs, "y");
      const width = numberAttr(attrs, "width");
      const height = numberAttr(attrs, "height");
      if (width <= 0 || height <= 0) return null;
      // Per spec `rx` defaults to `ry` and vice versa, then both clamp to half
      // the corresponding side.
      const hasRx = attrs.rx !== undefined;
      const hasRy = attrs.ry !== undefined;
      let rx = hasRx ? numberAttr(attrs, "rx") : hasRy ? numberAttr(attrs, "ry") : 0;
      let ry = hasRy ? numberAttr(attrs, "ry") : hasRx ? numberAttr(attrs, "rx") : 0;
      rx = Math.min(Math.abs(rx), width / 2);
      ry = Math.min(Math.abs(ry), height / 2);
      if (rx === 0 || ry === 0) {
        return `M${x} ${y}H${x + width}V${y + height}H${x}Z`;
      }
      return (
        `M${x + rx} ${y}` +
        `H${x + width - rx}` +
        `A${rx} ${ry} 0 0 1 ${x + width} ${y + ry}` +
        `V${y + height - ry}` +
        `A${rx} ${ry} 0 0 1 ${x + width - rx} ${y + height}` +
        `H${x + rx}` +
        `A${rx} ${ry} 0 0 1 ${x} ${y + height - ry}` +
        `V${y + ry}` +
        `A${rx} ${ry} 0 0 1 ${x + rx} ${y}Z`
      );
    }
    case "polyline":
    case "polygon": {
      const points = pointsAttr(attrs.points);
      if (points.length < 2) return null;
      const head = points[0]!;
      const rest = points.slice(1).map((p) => `L${p.x} ${p.y}`).join("");
      return `M${head.x} ${head.y}${rest}${node.tag.toLowerCase() === "polygon" ? "Z" : ""}`;
    }
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// Resampling
// ---------------------------------------------------------------------------

/**
 * Resample a polyline to exactly `count` points spaced evenly by arc length.
 * Even spacing is what makes pointwise interpolation between two different
 * shapes look like a morph rather than a shuffle: corresponding indices sit at
 * corresponding fractions of the outline.
 */
export function resamplePolyline(points: Point[], closed: boolean, count = SAMPLE_COUNT): Point[] {
  if (points.length === 0) return [];
  if (points.length === 1) return Array.from({ length: count }, () => points[0]!);

  const loop = closed ? [...points, points[0]!] : points;
  const lengths: number[] = [];
  let total = 0;
  for (let index = 0; index + 1 < loop.length; index += 1) {
    const segment = distance(loop[index]!, loop[index + 1]!);
    lengths.push(segment);
    total += segment;
  }
  if (total === 0) return Array.from({ length: count }, () => points[0]!);

  // A closed loop samples over [0, total) so the first and last samples are
  // distinct; an open stroke samples over [0, total] so both ends are hit.
  const span = closed ? count : count - 1;
  const result: Point[] = [];
  let cursor = 0;
  let travelled = 0;
  for (let index = 0; index < count; index += 1) {
    const target = (index / span) * total;
    while (cursor < lengths.length - 1 && travelled + lengths[cursor]! < target) {
      travelled += lengths[cursor]!;
      cursor += 1;
    }
    const segment = lengths[cursor]!;
    const local = segment > 0 ? Math.min(1, Math.max(0, (target - travelled) / segment)) : 0;
    const from = loop[cursor]!;
    const to = loop[cursor + 1]!;
    result.push({ x: from.x + (to.x - from.x) * local, y: from.y + (to.y - from.y) * local });
  }
  return result;
}

// ---------------------------------------------------------------------------
// Geometry construction
// ---------------------------------------------------------------------------

const DEFAULT_VIEW_BOX = "0 0 24 24";

/** Build geometry from already-extracted SVG child nodes. */
export function geometryFromNodes(
  nodes: readonly SvgNode[],
  viewBox: string = DEFAULT_VIEW_BOX,
  sampleCount = SAMPLE_COUNT,
): IconGeometry {
  const subpaths: Subpath[] = [];
  for (const node of nodes) {
    const data = shapeToPathData(node);
    if (!data) continue;
    for (const raw of flattenPath(data)) {
      subpaths.push({
        points: resamplePolyline(raw.points, raw.closed, sampleCount),
        closed: raw.closed,
        opacity: 1,
      });
    }
  }
  return { viewBox: viewBox || DEFAULT_VIEW_BOX, subpaths };
}

const SHAPE_TAGS = new Set(["path", "line", "circle", "ellipse", "rect", "polyline", "polygon"]);
const SHAPE_ATTRS = [
  "d", "x", "y", "x1", "y1", "x2", "y2", "cx", "cy", "r", "rx", "ry",
  "width", "height", "points",
];

/**
 * Read geometry off a live (or jsdom) SVG element. Only attributes are touched
 * — no `getTotalLength`, no layout — so this behaves identically in the story
 * harness and the browser.
 */
export function geometryFromElement(svg: Element | null | undefined, sampleCount = SAMPLE_COUNT): IconGeometry | null {
  if (!svg) return null;
  const root = svg.tagName.toLowerCase() === "svg" ? svg : svg.querySelector("svg");
  if (!root) return null;
  const nodes: SvgNode[] = [];
  for (const child of Array.from(root.querySelectorAll("*"))) {
    const tag = child.tagName.toLowerCase();
    if (!SHAPE_TAGS.has(tag)) continue;
    const attrs: Record<string, string> = {};
    for (const name of SHAPE_ATTRS) {
      const value = child.getAttribute(name);
      if (value !== null) attrs[name] = value;
    }
    nodes.push({ tag, attrs });
  }
  if (nodes.length === 0) return null;
  const geometry = geometryFromNodes(nodes, root.getAttribute("viewBox") ?? DEFAULT_VIEW_BOX, sampleCount);
  return geometry.subpaths.length > 0 ? geometry : null;
}

/** Serialize one subpath back to path data. */
export function geometryPath(subpath: Subpath): string {
  if (subpath.points.length === 0) return "";
  const [head, ...rest] = subpath.points;
  const body = rest.map((point) => `L${round(point.x)} ${round(point.y)}`).join("");
  return `M${round(head!.x)} ${round(head!.y)}${body}${subpath.closed ? "Z" : ""}`;
}

function round(value: number): number {
  return Math.round(value * 100) / 100;
}

// ---------------------------------------------------------------------------
// Alignment
// ---------------------------------------------------------------------------

function centroid(points: Point[]): Point {
  if (points.length === 0) return { x: 0, y: 0 };
  let x = 0;
  let y = 0;
  for (const point of points) {
    x += point.x;
    y += point.y;
  }
  return { x: x / points.length, y: y / points.length };
}

function perimeter(points: Point[], closed: boolean): number {
  let total = 0;
  for (let index = 0; index + 1 < points.length; index += 1) {
    total += distance(points[index]!, points[index + 1]!);
  }
  if (closed && points.length > 1) total += distance(points[points.length - 1]!, points[0]!);
  return total;
}

/** Sum of squared distances between two equal-length point runs. */
function pairCost(a: Point[], b: Point[]): number {
  let total = 0;
  for (let index = 0; index < a.length; index += 1) {
    const from = a[index]!;
    const to = b[index]!;
    total += (from.x - to.x) ** 2 + (from.y - to.y) ** 2;
  }
  return total;
}

function rotated(points: Point[], offset: number): Point[] {
  const size = points.length;
  const shift = ((offset % size) + size) % size;
  if (shift === 0) return points;
  return [...points.slice(shift), ...points.slice(0, shift)];
}

/**
 * Choose the traversal of `candidate` that best matches `reference`.
 *
 * Two outlines sampled independently rarely start at the same corner, and one
 * may be wound clockwise where the other is counter-clockwise. Interpolating
 * without correcting for either is what makes naive morphs turn inside out
 * halfway through. Closed loops may rotate to any start index; open strokes may
 * only reverse, because rotating an open stroke would tear it.
 */
function bestTraversal(reference: Point[], candidate: Point[], closed: boolean): Point[] {
  const reversed = [...candidate].reverse();
  let best = candidate;
  let bestCost = pairCost(reference, candidate);

  const consider = (points: Point[]) => {
    const cost = pairCost(reference, points);
    if (cost < bestCost) {
      bestCost = cost;
      best = points;
    }
  };

  consider(reversed);
  if (closed) {
    for (let offset = 1; offset < candidate.length; offset += 1) {
      consider(rotated(candidate, offset));
      consider(rotated(reversed, offset));
    }
  }
  return best;
}

/** A zero-area subpath parked at `at`, so a missing stroke grows from a point. */
function ghostAt(at: Point, count: number, closed: boolean): Subpath {
  return {
    points: Array.from({ length: count }, () => ({ ...at })),
    closed,
    opacity: 0,
  };
}

interface AlignedPair {
  from: Subpath;
  to: Subpath;
}

/**
 * Pair up the subpaths of two icons and normalize their traversal so a
 * pointwise lerp reads as a morph.
 *
 * Counts rarely match, so the shorter side is padded with ghosts collapsed onto
 * the partner's centroid: the extra stroke grows out of, or shrinks into, the
 * place it belongs rather than blinking. Matching is greedy over a cost of
 * centroid distance plus perimeter difference — with at most a handful of
 * subpaths per icon, greedy and optimal agree in practice and greedy is stable.
 */
export function alignGeometry(from: IconGeometry, to: IconGeometry): AlignedPair[] {
  const sources = [...from.subpaths];
  const targets = [...to.subpaths];
  const pairs: AlignedPair[] = [];
  const takenTargets = new Set<number>();

  const order = sources
    .map((subpath, index) => ({ index, size: perimeter(subpath.points, subpath.closed) }))
    .sort((a, b) => b.size - a.size);

  for (const { index: sourceIndex } of order) {
    const source = sources[sourceIndex]!;
    const sourceCentre = centroid(source.points);
    const sourceLength = perimeter(source.points, source.closed);
    let bestIndex = -1;
    let bestCost = Number.POSITIVE_INFINITY;
    for (let targetIndex = 0; targetIndex < targets.length; targetIndex += 1) {
      if (takenTargets.has(targetIndex)) continue;
      const target = targets[targetIndex]!;
      const cost =
        distance(sourceCentre, centroid(target.points)) +
        Math.abs(sourceLength - perimeter(target.points, target.closed)) * 0.35 +
        (target.closed === source.closed ? 0 : 2);
      if (cost < bestCost) {
        bestCost = cost;
        bestIndex = targetIndex;
      }
    }
    if (bestIndex === -1) {
      // More sources than targets: this stroke collapses into the icon it is
      // becoming, aimed at that icon's overall centre.
      const sink = to.subpaths.length > 0
        ? centroid(to.subpaths.flatMap((subpath) => subpath.points))
        : sourceCentre;
      pairs.push({
        from: source,
        to: ghostAt(sink, source.points.length, source.closed),
      });
      continue;
    }
    takenTargets.add(bestIndex);
    const target = targets[bestIndex]!;
    pairs.push({
      from: source,
      to: {
        ...target,
        points: bestTraversal(source.points, target.points, source.closed && target.closed),
      },
    });
  }

  // More targets than sources: those strokes grow out of the outgoing icon.
  for (let targetIndex = 0; targetIndex < targets.length; targetIndex += 1) {
    if (takenTargets.has(targetIndex)) continue;
    const target = targets[targetIndex]!;
    const source = from.subpaths.length > 0
      ? centroid(from.subpaths.flatMap((subpath) => subpath.points))
      : centroid(target.points);
    pairs.push({
      from: ghostAt(source, target.points.length, target.closed),
      to: target,
    });
  }

  return pairs;
}

// ---------------------------------------------------------------------------
// Interpolation
// ---------------------------------------------------------------------------

const clamp01 = (value: number) => Math.max(0, Math.min(1, value));

/**
 * Interpolate between two icons at `progress` in [0, 1]. Pass the pairs from
 * `alignGeometry` so alignment is computed once per swap rather than once per
 * frame.
 */
export function interpolateAligned(
  pairs: readonly AlignedPair[],
  progress: number,
  viewBox: string,
): IconGeometry {
  const t = clamp01(progress);
  return {
    viewBox,
    subpaths: pairs.map(({ from, to }) => {
      const count = Math.max(from.points.length, to.points.length);
      const points: Point[] = [];
      for (let index = 0; index < count; index += 1) {
        const a = from.points[Math.min(index, from.points.length - 1)] ?? { x: 0, y: 0 };
        const b = to.points[Math.min(index, to.points.length - 1)] ?? { x: 0, y: 0 };
        points.push({ x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t });
      }
      return {
        points,
        // A ghost pairing keeps the real end's closedness so the interpolated
        // stroke does not flip between open and closed mid-flight.
        closed: from.opacity === 0 ? to.closed : from.closed,
        opacity: from.opacity + (to.opacity - from.opacity) * t,
      };
    }),
  };
}

/** Convenience wrapper for callers that do not cache alignment. */
export function interpolateGeometry(
  from: IconGeometry,
  to: IconGeometry,
  progress: number,
): IconGeometry {
  return interpolateAligned(alignGeometry(from, to), progress, to.viewBox || from.viewBox);
}

// ---------------------------------------------------------------------------
// Compatibility
// ---------------------------------------------------------------------------

function viewBoxDiagonal(viewBox: string): number {
  const { values } = scanNumbers(viewBox, 0);
  const width = values[2] ?? 24;
  const height = values[3] ?? 24;
  const diagonal = Math.hypot(width, height);
  return diagonal > 0 ? diagonal : Math.hypot(24, 24);
}

/**
 * Score whether two icons should morph or crossfade.
 *
 * Path interpolation flatters *related* shapes — a hamburger folding into a
 * close cross, a plus becoming a check — and flatters nothing else. Between
 * unrelated glyphs the in-between frames are shapes that belong to neither
 * icon, which reads as a glitch rather than as motion, however correct the
 * math. Two things predict that outcome well enough to act on:
 *
 *   - **Structural agreement.** Icons with the same number of strokes, wound
 *     the same way, have somewhere for every stroke to go. Ghost padding makes
 *     a mismatch *survivable*, not *good*.
 *   - **Travel.** If corresponding points barely move, the morph is a nudge and
 *     always looks right. If they cross most of the canvas, the intermediate
 *     frames are noise.
 *
 * The caller decides what to do with the score; nothing here refuses to morph.
 */
export function morphCompatibility(from: IconGeometry, to: IconGeometry): MorphCompatibility {
  const fromSubpaths = from.subpaths.length;
  const toSubpaths = to.subpaths.length;
  if (fromSubpaths === 0 || toSubpaths === 0) {
    return { score: 0, fromSubpaths, toSubpaths, travel: 1, reason: "empty-geometry" };
  }

  const pairs = alignGeometry(from, to);
  const diagonal = viewBoxDiagonal(to.viewBox || from.viewBox);

  let travelTotal = 0;
  let pointTotal = 0;
  for (const { from: source, to: target } of pairs) {
    // A ghost pairing has no meaningful travel — it is a fade, and charging it
    // the full canvas would make every count mismatch score zero.
    if (source.opacity === 0 || target.opacity === 0) continue;
    const count = Math.min(source.points.length, target.points.length);
    for (let index = 0; index < count; index += 1) {
      travelTotal += distance(source.points[index]!, target.points[index]!);
      pointTotal += 1;
    }
  }
  const travel = pointTotal > 0 ? travelTotal / pointTotal / diagonal : 1;

  const countRatio = Math.min(fromSubpaths, toSubpaths) / Math.max(fromSubpaths, toSubpaths);
  const matchedPairs = pairs.filter((pair) => pair.from.opacity > 0 && pair.to.opacity > 0);
  const closedAgreement = matchedPairs.length === 0
    ? 0
    : matchedPairs.filter((pair) => pair.from.closed === pair.to.closed).length / matchedPairs.length;

  // Travel is the dominant term: it is the one that predicts whether the
  // in-between frames look like anything. 25% of the canvas diagonal is the
  // point where a morph stops reading as one shape becoming another.
  const travelScore = clamp01(1 - travel / 0.25);
  const score = clamp01(travelScore * 0.6 + countRatio * 0.25 + closedAgreement * 0.15);

  return { score, fromSubpaths, toSubpaths, travel };
}

/** Above this, `morph="auto"` upgrades a crossfade to a path morph. */
export const MORPH_SCORE_THRESHOLD = 0.55;
