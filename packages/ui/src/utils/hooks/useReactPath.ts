import { useEffect, useState } from "react";

/**
 * Hook for detecting changes of window.location.pathname.
 */
export const useReactPath = () => {
    const [path, setPath] = useState(window.location.pathname);
    const listenToPopstate = () => {
        console.log('POP STATE', window.location.pathname);
        const winPath = window.location.pathname;
        setPath(winPath);
    };
    useEffect(() => {
        window.addEventListener("popstate", listenToPopstate);
        return () => {
            window.removeEventListener("popstate", listenToPopstate);
        };
    }, []);
    return path;
};