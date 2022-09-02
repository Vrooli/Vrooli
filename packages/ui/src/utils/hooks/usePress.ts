import React, { useCallback, useRef } from "react";

interface UsePressProps {
    onLongPress: (target: React.MouseEvent['target']) => void;
    onClick?: (target: React.MouseEvent['target']) => void;
    onRightClick?: (target: React.MouseEvent['target']) => void;
    shouldPreventDefault?: boolean;
    delay?: number;
}

type UsePressReturn = {
    onMouseDown: (event: React.MouseEvent) => void;
    onMouseLeave: (event: React.MouseEvent) => void;
    onMouseMove: (event: React.MouseEvent) => void;
    onMouseUp: (event: React.MouseEvent) => void;
    onTouchEnd: (event: React.TouchEvent) => void;
    onTouchMove: (event: React.TouchEvent) => void;
    onTouchStart: (event: React.TouchEvent) => void;
    onContextMenu: (event: React.MouseEvent) => void;
}

const isTouchEvent = (event: React.MouseEvent | React.TouchEvent): event is React.TouchEvent => "touches" in event;
const isMouseEvent = (event: React.MouseEvent | React.TouchEvent): event is React.MouseEvent => "button" in event;

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
export const usePress = ({
    onLongPress,
    onClick,
    onRightClick,
    shouldPreventDefault = true,
    delay = 300,
}: UsePressProps): UsePressReturn => {
    // Stores is long press has been triggered
    const longPressTriggered = useRef<boolean>(false);
    // Timeout for long press
    const timeout = useRef<NodeJS.Timeout>();
    // Positions to calculate travel (drag) distance
    const startPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    const lastPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    // Event target, for adding/removing event listeners
    const target = useRef<React.MouseEvent['target']>();
    // Stores if click was a right click
    const isRightClick = useRef<boolean>(false);
    // Stores if object is currently being pressed
    const isPressed = useRef<boolean>(false);

    const start = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        // Set isPressed to true
        isPressed.current = true;
        // Check if right click
        if (isMouseEvent(event)) {
            isRightClick.current = event.button === 2;
        }
        // Handle prevent default
        if (shouldPreventDefault && event.target) {
            if (isTouchEvent(event)) {
                event.target.addEventListener("touchEnd", preventDefaultTouch as any, { passive: false });
            }
        }
        // Store target
        target.current = event.target ?? event.currentTarget as React.MouseEvent['target'];
        // Store position
        const currentPosition = getPosition(event);
        startPosition.current = currentPosition;
        lastPosition.current = currentPosition;
        // Start timeout to determine if long press
        timeout.current = setTimeout(() => {
            if (!longPressTriggered.current && target.current) onLongPress(target.current);
            longPressTriggered.current = true;
        }, delay);
    }, [onLongPress, shouldPreventDefault, delay]);

    const move = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        if (longPressTriggered.current || !timeout.current) return;
        // Update last position
        const position = getPosition(event);
        lastPosition.current = position;
        const distance = Math.sqrt(Math.pow(position.x - startPosition.current.x, 2) + Math.pow(position.y - startPosition.current.y, 2));
        // If the distance is too far (i.e. a drag), cancel the press
        if (distance > MAX_TRAVEL_DISTANCE) {
            clearTimeout(timeout.current);
            timeout.current = undefined;
        }
    }, []);

    const clear = useCallback((event: React.MouseEvent | React.TouchEvent, shouldTriggerClick = true) => {
        // Clear timeout
        if (timeout.current) {
            clearTimeout(timeout.current);
            timeout.current = undefined;
        }
        // Calculate distatnce travelled
        const travelDistance = Math.sqrt(
            Math.pow(lastPosition.current.x - startPosition.current.x, 2) +
            Math.pow(lastPosition.current.y - startPosition.current.y, 2)
        );
        // Check if short click or right click
        if (
            isPressed.current &&
            longPressTriggered.current === false &&
            travelDistance < MAX_TRAVEL_DISTANCE &&
            shouldTriggerClick && 
            !longPressTriggered.current &&
            target.current
        ) {
            if (isRightClick.current) {
                typeof onRightClick === "function" && onRightClick(target.current);
            } else {
                typeof onClick === "function" && onClick(target.current);
            }
        }
        // Reset state and remove event listeners
        longPressTriggered.current = false;
        isPressed.current = false;
        if (shouldPreventDefault && target.current && isTouchEvent(event)) {
            target.current.removeEventListener("touchend", preventDefaultTouch as any);
        }
    }, [onClick, shouldPreventDefault, onRightClick]);

    return {
        onMouseDown: e => start(e),
        onMouseLeave: e => clear(e),
        onMouseMove: e => move(e),
        onMouseUp: e => clear(e),
        onTouchEnd: e => clear(e),
        onTouchMove: e => move(e),
        onTouchStart: e => start(e),
        onContextMenu: e => { console.log('reeeeeeee'); e.preventDefault(); }
    };
};

export default usePress;