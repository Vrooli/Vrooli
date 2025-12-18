/**
 * useCursorAnimation hook
 *
 * Computes cursor animation plans for each frame and calculates
 * the current cursor position based on playback progress.
 */

import { useMemo, useEffect, useState, useRef } from 'react';
import type {
  ReplayFrame,
  ReplayPoint,
  NormalizedPoint,
  CursorSpeedProfile,
  CursorPathStyle,
  CursorAnimationOverride,
  CursorOverrideMap,
  CursorPlan,
  Dimensions,
  ReplayCursorInitialPosition,
} from '../types';
import {
  clampNormalizedPoint,
  applySpeedProfile,
  interpolatePath,
  generateStylizedPath,
} from '../utils/cursorMath';
import { toNormalizedPoint, toAbsolutePoint } from '../utils/geometry';
import { FALLBACK_DIMENSIONS } from '../constants';

export interface UseCursorAnimationOptions {
  frames: ReplayFrame[];
  currentIndex: number;
  frameProgress: number;
  isPlaying: boolean;
  isCursorEnabled: boolean;
  cursorOverrides: CursorOverrideMap;
  cursorInitialPosition: ReplayCursorInitialPosition;
  basePathStyle: CursorPathStyle;
  baseSpeedProfile: CursorSpeedProfile;
}

export interface UseCursorAnimationResult {
  cursorPlans: CursorPlan[];
  cursorPosition: ReplayPoint | undefined;
}

export function useCursorAnimation({
  frames,
  currentIndex,
  frameProgress,
  isPlaying,
  isCursorEnabled,
  cursorOverrides,
  cursorInitialPosition,
  basePathStyle,
  baseSpeedProfile,
}: UseCursorAnimationOptions): UseCursorAnimationResult {
  const [cursorPosition, setCursorPosition] = useState<ReplayPoint | undefined>(undefined);
  const randomSeedsRef = useRef<Record<string, NormalizedPoint>>({});

  // Reset random seeds when initial position changes
  useEffect(() => {
    randomSeedsRef.current = {};
  }, [cursorInitialPosition]);

  // Clear cursor position when disabled
  useEffect(() => {
    if (!isCursorEnabled) {
      setCursorPosition(undefined);
    }
  }, [isCursorEnabled]);

  const computeFallbackNormalized = (frameId: string, dims: Dimensions): NormalizedPoint => {
    const width = dims.width || FALLBACK_DIMENSIONS.width;
    const height = dims.height || FALLBACK_DIMENSIONS.height;
    const computePadRatio = (size: number) => {
      if (!size) {
        return 0.08;
      }
      return Math.min(0.48, Math.max(12 / size, 0.08));
    };
    const padX = computePadRatio(width);
    const padY = computePadRatio(height);

    switch (cursorInitialPosition) {
      case 'top-left':
        return clampNormalizedPoint({ x: padX, y: padY });
      case 'top-right':
        return clampNormalizedPoint({ x: 1 - padX, y: padY });
      case 'bottom-left':
        return clampNormalizedPoint({ x: padX, y: 1 - padY });
      case 'bottom-right':
        return clampNormalizedPoint({ x: 1 - padX, y: 1 - padY });
      case 'random': {
        const seed = randomSeedsRef.current[frameId] || { x: Math.random(), y: Math.random() };
        randomSeedsRef.current[frameId] = seed;
        const usableX = Math.max(0.02, 1 - padX * 2);
        const usableY = Math.max(0.02, 1 - padY * 2);
        return clampNormalizedPoint({ x: padX + seed.x * usableX, y: padY + seed.y * usableY });
      }
      case 'center':
      default:
        return { x: 0.5, y: 0.5 };
    }
  };

  const cursorPlans = useMemo<CursorPlan[]>(() => {
    const plans: CursorPlan[] = [];
    let previousNormalized: NormalizedPoint | undefined;

    frames.forEach((frame) => {
      const dims: Dimensions = {
        width: frame?.screenshot?.width || FALLBACK_DIMENSIONS.width,
        height: frame?.screenshot?.height || FALLBACK_DIMENSIONS.height,
      };

      const override = cursorOverrides[frame.id];

      const recordedTrail = Array.isArray(frame.cursorTrail)
        ? frame.cursorTrail.filter(
            (point): point is ReplayPoint => typeof point?.x === 'number' && typeof point?.y === 'number',
          )
        : [];

      const recordedTrailNormalized = recordedTrail
        .map((point) => toNormalizedPoint(point, dims))
        .filter((point): point is NormalizedPoint => Boolean(point))
        .map(clampNormalizedPoint);

      const recordedClickNormalized = toNormalizedPoint(frame.clickPosition ?? undefined, dims);

      const overrideTargetNormalized = override?.target ? clampNormalizedPoint(override.target) : undefined;
      const recordedTargetNormalized =
        recordedTrailNormalized.length > 0
          ? recordedTrailNormalized[recordedTrailNormalized.length - 1]
          : recordedClickNormalized ?? undefined;

      const fallbackNormalized = computeFallbackNormalized(frame.id, dims);
      const targetNormalized = overrideTargetNormalized ?? recordedTargetNormalized ?? fallbackNormalized;

      const recordedIntermediate =
        recordedTrailNormalized.length > 2
          ? recordedTrailNormalized.slice(1, recordedTrailNormalized.length - 1)
          : [];

      let startNormalized = previousNormalized;
      if (!startNormalized) {
        if (recordedTrailNormalized.length > 0) {
          startNormalized = recordedTrailNormalized[0];
        } else {
          startNormalized = fallbackNormalized;
        }
      }

      startNormalized = clampNormalizedPoint(startNormalized);

      const overridePathStyle = override?.pathStyle;
      const usingRecordedTrail = !overridePathStyle && recordedIntermediate.length > 0;
      const effectivePathStyle = overridePathStyle ?? basePathStyle;
      const generatedPath = usingRecordedTrail
        ? recordedIntermediate
        : generateStylizedPath(effectivePathStyle, startNormalized, targetNormalized, frame.id);

      plans.push({
        frameId: frame.id,
        dims,
        startNormalized,
        targetNormalized,
        pathNormalized: generatedPath,
        speedProfile: override?.speedProfile ?? baseSpeedProfile,
        pathStyle: effectivePathStyle,
        hasRecordedTrail: usingRecordedTrail,
        previousTargetNormalized: previousNormalized,
      });

      previousNormalized = targetNormalized;
    });

    return plans;
  }, [frames, cursorOverrides, cursorInitialPosition, basePathStyle, baseSpeedProfile]);

  // Update cursor position based on current frame and progress
  useEffect(() => {
    if (!isCursorEnabled) {
      return;
    }
    const plan = cursorPlans[currentIndex];
    if (!plan) {
      setCursorPosition(undefined);
      return;
    }
    const normalizedPath = [plan.startNormalized, ...plan.pathNormalized, plan.targetNormalized];
    if (normalizedPath.length === 0) {
      setCursorPosition(undefined);
      return;
    }
    const absolutePath = normalizedPath.map((point) => toAbsolutePoint(point, plan.dims));
    const profiled = applySpeedProfile(isPlaying ? frameProgress : 1, plan.speedProfile);
    const evaluated = interpolatePath(absolutePath, profiled);
    setCursorPosition(evaluated);
  }, [cursorPlans, currentIndex, frameProgress, isCursorEnabled, isPlaying]);

  return {
    cursorPlans,
    cursorPosition,
  };
}
