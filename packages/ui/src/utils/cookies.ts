import { COOKIE, ValueOf } from "@shared/consts";
import { getDeviceInfo } from "./display";

/**
 * Handles storing and retrieving cookies, which may or 
 * may not be permitted by the user's cookie preferences
 */
export const Cookies = {
    ...COOKIE,
    Preferences: "cookiePreferences",
    Theme: 'theme',
    FontSize: 'fontSize',
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

export const getCookie = <T>(name: Cookies, typeCheck: (value: any) => value is T): T | null => {
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
    return null;
}

export const setCookie = (name: Cookies, value: any) => {
    localStorage.setItem(name, JSON.stringify(value));
}

export const getCookiePreferences = (): CookiePreferences | null => {
    // Standalone apps don't set preferences
    if(getDeviceInfo().isStandalone) {
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

export const setCookiePreferences = (preferences: CookiePreferences) => {
    // Standalone apps don't set preferences
    if(getDeviceInfo().isStandalone) return;
    setCookie(Cookies.Preferences, preferences);
}

export const getCookieTheme = (): 'light' | 'dark' | null => {
    // Only get theme if strictly necessary cookies are allowed
    const preferences = getCookiePreferences();
    if (!preferences?.strictlyNecessary) return null;
    return getCookie(Cookies.Theme, (value: any): value is 'light' | 'dark' => value === 'light' || value === 'dark');
}

export const setCookieTheme = (theme: 'light' | 'dark') => {
    // Only set theme if strictly necessary cookies are allowed
    console.log('in setting cookie theme', theme);
    const preferences = getCookiePreferences();
    console.log('preferences', preferences);
    if (!preferences?.strictlyNecessary) return;
    setCookie(Cookies.Theme, theme);
}

export const getCookieFontSize = (): number | null => {
    // Only get font size if strictly necessary cookies are allowed
    const preferences = getCookiePreferences();
    if (!preferences?.strictlyNecessary) return null;
    return getCookie(Cookies.FontSize, (value: any): value is number => typeof value === 'number');
}

export const setCookieFontSize = (fontSize: number) => {
    // Only set font size if strictly necessary cookies are allowed
    const preferences = getCookiePreferences();
    if (!preferences?.strictlyNecessary) return;
    setCookie(Cookies.FontSize, fontSize);
}