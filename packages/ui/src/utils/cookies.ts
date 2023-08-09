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
export const ifAllowed = (cookieType: keyof CookiePreferences, callback: () => unknown, fallback?: any) => {
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
export const getCookiePartialData = <T extends PartialData>(knownData: T): T =>
    ifAllowed("functional",
        () => {
            const shapedKnownData = shapeKnownData(knownData);
            console.log("getCookiePartialData shapedKnownData", shapedKnownData);
            // If known data does not contain an id or handle, return known data
            if (
                !shapedKnownData?.id &&
                !shapedKnownData?.handle &&
                !shapedKnownData?.root?.id &&
                !shapedKnownData?.root?.handle
            ) return shapedKnownData;
            // Find stored data
            const storedData = getOrSetCookie(Cookies.PartialData, (value: unknown): value is PartialData => typeof value === "object");
            // If stored data matches known data (i.e. same type and id or handle), return stored data
            if (storedData?.__typename === shapedKnownData?.__typename && (
                storedData?.id === shapedKnownData?.id ||
                storedData?.handle === shapedKnownData?.handle ||
                storedData?.root?.id === shapedKnownData?.root?.id ||
                storedData?.root?.handle === shapedKnownData?.root?.handle
            )) {
                return storedData;
            }
            // Otherwise return known data
            return shapedKnownData;
        },
        shapeKnownData(knownData),
    );
export const setCookiePartialData = (partialData: PartialData) => ifAllowed("functional", () => setCookie(Cookies.PartialData, partialData));
