/**
 * This handles handles storing and retrieving "cookies", which may or
 * may not be permitted by the user's cookie preferences. I say "cookies" in 
 * quotes because they are not actually cookies, but rather localStorage. It is 
 * unclear whether EU's Cookie Law applies to localStorage, but it is better to
 * be safe than sorry.
 */
import { ActiveFocusMode, COOKIE, exists, FocusMode, ValueOf } from "@local/shared";
import { NavigableObject } from "types";

/**
 * Handles storing and retrieving cookies, which may or 
 * may not be permitted by the user's cookie preferences
 */
export const Cookies = {
    ...COOKIE,
    PartialData: "partialData", // Used to store partial data for the page before the full data is loaded
    Preferences: "cookiePreferences",
    Theme: "theme",
    FontSize: "fontSize",
    Language: "language",
    IsLeftHanded: "isLeftHanded",
    FocusModeActive: "focusModeActive",
    FocusModeAll: "focusModeAll",
};
export type Cookies = ValueOf<typeof Cookies>;

/**
 * Preferences for the user's cookie settings
 */
export type CookiePreferences = {
    strictlyNecessary: boolean;
    performance: boolean;
    functional: boolean;
    targeting: boolean;
}

const getCookie = <T>(name: Cookies, typeCheck: (value: unknown) => value is T): T | undefined => {
    const cookie = localStorage.getItem(name);
    // Try to parse
    try {
        const parsed = JSON.parse(cookie ?? "");
        if (typeCheck(parsed)) {
            return parsed;
        }
    } catch (e) {
        console.warn(`Failed to parse cookie ${name}`, cookie);
    }
    return undefined;
};

const setCookie = (name: Cookies, value: unknown) => { localStorage.setItem(name, JSON.stringify(value)); };

/**
 * Gets a cookie if it exists, otherwise sets it to the default value. 
 * Assumes that you have already checked that the cookie is allowed.
 */
export const getOrSetCookie = <T>(name: Cookies, typeCheck: (value: unknown) => value is T, defaultValue?: T): T | undefined => {
    const cookie = getCookie(name, typeCheck);
    if (exists(cookie)) return cookie;
    if (exists(defaultValue)) setCookie(name, defaultValue);
    return defaultValue;
};

/**
 * Finds the user's cookie preferences.
 * @returns CookiePreferences object, or null if not set
 */
export const getCookiePreferences = (): CookiePreferences | null => {
    const cookie = getCookie(Cookies.Preferences, (value: unknown): value is CookiePreferences => typeof value === "object");
    if (!cookie) return null;
    return {
        strictlyNecessary: cookie.strictlyNecessary || true,
        performance: cookie.performance || false,
        functional: cookie.functional || false,
        targeting: cookie.targeting || false,
    };
};

/**
 * Sets the user's cookie preferences.
 * @param preferences CookiePreferences object
 */
export const setCookiePreferences = (preferences: CookiePreferences) => {
    setCookie(Cookies.Preferences, preferences);
};

/**
 * Sets a cookie only if the user has permitted the cookie's type. 
 * For strictly necessary cookies it will be set regardless of user 
 * preferences.
 * @param cookieType Cookie type to check
 * @param callback Callback function to call if cookie is allowed
 * @param fallback Optional fallback value to use if cookie is not allowed
 */
const ifAllowed = (cookieType: keyof CookiePreferences, callback: () => unknown, fallback?: any) => {
    const preferences = getCookiePreferences();
    if (cookieType === "strictlyNecessary" || preferences?.[cookieType]) {
        return callback();
    }
    else {
        console.warn(`Not allowed to get/set cookie ${cookieType}`);
        return fallback;
    }
};

type ThemeType = "light" | "dark";
export const getCookieTheme = <T extends ThemeType | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.Theme, (value: unknown): value is ThemeType => value === "light" || value === "dark", fallback),
        fallback,
    );
export const setCookieTheme = (theme: ThemeType) => ifAllowed("functional", () => setCookie(Cookies.Theme, theme));

export const getCookieFontSize = <T extends number | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => {
            const size = getOrSetCookie(Cookies.FontSize, (value: unknown): value is number => typeof value === "number", fallback);
            // Ensure font size is not too small or too large. This would make the UI unusable.
            return size ? Math.max(8, Math.min(24, size)) : undefined;
        },
        fallback,
    );
export const setCookieFontSize = (fontSize: number) => ifAllowed("functional", () => setCookie(Cookies.FontSize, fontSize));

export const getCookieLanguage = <T extends string | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.Language, (value: unknown): value is string => typeof value === "string", fallback),
        fallback,
    );
export const setCookieLanguage = (language: string) => ifAllowed("functional", () => setCookie(Cookies.Language, language));

export const getCookieIsLeftHanded = <T extends boolean | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.IsLeftHanded, (value: unknown): value is boolean => typeof value === "boolean", fallback),
        fallback,
    );
export const setCookieIsLeftHanded = (isLeftHanded: boolean) => ifAllowed("functional", () => setCookie(Cookies.IsLeftHanded, isLeftHanded));

export const getCookieActiveFocusMode = <T extends ActiveFocusMode | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.FocusModeActive, (value: unknown): value is ActiveFocusMode => typeof value === "object", fallback),
        fallback,
    );
export const setCookieActiveFocusMode = (focusMode: ActiveFocusMode | null) => ifAllowed("functional", () => setCookie(Cookies.FocusModeActive, focusMode));

export const getCookieAllFocusModes = <T extends FocusMode[]>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.FocusModeAll, (value: unknown): value is FocusMode[] => Array.isArray(value), fallback),
        fallback,
    );
export const setCookieAllFocusModes = (modes: FocusMode[]) => ifAllowed("functional", () => setCookie(Cookies.FocusModeAll, modes));

/** Supports ID data from URL params, as well as partial object */
type PartialData = {
    __typename: NavigableObject["__typename"],
    id?: string | null,
    idRoot?: string | null,
    handle?: string | null
    handleRoot?: string | null,
    root?: {
        id?: string | null,
        handle?: string | null,
    } | null,
};
/** Type of data stored */
type DataType = "list" | "full";
/** 
 * Stores multiple cached objects, in a way that allows 
 * for removing the oldest object when the cache is full.
 * 
 * Objects can be cached using their ID or handle, or root's ID or handle
 */
type Cache = {
    idMap: Record<string, PartialData>,
    handleMap: Record<string, PartialData>,
    idRootMap: Record<string, PartialData>,
    handleRootMap: Record<string, PartialData>,
    order: string[], // Order of objects in cache, for FIFO
};

const MAX_CACHE_SIZE = 300;

/** Get object containing all cached objects */
const getCache = (): Cache => getOrSetCookie(
    Cookies.PartialData, // Cookie name
    (value: unknown): value is Cache =>
        typeof value === "object" &&
        typeof (value as Partial<Cache>)?.idMap === "object" &&
        typeof (value as Partial<Cache>)?.handleMap === "object" &&
        typeof (value as Partial<Cache>)?.idRootMap === "object" &&
        typeof (value as Partial<Cache>)?.handleRootMap === "object" &&
        Array.isArray((value as Partial<Cache>)?.order),
    {
        idMap: {},
        handleMap: {},
        idRootMap: {},
        handleRootMap: {},
        order: [],
    }, // Default value
) as Cache;

/** Shape knownData to replace idRoot and handleRoot with proper root object */
const shapeKnownData = <T extends PartialData>(knownData: T): T => ({
    ...knownData,
    idRoot: undefined,
    handleRoot: undefined,
    ...(knownData.idRoot || knownData.handleRoot ? {
        root: {
            ...knownData.root,
            id: knownData.idRoot,
            handle: knownData.handleRoot,
        },
    } : {}),
});
export const getCookiePartialData = <T extends PartialData>(knownData: PartialData): T =>
    ifAllowed("functional",
        () => {
            const shapedKnownData = shapeKnownData(knownData);
            // Get the cache, which hopefully includes more info on the requested object
            const cache = getCache();
            // Try to find stored data in cache
            const storedData = cache.idMap[shapedKnownData.id || ""] ||
                cache.handleMap[shapedKnownData.handle || ""] ||
                cache.idRootMap[shapedKnownData.root?.id || ""] ||
                cache.handleRootMap[shapedKnownData.root?.handle || ""];
            // Return match if found
            if (storedData && storedData.__typename === shapedKnownData.__typename) {
                return storedData;
            }
            // Otherwise return known data
            return shapedKnownData;
        },
        shapeKnownData(knownData),
    );
export const setCookiePartialData = (partialData: PartialData, dataType: DataType) => ifAllowed("functional", () => {
    ifAllowed("functional", () => {
        // Get the cache
        const cache = getCache();
        // Create a key that includes every identifier for the object
        const key = `${partialData.id}|${partialData.handle}|${partialData.root?.id}|${partialData.root?.handle}`;
        // Check if the object is already in the cache
        const isAlreadyInCache = Boolean((partialData.id && cache.idMap[partialData.id]) ||
            (partialData.handle && cache.handleMap[partialData.handle]) ||
            (partialData.root?.id && cache.idRootMap[partialData.root.id]) ||
            (partialData.root?.handle && cache.handleRootMap[partialData.root.handle]));
        // If data type is "list", don't overwrite the existing cached data
        console.log("setcookiepartialdata", dataType, isAlreadyInCache);
        if (isAlreadyInCache && dataType === "list") {
            return;
        }
        // Store the object in the cache, even if it already existed. Store in every map that applies, since we don't know which one will be needed
        if (partialData.id) cache.idMap[partialData.id] = partialData;
        if (partialData.handle) cache.handleMap[partialData.handle] = partialData;
        if (partialData.root?.id) cache.idRootMap[partialData.root.id] = partialData;
        if (partialData.root?.handle) cache.handleRootMap[partialData.root.handle] = partialData;
        // If the object was already in the cache, update the order so it's at the end
        if (isAlreadyInCache) {
            // Find the existing key in the order array
            const existingIndex = cache.order.indexOf(key);
            if (existingIndex !== -1) {
                // Remove the existing key
                cache.order.splice(existingIndex, 1);
                // Add it back to the end
                cache.order.push(key);
            }
            setCookie(Cookies.PartialData, cache);
            return;
        }
        // If the cache is full, remove the oldest object
        if (cache.order.length >= MAX_CACHE_SIZE) {
            const oldestKey = cache.order.shift();
            if (oldestKey && typeof oldestKey === "string") {
                const [id, handle, idRoot, handleRoot] = oldestKey.split("|");
                delete cache.idMap[id];
                delete cache.handleMap[handle];
                delete cache.idRootMap[idRoot];
                delete cache.handleRootMap[handleRoot];
            }
        }
        // Push the new object to the end of the FIFO order
        cache.order.push(key);

        setCookie(Cookies.PartialData, cache);
    });
});
