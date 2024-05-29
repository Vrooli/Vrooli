import { stringifySearchParams } from "@local/shared";
import { useCallback, useEffect, useRef, useState } from "react";
import { SetLocation } from "./types";

export type Href = string;
export type Pathname = string;
export type Search = string;
export type Location = {
    href: Href,
    pathname: Pathname,
    search: Search
};
export type UseLocationResult = [Location, SetLocation];
export type UseLocationHook = () => UseLocationResult;
/** Returns the type of the navigation options that hook's push function accepts. */
export type HookNavigationOptions =
    SetLocation extends (to: string, options: infer R, ...rest: any[]) => any
    ? R extends { [k: string]: any }
    ? R
    : object
    : object;

/**
 * History API docs @see https://developer.mozilla.org/en-US/docs/Web/API/History
 */
const eventPopstate = "popstate";
const eventPushState = "pushState";
const eventReplaceState = "replaceState";
export const events = [eventPopstate, eventPushState, eventReplaceState];

export default function useLocation(): UseLocationResult {
    const [loc, setLoc] = useState<Location>(() => ({
        href: location.href,
        pathname: location.pathname ?? "",
        search: location.search,
    })); // @see https://reactjs.org/docs/hooks-reference.html#lazy-initial-state
    const prevHref = useRef(location.href);

    useEffect(() => {
        const checkForUpdates = () => {
            const { href, pathname, search } = location;

            if (prevHref.current !== href) {
                prevHref.current = href;
                setLoc({ href, pathname: pathname ?? "", search });
            }
        };

        events.forEach((e) => addEventListener(e, checkForUpdates));

        // it's possible that an update has occurred between render and the effect handler,
        // so we run additional check on mount to catch these updates. Based on:
        // https://gist.github.com/bvaughn/e25397f70e8c65b0ae0d7c90b731b189
        checkForUpdates();

        return () => events.forEach((e) => removeEventListener(e, checkForUpdates));
    }, []);

    // the 2nd argument of the `useLocation` return value is a function
    // that allows to perform a navigation.
    //
    // the function reference should stay the same between re-renders, so that
    // it can be passed down as an element prop without any performance concerns.
    const setLocation = useCallback((to: string, { replace = false, searchParams = {} } = {}) => {
        // Get last path and search params from sessionStorage
        const lastPath = sessionStorage.getItem("currentPath");
        const lastSearchParams = sessionStorage.getItem("currentSearchParams");
        // Get current path and search params. "to" might contain search params, 
        // so we need to split it
        const currPath = to.split("?")[0];
        let currSearchParams: string | undefined = to.split("?")[1];
        if (currSearchParams) currSearchParams = `?${currSearchParams}`;
        else if (Object.keys(searchParams).length > 0) currSearchParams = stringifySearchParams(searchParams);
        // Store last data in sessionStorage
        if (lastPath && lastPath !== currPath) {
            sessionStorage.setItem("lastPath", lastPath);
        }
        if (lastSearchParams && lastSearchParams !== currSearchParams) {
            sessionStorage.setItem("lastSearchParams", lastSearchParams);
        }
        // Store current data in sessionStorage
        sessionStorage.setItem("currentPath", currPath);
        sessionStorage.setItem("currentSearchParams", currSearchParams ?? JSON.stringify({}));
        // Update history
        history[replace ? eventReplaceState : eventPushState](
            null,
            "",
            currSearchParams ? `${currPath}${currSearchParams}` : currPath,
        );
        // Update location
        const { href, pathname, search } = location;
        setLoc({ href, pathname: pathname ?? "", search });
    }, []);

    return [loc, setLocation];
}

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
