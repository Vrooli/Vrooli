import { useEffect, useRef, useState } from "react";
import { PubSub } from "utils/pubsub";

interface UsePinchZoomProps {
    onScaleChange: (scale: number) => void;
    validTargetIds: string[];
}

type UsePinchZoomReturn = {
    isPinching: boolean;
}

type PinchRefs = {
    currPosition: { x: number, y: number } | null; // Most recent pinch position
    lastPosition: { x: number, y: number } | null; // Pinch position when scale was last updated
}

/**
 * Hook for zooming in and out of a component, using pinch gestures. 
 * Supports both touch and trackpad.
 */
export const usePinchZoom = ({
    onScaleChange,
    validTargetIds = [],
}: UsePinchZoomProps): UsePinchZoomReturn => {
    const [isPinching, setIsPinching] = useState(false);
    const refs = useRef<PinchRefs>({
        currPosition: null,
        lastPosition: null,
    });

    useEffect(() => {
        const handleTouchStart = (e: TouchEvent) => {
            // Find the target
            const targetId = (e as any)?.target?.id;
            if (!targetId) return;
            if (!validTargetIds.some(id => targetId.startsWith(id))) return;
            // Pinch requires two touches
            if (e.touches.length !== 2) return;
            PubSub.get().publishSnack({ message: `Is pinching` })
            setIsPinching(true);
            refs.current.currPosition = {
                x: (e.touches[0].clientX + e.touches[1].clientX) / 2,
                y: (e.touches[0].clientY + e.touches[1].clientY) / 2,
            };
            refs.current.lastPosition = refs.current.currPosition;
        };
        const handleTouchMove = (e: TouchEvent) => {
            e.preventDefault();
            if (isPinching && e.touches.length === 2) {
                const currPosition = {
                    x: (e.touches[0].clientX + e.touches[1].clientX) / 2,
                    y: (e.touches[0].clientY + e.touches[1].clientY) / 2,
                };
                const deltaX = currPosition.x - (refs.current.currPosition?.x ?? 0);
                const deltaY = currPosition.y - (refs.current.currPosition?.y ?? 0);
                PubSub.get().publishSnack({ message: `touchmove deltaX: ${deltaX} deltaY: ${deltaY}` })
                // If deltas have same sign, we're pinching
                if (deltaX * deltaY > 0) {
                    onScaleChange(deltaX);
                    refs.current.lastPosition = refs.current.currPosition;
                    refs.current.currPosition = currPosition;
                }
            }
        };
        const handleTouchEnd = (e: TouchEvent) => {
            if (e.touches.length === 0) {
                setIsPinching(false);
                refs.current.currPosition = null;
                refs.current.lastPosition = null;
            }
        };
        const handleWheel = (e: WheelEvent) => {
            e.preventDefault();
            // If mouse wheel (instead of trackpad), ignore. This is because the grid has scrollbars
            // Mouse wheel uses much larger deltas than trackpad, so it's easier to detect
            if (e.deltaY > 50) return;
            // If not pinching, ignore. Scrolling uses integers for deltas, so it's easier to detect
            if (Number.isInteger(e.deltaY)) return;
            const moveBy = e.deltaY / 200;
            const targetId = (e as any)?.target?.id;
            if (!targetId) return;
            if (!validTargetIds.some(id => targetId.startsWith(id))) return;
            if (e.deltaY > 0) {
                onScaleChange(-moveBy);
            } else if (e.deltaY < 0) {
                onScaleChange(-moveBy);
            }
        }
        document.addEventListener("touchstart", handleTouchStart);
        document.addEventListener("touchmove", handleTouchMove);
        document.addEventListener("touchend", handleTouchEnd);
        document.addEventListener("wheel", handleWheel);
        return () => {
            document.removeEventListener("touchstart", handleTouchStart);
            document.removeEventListener("touchmove", handleTouchMove);
            document.removeEventListener("touchend", handleTouchEnd);
            document.removeEventListener("wheel", handleWheel);
        }
    }, [onScaleChange, isPinching, validTargetIds]);

    return {
        isPinching,
    }
};