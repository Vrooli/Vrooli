/**
 * useCursorAnimation
 *
 * Replay cursor positions come from recorded points or explicit editor
 * overrides. The hook does not synthesize movement between unrelated events.
 */

import { useMemo } from 'react';
import type {
  ReplayFrame,
  ReplayPoint,
  CursorSpeedProfile,
  CursorPathStyle,
  CursorOverrideMap,
  CursorPlan,
} from '../types';
import { toNormalizedPoint, toAbsolutePoint } from '../utils/geometry';
import { FALLBACK_DIMENSIONS } from '../constants';

export type CursorProvenance = 'observed' | 'edited' | 'derived' | 'missing';

export interface UseCursorAnimationOptions {
  frames: ReplayFrame[];
  currentIndex: number;
  frameProgress: number;
  isPlaying: boolean;
  isCursorEnabled: boolean;
  cursorOverrides: CursorOverrideMap;
  basePathStyle: CursorPathStyle;
  baseSpeedProfile: CursorSpeedProfile;
}

export interface UseCursorAnimationResult {
  cursorPlans: Array<CursorPlan | undefined>;
  cursorPosition: ReplayPoint | undefined;
  cursorProvenance: CursorProvenance;
}

const isPoint = (point: ReplayPoint | null | undefined): point is ReplayPoint =>
  typeof point?.x === 'number' && Number.isFinite(point.x)
  && typeof point?.y === 'number' && Number.isFinite(point.y);

function recordedPoints(frame: ReplayFrame): ReplayPoint[] {
  if (frame.cursorProvenance === 'missing') return [];
  const samples = frame.cursorTrailSamples?.filter(isPoint) ?? [];
  if (samples.length > 0) return samples;
  const trail = Array.isArray(frame.cursorTrail) ? frame.cursorTrail.filter(isPoint) : [];
  if (trail.length > 0) return trail;
  if (isPoint(frame.cursorPosition)) return [frame.cursorPosition];
  if (isPoint(frame.clickPosition)) return [frame.clickPosition];
  return [];
}

export function cursorSampleAtProgress(frame: ReplayFrame, progress: number): ReplayPoint | undefined {
  if (frame.cursorProvenance === 'missing') return undefined;
  const timedSamples = frame.cursorTrailSamples?.filter((sample) => isPoint(sample)
    && typeof sample.elapsedMs === 'number' && Number.isFinite(sample.elapsedMs));
  if (!timedSamples?.length) {
    const points = recordedPoints(frame);
    return points[points.length - 1];
  }
  const totalDuration = Math.max(0, frame.totalDurationMs ?? frame.durationMs ?? 0);
  const actionDuration = Math.min(totalDuration, Math.max(0, frame.durationMs ?? totalDuration));
  const elapsed = Math.min(actionDuration, Math.max(0, Math.min(1, progress)) * totalDuration);
  let current: ReplayPoint | undefined;
  for (const sample of timedSamples) {
    if ((sample.elapsedMs ?? 0) > elapsed) break;
    current = sample;
  }
  return current ? { x: current.x, y: current.y } : undefined;
}

export function useCursorAnimation({
  frames,
  currentIndex,
  frameProgress,
  isPlaying,
  isCursorEnabled,
  cursorOverrides,
  basePathStyle,
  baseSpeedProfile,
}: UseCursorAnimationOptions): UseCursorAnimationResult {
  const cursorPlans = useMemo<Array<CursorPlan | undefined>>(() => frames.map((frame) => {
    const points = recordedPoints(frame);
    const override = cursorOverrides[frame.id];
    if (points.length === 0 && !override?.target) return undefined;

    const dims = {
      width: frame.viewport?.width || frame.screenshot?.width || FALLBACK_DIMENSIONS.width,
      height: frame.viewport?.height || frame.screenshot?.height || FALLBACK_DIMENSIONS.height,
    };
    const normalized = points
      .map((point) => toNormalizedPoint(point, dims))
      .filter((point): point is { x: number; y: number } => Boolean(point));
    const first = normalized[0] ?? override?.target;
    const last = override?.target ?? normalized[normalized.length - 1];
    if (!first || !last) return undefined;

    return {
      frameId: frame.id,
      dims,
      startNormalized: first,
      targetNormalized: last,
      pathNormalized: normalized.slice(1, -1),
      speedProfile: override?.speedProfile ?? baseSpeedProfile,
      pathStyle: override?.pathStyle ?? basePathStyle,
      hasRecordedTrail: points.length > 1,
    };
  }), [frames, cursorOverrides, basePathStyle, baseSpeedProfile]);

  const frame = frames[currentIndex];
  const override = frame ? cursorOverrides[frame.id] : undefined;
  const points = frame ? recordedPoints(frame) : [];
  const cursorProvenance: CursorProvenance = override?.target
    ? 'edited'
    : points.length > 0 ? frame?.cursorProvenance ?? 'observed' : 'missing';

  let cursorPosition: ReplayPoint | undefined;
  const plan = cursorPlans[currentIndex];
  if (isCursorEnabled && plan) {
    cursorPosition = override?.target
      ? toAbsolutePoint(override.target, plan.dims)
      : isPlaying || frameProgress > 0
        ? cursorSampleAtProgress(frame!, frameProgress)
        : points[points.length - 1];
  }

  return { cursorPlans, cursorPosition, cursorProvenance };
}
