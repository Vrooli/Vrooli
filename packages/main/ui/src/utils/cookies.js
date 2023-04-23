import { COOKIE } from "@local/consts";
import { exists } from "@local/utils";
import { getDeviceInfo } from "./display/device";
export const Cookies = {
    ...COOKIE,
    Preferences: "cookiePreferences",
    Theme: "theme",
    FontSize: "fontSize",
    Language: "language",
    IsLeftHanded: "isLeftHanded",
    FocusModeActive: "focusModeActive",
    FocusModeAll: "focusModeAll",
};
export const getCookie = (name, typeCheck) => {
    const cookie = localStorage.getItem(name);
    try {
        const parsed = JSON.parse(cookie ?? "");
        if (typeCheck(parsed)) {
            return parsed;
        }
    }
    catch (e) {
        console.warn(`Failed to parse cookie ${name}`, cookie);
    }
    return undefined;
};
export const setCookie = (name, value) => {
    localStorage.setItem(name, JSON.stringify(value));
};
export const getOrSetCookie = (name, typeCheck, defaultValue) => {
    const cookie = getCookie(name, typeCheck);
    if (exists(cookie))
        return cookie;
    if (exists(defaultValue))
        setCookie(name, defaultValue);
    return defaultValue;
};
export const getCookiePreferences = () => {
    if (getDeviceInfo().isStandalone) {
        return {
            strictlyNecessary: true,
            performance: true,
            functional: true,
            targeting: true,
        };
    }
    const cookie = getCookie(Cookies.Preferences, (value) => typeof value === "object");
    if (!cookie)
        return null;
    return {
        strictlyNecessary: cookie.strictlyNecessary || true,
        performance: cookie.performance || false,
        functional: cookie.functional || false,
        targeting: cookie.targeting || false,
    };
};
export const setCookiePreferences = (preferences) => {
    if (getDeviceInfo().isStandalone)
        return;
    setCookie(Cookies.Preferences, preferences);
};
export const onlyIfCookieAllowed = (cookieType, callback) => {
    const preferences = getCookiePreferences();
    if (cookieType === "strictlyNecessary" || preferences?.[cookieType]) {
        return callback();
    }
    else {
        console.warn(`Not allowed to get/set cookie ${cookieType}`);
    }
};
export const getCookieTheme = (fallback) => onlyIfCookieAllowed("functional", () => getOrSetCookie(Cookies.Theme, (value) => value === "light" || value === "dark", fallback));
export const setCookieTheme = (theme) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.Theme, theme));
export const getCookieFontSize = (fallback) => onlyIfCookieAllowed("functional", () => {
    const size = getOrSetCookie(Cookies.FontSize, (value) => typeof value === "number", fallback);
    return size ? Math.max(8, Math.min(24, size)) : undefined;
});
export const setCookieFontSize = (fontSize) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.FontSize, fontSize));
export const getCookieLanguage = (fallback) => onlyIfCookieAllowed("functional", () => getOrSetCookie(Cookies.Language, (value) => typeof value === "string", fallback));
export const setCookieLanguage = (language) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.Language, language));
export const getCookieIsLeftHanded = (fallback) => onlyIfCookieAllowed("functional", () => getOrSetCookie(Cookies.IsLeftHanded, (value) => typeof value === "boolean", fallback));
export const setCookieIsLeftHanded = (isLeftHanded) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.IsLeftHanded, isLeftHanded));
export const getCookieActiveFocusMode = (fallback) => onlyIfCookieAllowed("functional", () => getOrSetCookie(Cookies.FocusModeActive, (value) => typeof value === "object", fallback));
export const setCookieActiveFocusMode = (focusMode) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.FocusModeActive, focusMode));
export const getCookieAllFocusModes = (fallback) => onlyIfCookieAllowed("functional", () => getOrSetCookie(Cookies.FocusModeAll, (value) => Array.isArray(value), fallback));
export const setCookieAllFocusModes = (modes) => onlyIfCookieAllowed("functional", () => setCookie(Cookies.FocusModeAll, modes));
//# sourceMappingURL=cookies.js.map