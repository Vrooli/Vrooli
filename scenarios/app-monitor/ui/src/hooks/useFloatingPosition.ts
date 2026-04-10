import { useCallback, useMemo } from 'react';
import { PREVIEW_UI } from '@/components/views/previewConstants';

interface FloatingDimensions {
  width: number;
  height: number;
}

interface ViewportDimensions {
  width: number;
  height: number;
}

interface FloatingPositionOptions {
  /** Margin from viewport edges in pixels */
  floatingMargin?: number;
}

/**
 * Hook for managing floating element positioning with clamping and collision detection
 */
export const useFloatingPosition = (options: FloatingPositionOptions = {}) => {
  const floatingMargin = options.floatingMargin ?? PREVIEW_UI.FLOATING_MARGIN;

  /**
   * Clamps a position to keep element within viewport bounds
   */
  const clampPosition = useCallback((
    x: number,
    y: number,
    size: FloatingDimensions,
    viewport?: ViewportDimensions,
  ) => {
    if (typeof window === 'undefined') {
      return { x, y };
    }

    const vp = viewport ?? { width: window.innerWidth, height: window.innerHeight };
    const { width: innerWidth, height: innerHeight } = vp;
    const maxX = Math.max(floatingMargin, innerWidth - size.width - floatingMargin);
    const maxY = Math.max(floatingMargin, innerHeight - size.height - floatingMargin);
    const clampedX = Math.min(Math.max(x, floatingMargin), maxX);
    const clampedY = Math.min(Math.max(y, floatingMargin), maxY);
    return { x: clampedX, y: clampedY };
  }, [floatingMargin]);

  /**
   * Computes bottom-right position for a floating element
   */
  const computeBottomRightPosition = useCallback((
    elementSize: FloatingDimensions,
    viewport?: ViewportDimensions,
  ) => {
    if (typeof window === 'undefined') {
      return null;
    }

    const vp = viewport ?? { width: window.innerWidth, height: window.innerHeight };
    const { width: innerWidth, height: innerHeight } = vp;
    const desiredX = innerWidth - elementSize.width - floatingMargin;
    const desiredY = innerHeight - elementSize.height - floatingMargin;
    return clampPosition(desiredX, desiredY, elementSize, vp);
  }, [clampPosition, floatingMargin]);

  return useMemo(() => ({
    clampPosition,
    computeBottomRightPosition,
  }), [clampPosition, computeBottomRightPosition]);
};

export type UseFloatingPositionReturn = ReturnType<typeof useFloatingPosition>;
