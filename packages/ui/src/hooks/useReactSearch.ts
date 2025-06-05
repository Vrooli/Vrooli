import { parseSearchParams } from "@vrooli/shared";
import { useCallback, useEffect, useState } from "react";

type Primitive = string | number | boolean | object | null | undefined;
type UseReactSearchResults = { [x: string]: Primitive | Primitive[] | UseReactSearchResults };

const DEFAULT_POLL_INTERVAL = 100;

/**
 * Hook for detecting changes of window.location.search.
 * @param pollInterval How often to check for changes in window.location.search. 
 * This can only be accomplished with polling, so try not to rely on 
 * this hook for too many things.
 */
export function useReactSearch(pollInterval: number | null = DEFAULT_POLL_INTERVAL) {
    const [search, setSearch] = useState<UseReactSearchResults>(parseSearchParams());
    const listenToSearch = useCallback(() => {
        const newSearch = parseSearchParams();
        if (JSON.stringify(newSearch) !== JSON.stringify(search)) {
            setSearch(newSearch);
        }
    }, [search]);
    useEffect(() => {
        if (pollInterval) {
            // Create interval to listen to search changes
            const interval = setInterval(listenToSearch, pollInterval);
            // Clean up interval when component unmounts
            return () => clearInterval(interval);
        }
    }, [listenToSearch, pollInterval]);
    return search;
}
