import React, { useCallback, useRef } from "react";

interface UsePressProps {
    onLongPress: (target: EventTarget) => unknown;
    onClick?: (target: EventTarget) => unknown;
    onHover?: (target: EventTarget) => unknown;
    onHoverEnd?: (target: EventTarget) => unknown;
    onRightClick?: (target: EventTarget) => unknown;
    shouldPreventDefault?: boolean;
    pressDelay?: number;
    hoverDelay?: number;
}

type UsePressReturn = {
    onMouseDown: (event: React.MouseEvent) => unknown;
    onMouseEnter: (event: React.MouseEvent) => unknown;
    onMouseLeave: (event: React.MouseEvent) => unknown;
    onMouseMove: (event: React.MouseEvent) => unknown;
    onMouseUp: (event: React.MouseEvent) => unknown;
    onTouchEnd: (event: React.TouchEvent) => unknown;
    onTouchMove: (event: React.TouchEvent) => unknown;
    onTouchStart: (event: React.TouchEvent) => unknown;
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
    onHoverEnd,
    onRightClick,
    shouldPreventDefault = true,
    pressDelay = 300,
    hoverDelay = 900,
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
    const target = useRef<React.MouseEvent["target"]>();
    // Stores if click was a right click
    const isRightClick = useRef<boolean>(false);
    // Stores if object is currently being pressed
    const isPressing = useRef<boolean>(false);
    // Stores if object is currently being hovered
    const isHovering = useRef<boolean>(false);
    // Stores if object is currently being dragged
    const isDragging = useRef<boolean>(false);

    const hover = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        // Ignore if pressing or already hovered
        if (isPressing.current || isHovering.current) return;
        // Cancel if already triggered
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            hoverTimeout.current = undefined;
        }
        // Set timeout. Hover delay is longer than press delay
        hoverTimeout.current = setTimeout(() => {
            if (target.current) onHover?.(target.current);
            isHovering.current = true;
        }, hoverDelay);
        // Store target
        target.current = event.target ?? event.currentTarget as React.MouseEvent["target"];
    }, [onHover, hoverDelay]);

    const start = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        // Cancel hover timeout
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            pressTimeout.current = undefined;
        }
        // Set isPressing to true
        isPressing.current = true;
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
        target.current = event.target ?? event.currentTarget as React.MouseEvent["target"];
        // Store position
        const currentPosition = getPosition(event);
        startPosition.current = currentPosition;
        lastPosition.current = currentPosition;
        // Start pressTimeout to determine if long press
        pressTimeout.current = setTimeout(() => {
            if (!longPressTriggered.current && target.current) onLongPress(target.current);
            longPressTriggered.current = true;
        }, pressDelay);
    }, [onLongPress, shouldPreventDefault, pressDelay]);

    const move = useCallback((event: React.MouseEvent | React.TouchEvent) => {
        if (longPressTriggered.current || !pressTimeout.current) return;
        // Update last position
        const position = getPosition(event);
        lastPosition.current = position;
        const distance = Math.sqrt(Math.pow(position.x - startPosition.current.x, 2) + Math.pow(position.y - startPosition.current.y, 2));
        // If the distance is too far (i.e. a drag), cancel the press and set isDragging to true
        if (distance > MAX_TRAVEL_DISTANCE) {
            clearTimeout(pressTimeout.current);
            pressTimeout.current = undefined;
            isDragging.current = true;
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
        // If hover was triggered, trigger hoverEnd
        if (isHovering.current) {
            if (target.current) onHoverEnd?.(target.current);
            isHovering.current = false;
        }
        // Calculate distance travelled
        const travelDistance = Math.sqrt(
            Math.pow(lastPosition.current.x - startPosition.current.x, 2) +
            Math.pow(lastPosition.current.y - startPosition.current.y, 2),
        );
        // Check if short click or right click
        if (
            isPressing.current &&
            longPressTriggered.current === false &&
            travelDistance < MAX_TRAVEL_DISTANCE &&
            shouldTriggerClick &&
            !longPressTriggered.current &&
            !isDragging.current &&
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
        isPressing.current = false;
        isDragging.current = false;
        if (shouldPreventDefault && target.current && isTouchEvent(event)) {
            target.current.removeEventListener("touchend", preventDefaultTouch as any);
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
