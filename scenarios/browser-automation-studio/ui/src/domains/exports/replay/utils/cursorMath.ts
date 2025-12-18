/**
 * Cursor math utilities for the ReplayPlayer component
 *
 * Contains mathematical functions for cursor animation:
 * - Point clamping and comparison
 * - Speed profile easing functions
 * - Path interpolation
 * - Bezier curve generation
 * - Catmull-Rom spline generation
 */

import type {
  NormalizedPoint,
  ReplayPoint,
  CursorSpeedProfile,
  CursorPathStyle,
  CursorAnimationOverride,
} from '../types';
import { DEFAULT_PATH_STYLE, DEFAULT_SPEED_PROFILE } from '../constants';

// =============================================================================
// Point Clamping
// =============================================================================

export const clamp01 = (value: number): number => Math.min(1, Math.max(0, value));

export const clampNormalizedPoint = (point: NormalizedPoint): NormalizedPoint => ({
  x: clamp01(point.x),
  y: clamp01(point.y),
});

export const isSamePoint = (a: NormalizedPoint, b: NormalizedPoint, epsilon = 1e-4): boolean =>
  Math.abs(a.x - b.x) < epsilon && Math.abs(a.y - b.y) < epsilon;

// =============================================================================
// Speed Profile (Easing Functions)
// =============================================================================

export const applySpeedProfile = (progress: number, profile: CursorSpeedProfile): number => {
  const clamped = clamp01(progress);
  switch (profile) {
    case 'instant':
      return clamped >= 0.999 ? 1 : 0;
    case 'easeIn':
      return clamped * clamped;
    case 'easeOut':
      return 1 - (1 - clamped) * (1 - clamped);
    case 'easeInOut':
      if (clamped < 0.5) {
        return 2 * clamped * clamped;
      }
      return 1 - Math.pow(-2 * clamped + 2, 2) / 2;
    case 'linear':
    default:
      return clamped;
  }
};

// =============================================================================
// Path Interpolation
// =============================================================================

export const interpolatePath = (points: ReplayPoint[], progress: number): ReplayPoint => {
  if (points.length === 0) {
    return { x: 0, y: 0 };
  }
  if (points.length === 1) {
    return points[0];
  }

  let totalDistance = 0;
  const segments = [] as Array<{ start: ReplayPoint; end: ReplayPoint; length: number; cumulative: number }>;
  for (let index = 0; index < points.length - 1; index += 1) {
    const start = points[index];
    const end = points[index + 1];
    if (typeof start.x !== 'number' || typeof start.y !== 'number' || typeof end.x !== 'number' || typeof end.y !== 'number') {
      continue;
    }
    const length = Math.hypot(end.x - start.x, end.y - start.y);
    if (length <= 0.0001) {
      continue;
    }
    totalDistance += length;
    segments.push({ start, end, length, cumulative: totalDistance });
  }

  if (segments.length === 0 || totalDistance <= 0) {
    return points[points.length - 1];
  }

  const targetDistance = totalDistance * clamp01(progress);
  for (const segment of segments) {
    const prevCumulative = segment.cumulative - segment.length;
    if (targetDistance <= segment.cumulative) {
      const segmentProgress = segment.length > 0 ? (targetDistance - prevCumulative) / segment.length : 1;
      return {
        x: (segment.start.x ?? 0) + (segment.end.x ?? 0 - (segment.start.x ?? 0)) * segmentProgress,
        y: (segment.start.y ?? 0) + (segment.end.y ?? 0 - (segment.start.y ?? 0)) * segmentProgress,
      };
    }
  }

  return segments[segments.length - 1].end;
};

// =============================================================================
// Random Number Generation
// =============================================================================

export const createDeterministicRng = (seed: string): () => number => {
  let h = 2166136261;
  for (let i = 0; i < seed.length; i += 1) {
    h ^= seed.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return () => {
    // Xorshift32
    h ^= h << 13;
    h ^= h >>> 17;
    h ^= h << 5;
    return ((h >>> 0) % 1_000_000) / 1_000_000;
  };
};

// =============================================================================
// Bezier Curve Sampling
// =============================================================================

export const sampleQuadraticBezier = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  segments = 20,
): NormalizedPoint[] => {
  const points: NormalizedPoint[] = [];
  for (let i = 1; i < segments; i += 1) {
    const t = i / segments;
    const invT = 1 - t;
    const x = invT * invT * p0.x + 2 * invT * t * p1.x + t * t * p2.x;
    const y = invT * invT * p0.y + 2 * invT * t * p1.y + t * t * p2.y;
    points.push(clampNormalizedPoint({ x, y }));
  }
  return points;
};

export const sampleCubicBezier = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  p3: NormalizedPoint,
  segments = 28,
): NormalizedPoint[] => {
  const points: NormalizedPoint[] = [];
  for (let i = 1; i < segments; i += 1) {
    const t = i / segments;
    const invT = 1 - t;
    const x =
      invT * invT * invT * p0.x +
      3 * invT * invT * t * p1.x +
      3 * invT * t * t * p2.x +
      t * t * t * p3.x;
    const y =
      invT * invT * invT * p0.y +
      3 * invT * invT * t * p1.y +
      3 * invT * t * t * p2.y +
      t * t * t * p3.y;
    points.push(clampNormalizedPoint({ x, y }));
  }
  return points;
};

// =============================================================================
// Catmull-Rom Spline
// =============================================================================

export const catmullRom = (
  p0: NormalizedPoint,
  p1: NormalizedPoint,
  p2: NormalizedPoint,
  p3: NormalizedPoint,
  t: number,
): NormalizedPoint => {
  const t2 = t * t;
  const t3 = t2 * t;
  const x =
    0.5 *
    ((2 * p1.x) +
      (-p0.x + p2.x) * t +
      (2 * p0.x - 5 * p1.x + 4 * p2.x - p3.x) * t2 +
      (-p0.x + 3 * p1.x - 3 * p2.x + p3.x) * t3);
  const y =
    0.5 *
    ((2 * p1.y) +
      (-p0.y + p2.y) * t +
      (2 * p0.y - 5 * p1.y + 4 * p2.y - p3.y) * t2 +
      (-p0.y + 3 * p1.y - 3 * p2.y + p3.y) * t3);
  return { x, y };
};

export const generateCatmullRomPath = (
  controlPoints: NormalizedPoint[],
  samplesPerSegment = 6,
): NormalizedPoint[] => {
  if (controlPoints.length <= 2) {
    return [];
  }
  const points: NormalizedPoint[] = [];
  for (let i = 0; i < controlPoints.length - 1; i += 1) {
    const p0 = controlPoints[i - 1] ?? controlPoints[i];
    const p1 = controlPoints[i];
    const p2 = controlPoints[i + 1];
    const p3 = controlPoints[i + 2] ?? controlPoints[i + 1];
    for (let step = 1; step <= samplesPerSegment; step += 1) {
      const t = step / (samplesPerSegment + 1);
      points.push(clampNormalizedPoint(catmullRom(p0, p1, p2, p3, t)));
    }
  }
  const first = controlPoints[0];
  const last = controlPoints[controlPoints.length - 1];
  return points.filter((point) => !isSamePoint(point, first) && !isSamePoint(point, last));
};

// =============================================================================
// Stylized Path Generation
// =============================================================================

export const generateStylizedPath = (
  style: CursorPathStyle,
  start: NormalizedPoint,
  target: NormalizedPoint,
  frameId: string,
): NormalizedPoint[] => {
  if (isSamePoint(start, target)) {
    return [];
  }

  const dx = target.x - start.x;
  const dy = target.y - start.y;
  const distance = Math.hypot(dx, dy);

  if (style === 'linear') {
    return [];
  }

  if (style === 'parabolicUp' || style === 'parabolicDown') {
    const mid = { x: (start.x + target.x) / 2, y: (start.y + target.y) / 2 };
    const curveStrength = Math.min(0.45, Math.max(0.08, distance * 0.45));
    const offsetY = style === 'parabolicUp' ? -curveStrength : curveStrength;
    const control = clampNormalizedPoint({ x: mid.x, y: mid.y + offsetY });
    return sampleQuadraticBezier(start, control, target);
  }

  if (style === 'cubic') {
    const magnitude = Math.min(0.45, distance * 0.35 + 0.08);
    const perpLength = Math.hypot(-dy, dx) || 1;
    const perp = { x: (-dy) / perpLength, y: dx / perpLength };
    const control1 = clampNormalizedPoint({
      x: start.x + dx * (1 / 3) + perp.x * magnitude,
      y: start.y + dy * (1 / 3) + perp.y * magnitude,
    });
    const control2 = clampNormalizedPoint({
      x: start.x + dx * (2 / 3) - perp.x * magnitude,
      y: start.y + dy * (2 / 3) - perp.y * magnitude,
    });
    return sampleCubicBezier(start, control1, control2, target);
  }

  // Pseudorandom path
  const rng = createDeterministicRng(`${frameId}:${start.x.toFixed(4)}:${start.y.toFixed(4)}:${target.x.toFixed(4)}:${target.y.toFixed(4)}`);
  const waypointCount = 3;
  const controlPoints: NormalizedPoint[] = [start];
  for (let i = 1; i <= waypointCount; i += 1) {
    const t = i / (waypointCount + 1);
    const baseX = start.x + dx * t;
    const baseY = start.y + dy * t;
    const jitterRadius = Math.min(0.4, distance * 0.5 + 0.05);
    const angle = (rng() - 0.5) * Math.PI * 1.4;
    const offsetMagnitude = jitterRadius * (0.35 + rng() * 0.65);
    let candidate = clampNormalizedPoint({
      x: baseX + Math.cos(angle) * offsetMagnitude,
      y: baseY + Math.sin(angle) * offsetMagnitude,
    });
    const prev = controlPoints[controlPoints.length - 1];
    const prevDist = Math.hypot(prev.x - target.x, prev.y - target.y);
    const candidateDist = Math.hypot(candidate.x - target.x, candidate.y - target.y);
    if (candidateDist > prevDist) {
      const blend = 0.6;
      candidate = clampNormalizedPoint({
        x: candidate.x * blend + baseX * (1 - blend),
        y: candidate.y * blend + baseY * (1 - blend),
      });
    }
    controlPoints.push(candidate);
  }
  controlPoints.push(target);

  return generateCatmullRomPath(controlPoints, 6);
};

// =============================================================================
// Override Normalization
// =============================================================================

export const normalizeOverride = (
  override: CursorAnimationOverride | null | undefined,
  defaultPathStyle: CursorPathStyle = DEFAULT_PATH_STYLE,
  defaultSpeedProfile: CursorSpeedProfile = DEFAULT_SPEED_PROFILE,
): CursorAnimationOverride | undefined => {
  if (!override) {
    return undefined;
  }
  const next: CursorAnimationOverride = {};
  if (override.target) {
    next.target = clampNormalizedPoint(override.target);
  }
  if (override.pathStyle && override.pathStyle !== defaultPathStyle) {
    next.pathStyle = override.pathStyle;
  }
  if (override.speedProfile && override.speedProfile !== defaultSpeedProfile) {
    next.speedProfile = override.speedProfile;
  }
  if (next.pathStyle || next.target || next.speedProfile) {
    return next;
  }
  return undefined;
};
