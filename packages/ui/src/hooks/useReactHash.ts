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
