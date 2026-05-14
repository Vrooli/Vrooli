import { useCallback, useMemo } from "react";
import { readSafeAreaInsets } from "../lib/safeArea";

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
      const safe = readSafeAreaInsets();
      const minX = floatingMargin + safe.left;
      const minY = floatingMargin + safe.top;
      const maxX = Math.max(
        minX,
        vp.width - size.width - floatingMargin - safe.right,
      );
      const maxY = Math.max(
        minY,
        vp.height - size.height - floatingMargin - safe.bottom,
      );
      return {
        x: Math.min(Math.max(x, minX), maxX),
        y: Math.min(Math.max(y, minY), maxY),
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
