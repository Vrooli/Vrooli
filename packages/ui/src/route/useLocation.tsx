import { stringifySearchParams } from "@vrooli/shared";
import { useCallback, useEffect, useRef, useState } from "react";
import { type SetLocation } from "./types.js";

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

// Storage key for mocked location in Storybook
const STORYBOOK_MOCK_PATH_KEY = "vrooli_storybook_mock_path";

/**
 * Mocks the location for Storybook testing
 * @param mockPath The path to mock (e.g. '/chat')
 */
export function mockLocationForStorybook(mockPath: string): void {
    if (typeof window !== "undefined" && process.env.DEV) {
        localStorage.setItem(STORYBOOK_MOCK_PATH_KEY, mockPath);
        // Trigger a popstate event to update any hooks listening for location changes
        window.dispatchEvent(new Event(eventPopstate));
    }
}

/**
 * Clears any mocked location for Storybook testing
 */
export function clearMockedLocationForStorybook(): void {
    if (typeof window !== "undefined" && process.env.DEV) {
        localStorage.removeItem(STORYBOOK_MOCK_PATH_KEY);
        // Trigger a popstate event to update any hooks listening for location changes
        window.dispatchEvent(new Event(eventPopstate));
    }
}

/**
 * Gets the current location, checking for Storybook mocked path first
 */
function getCurrentLocation(): Location {
    // In development mode and Storybook, check if we have a mocked path
    if (typeof window !== "undefined" && process.env.DEV) {
        const mockedPath = localStorage.getItem(STORYBOOK_MOCK_PATH_KEY);
        if (mockedPath) {
            // Only override the pathname, keep the other location properties
            return {
                href: location.href,
                pathname: mockedPath,
                search: location.search,
            };
        }
    }

    // Default to actual location
    return {
        href: location.href,
        pathname: location.pathname ?? "",
        search: location.search,
    };
}

export default function useLocation(): UseLocationResult {
    const [loc, setLoc] = useState<Location>(() => getCurrentLocation());
    const prevHref = useRef(location.href);

    useEffect(() => {
        function checkForUpdates() {
            const currentLocation = getCurrentLocation();
            const { href } = currentLocation;

            if (prevHref.current !== href) {
                prevHref.current = href;
                setLoc(currentLocation);
            } else {
                // Even if the href hasn't changed, the mocked path might have
                setLoc(currentLocation);
            }
        }

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
        // If we're in Storybook and have a mocked path, update that instead
        if (typeof window !== "undefined" && process.env.DEV && localStorage.getItem(STORYBOOK_MOCK_PATH_KEY)) {
            localStorage.setItem(STORYBOOK_MOCK_PATH_KEY, to.split("?")[0]);
            // Trigger a location update without affecting actual browser history
            window.dispatchEvent(new Event(eventPopstate));
            return;
        }

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

        (history as any)[type as keyof History] = function monkeyPatchedHistory(...args: any[]) {
            const result = original.apply(this, args);
            const event: Event & { arguments?: any } = new Event(type);
            event.arguments = args;

            dispatchEvent(event);
            return result;
        };
    }
}
