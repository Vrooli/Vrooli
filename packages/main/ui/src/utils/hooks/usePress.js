import { useCallback, useRef } from "react";
const isTouchEvent = (event) => "touches" in event;
const isMouseEvent = (event) => "button" in event;
const preventDefaultTouch = (event) => {
    if (event.touches.length < 2 && event.preventDefault) {
        event.preventDefault();
    }
};
const getPosition = (event) => {
    if (isTouchEvent(event)) {
        const touch = event.touches[0];
        return { x: touch.clientX, y: touch.clientY };
    }
    else {
        return { x: event.clientX, y: event.clientY };
    }
};
const MAX_TRAVEL_DISTANCE = 10;
export const usePress = ({ onLongPress, onClick, onHover, onHoverEnd, onRightClick, shouldPreventDefault = true, pressDelay = 300, hoverDelay = 900, }) => {
    const longPressTriggered = useRef(false);
    const pressTimeout = useRef();
    const hoverTimeout = useRef();
    const startPosition = useRef({ x: 0, y: 0 });
    const lastPosition = useRef({ x: 0, y: 0 });
    const target = useRef();
    const isRightClick = useRef(false);
    const isPressing = useRef(false);
    const isHovering = useRef(false);
    const isDragging = useRef(false);
    const hover = useCallback((event) => {
        if (isPressing.current || isHovering.current)
            return;
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            hoverTimeout.current = undefined;
        }
        hoverTimeout.current = setTimeout(() => {
            if (target.current)
                onHover?.(target.current);
            isHovering.current = true;
        }, hoverDelay);
        target.current = event.target ?? event.currentTarget;
    }, [onHover, hoverDelay]);
    const start = useCallback((event) => {
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            pressTimeout.current = undefined;
        }
        isPressing.current = true;
        if (isMouseEvent(event)) {
            isRightClick.current = event.button === 2;
        }
        if (shouldPreventDefault && event.target) {
            if (isTouchEvent(event)) {
                event.target.addEventListener("touchEnd", preventDefaultTouch, { passive: false });
            }
        }
        target.current = event.target ?? event.currentTarget;
        const currentPosition = getPosition(event);
        startPosition.current = currentPosition;
        lastPosition.current = currentPosition;
        pressTimeout.current = setTimeout(() => {
            if (!longPressTriggered.current && target.current)
                onLongPress(target.current);
            longPressTriggered.current = true;
        }, pressDelay);
    }, [onLongPress, shouldPreventDefault, pressDelay]);
    const move = useCallback((event) => {
        if (longPressTriggered.current || !pressTimeout.current)
            return;
        const position = getPosition(event);
        lastPosition.current = position;
        const distance = Math.sqrt(Math.pow(position.x - startPosition.current.x, 2) + Math.pow(position.y - startPosition.current.y, 2));
        if (distance > MAX_TRAVEL_DISTANCE) {
            clearTimeout(pressTimeout.current);
            pressTimeout.current = undefined;
            isDragging.current = true;
        }
    }, []);
    const clear = useCallback((event, shouldTriggerClick = true) => {
        if (pressTimeout.current) {
            clearTimeout(pressTimeout.current);
            pressTimeout.current = undefined;
        }
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            hoverTimeout.current = undefined;
        }
        if (isHovering.current) {
            if (target.current)
                onHoverEnd?.(target.current);
            isHovering.current = false;
        }
        const travelDistance = Math.sqrt(Math.pow(lastPosition.current.x - startPosition.current.x, 2) +
            Math.pow(lastPosition.current.y - startPosition.current.y, 2));
        if (isPressing.current &&
            longPressTriggered.current === false &&
            travelDistance < MAX_TRAVEL_DISTANCE &&
            shouldTriggerClick &&
            !longPressTriggered.current &&
            !isDragging.current &&
            target.current) {
            if (isRightClick.current) {
                typeof onRightClick === "function" && onRightClick(target.current);
            }
            else {
                typeof onClick === "function" && onClick(target.current);
            }
        }
        longPressTriggered.current = false;
        isPressing.current = false;
        isDragging.current = false;
        if (shouldPreventDefault && target.current && isTouchEvent(event)) {
            target.current.removeEventListener("touchend", preventDefaultTouch);
        }
    }, [shouldPreventDefault, onHoverEnd, onRightClick, onClick]);
    return {
        onMouseDown: e => start(e),
        onMouseEnter: e => hover(e),
        onMouseLeave: e => clear(e, false),
        onMouseMove: e => move(e),
        onMouseUp: e => clear(e, true),
        onTouchEnd: e => clear(e, true),
        onTouchMove: e => move(e),
        onTouchStart: e => start(e),
    };
};
export default usePress;
//# sourceMappingURL=usePress.js.map