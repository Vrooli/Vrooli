import { useEffect, useRef, useState, useCallback } from "react";
import { stringifySearchParams } from "./searchParams";
const eventPopstate = "popstate";
const eventPushState = "pushState";
const eventReplaceState = "replaceState";
export const events = [eventPopstate, eventPushState, eventReplaceState];
export default ({ base = "" } = {}) => {
    const [{ path, search }, update] = useState(() => ({
        path: currentPathname(base),
        search: location.search,
    }));
    const prevHash = useRef(path + search);
    useEffect(() => {
        const checkForUpdates = () => {
            const pathname = currentPathname(base);
            const search = location.search;
            const hash = pathname + search;
            if (prevHash.current !== hash) {
                prevHash.current = hash;
                update({ path: pathname, search });
            }
        };
        events.forEach((e) => addEventListener(e, checkForUpdates));
        checkForUpdates();
        return () => events.forEach((e) => removeEventListener(e, checkForUpdates));
    }, [base]);
    const setLocation = useCallback((to, { replace = false, searchParams = {} } = {}) => {
        console.log("calling setLocation", to, replace, searchParams);
        const lastPath = sessionStorage.getItem("currentPath");
        const lastSearchParams = sessionStorage.getItem("currentSearchParams");
        const toShaped = to[0] === "~" ? to.slice(1) : to;
        const currPath = toShaped.split("?")[0];
        let currSearchParams = toShaped.split("?")[1];
        if (currSearchParams)
            currSearchParams = `?${currSearchParams}`;
        else if (Object.keys(searchParams).length > 0)
            currSearchParams = stringifySearchParams(searchParams);
        if (lastPath && lastPath !== currPath) {
            console.log("USELOCATION: changing last path", lastPath, currPath);
            sessionStorage.setItem("lastPath", lastPath);
        }
        if (lastSearchParams && lastSearchParams !== currSearchParams) {
            console.log("USELOCATION: changing last search params", lastSearchParams, currSearchParams);
            sessionStorage.setItem("lastSearchParams", lastSearchParams);
        }
        sessionStorage.setItem("currentPath", currPath);
        sessionStorage.setItem("currentSearchParams", currSearchParams ?? JSON.stringify({}));
        console.log("USELOCATION: updating history", currSearchParams ? `${currPath}${currSearchParams}` : currPath);
        history[replace ? eventReplaceState : eventPushState](null, "", currSearchParams ? `${currPath}${currSearchParams}` : currPath);
    }, [base]);
    return [path, setLocation];
};
if (typeof history !== "undefined") {
    for (const type of [eventPushState, eventReplaceState]) {
        const original = history[type];
        history[type] = function () {
            const result = original.apply(this, arguments);
            const event = new Event(type);
            event.arguments = arguments;
            dispatchEvent(event);
            return result;
        };
    }
}
const currentPathname = (base, path = location.pathname) => !path.toLowerCase().indexOf(base.toLowerCase())
    ? path.slice(base.length) || "/"
    : "~" + path;
//# sourceMappingURL=useLocation.js.map