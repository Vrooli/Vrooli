import {
  normalizeIcon,
  type IconGeometry,
  type MorphingIconName,
  type Point,
  type Subpath,
} from "../2.0.0/geometry";

export { normalizeIcon };
export type { IconGeometry, MorphingIconName, Point, Subpath };

function centroid(points: Point[]): Point {
  return points.reduce(
    (sum, point) => ({
      x: sum.x + point.x / points.length,
      y: sum.y + point.y / points.length,
    }),
    { x: 0, y: 0 },
  );
}

function radius(points: Point[], center: Point) {
  return (
    Math.sqrt(
      points.reduce(
        (sum, point) =>
          sum + (point.x - center.x) ** 2 + (point.y - center.y) ** 2,
        0,
      ) / points.length,
    ) || 1
  );
}

function candidate(points: Point[], reverse: boolean, offset: number) {
  const ordered = reverse ? [...points].reverse() : points;
  return ordered.map((_, index) => ordered[(index + offset) % ordered.length]!);
}

function correspondence(source: Subpath, target: Subpath) {
  const sourceCenter = centroid(source.points);
  const targetCenter = centroid(target.points);
  const sourceRadius = radius(source.points, sourceCenter);
  const targetRadius = radius(target.points, targetCenter);
  const offsets = target.closed ? target.points.map((_, index) => index) : [0];
  let best = target.points;
  let bestScore = Number.POSITIVE_INFINITY;
  for (const reverse of [false, true]) {
    for (const offset of offsets) {
      const points = candidate(target.points, reverse, offset);
      const score = points.reduce((sum, point, index) => {
        const sourcePoint = source.points[index]!;
        const sourceX = (sourcePoint.x - sourceCenter.x) / sourceRadius;
        const sourceY = (sourcePoint.y - sourceCenter.y) / sourceRadius;
        const targetX = (point.x - targetCenter.x) / targetRadius;
        const targetY = (point.y - targetCenter.y) / targetRadius;
        return sum + (sourceX - targetX) ** 2 + (sourceY - targetY) ** 2;
      }, 0);
      if (score < bestScore) {
        bestScore = score;
        best = points;
      }
    }
  }
  return { points: best, sourceCenter, targetCenter };
}

function matchSubpaths(from: Subpath[], to: Subpath[]) {
  return from.map((source, sourceIndex) => {
    let best = 0;
    let bestScore = Number.POSITIVE_INFINITY;
    to.forEach((target, targetIndex) => {
      const sourceCenter = centroid(source.points);
      const targetCenter = centroid(target.points);
      const score =
        Math.hypot(
          sourceCenter.x - targetCenter.x,
          sourceCenter.y - targetCenter.y,
        ) + (sourceIndex === targetIndex ? 0 : 0.01);
      if (score < bestScore) {
        best = targetIndex;
        bestScore = score;
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
  return {
    viewBox: to.viewBox,
    subpaths: from.subpaths.map((source, index) => {
      const target = (to.subpaths[matches[index] ?? 0] ?? to.subpaths[0])!;
      const aligned = correspondence(source, target);
      return {
        closed: target.closed,
        opacity: source.opacity + (target.opacity - source.opacity) * t,
        points: source.points.map((point, pointIndex) => ({
          x:
            point.x +
            (aligned.points[pointIndex]!.x - point.x) * t +
            (aligned.targetCenter.x - aligned.sourceCenter.x) * t,
          y:
            point.y +
            (aligned.points[pointIndex]!.y - point.y) * t +
            (aligned.targetCenter.y - aligned.sourceCenter.y) * t,
        })),
      };
    }),
  };
}

export function geometryPath(subpath: Subpath) {
  return subpath.points
    .map(
      ({ x, y }, index) =>
        `${index === 0 ? "M" : "L"}${x.toFixed(3)} ${y.toFixed(3)}`,
    )
    .join(" ");
}
