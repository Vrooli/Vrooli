import { useEffect, useRef, useState, useCallback } from "react";
import { stringifySearchParams } from "./searchParams";

export type Path = string;

export type SetLocationOptions = {
    replace?: boolean;
    searchParams?: Record<string, any>;
};

export type SetLocation = (to: Path, options?: SetLocationOptions) => void;

export type UseLocationResult = [Path, SetLocation];

export type UseLocationHook = (options?: {
    base?: Path;
}) => UseLocationResult;

// Returns the type of the navigation options that hook's push function accepts.
export type HookNavigationOptions =
    SetLocation extends (path: Path, options: infer R, ...rest: any[]) => any
    ? R extends { [k: string]: any }
    ? R
    : {}
    : {};

// Returns the type of the navigation options that hook's push function accepts.
export type LocationHook = (options?: {
    base?: Path;
}) => [Path, (to: Path, options?: { replace?: boolean }) => void];

/**
 * History API docs @see https://developer.mozilla.org/en-US/docs/Web/API/History
 */
const eventPopstate = "popstate";
const eventPushState = "pushState";
const eventReplaceState = "replaceState";
export const events = [eventPopstate, eventPushState, eventReplaceState];

export default ({ base = "" } = {}): [Path, SetLocation] => {
    const [{ path, search }, update] = useState(() => ({
        path: currentPathname(base),
        search: location.search,
    })); // @see https://reactjs.org/docs/hooks-reference.html#lazy-initial-state
    const prevHash = useRef(path + search);

    // // When path changes, store last path in session storage.
    // // This can be useful for components like dialogs that have 
    // // conditional logic based on if the user navigated back to the page
    // useEffect(() => {
    //     // Get last stored data in sessionStorage
    //     const lastPath = sessionStorage.getItem("currentPath");
    //     console.log("LAST PATH", lastPath);
    //     console.log('CURRENT PATH', path)
    //     // Store last data in sessionStorage
    //     if (lastPath) sessionStorage.setItem("lastPath", lastPath);
    //     // Store current data in sessionStorage
    //     sessionStorage.setItem("currentPath", path);
    // }, [path]);

    // // Also store search params in sessionStorage
    // useEffect(() => {
    //     // Get last stored data in sessionStorage
    //     const lastSearchParams = sessionStorage.getItem("currentSearchParams");
    //     console.log("LAST SEARCH PARAMS", lastSearchParams);
    //     console.log('CURRENT SEARCH PARAMS', search)
    //     // Store last data in sessionStorage
    //     if (lastSearchParams) sessionStorage.setItem("lastSearchParams", lastSearchParams);
    //     // Store current data in sessionStorage
    //     sessionStorage.setItem("currentSearchParams", search);
    // }, [search]);

    useEffect(() => {
        // this function checks if the location has been changed since the
        // last render and updates the state only when needed.
        // unfortunately, we can't rely on `path` value here, since it can be stale,
        // that's why we store the last pathname in a ref.
        // const checkForUpdates = () => {
        //     const pathname = currentPathname(base);
        //     const search = location.search;
        //     // If previous path is different from current path, update sessionStorage
        //     let hasPrevPathChanged = prevPath.current !== pathname;
        //     if (hasPrevPathChanged) {
        //         console.log("PREV PATH", prevPath.current, "CURRENT PATH", pathname);
        //         prevPath.current = pathname;
        //         // Update last path in sessionStorage
        //         sessionStorage.setItem("lastPath", prevPath.current);
        //         // Update current path in sessionStorage
        //         sessionStorage.setItem("currentPath", pathname);
        //     }
        //     // If previous search params are different from current search params, update sessionStorage
        //     let hasPrevSearchChanged = prevSearch.current !== search;
        //     if (hasPrevSearchChanged) {
        //         console.log("PREV SEARCH PARAMS", prevSearch.current, "CURRENT SEARCH PARAMS", search);
        //         prevSearch.current = search;
        //         // Update last search params in sessionStorage
        //         sessionStorage.setItem("lastSearchParams", prevSearch.current);
        //         // Update current search params in sessionStorage
        //         sessionStorage.setItem("currentSearchParams", search);
        //     }
        //     // If either path or search params have changed, update the state
        //     if (hasPrevPathChanged || hasPrevSearchChanged) {
        //         update({ path: pathname, search });
        //     }
        // };
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

        // it's possible that an update has occurred between render and the effect handler,
        // so we run additional check on mount to catch these updates. Based on:
        // https://gist.github.com/bvaughn/e25397f70e8c65b0ae0d7c90b731b189
        checkForUpdates();

        return () => events.forEach((e) => removeEventListener(e, checkForUpdates));
    }, [base]);

    // the 2nd argument of the `useLocation` return value is a function
    // that allows to perform a navigation.
    //
    // the function reference should stay the same between re-renders, so that
    // it can be passed down as an element prop without any performance concerns.
    const setLocation = useCallback((to: Path, { replace = false, searchParams = {} } = {}) => {
        console.log('calling setLocation', to, replace, searchParams)
        // Get last path and search params from sessionStorage
        const lastPath = sessionStorage.getItem("currentPath");
        const lastSearchParams = sessionStorage.getItem("currentSearchParams");
        // Get current path and search params. "to" might contain search params, 
        // so we need to split it
        const toShaped = to[0] === "~" ? to.slice(1) : to;
        const currPath = toShaped.split('?')[0];
        let currSearchParams: string | undefined = toShaped.split('?')[1];
        if (currSearchParams) currSearchParams = `?${currSearchParams}`;
        else if (Object.keys(searchParams).length > 0) currSearchParams = stringifySearchParams(searchParams);
        // Store last data in sessionStorage
        if (lastPath && lastPath !== currPath) {
            console.log('USELOCATION: changing last path', lastPath, currPath)
            sessionStorage.setItem("lastPath", lastPath);
        }
        if (lastSearchParams && lastSearchParams !== currSearchParams) {
            console.log('USELOCATION: changing last search params', lastSearchParams, currSearchParams)
            sessionStorage.setItem("lastSearchParams", lastSearchParams);
        }
        // Store current data in sessionStorage
        sessionStorage.setItem("currentPath", currPath);
        sessionStorage.setItem("currentSearchParams", currSearchParams ?? JSON.stringify({}));
        // Update history
        console.log('USELOCATION: updating history', currSearchParams ? `${currPath}${currSearchParams}` : currPath)
        history[replace ? eventReplaceState : eventPushState](
            null,
            "",
            // Combine path and search params
            currSearchParams ? `${currPath}${currSearchParams}` : currPath
        )
    }, [base]);

    return [path, setLocation];
};

// While History API does have `popstate` event, the only
// proper way to listen to changes via `push/replaceState`
// is to monkey-patch these methods.
//
// See https://stackoverflow.com/a/4585031
if (typeof history !== "undefined") {
    for (const type of [eventPushState, eventReplaceState]) {
        const original = history[type as keyof History];

        (history as any)[type as keyof History] = function () {
            const result = original.apply(this, arguments);
            const event: Event & { arguments?: any } = new Event(type);
            event.arguments = arguments;

            dispatchEvent(event);
            return result;
        };
    }
}

const currentPathname = (base: any, path = location.pathname) =>
    !path.toLowerCase().indexOf(base.toLowerCase())
        ? path.slice(base.length) || "/"
        : "~" + path;