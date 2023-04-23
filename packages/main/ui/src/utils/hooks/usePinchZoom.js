import { useEffect, useRef, useState } from "react";
export const usePinchZoom = ({ onScaleChange, validTargetIds = [], }) => {
    const [isPinching, setIsPinching] = useState(false);
    const refs = useRef({
        currDistance: 0,
        lastDistance: 0,
    });
    const waitRef = useRef(0);
    useEffect(() => {
        const getTouchDistance = (e) => {
            const touch1 = e.touches[0];
            const touch2 = e.touches[1];
            return Math.sqrt(Math.pow(touch1.clientX - touch2.clientX, 2) + Math.pow(touch1.clientY - touch2.clientY, 2));
        };
        const handleTouchStart = (e) => {
            const targetId = e?.target?.id;
            if (!targetId)
                return;
            if (!validTargetIds.some(id => targetId.startsWith(id)))
                return;
            if (e.touches.length !== 2)
                return;
            setIsPinching(true);
            refs.current.currDistance = getTouchDistance(e);
            refs.current.lastDistance = refs.current.currDistance;
        };
        const handleTouchMove = (e) => {
            e.preventDefault();
            if (isPinching && e.touches.length === 2) {
                if (waitRef.current < 5) {
                    waitRef.current++;
                    return;
                }
                waitRef.current = 0;
                const newDistance = getTouchDistance(e);
                const scaleDelta = (newDistance - refs.current.currDistance) / 250;
                const touch1 = e.touches[0];
                const touch2 = e.touches[1];
                const center = {
                    x: (touch1.clientX + touch2.clientX) / 2,
                    y: (touch1.clientY + touch2.clientY) / 2,
                };
                onScaleChange(scaleDelta, center);
                refs.current.lastDistance = refs.current.currDistance;
                refs.current.currDistance = newDistance;
            }
        };
        const handleTouchEnd = (e) => {
            if (e.touches.length === 0) {
                setIsPinching(false);
                refs.current.currDistance = 0;
                refs.current.lastDistance = 0;
            }
        };
        const handleWheel = (e) => {
            e.preventDefault();
            const moveBy = e.deltaY / 500;
            const targetId = e?.target?.id;
            if (!targetId)
                return;
            if (!validTargetIds.some(id => targetId.startsWith(id)))
                return;
            const cursor = {
                x: e.clientX,
                y: e.clientY,
            };
            if (e.deltaY > 0) {
                onScaleChange(-moveBy, cursor);
            }
            else if (e.deltaY < 0) {
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
//# sourceMappingURL=usePinchZoom.js.map