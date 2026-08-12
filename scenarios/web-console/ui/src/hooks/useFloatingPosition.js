import { useCallback, useLayoutEffect, useMemo, useState } from "react";
import { readSafeAreaInsets } from "../lib/safeArea";
const DEFAULT_FLOATING_MARGIN = 12;
const DEFAULT_ANCHOR_GAP = 4;
export function computeAnchoredFloatingPosition({ anchor, size, placements = ["right-start", "left-start", "bottom-start", "top-start"], gap = DEFAULT_ANCHOR_GAP, margin = DEFAULT_FLOATING_MARGIN, viewport, }) {
    const vp = viewport ?? (typeof window === "undefined"
        ? { width: Number.POSITIVE_INFINITY, height: Number.POSITIVE_INFINITY }
        : { width: window.innerWidth, height: window.innerHeight });
    const safe = typeof window === "undefined"
        ? { top: 0, right: 0, bottom: 0, left: 0 }
        : readSafeAreaInsets();
    const minX = margin + safe.left;
    const minY = margin + safe.top;
    const maxX = Math.max(minX, vp.width - size.width - margin - safe.right);
    const maxY = Math.max(minY, vp.height - size.height - margin - safe.bottom);
    const candidateFor = (placement) => {
        switch (placement) {
            case "right-start":
                return { x: anchor.right + gap, y: anchor.top };
            case "left-start":
                return { x: anchor.left - size.width - gap, y: anchor.top };
            case "bottom-start":
                return { x: anchor.left, y: anchor.bottom + gap };
            case "top-start":
                return { x: anchor.left, y: anchor.top - size.height - gap };
            case "bottom-end":
                return { x: anchor.right - size.width, y: anchor.bottom + gap };
            case "top-end":
                return { x: anchor.right - size.width, y: anchor.top - size.height - gap };
        }
    };
    const fits = (point) => point.x >= minX && point.x <= maxX && point.y >= minY && point.y <= maxY;
    for (const placement of placements) {
        const candidate = candidateFor(placement);
        if (fits(candidate))
            return { ...candidate, placement };
    }
    const fallbackPlacement = placements[0] ?? "right-start";
    const fallback = candidateFor(fallbackPlacement);
    return {
        x: Math.min(Math.max(fallback.x, minX), maxX),
        y: Math.min(Math.max(fallback.y, minY), maxY),
        placement: fallbackPlacement,
    };
}
/**
 * Measure-then-position for anchored desktop popovers: renders the popover
 * invisibly for one layout pass, measures it, then places it against the
 * anchor via computeAnchoredFloatingPosition (the single anchor-math
 * implementation). Callers pass a stable (module-const) placements array.
 */
export function useAnchoredPopoverPosition(open, anchorRef, popoverRef, placements) {
    const [position, setPosition] = useState(null);
    useLayoutEffect(() => {
        if (!open) {
            setPosition(null);
            return;
        }
        const anchor = anchorRef.current;
        const popover = popoverRef.current;
        if (!anchor || !popover)
            return;
        const result = computeAnchoredFloatingPosition({
            anchor: anchor.getBoundingClientRect(),
            size: { width: popover.offsetWidth, height: popover.offsetHeight },
            placements,
        });
        setPosition({ x: result.x, y: result.y });
    }, [open, anchorRef, popoverRef, placements]);
    return position
        ? { position: "fixed", left: position.x, top: position.y }
        : { position: "fixed", left: 0, top: 0, opacity: 0, pointerEvents: "none" };
}
export const useFloatingPosition = (options = {}) => {
    const floatingMargin = options.floatingMargin ?? DEFAULT_FLOATING_MARGIN;
    const clampPosition = useCallback((x, y, size, viewport) => {
        if (typeof window === "undefined")
            return { x, y };
        const vp = viewport ?? {
            width: window.innerWidth,
            height: window.innerHeight,
        };
        const safe = readSafeAreaInsets();
        const minX = floatingMargin + safe.left;
        const minY = floatingMargin + safe.top;
        const maxX = Math.max(minX, vp.width - size.width - floatingMargin - safe.right);
        const maxY = Math.max(minY, vp.height - size.height - floatingMargin - safe.bottom);
        return {
            x: Math.min(Math.max(x, minX), maxX),
            y: Math.min(Math.max(y, minY), maxY),
        };
    }, [floatingMargin]);
    const computeBottomRightPosition = useCallback((elementSize, viewport) => {
        if (typeof window === "undefined")
            return null;
        const vp = viewport ?? {
            width: window.innerWidth,
            height: window.innerHeight,
        };
        return clampPosition(vp.width - elementSize.width - floatingMargin, vp.height - elementSize.height - floatingMargin, elementSize, vp);
    }, [clampPosition, floatingMargin]);
    const computeAnchoredPosition = useCallback((options) => computeAnchoredFloatingPosition({ ...options, margin: floatingMargin }), [floatingMargin]);
    return useMemo(() => ({ clampPosition, computeBottomRightPosition, computeAnchoredPosition }), [clampPosition, computeBottomRightPosition, computeAnchoredPosition]);
};
