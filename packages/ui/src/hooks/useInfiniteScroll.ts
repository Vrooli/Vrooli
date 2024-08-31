import { useCallback, useEffect } from "react";

const DEFAULT_THRESHOLD_PX = 500;

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
    threshold = DEFAULT_THRESHOLD_PX,
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
