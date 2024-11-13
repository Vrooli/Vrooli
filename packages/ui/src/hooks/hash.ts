import { useEffect, useState } from "react";

/**
 * Hook for detecting changes of window.location.hash.
 */
export function useReactHash() {
    const [hash, setHash] = useState(window.location.hash);
    function listenToPopstate() {
        const winHash = window.location.hash;
        setHash(winHash);
    }
    useEffect(() => {
        window.addEventListener("popstate", listenToPopstate);
        return () => {
            window.removeEventListener("popstate", listenToPopstate);
        };
    }, []);
    return hash;
}

const HASH_SCROLL_MAX_ATTEMPTS = 50;
const HASH_SCROLL_INTERVAL_MS = 100;
const HASH_SCROLL_OFFSET_PX = 100;

/** Finds the nearest scrollable parent container */
function findScrollContainer(element: HTMLElement | null): HTMLElement | Window {
    if (!element) return window;

    let parent = element.parentElement;
    while (parent) {
        const { overflow, overflowY } = window.getComputedStyle(parent);
        if (
            ["auto", "scroll"].includes(overflow) ||
            ["auto", "scroll"].includes(overflowY)
        ) {
            return parent;
        }
        parent = parent.parentElement;
    }
    return window;
}

function getOffsetTopRelativeToContainer(element, container) {
    let offsetTop = 0;
    while (element && element !== container) {
        offsetTop += element.offsetTop;
        element = element.offsetParent;
    }
    return offsetTop;
}

/**
 * Hook that handles scrolling to elements based on URL hash fragments.
 * Implements a polling mechanism to handle dynamically loaded content.
 */
export function useHashScroll(
    maxAttempts = HASH_SCROLL_MAX_ATTEMPTS,
    intervalMs = HASH_SCROLL_INTERVAL_MS,
    offsetPx = HASH_SCROLL_OFFSET_PX,
) {
    const hash = useReactHash();

    useEffect(function scrollToUrlHash() {
        // If no hash, scroll to top
        if (hash === "") {
            window.scrollTo(0, 0);
            return;
        }

        let attempts = 0;
        const id = hash.replace("#", "");

        function checkAndScroll() {
            const element = document.getElementById(id);

            if (element) {
                const scrollContainer = findScrollContainer(element);
                const elementRect = element.getBoundingClientRect();

                // Calculate positions differently for window vs custom container
                if (scrollContainer === window) {
                    const scrollContainerWindow = scrollContainer as Window;
                    const offsetPosition = elementRect.top + window.scrollY - offsetPx;
                    scrollContainerWindow.scrollTo({
                        top: offsetPosition,
                        behavior: "smooth",
                    });
                } else {
                    const scrollContainerElement = scrollContainer as HTMLElement;
                    const offsetPosition = getOffsetTopRelativeToContainer(element, scrollContainerElement) - offsetPx;
                    scrollContainerElement.scrollTo({
                        top: offsetPosition,
                        behavior: "smooth",
                    });
                }
                return true;
            }

            attempts++;
            return false;
        }

        // Try immediately first
        if (checkAndScroll()) return;

        // If not found, start polling
        const intervalId = setInterval(() => {
            if (checkAndScroll() || attempts >= maxAttempts) {
                clearInterval(intervalId);
            }
        }, intervalMs);

        // Cleanup
        return () => clearInterval(intervalId);
    }, [hash, intervalMs, maxAttempts, offsetPx]);
}
