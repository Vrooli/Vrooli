import { RefObject, useCallback, useEffect, useRef, useState } from "react";

type useDraggableScrollProps = {
    ref: RefObject<HTMLElement>;
    options: {
        direction?: "vertical" | "horizontal" | "both";
    };
}

// Taken from: https://github.com/g-delmo/use-draggable-scroll
export default function useDraggableScroll({
    ref,
    options = { direction: "both" },
}: useDraggableScrollProps) {
    if (typeof ref !== "object" || typeof ref.current === "undefined") {
        console.error("`useDraggableScroll` expects a single ref argument.");
    }

    const { direction } = options;

    // The initial position (scroll progress and mouse location) when the mouse is pressed down on the element
    let initialPosition = { scrollTop: 0, scrollLeft: 0, mouseX: 0, mouseY: 0 };

    function mouseMoveHandler(event: { clientX: number; clientY: number }) {
        if (ref.current) {
            // Calculate differences to see how far the user has moved
            const dx = event.clientX - initialPosition.mouseX;
            const dy = event.clientY - initialPosition.mouseY;

            // Scroll the element according to those differences
            if (direction !== "horizontal")
                ref.current.scrollTop = initialPosition.scrollTop - dy;
            if (direction !== "vertical")
                ref.current.scrollLeft = initialPosition.scrollLeft - dx;
        }
    }

    function mouseUpHandler() {
        // Return to cursor: grab after the user is no longer pressing
        if (ref.current) ref.current.style.cursor = "grab";

        // Remove the event listeners since it is not necessary to track the mouse position anymore
        document.removeEventListener("mousemove", mouseMoveHandler);
        document.removeEventListener("mouseup", mouseUpHandler);
    }

    function onMouseDown(event: { clientX: number; clientY: number }) {
        if (ref.current) {
            // Save the position at the moment the user presses down
            initialPosition = {
                scrollLeft: ref.current.scrollLeft,
                scrollTop: ref.current.scrollTop,
                mouseX: event.clientX,
                mouseY: event.clientY,
            };

            // Show a cursor: grabbing style and set user-select: none to avoid highlighting text while dragging
            ref.current.style.cursor = "grabbing";
            ref.current.style.userSelect = "none";

            // Add the event listeners that will track the mouse position for the rest of the interaction
            document.addEventListener("mousemove", mouseMoveHandler);
            document.addEventListener("mouseup", mouseUpHandler);
        }
    }

    return { onMouseDown };
}

const DEFAULT_INFINITE_SCROLL_THRESHOLD_PX = 500;

interface UseInfiniteScrollProps {
    /** If we're currently loading more items, in which case we don't want to trigger another load */
    loading: boolean;
    /** Callback to load more items */
    loadMore: () => unknown;
    /** ID of the container to listen for scroll events on */
    scrollContainerId: string;
    /** The distance from the bottom of the container at which to trigger the loadMore callback */
    threshold?: number;
}

/**
 * Tracks when to load more items in an infinite scroll list
 */
export function useInfiniteScroll({
    loading,
    loadMore,
    scrollContainerId,
    threshold = DEFAULT_INFINITE_SCROLL_THRESHOLD_PX,
}: UseInfiniteScrollProps) {
    const handleScroll = useCallback(() => {
        const scrollContainer = document.getElementById(scrollContainerId);
        if (!scrollContainer) {
            console.error(`Could not find scrolling container for id ${scrollContainerId} - infinite scroll disabled`);
            return;
        }

        const scrolledY = scrollContainer.scrollTop;
        const scrollableHeight = scrollContainer.scrollHeight - scrollContainer.clientHeight;

        if (
            !loading &&
            scrollableHeight > 0 &&
            (scrolledY > scrollableHeight - threshold)
        ) {
            loadMore();
        }
    }, [loading, loadMore, scrollContainerId, threshold]);

    useEffect(() => {
        const scrollingContainer = document.getElementById(scrollContainerId);

        if (scrollingContainer) {
            scrollingContainer.addEventListener("scroll", handleScroll);
            return () => scrollingContainer.removeEventListener("scroll", handleScroll);
        } else {
            console.error(`Could not find scrolling container for id ${scrollContainerId} - infinite scroll disabled`);
        }
    }, [handleScroll, scrollContainerId]);

    return null;
}

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
export function usePinchZoom({
    onScaleChange,
    validTargetIds = [],
}: UsePinchZoomProps): UsePinchZoomReturn {
    const [isPinching, setIsPinching] = useState(false);
    const refs = useRef<PinchRefs>({
        currDistance: 0,
        lastDistance: 0,
    });
    // Wait ref so we can update every k iterations
    const waitRef = useRef<number>(0);

    useEffect(() => {
        function getTouchDistance(e: TouchEvent) {
            const touch1 = e.touches[0];
            const touch2 = e.touches[1];
            return Math.sqrt(Math.pow(touch1.clientX - touch2.clientX, 2) + Math.pow(touch1.clientY - touch2.clientY, 2));
        }
        function handleTouchStart(e: TouchEvent) {
            // Find the target
            const targetId = (e as any)?.target?.id;
            if (!targetId) return;
            if (!validTargetIds.some(id => targetId.startsWith(id))) return;
            // Pinch requires two touches
            if (e.touches.length !== 2) return;
            setIsPinching(true);
            refs.current.currDistance = getTouchDistance(e);
            refs.current.lastDistance = refs.current.currDistance;
        }
        function handleTouchMove(e: TouchEvent) {
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
        }
        function handleTouchEnd(e: TouchEvent) {
            if (e.touches.length === 0) {
                setIsPinching(false);
                refs.current.currDistance = 0;
                refs.current.lastDistance = 0;
            }
        }
        function handleWheel(e: WheelEvent) {
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
        };
    }, [onScaleChange, isPinching, validTargetIds]);

    return {
        isPinching,
    };
}

/**
 * Maximum travel distance allowed before a press is cancelled
 */
const MAX_TRAVEL_DISTANCE = 10;
const DEFAULT_PRESS_DELAY = 300;
const DEFAULT_HOVER_DELAY = 900;
const EVENT_TIME_THRESHOLD = 20;

/** Partial event data, since callbacks can be triggered by different event types */
export type UsePressEvent = {
    shiftKey: boolean;
    target: EventTarget;
}

type UsePressProps = {
    onLongPress: (event: UsePressEvent) => unknown;
    onClick?: (event: UsePressEvent) => unknown;
    onHover?: (event: UsePressEvent) => unknown;
    onHoverEnd?: (event: UsePressEvent) => unknown;
    onRightClick?: (event: UsePressEvent) => unknown;
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
    // Store event data
    const target = useRef<React.MouseEvent["target"]>();
    const shiftKey = useRef<boolean>(false);
    // Stores if click was a right click
    const isRightClick = useRef<boolean>(false);
    // Stores if object is currently being pressed
    const isPressing = useRef<boolean>(false);
    // Stores if object is currently being hovered
    const isHovering = useRef<boolean>(false);
    // Stores if object is currently being dragged
    const isDragging = useRef<boolean>(false);

    const hover = useCallback(function handleHoverCallback(event: React.MouseEvent | React.TouchEvent) {
        // Ignore if pressing or already hovered
        if (isPressing.current || isHovering.current) return;
        // Cancel if already triggered
        if (hoverTimeout.current) {
            clearTimeout(hoverTimeout.current);
            hoverTimeout.current = undefined;
        }
        // Set timeout. Hover delay is longer than press delay
        hoverTimeout.current = setTimeout(() => {
            if (target.current) onHover?.({
                shiftKey: false, // Hover events don't have shift key
                target: target.current,
            });
            isHovering.current = true;
        }, hoverDelay);
        // Store target
        target.current = event.target ?? event.currentTarget as React.MouseEvent["target"];
    }, [onHover, hoverDelay]);

    const start = useCallback(function handleStartCallback(event: React.MouseEvent | React.TouchEvent) {
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
        // Store event information
        target.current = event.target ?? event.currentTarget as React.MouseEvent["target"];
        shiftKey.current = event.shiftKey;
        // Store position
        const currentPosition = getPosition(event);
        startPosition.current = currentPosition;
        lastPosition.current = currentPosition;
        // Start pressTimeout to determine if long press
        pressTimeout.current = setTimeout(() => {
            if (!longPressTriggered.current && target.current) onLongPress({
                shiftKey: shiftKey.current,
                target: target.current,
            });
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
            if (target.current) onHoverEnd?.({
                shiftKey: false, // Hover events don't have shift key
                target: target.current,
            });
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
                typeof onRightClick === "function" && onRightClick({
                    shiftKey: shiftKey.current,
                    target: target.current,
                });
            } else {
                typeof onClick === "function" && onClick({
                    shiftKey: shiftKey.current,
                    target: target.current,
                });
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
