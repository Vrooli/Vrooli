import { useEffect, useState } from "react";

/**
 * Hook for detecting changes of window.location.hash.
 */
export const useReactHash = () => {
    const [hash, setHash] = useState(window.location.hash);
    const listenToPopstate = () => {
        const winHash = window.location.hash;
        setHash(winHash);
    };
    useEffect(() => {
        window.addEventListener("popstate", listenToPopstate);
        return () => {
            window.removeEventListener("popstate", listenToPopstate);
        };
    }, []);
    return hash;
};