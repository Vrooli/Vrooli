/**
 * useClickEffect hook
 *
 * Manages click animation effects (pulse, ripple) triggered when
 * frames contain click actions.
 */

import { useState, useEffect } from 'react';
import type { ReplayFrame, ReplayCursorClickAnimation } from '../types';

export interface ActiveClickEffect {
  frameId: string;
  key: number;
}

export interface UseClickEffectOptions {
  cursorClickAnimation: ReplayCursorClickAnimation;
  currentFrame: ReplayFrame | null;
  frameProgress: number;
  isCursorEnabled: boolean;
}

export interface UseClickEffectResult {
  activeClickEffect: ActiveClickEffect | null;
  isClickEffectActive: boolean;
}

export function useClickEffect({
  cursorClickAnimation,
  currentFrame,
  frameProgress,
  isCursorEnabled,
}: UseClickEffectOptions): UseClickEffectResult {
  const [activeClickEffect, setActiveClickEffect] = useState<ActiveClickEffect | null>(null);

  const currentFrameId = currentFrame?.id;
  const currentStepType = currentFrame?.stepType;
  const currentClickPosition = currentFrame?.clickPosition;

  // Trigger click effect when frame changes and contains a click action
  useEffect(() => {
    if (cursorClickAnimation === 'none' || !isCursorEnabled || !currentFrameId) {
      if (activeClickEffect !== null) {
        setActiveClickEffect(null);
      }
      return;
    }
    const indicatesClick = Boolean(
      (currentClickPosition &&
        typeof currentClickPosition.x === 'number' &&
        typeof currentClickPosition.y === 'number') ||
        (typeof currentStepType === 'string' && currentStepType.toLowerCase().includes('click')),
    );
    if (!indicatesClick) {
      return;
    }
    if (frameProgress === 0) {
      setActiveClickEffect({ frameId: currentFrameId, key: Date.now() });
    }
  }, [
    activeClickEffect,
    cursorClickAnimation,
    currentClickPosition?.x,
    currentClickPosition?.y,
    currentFrameId,
    currentStepType,
    frameProgress,
    isCursorEnabled,
  ]);

  // Auto-clear click effect after animation duration
  useEffect(() => {
    if (!activeClickEffect) {
      return;
    }
    const timeout = window.setTimeout(() => {
      setActiveClickEffect((prev) => (prev && prev.key === activeClickEffect.key ? null : prev));
    }, cursorClickAnimation === 'ripple' ? 750 : 620);
    return () => {
      window.clearTimeout(timeout);
    };
  }, [activeClickEffect, cursorClickAnimation]);

  const isClickEffectActive = Boolean(
    cursorClickAnimation !== 'none' &&
      isCursorEnabled &&
      activeClickEffect &&
      currentFrameId &&
      activeClickEffect.frameId === currentFrameId,
  );

  return {
    activeClickEffect,
    isClickEffectActive,
  };
}
