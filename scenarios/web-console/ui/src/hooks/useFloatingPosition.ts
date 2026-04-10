import { useCallback, useMemo } from "react";

interface FloatingDimensions {
  width: number;
  height: number;
}

interface ViewportDimensions {
  width: number;
  height: number;
}

interface FloatingPositionOptions {
  floatingMargin?: number;
}

const DEFAULT_FLOATING_MARGIN = 12;

export const useFloatingPosition = (options: FloatingPositionOptions = {}) => {
  const floatingMargin = options.floatingMargin ?? DEFAULT_FLOATING_MARGIN;

  const clampPosition = useCallback(
    (
      x: number,
      y: number,
      size: FloatingDimensions,
      viewport?: ViewportDimensions,
    ) => {
      if (typeof window === "undefined") return { x, y };

      const vp = viewport ?? {
        width: window.innerWidth,
        height: window.innerHeight,
      };
      const maxX = Math.max(
        floatingMargin,
        vp.width - size.width - floatingMargin,
      );
      const maxY = Math.max(
        floatingMargin,
        vp.height - size.height - floatingMargin,
      );
      return {
        x: Math.min(Math.max(x, floatingMargin), maxX),
        y: Math.min(Math.max(y, floatingMargin), maxY),
      };
    },
    [floatingMargin],
  );

  const computeBottomRightPosition = useCallback(
    (elementSize: FloatingDimensions, viewport?: ViewportDimensions) => {
      if (typeof window === "undefined") return null;

      const vp = viewport ?? {
        width: window.innerWidth,
        height: window.innerHeight,
      };
      return clampPosition(
        vp.width - elementSize.width - floatingMargin,
        vp.height - elementSize.height - floatingMargin,
        elementSize,
        vp,
      );
    },
    [clampPosition, floatingMargin],
  );

  return useMemo(
    () => ({ clampPosition, computeBottomRightPosition }),
    [clampPosition, computeBottomRightPosition],
  );
};
