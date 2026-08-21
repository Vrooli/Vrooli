import {
  normalizeIcon as baseNormalizeIcon,
  type IconGeometry,
  type MorphingIconName,
  type Point,
  type Subpath,
} from "../2.0.0/geometry";

export { baseNormalizeIcon as normalizeIcon };
export type { IconGeometry, MorphingIconName, Point, Subpath };

const progressValue = (value: number) => Math.max(-0.08, Math.min(1.08, value));
const opacityValue = (value: number) => Math.max(0, Math.min(1, value));

function center(points: Point[]): Point {
  return points.reduce(
    (sum, point) => ({
      x: sum.x + point.x / points.length,
      y: sum.y + point.y / points.length,
    }),
    { x: 0, y: 0 },
  );
}

function length(points: Point[], closed: boolean) {
  return points.reduce((total, point, index) => {
    const next = points[index + 1] ?? (closed ? points[0] : undefined);
    return next ? total + Math.hypot(next.x - point.x, next.y - point.y) : total;
  }, 0);
}

function radius(points: Point[], origin: Point) {
  return Math.sqrt(
    points.reduce(
      (sum, point) =>
        sum + (point.x - origin.x) ** 2 + (point.y - origin.y) ** 2,
      0,
    ) / points.length,
  ) || 1;
}

function candidate(points: Point[], reverse: boolean, offset: number) {
  const ordered = reverse ? [...points].reverse() : points;
  return ordered.map((_, index) => ordered[(index + offset) % ordered.length]!);
}

type Alignment = { angle: number; scale: number; residual: number };

function alignment(source: Point[], target: Point[]): Alignment {
  const a = center(source);
  const b = center(target);
  let xx = 0;
  let xy = 0;
  let yx = 0;
  let yy = 0;
  let sourceEnergy = 0;
  let targetEnergy = 0;
  for (let index = 0; index < source.length; index += 1) {
    const left = source[index]!;
    const right = target[index]!;
    const ax = left.x - a.x;
    const ay = left.y - a.y;
    const bx = right.x - b.x;
    const by = right.y - b.y;
    xx += ax * bx;
    xy += ax * by;
    yx += ay * bx;
    yy += ay * by;
    sourceEnergy += ax * ax + ay * ay;
    targetEnergy += bx * bx + by * by;
  }
  const angle = Math.atan2(xy - yx, xx + yy);
  const scaleNumerator =
    Math.cos(angle) * (xx + yy) + Math.sin(angle) * (xy - yx);
  const scale = sourceEnergy ? scaleNumerator / sourceEnergy : 1;
  const safeScale = Math.max(0.05, Math.min(20, scale));
  const residual = Math.sqrt(
    source.reduce((sum, point, index) => {
      const targetPoint = target[index]!;
      const x = point.x - a.x;
      const y = point.y - a.y;
      const transformedX = safeScale * (x * Math.cos(angle) - y * Math.sin(angle));
      const transformedY = safeScale * (x * Math.sin(angle) + y * Math.cos(angle));
      return (
        sum +
        (transformedX - (targetPoint.x - b.x)) ** 2 +
        (transformedY - (targetPoint.y - b.y)) ** 2
      );
    }, 0) / Math.max(targetEnergy, 1),
  );
  return { angle, scale: safeScale, residual };
}

function correspond(source: Subpath, target: Subpath) {
  const candidates = target.closed
    ? target.points.flatMap((_, offset) =>
        [false, true].map((reverse) => candidate(target.points, reverse, offset)),
      )
    : [target.points, [...target.points].reverse()];
  let best = candidates[0] ?? target.points;
  let bestAlignment = alignment(source.points, best);
  let bestScore = bestAlignment.residual + 0.05 * Math.abs(bestAlignment.angle) / Math.PI;
  for (const points of candidates.slice(1)) {
    const next = alignment(source.points, points);
    const score = next.residual + 0.05 * Math.abs(next.angle) / Math.PI;
    if (score < bestScore) {
      best = points;
      bestAlignment = next;
      bestScore = score;
    }
  }
  return { points: best, alignment: bestAlignment };
}

function matchSubpaths(from: Subpath[], to: Subpath[]) {
  const used = new Set<number>();
  return from.map((source, sourceIndex) => {
    let best = 0;
    let bestCost = Number.POSITIVE_INFINITY;
    to.forEach((target, targetIndex) => {
      const a = center(source.points);
      const b = center(target.points);
      const centerCost = Math.hypot(a.x - b.x, a.y - b.y) / 24;
      const sourceLength = length(source.points, source.closed);
      const targetLength = length(target.points, target.closed);
      const lengthCost = Math.abs(sourceLength - targetLength) / 24;
      const reusePenalty = used.has(targetIndex) ? 0.18 : 0;
      const indexPenalty = sourceIndex === targetIndex ? 0 : 0.01;
      const cost = centerCost + 0.35 * lengthCost + reusePenalty + indexPenalty;
      if (cost < bestCost) {
        best = targetIndex;
        bestCost = cost;
      }
    });
    used.add(best);
    return best;
  });
}

function transform(
  point: Point,
  sourceCenter: Point,
  targetCenter: Point,
  plan: Alignment,
  progress: number,
) {
  const angle = plan.angle * progress;
  const scale = 1 + (plan.scale - 1) * progress;
  const localX = point.x - sourceCenter.x;
  const localY = point.y - sourceCenter.y;
  const rotatedX = localX * Math.cos(angle) - localY * Math.sin(angle);
  const rotatedY = localX * Math.sin(angle) + localY * Math.cos(angle);
  return {
    x: sourceCenter.x + (targetCenter.x - sourceCenter.x) * progress + rotatedX * scale,
    y: sourceCenter.y + (targetCenter.y - sourceCenter.y) * progress + rotatedY * scale,
  };
}

export function interpolateGeometry(from: IconGeometry, to: IconGeometry, progress: number) {
  const t = progressValue(progress);
  const pairing = matchSubpaths(from.subpaths, to.subpaths);
  return {
    viewBox: to.viewBox,
    subpaths: from.subpaths.map((source, index) => {
      const target = to.subpaths[pairing[index] ?? 0] ?? to.subpaths[0]!;
      const targetPoints = target.points.length === source.points.length
        ? target.points
        : Array.from({ length: source.points.length }, (_, pointIndex) =>
            target.points[Math.floor(pointIndex * target.points.length / source.points.length)]!,
          );
      const correspondence = correspond(source, { ...target, points: targetPoints });
      const sourceCenter = center(source.points);
      const targetCenter = center(correspondence.points);
      const shouldFade = from.subpaths.length > to.subpaths.length && index > 0;
      return {
        closed: target.closed,
        opacity: opacityValue(source.opacity + ((shouldFade ? 0 : target.opacity) - source.opacity) * Math.max(0, t)),
        points: source.points.map((point, pointIndex) => {
          const targetPoint = correspondence.points[pointIndex]!;
          const rigid = transform(point, sourceCenter, targetCenter, correspondence.alignment, t);
          const rigidEndpoint = transform(point, sourceCenter, targetCenter, correspondence.alignment, 1);
          return {
            x: rigid.x + (targetPoint.x - rigidEndpoint.x) * t,
            y: rigid.y + (targetPoint.y - rigidEndpoint.y) * t,
          };
        }),
      };
    }),
  };
}

export function geometryPath(subpath: Subpath) {
  return subpath.points
    .map(({ x, y }, index) => `${index === 0 ? "M" : "L"}${x.toFixed(3)} ${y.toFixed(3)}`)
    .join(" ");
}
