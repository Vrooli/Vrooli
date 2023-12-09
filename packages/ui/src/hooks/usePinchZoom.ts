import { useEffect, useRef, useState } from "react";

interface UsePinchZoomProps {
    onScaleChange: (scale: number, position: { x: number, y: number }) => unknown;
    validTargetIds: string[];
}

type UsePinchZoomReturn = {
    isPinching: boolean;
}

type PinchRefs = {
    currDistance: number; // Most recent distance between two fingers
    lastDistance: number; // Last distance between two fingers
}

/**
 * Hook for zooming in and out of a component, using pinch gestures. 
 * Supports both touch and trackpad. 
 * NOTE: Make sure to disable the accessibility zoom on the component you're using this hook on. Not sure how to do this yet
 */
export const usePinchZoom = ({
    onScaleChange,
    validTargetIds = [],
}: UsePinchZoomProps): UsePinchZoomReturn => {
    const [isPinching, setIsPinching] = useState(false);
    const refs = useRef<PinchRefs>({
        currDistance: 0,
        lastDistance: 0,
    });
    // Wait ref so we can update every k iterations
    const waitRef = useRef<number>(0);

    useEffect(() => {
        const getTouchDistance = (e: TouchEvent) => {
            const touch1 = e.touches[0];
            const touch2 = e.touches[1];
            return Math.sqrt(Math.pow(touch1.clientX - touch2.clientX, 2) + Math.pow(touch1.clientY - touch2.clientY, 2));
        };
        const handleTouchStart = (e: TouchEvent) => {
            // Find the target
            const targetId = (e as any)?.target?.id;
            if (!targetId) return;
            if (!validTargetIds.some(id => targetId.startsWith(id))) return;
            // Pinch requires two touches
            if (e.touches.length !== 2) return;
            setIsPinching(true);
            refs.current.currDistance = getTouchDistance(e);
            refs.current.lastDistance = refs.current.currDistance;
        };
        const handleTouchMove = (e: TouchEvent) => {
            e.preventDefault();
            // If pinching
            if (isPinching && e.touches.length === 2) {
                // Only update every 5 iterations
                if (waitRef.current < 5) {
                    waitRef.current++;
                    return;
                }
                waitRef.current = 0;
                // Get the current position
                const newDistance = getTouchDistance(e);
                // Calculate the scale delta
                const scaleDelta = (newDistance - refs.current.currDistance) / 250;
                // Find the center of the two touches, so we can zoom in on that point
                const touch1 = e.touches[0];
                const touch2 = e.touches[1];
                const center = {
                    x: (touch1.clientX + touch2.clientX) / 2,
                    y: (touch1.clientY + touch2.clientY) / 2,
                };
                // Update the scale and refs
                onScaleChange(scaleDelta, center);
                refs.current.lastDistance = refs.current.currDistance;
                refs.current.currDistance = newDistance;
            }
        };
        const handleTouchEnd = (e: TouchEvent) => {
            if (e.touches.length === 0) {
                setIsPinching(false);
                refs.current.currDistance = 0;
                refs.current.lastDistance = 0;
            }
        };
        const handleWheel = (e: WheelEvent) => {
            e.preventDefault();
            // Scale down movement so it's no too fast
            const moveBy = e.deltaY / 500;
            // Check if the target is valid
            const targetId = (e as any)?.target?.id;
            if (!targetId) return;
            if (!validTargetIds.some(id => targetId.startsWith(id))) return;
            // Find cursor position, so we can zoom in on that point
            const cursor = {
                x: e.clientX,
                y: e.clientY,
            };
            if (e.deltaY > 0) {
                onScaleChange(-moveBy, cursor);
            } else if (e.deltaY < 0) {
                onScaleChange(-moveBy, cursor);
            }
        };
        document.addEventListener("touchstart", handleTouchStart);
        document.addEventListener("touchmove", handleTouchMove);
        document.addEventListener("touchend", handleTouchEnd);
        document.addEventListener("wheel", handleWheel);
        return () => {
            document.removeEventListener("touchstart", handleTouchStart);
            document.removeEventListener("touchmove", handleTouchMove);
            document.removeEventListener("touchend", handleTouchEnd);
            document.removeEventListener("wheel", handleWheel);
        };
    }, [onScaleChange, isPinching, validTargetIds]);

    return {
        isPinching,
    };
};
