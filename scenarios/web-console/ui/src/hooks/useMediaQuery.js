import { useEffect, useState } from "react";
function getMatch(query) {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
        return false;
    }
    return window.matchMedia(query).matches;
}
export function useMediaQuery(query) {
    const [matches, setMatches] = useState(() => getMatch(query));
    useEffect(() => {
        if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
            return;
        }
        const mediaQuery = window.matchMedia(query);
        const handleChange = (event) => {
            setMatches(event.matches);
        };
        setMatches(mediaQuery.matches);
        mediaQuery.addEventListener("change", handleChange);
        return () => mediaQuery.removeEventListener("change", handleChange);
    }, [query]);
    return matches;
}
