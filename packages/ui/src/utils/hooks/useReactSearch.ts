import { useEffect, useState } from "react";
import { parseSearchParams } from "utils/navigation";

/**
 * Hook for detecting changes of window.location.search.
 */
export const useReactSearch = () => {
    const [search, setSearch] = useState<{[x: string]: string}>(parseSearchParams(window.location.search));
    const listenToPopstate = () => {
        const winSearch = parseSearchParams(window.location.search);
        setSearch(winSearch);
    };
    useEffect(() => {
        window.addEventListener("popstate", listenToPopstate);
        return () => {
            window.removeEventListener("popstate", listenToPopstate);
        };
    }, []);
    return search;
};