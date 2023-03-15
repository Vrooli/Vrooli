import { COOKIE, UserSchedule, ValueOf } from "@shared/consts";
import { getDeviceInfo } from "./display";

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
    Schedule: 'schedule',
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

export const getCookieTheme = (): 'light' | 'dark' | null =>
    onlyIfCookieAllowed('functional', () =>
        getCookie(Cookies.Theme, (value: any): value is 'light' | 'dark' => value === 'light' || value === 'dark'));

export const setCookieTheme = (theme: 'light' | 'dark') =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.Theme, theme))

export const getCookieFontSize = (): number | null =>
    onlyIfCookieAllowed('functional', () => {
        const size = getCookie(Cookies.FontSize, (value: any): value is number => typeof value === 'number');
        // Ensure font size is not too small or too large. This would make the UI unusable.
        return size ? Math.max(8, Math.min(24, size)) : null;
    })

export const setCookieFontSize = (fontSize: number) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.FontSize, fontSize))

export const getCookieLanguage = (): string | null =>
    onlyIfCookieAllowed('functional', () => getCookie(Cookies.Language, (value: any): value is string => typeof value === 'string'));

export const setCookieLanguage = (language: string) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.Language, language));

export const getCookieIsLeftHanded = (): boolean | null =>
    onlyIfCookieAllowed('functional', () => getCookie(Cookies.IsLeftHanded, (value: any): value is boolean => typeof value === 'boolean'));

export const setCookieIsLeftHanded = (isLeftHanded: boolean) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.IsLeftHanded, isLeftHanded));

type ScheduleCookie = {
    scheduleId: string;
    stopWhen: 'automatic' | 'manual' | 'endOfCustomTime' | 'nextScheduleStart';
    stopTime: number;
}

export const getCookieSchedule = (): ScheduleCookie | null =>
    onlyIfCookieAllowed('functional', () => getCookie(Cookies.Schedule, (value: any): value is UserSchedule => typeof value === 'object'));

export const setCookieSchedule = (schedule: ScheduleCookie) =>
    onlyIfCookieAllowed('functional', () => setCookie(Cookies.Schedule, schedule));