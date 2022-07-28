import React, { useCallback, useRef } from "react";

interface UseLongPressProps {
    onLongPress: (event: React.MouseEvent | React.TouchEvent) => void;
    onClick?: (event: React.MouseEvent | React.TouchEvent) => void;
    shouldPreventDefault?: boolean;
    delay?: number;
}

type UseLongPressReturn = {
    onMouseDown: (event: React.MouseEvent) => void;
    onMouseLeave: (event: React.MouseEvent) => void;
    onMouseMove: (event: React.MouseEvent) => void;
    onMouseUp: (event: React.MouseEvent) => void;
    onTouchEnd: (event: React.TouchEvent) => void;
    onTouchMove: (event: React.TouchEvent) => void;
    onTouchStart: (event: React.TouchEvent) => void;
}

const isTouchEvent = (event: React.MouseEvent | React.TouchEvent): event is React.TouchEvent => "touches" in event;

const preventDefaultTouch = (event: React.TouchEvent) => {
    if (event.touches.length < 2 && event.preventDefault) {
        event.preventDefault();
    }
};

/**
 * Determines the position of the click or touch event
 * @param event The event to get the position of
 * @returns The position of the event
 */
const getPosition = (event: React.MouseEvent | React.TouchEvent): { x: number, y: number } => {
    if (isTouchEvent(event)) {
        const touch = event.touches[0];
        return { x: touch.clientX, y: touch.clientY };
    } else {
        return { x: event.clientX, y: event.clientY };
    }
};

/**
 * Maximum travel distance allowed before a press is cancelled
 */
const MAX_TRAVEL_DISTANCE = 10;

/**
 * Triggered when its parent is long clicked or pressed. 
 * Also supports short clicks.
 */
export const useLongPress = ({
    onLongPress,
    onClick,
    shouldPreventDefault = true,
    delay = 300,
}: UseLongPressProps): UseLongPressReturn => {
    const longPressTriggered = useRef<boolean>(false);
    const timeout = useRef<NodeJS.Timeout>();
    const startPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    const lastPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    const target = useRef<React.MouseEvent['target'] | React.TouchEvent['target']>();

    const start = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        if (shouldPreventDefault && event.target) {
            if (isTouchEvent(event)) {
                event.target.addEventListener("touchEnd", preventDefaultTouch as any, { passive: false });
            }
            target.current = event.target as any;
        }
        startPosition.current = getPosition(event);
        timeout.current = setTimeout(() => {
            onLongPress(event);
            longPressTriggered.current = true;
        }, delay);
    }, [onLongPress, shouldPreventDefault, delay]);

    const move = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        if (longPressTriggered.current || !timeout.current) return;
        const position = getPosition(event);
        lastPosition.current = position;
        const distance = Math.sqrt(Math.pow(position.x - startPosition.current.x, 2) + Math.pow(position.y - startPosition.current.y, 2));
        if (distance > MAX_TRAVEL_DISTANCE) {
            clearTimeout(timeout.current);
            timeout.current = undefined;
        }
    }, []);

    const clear = useCallback((event: React.MouseEvent | React.TouchEvent, shouldTriggerClick = true) => {
        if (timeout.current) {
            clearTimeout(timeout.current);
            timeout.current = undefined;
        }
        const travelDistance = Math.sqrt(
            Math.pow(lastPosition.current.x - startPosition.current.x, 2) +
            Math.pow(lastPosition.current.y - startPosition.current.y, 2)
        );
        // If long press not triggered, count as short click
        if (travelDistance < MAX_TRAVEL_DISTANCE && shouldTriggerClick && !longPressTriggered.current && typeof onClick === "function") {
            onClick(event)
        }
        longPressTriggered.current = false;
        if (shouldPreventDefault && target.current && isTouchEvent(event)) {
            target.current.removeEventListener("touchend", preventDefaultTouch as any);
        }
    }, [shouldPreventDefault, onClick]);

    return {
        onMouseDown: e => start(e),
        onMouseLeave: e => clear(e),
        onMouseMove: e => move(e),
        onMouseUp: e => clear(e),
        onTouchEnd: e => clear(e),
        onTouchMove: e => move(e),
        onTouchStart: e => start(e),
    };
};

export default useLongPress;