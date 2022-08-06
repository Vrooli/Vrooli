import { useCallback, useEffect, useState } from "react";
import { parseSearchParams } from "utils/navigation";

type Primitive = string | number | boolean;
type UseReactSearchResults = { [x: string]: Primitive | Primitive[] | UseReactSearchResults };

/**
 * Hook for detecting changes of window.location.search.
 * @param pollInterval How often to check for changes in window.location.search. 
 * This can only be accomplished with polling, so try not to rely on 
 * this hook for too many things.
 */
export const useReactSearch = (pollInterval: number | null = 50) => {
    const [search, setSearch] = useState<UseReactSearchResults>(parseSearchParams(window.location.search));
    const listenToSearch = useCallback(() => {
        const newSearch = parseSearchParams(window.location.search);
        if (JSON.stringify(newSearch) !== JSON.stringify(search)) {
            setSearch(newSearch);
        }
    }, [search]);
    useEffect(() => {
        if (pollInterval) {
            // Create interval to listen to search changes
            const interval = setInterval(listenToSearch, 50);
            // Clean up interval when component unmounts
            return () => clearInterval(interval);
        }
    }, [listenToSearch, pollInterval])
    return search;
};