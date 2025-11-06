import { useCallback, useMemo } from 'react';
import type { CSSProperties } from 'react';
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
  /** Offset from anchor element in pixels */
  menuOffset?: number;
}

/**
 * Hook for managing floating element positioning with clamping and collision detection
 */
export const useFloatingPosition = (options: FloatingPositionOptions = {}) => {
  const floatingMargin = options.floatingMargin ?? PREVIEW_UI.FLOATING_MARGIN;
  const menuOffset = options.menuOffset ?? PREVIEW_UI.MENU_OFFSET;

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
   * Computes optimal menu/popover positioning relative to an anchor element
   * with automatic flip to avoid viewport overflow
   */
  const computeMenuStyle = useCallback((
    anchorRect: DOMRect | null,
    popover: HTMLDivElement | null,
    viewport?: ViewportDimensions,
  ): CSSProperties | undefined => {
    if (!anchorRect) {
      return undefined;
    }

    const viewportWidth = viewport?.width ?? (typeof window !== 'undefined' ? window.innerWidth : undefined);
    const viewportHeight = viewport?.height ?? (typeof window !== 'undefined' ? window.innerHeight : undefined);
    const popRect = popover?.getBoundingClientRect();
    const popHeight = popRect?.height ?? 0;
    const popWidth = popRect?.width ?? 0;

    // Vertical positioning with flip
    let top = anchorRect.bottom + menuOffset;
    if (typeof viewportHeight === 'number') {
      const spaceBelow = viewportHeight - anchorRect.bottom - menuOffset;
      const shouldPlaceBelow = popHeight === 0
        ? anchorRect.top + anchorRect.height / 2 < viewportHeight / 2
        : spaceBelow >= popHeight + floatingMargin;
      if (!shouldPlaceBelow) {
        top = anchorRect.top - menuOffset - popHeight;
      }
      const maxTop = viewportHeight - floatingMargin - popHeight;
      const minTop = floatingMargin;
      top = Math.min(Math.max(top, minTop), maxTop);
    }

    // Horizontal positioning
    let left = anchorRect.right;
    if (typeof viewportWidth === 'number') {
      const maxLeft = viewportWidth - floatingMargin;
      const minLeft = popWidth > 0 ? popWidth + floatingMargin : floatingMargin;
      left = Math.min(Math.max(left, minLeft), maxLeft);
    }

    return {
      top: `${Math.round(top)}px`,
      left: `${Math.round(left)}px`,
      transform: 'translateX(-100%)',
    } satisfies CSSProperties;
  }, [floatingMargin, menuOffset]);

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
    computeMenuStyle,
    computeBottomRightPosition,
  }), [clampPosition, computeMenuStyle, computeBottomRightPosition]);
};

export type UseFloatingPositionReturn = ReturnType<typeof useFloatingPosition>;
