import React, { useCallback, useRef } from "react";

/**
 * Maximum travel distance allowed before a press is cancelled
 */
const MAX_TRAVEL_DISTANCE = 10;
const DEFAULT_PRESS_DELAY = 300;
const DEFAULT_HOVER_DELAY = 900;
const EVENT_TIME_THRESHOLD = 20;

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

function isTouchEvent(event: React.MouseEvent | React.TouchEvent): event is React.TouchEvent {
    return "touches" in event;
}
function isMouseEvent(event: React.MouseEvent | React.TouchEvent): event is React.MouseEvent {
    return "button" in event;
}

function preventDefaultTouch(event: React.TouchEvent) {
    if (event.touches.length < 2 && event.preventDefault) {
        event.preventDefault();
    }
}

/**
 * Determines the position of the click or touch event
 * @param event The event to get the position of
 * @returns The position of the event
 */
function getPosition(event: React.MouseEvent | React.TouchEvent): { x: number, y: number } {
    if (isTouchEvent(event)) {
        const touch = event.touches[0];
        return { x: touch.clientX, y: touch.clientY };
    } else {
        return { x: event.clientX, y: event.clientY };
    }
}

let lastStartEvent = 0;
let lastStopEvent = 0;

/**
 * Determines if the start/stop event has happened sufficiently longer than 
 * the last start/stop event. This is to prevent double triggers for devices 
 * that send both mouse and touch events.
 * @param event The event to check
 * @param eventType The type of event
 * @returns True if the event is sufficiently far from the last event
 */
function isNewEvent(event: React.MouseEvent | React.TouchEvent, eventType: "start" | "stop"): boolean {
    // Check if event is sufficiently far from last event
    const lastEvent = eventType === "start" ? lastStartEvent : lastStopEvent;
    let isNewEvent = event.timeStamp - lastEvent > EVENT_TIME_THRESHOLD;
    // Update last event time
    if (eventType === "start") {
        lastStartEvent = event.timeStamp;
        // Additionally for start events, make sure stop event hasn't happened recently
        isNewEvent = isNewEvent && event.timeStamp - lastStopEvent > EVENT_TIME_THRESHOLD;
    }
    if (eventType === "stop") {
        lastStopEvent = event.timeStamp;
    }
    return isNewEvent;
}

/**
 * Triggered when its parent is long clicked or pressed. 
 * Also supports short clicks.
 */
export function usePress({
    onLongPress,
    onClick,
    onHover,
    onHoverEnd,
    onRightClick,
    shouldPreventDefault = true,
    pressDelay = DEFAULT_PRESS_DELAY,
    hoverDelay = DEFAULT_HOVER_DELAY,
}: UsePressProps): UsePressReturn {
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
        // Ignore if start event has already been triggered
        if (!isNewEvent(event, "start")) return;
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

    const stop = useCallback((event: React.MouseEvent | React.TouchEvent, shouldTriggerClick = true) => {
        // Ignore if stop event has already been triggered
        if (!isNewEvent(event, "stop")) return;
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
        onMouseLeave: e => stop(e, false),
        onMouseMove: e => move(e),
        onMouseUp: e => stop(e, true),
        onTouchEnd: e => stop(e, true),
        onTouchMove: e => move(e),
        onTouchStart: e => start(e),
    };
}

export default usePress;
