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

type FloatingPlacement = "right-start" | "left-start" | "bottom-start" | "top-start";

interface AnchoredFloatingOptions {
  anchor: DOMRect | { left: number; right: number; top: number; bottom: number; width: number; height: number };
  size: FloatingDimensions;
  placements?: FloatingPlacement[];
  gap?: number;
  margin?: number;
  viewport?: ViewportDimensions;
}

interface FloatingPositionOptions {
  floatingMargin?: number;
}

const DEFAULT_FLOATING_MARGIN = 12;
const DEFAULT_ANCHOR_GAP = 4;

export function computeAnchoredFloatingPosition({
  anchor,
  size,
  placements = ["right-start", "left-start", "bottom-start", "top-start"],
  gap = DEFAULT_ANCHOR_GAP,
  margin = DEFAULT_FLOATING_MARGIN,
  viewport,
}: AnchoredFloatingOptions): { x: number; y: number; placement: FloatingPlacement } {
  const vp = viewport ?? (
    typeof window === "undefined"
      ? { width: Number.POSITIVE_INFINITY, height: Number.POSITIVE_INFINITY }
      : { width: window.innerWidth, height: window.innerHeight }
  );
  const safe = typeof window === "undefined"
    ? { top: 0, right: 0, bottom: 0, left: 0 }
    : readSafeAreaInsets();
  const minX = margin + safe.left;
  const minY = margin + safe.top;
  const maxX = Math.max(minX, vp.width - size.width - margin - safe.right);
  const maxY = Math.max(minY, vp.height - size.height - margin - safe.bottom);

  const candidateFor = (placement: FloatingPlacement) => {
    switch (placement) {
      case "right-start":
        return { x: anchor.right + gap, y: anchor.top };
      case "left-start":
        return { x: anchor.left - size.width - gap, y: anchor.top };
      case "bottom-start":
        return { x: anchor.left, y: anchor.bottom + gap };
      case "top-start":
        return { x: anchor.left, y: anchor.top - size.height - gap };
    }
  };

  const fits = (point: { x: number; y: number }) =>
    point.x >= minX && point.x <= maxX && point.y >= minY && point.y <= maxY;

  for (const placement of placements) {
    const candidate = candidateFor(placement);
    if (fits(candidate)) return { ...candidate, placement };
  }

  const fallbackPlacement = placements[0] ?? "right-start";
  const fallback = candidateFor(fallbackPlacement);
  return {
    x: Math.min(Math.max(fallback.x, minX), maxX),
    y: Math.min(Math.max(fallback.y, minY), maxY),
    placement: fallbackPlacement,
  };
}

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

  const computeAnchoredPosition = useCallback(
    (options: Omit<AnchoredFloatingOptions, "margin">) =>
      computeAnchoredFloatingPosition({ ...options, margin: floatingMargin }),
    [floatingMargin],
  );

  return useMemo(
    () => ({ clampPosition, computeBottomRightPosition, computeAnchoredPosition }),
    [clampPosition, computeBottomRightPosition, computeAnchoredPosition],
  );
};
