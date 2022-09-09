import React, { useCallback, useRef } from "react";

interface UsePressProps {
    onLongPress: (target: React.MouseEvent['target']) => void;
    onClick?: (target: React.MouseEvent['target']) => void;
    onHover?: (target: React.MouseEvent['target']) => void;
    onRightClick?: (target: React.MouseEvent['target']) => void;
    shouldPreventDefault?: boolean;
    delay?: number;
}

type UsePressReturn = {
    onMouseDown: (event: React.MouseEvent) => void;
    onMouseEnter: (event: React.MouseEvent) => void;
    onMouseLeave: (event: React.MouseEvent) => void;
    onMouseMove: (event: React.MouseEvent) => void;
    onMouseUp: (event: React.MouseEvent) => void;
    onTouchEnd: (event: React.TouchEvent) => void;
    onTouchMove: (event: React.TouchEvent) => void;
    onTouchStart: (event: React.TouchEvent) => void;
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
    onHover,
    onRightClick,
    shouldPreventDefault = true,
    delay = 300,
}: UsePressProps): UsePressReturn => {
    // Stores is long press has been triggered
    const longPressTriggered = useRef<boolean>(false);
    // Timeout for long press
    const pressTimeout = useRef<NodeJS.Timeout>();
    // Timeout for hover
    const hoverTimeout = useRef<NodeJS.Timeout>();
    // Positions to calculate travel (drag) distance
    const startPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    const lastPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    // Event target, for adding/removing event listeners
    const target = useRef<React.MouseEvent['target']>();
    // Stores if click was a right click
    const isRightClick = useRef<boolean>(false);
    // Stores if object is currently being pressed
    const isPressed = useRef<boolean>(false);

    const hover = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        console.log('usepress hover', isPressed.current, event);
        // Ignore if pressing
        if (isPressed.current) return;
        // Cancel if already triggered
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current); 
            hoverTimeout.current = undefined;
        }  
        // Set timeout. Hover delay is longer than press delay
        hoverTimeout.current = setTimeout(() => {
            if (target.current) onHover?.(target.current);
        }, delay * 2);
        // Store target
        target.current = event.target ?? event.currentTarget as React.MouseEvent['target'];
    }, [onHover, delay]);

    const start = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        console.log("start", event);
        // Cancel hover timeout
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            pressTimeout.current = undefined;
        }
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
        // Start pressTimeout to determine if long press
        pressTimeout.current = setTimeout(() => {
            if (!longPressTriggered.current && target.current) onLongPress(target.current);
            longPressTriggered.current = true;
        }, delay);
    }, [onLongPress, shouldPreventDefault, delay]);

    const move = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        if (longPressTriggered.current || !pressTimeout.current) return;
        // Update last position
        const position = getPosition(event);
        lastPosition.current = position;
        const distance = Math.sqrt(Math.pow(position.x - startPosition.current.x, 2) + Math.pow(position.y - startPosition.current.y, 2));
        // If the distance is too far (i.e. a drag), cancel the press
        if (distance > MAX_TRAVEL_DISTANCE) {
            clearTimeout(pressTimeout.current);
            pressTimeout.current = undefined;
        }
    }, []);

    const clear = useCallback((event: React.MouseEvent | React.TouchEvent, shouldTriggerClick = true) => {
        // Clear pressTimeout and hoverTimeout
        if (pressTimeout.current) {
            clearTimeout(pressTimeout.current);
            pressTimeout.current = undefined;
        }
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            hoverTimeout.current = undefined;
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
                console.log('usepress onrightclick');
                typeof onRightClick === "function" && onRightClick(target.current);
            } else {
                console.log('usepress onclick');
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
        onMouseEnter: e => hover(e),
        onMouseLeave: e => clear(e),
        onMouseMove: e => move(e),
        onMouseUp: e => clear(e),
        onTouchEnd: e => clear(e),
        onTouchMove: e => move(e),
        onTouchStart: e => start(e),
    };
};

export default usePress;