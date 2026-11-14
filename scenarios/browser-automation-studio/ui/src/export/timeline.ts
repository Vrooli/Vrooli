/**
 * Timeline utilities for the replay export page.
 *
 * Pure functions for computing timeline data from movie specs,
 * finding frames by time, and clamping values.
 */

import type { ReplayFrame as ReplayMovieFrame, ReplaySummary as ReplayMovieSummary } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import type { FrameTimeline } from "./types";
import { toNumber } from "@/utils/executionTypeMappers";

/** Default duration for frames without explicit timing */
export const DEFAULT_FRAME_DURATION_MS = 1600;

/**
 * Builds a timeline from movie frames.
 */
export const buildTimeline = (
  frames: ReplayMovieFrame[] | undefined | null,
): FrameTimeline[] => {
  if (!frames || frames.length === 0) {
    return [];
  }
  const sorted = [...frames].sort((a, b) => {
    const aStart = toNumber(a.startOffsetMs) ?? 0;
    const bStart = toNumber(b.startOffsetMs) ?? 0;
    return aStart - bStart;
  });
  return sorted.map((frame, index) => {
    const duration = toNumber(frame.durationMs) ?? DEFAULT_FRAME_DURATION_MS;
    const start =
      toNumber(frame.startOffsetMs) ??
      (index > 0 ? index * DEFAULT_FRAME_DURATION_MS : 0);
    return {
      index,
      startMs: Math.max(0, start),
      durationMs: Math.max(1, duration),
    };
  });
};

/**
 * Computes the total duration from summary or timeline.
 */
export const computeTotalDuration = (
  summary: ReplayMovieSummary | undefined,
  timeline: FrameTimeline[],
): number => {
  if (summary?.totalDurationMs && summary.totalDurationMs > 0) {
    return summary.totalDurationMs;
  }
  if (timeline.length === 0) {
    return 0;
  }
  const last = timeline[timeline.length - 1];
  if (!last) return 0;
  return last.startMs + Math.max(1, last.durationMs);
};

/**
 * Finds the frame index and progress for a given time in milliseconds.
 */
export const findFrameForTime = (
  ms: number,
  timeline: FrameTimeline[],
): { index: number; progress: number } => {
  if (timeline.length === 0) {
    return { index: 0, progress: 0 };
  }
  const clamped = Math.max(0, ms);
  for (let i = timeline.length - 1; i >= 0; i -= 1) {
    const entry = timeline[i];
    if (!entry) continue;
    if (clamped >= entry.startMs || i === 0) {
      const elapsed = clamped - entry.startMs;
      const progress =
        entry.durationMs > 0
          ? Math.min(Math.max(elapsed / entry.durationMs, 0), 1)
          : 1;
      return { index: entry.index, progress };
    }
  }
  const last = timeline[timeline.length - 1];
  if (!last) return { index: 0, progress: 1 };
  return { index: last.index, progress: 1 };
};

/**
 * Clamps a progress value between 0 and 1.
 */
export const clampProgress = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0;
  }
  if (value <= 0) {
    return 0;
  }
  if (value >= 1) {
    return 1;
  }
  return value;
};
