import {
  ICON_REGISTRY,
  type IconDefinition,
  type IconName,
} from "@vrooli/react-component-library/IconRegistry/1.0.0";

export type MorphingIconName = IconName | "copy";
export type Point = { x: number; y: number };
export type Subpath = { points: Point[]; closed: boolean; opacity: number };
export type IconGeometry = { viewBox: string; subpaths: Subpath[] };

const copyIcon: IconDefinition = {
  name: "check" as IconName,
  viewBox: "0 0 24 24",
  path: "M9 9h10v10H9zM5 15H4V4h11v1",
};
const commandPattern = /([a-z])([^a-z]*)/gi;
const numberPattern = /-?(?:\d+\.?\d*|\.\d+)/g;
const SAMPLE_COUNT = 32;

function definition(name: MorphingIconName): IconDefinition {
  return name === "copy" ? copyIcon : ICON_REGISTRY[name];
}

function numbers(value: string) {
  return (value.match(numberPattern) ?? []).map(Number);
}

function distance(a: Point, b: Point) {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

function sample(points: Point[], closed: boolean): Point[] {
  if (points.length < 2) return points;
  const segments = points.slice(0, -1).map((point, index) => ({
    from: point,
    to: points[index + 1]!,
    length: distance(point, points[index + 1]!),
  }));
  if (closed) {
    segments.push({
      from: points[points.length - 1]!,
      to: points[0]!,
      length: distance(points[points.length - 1]!, points[0]!),
    });
  }
  const total = segments.reduce((sum, segment) => sum + segment.length, 0);
  if (!total) return Array.from({ length: SAMPLE_COUNT }, () => points[0]!);
  return Array.from({ length: SAMPLE_COUNT }, (_, sampleIndex) => {
    const target = (sampleIndex / (SAMPLE_COUNT - (closed ? 0 : 1))) * total;
    let traversed = 0;
    for (const segment of segments) {
      if (target <= traversed + segment.length || segment === segments.at(-1)) {
        const local = segment.length
          ? (target - traversed) / segment.length
          : 0;
        return {
          x: segment.from.x + (segment.to.x - segment.from.x) * local,
          y: segment.from.y + (segment.to.y - segment.from.y) * local,
        };
      }
      traversed += segment.length;
    }
    return segments.at(-1)!.to;
  });
}

export function normalizeIcon(name: MorphingIconName): IconGeometry {
  const icon = definition(name);
  const subpaths: Array<{ points: Point[]; closed: boolean }> = [];
  let current: Point = { x: 0, y: 0 };
  let active: Point[] = [];
  let closed = false;
  const flush = () => {
    if (active.length > 1) subpaths.push({ points: active, closed });
    active = [];
    closed = false;
  };

  for (const match of icon.path.matchAll(commandPattern)) {
    const command = match[1];
    if (!command) continue;
    const values = numbers(match[2] ?? "");
    const type = command.toUpperCase();
    const relative = command === command.toLowerCase();
    if (type === "M") {
      flush();
      const base = current;
      current = {
        x: (relative ? base.x : 0) + (values[0] ?? 0),
        y: (relative ? base.y : 0) + (values[1] ?? 0),
      };
      active.push(current);
    } else if (type === "L" || type === "T") {
      for (let index = 0; index + 1 < values.length; index += 2) {
        const next = {
          x: (relative ? current.x : 0) + values[index]!,
          y: (relative ? current.y : 0) + values[index + 1]!,
        };
        active.push(next);
        current = next;
      }
    } else if (type === "H" || type === "V") {
      for (const value of values) {
        const next =
          type === "H"
            ? { x: (relative ? current.x : 0) + value, y: current.y }
            : { x: current.x, y: (relative ? current.y : 0) + value };
        active.push(next);
        current = next;
      }
    } else if (type === "Z") {
      closed = true;
    }
  }
  flush();
  return {
    viewBox: icon.viewBox,
    subpaths: subpaths.map(({ points, closed: isClosed }) => ({
      points: sample(points, isClosed),
      closed: isClosed,
      opacity: 1,
    })),
  };
}

function centroid(points: Point[]): Point {
  return points.reduce(
    (sum, point) => ({
      x: sum.x + point.x / points.length,
      y: sum.y + point.y / points.length,
    }),
    { x: 0, y: 0 },
  );
}

function pathLength(points: Point[], closed: boolean) {
  return points.reduce((sum, point, index) => {
    const next = points[index + 1] ?? (closed ? points[0] : undefined);
    return next ? sum + distance(point, next) : sum;
  }, 0);
}

function matchSubpaths(from: Subpath[], to: Subpath[]) {
  return from.map((source, sourceIndex) => {
    let best = 0;
    let bestCost = Number.POSITIVE_INFINITY;
    from.forEach(() => undefined);
    to.forEach((target, targetIndex) => {
      const centerCost = distance(
        centroid(source.points),
        centroid(target.points),
      );
      const lengthCost = Math.abs(
        pathLength(source.points, source.closed) -
          pathLength(target.points, target.closed),
      );
      const cost =
        centerCost +
        lengthCost * 0.35 +
        (targetIndex === sourceIndex ? 0 : 0.01);
      if (cost < bestCost) {
        best = targetIndex;
        bestCost = cost;
      }
    });
    return best;
  });
}

export function interpolateGeometry(
  from: IconGeometry,
  to: IconGeometry,
  progress: number,
): IconGeometry {
  const t = Math.max(0, Math.min(1, progress));
  const matches = matchSubpaths(from.subpaths, to.subpaths);
  const subpaths = from.subpaths.map((source, index) => {
    const targetIndex = matches[index] ?? 0;
    const target = (to.subpaths[targetIndex] ?? to.subpaths[0]) as Subpath;
    const sourceCenter = centroid(source.points);
    const targetCenter = centroid(target.points);
    const sourceScale = Math.max(
      1,
      Math.sqrt(
        source.points.reduce(
          (sum, point) => sum + distance(point, sourceCenter) ** 2,
          0,
        ),
      ),
    );
    const targetScale = Math.max(
      1,
      Math.sqrt(
        target.points.reduce(
          (sum, point) => sum + distance(point, targetCenter) ** 2,
          0,
        ),
      ),
    );
    const dot = source.points.reduce((sum, point, pointIndex) => {
      const a = { x: point.x - sourceCenter.x, y: point.y - sourceCenter.y };
      const b = {
        x: target.points[pointIndex]!.x - targetCenter.x,
        y: target.points[pointIndex]!.y - targetCenter.y,
      };
      return sum + a.x * b.x + a.y * b.y;
    }, 0);
    const cross = source.points.reduce((sum, point, pointIndex) => {
      const a = { x: point.x - sourceCenter.x, y: point.y - sourceCenter.y };
      const b = {
        x: target.points[pointIndex]!.x - targetCenter.x,
        y: target.points[pointIndex]!.y - targetCenter.y,
      };
      return sum + a.x * b.y - a.y * b.x;
    }, 0);
    const angle = Math.atan2(cross, dot);
    const scale = targetScale / sourceScale;
    const cos = Math.cos(angle * t);
    const sin = Math.sin(angle * t);
    return {
      closed: target.closed,
      opacity: source.opacity + (target.opacity - source.opacity) * t,
      points: source.points.map((point, pointIndex) => {
        const local = {
          x: point.x - sourceCenter.x,
          y: point.y - sourceCenter.y,
        };
        const rotated = {
          x: local.x * cos - local.y * sin,
          y: local.x * sin + local.y * cos,
        };
        const transformed = {
          x:
            rotated.x * (1 + (scale - 1) * t) +
            sourceCenter.x +
            (targetCenter.x - sourceCenter.x) * t,
          y:
            rotated.y * (1 + (scale - 1) * t) +
            sourceCenter.y +
            (targetCenter.y - sourceCenter.y) * t,
        };
        const targetPoint = target.points[pointIndex]!;
        return {
          x: transformed.x + (targetPoint.x - transformed.x) * t,
          y: transformed.y + (targetPoint.y - transformed.y) * t,
        };
      }),
    };
  });
  return { viewBox: to.viewBox, subpaths };
}

export function geometryPath(subpath: Subpath) {
  return subpath.points
    .map(
      ({ x, y }, index) =>
        `${index === 0 ? "M" : "L"}${x.toFixed(3)} ${y.toFixed(3)}`,
    )
    .join(" ");
}
