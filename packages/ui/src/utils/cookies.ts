import { ActiveFocusMode, COOKIE, FocusMode, ValueOf } from "@shared/consts";
import { exists } from "@shared/utils";
import { getDeviceInfo } from "./display/device";

/**
 * Handles storing and retrieving cookies, which may or 
 * may not be permitted by the user's cookie preferences
 */
export const Cookies = {
    ...COOKIE,
    Preferences: 'cookiePreferences',
    Theme: 'theme',
    FontSize: 'fontSize',
    Language: 'language',
    IsLeftHanded: 'isLeftHanded',
    FocusModeActive: 'focusModeActive',
    FocusModeAll: 'focusModeAll',
}
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

export const getCookie = <T>(name: Cookies, typeCheck: (value: any) => value is T): T | undefined => {
    const cookie = localStorage.getItem(name);
    // Try to parse
    try {
        const parsed = JSON.parse(cookie ?? '');
        if (typeCheck(parsed)) {
            return parsed;
        }
    } catch (e) {
        console.warn(`Failed to parse cookie ${name}`, cookie);
    }
    return undefined;
}

export const setCookie = (name: Cookies, value: any) => {
    localStorage.setItem(name, JSON.stringify(value));
}

/**
 * Gets a cookie if it exists, otherwise sets it to the default value. 
 * Assumes that you have already checked that the cookie is allowed.
 */
export const getOrSetCookie = <T>(name: Cookies, typeCheck: (value: any) => value is T, defaultValue?: T): T | undefined => {
    const cookie = getCookie(name, typeCheck);
    if (exists(cookie)) return cookie;
    if (exists(defaultValue)) setCookie(name, defaultValue);
    return defaultValue;
}

/**
 * Finds the user's cookie preferences.
 * @returns CookiePreferences object, or null if not set
 */
export const getCookiePreferences = (): CookiePreferences | null => {
    // Standalone apps don't set preferences
    if (getDeviceInfo().isStandalone) {
        return {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        }
    }
    const cookie = getCookie(Cookies.Preferences, (value: any): value is CookiePreferences => typeof value === 'object');
    if (!cookie) return null;
    return {
        strictlyNecessary: cookie.strictlyNecessary || true,
        performance: cookie.performance || false,
        functional: cookie.functional || false,
        targeting: cookie.targeting || false,
    }
}

/**
 * Sets the user's cookie preferences.
 * @param preferences CookiePreferences object
 */
export const setCookiePreferences = (preferences: CookiePreferences) => {
    // Standalone apps don't set preferences
    if (getDeviceInfo().isStandalone) return;
    setCookie(Cookies.Preferences, preferences);
}

/**
 * Sets a cookie only if the user has permitted the cookie's type. 
 * For strictly necessary cookies it will be set regardless of user 
 * preferences.
 * @param cookieType Cookie type to check
 * @param callback Callback function to call if cookie is allowed
 */
export const onlyIfCookieAllowed = (cookieType: keyof CookiePreferences, callback: () => any) => {
    const preferences = getCookiePreferences();
    if (cookieType === 'strictlyNecessary' || preferences?.[cookieType]) {
        return callback();
    }
    else {
        console.warn(`Not allowed to get/set cookie ${cookieType}`);
    }
}

type ThemeType = 'light' | 'dark';
export const getCookieTheme = <T extends ThemeType | undefined>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () =>
        getOrSetCookie(Cookies.Theme, (value: any): value is ThemeType => value === 'light' || value === 'dark', fallback));

export const setCookieTheme = (theme: ThemeType) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.Theme, theme))

export const getCookieFontSize = <T extends number | undefined>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () => {
        const size = getOrSetCookie(Cookies.FontSize, (value: any): value is number => typeof value === 'number', fallback);
        // Ensure font size is not too small or too large. This would make the UI unusable.
        return size ? Math.max(8, Math.min(24, size)) : undefined;
    })

export const setCookieFontSize = (fontSize: number) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.FontSize, fontSize))

export const getCookieLanguage = <T extends string | undefined>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () => getOrSetCookie(Cookies.Language, (value: any): value is string => typeof value === 'string', fallback));

export const setCookieLanguage = (language: string) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.Language, language));

export const getCookieIsLeftHanded = <T extends boolean | undefined>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () => getOrSetCookie(Cookies.IsLeftHanded, (value: any): value is boolean => typeof value === 'boolean', fallback));

export const setCookieIsLeftHanded = (isLeftHanded: boolean) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.IsLeftHanded, isLeftHanded));

export const getCookieActiveFocusMode = <T extends ActiveFocusMode | undefined>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () => getOrSetCookie(Cookies.FocusModeActive, (value: any): value is ActiveFocusMode => typeof value === 'object', fallback));

export const setCookieActiveFocusMode = (focusMode: ActiveFocusMode | null) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.FocusModeActive, focusMode));

export const getCookieAllFocusModes = <T extends FocusMode[]>(fallback?: T): T =>
    onlyIfCookieAllowed('functional', () => getOrSetCookie(Cookies.FocusModeAll, (value: any): value is FocusMode[] => Array.isArray(value), fallback));

export const setCookieAllFocusModes = (modes: FocusMode[]) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.FocusModeAll, modes));