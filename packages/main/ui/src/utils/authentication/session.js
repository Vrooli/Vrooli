import { uuidValidate } from "@local/uuid";
import { getCookieActiveFocusMode, getCookieAllFocusModes, getCookieLanguage } from "../cookies";
import { getUserLanguages } from "../display/translationTools";
export const guestSession = {
    __typename: "Session",
    isLoggedIn: false,
    users: [],
};
export const getCurrentUser = (session) => {
    if (!session || !session.isLoggedIn || !Array.isArray(session.users) || session.users.length === 0) {
        return {};
    }
    const userData = session.users[0];
    return uuidValidate(userData.id) ? userData : {};
};
export const checkIfLoggedIn = (session) => {
    if (!session)
        return localStorage.getItem("isLoggedIn") === "true";
    return session.isLoggedIn;
};
export const siteLanguages = ["en"];
export const getSiteLanguage = (session) => {
    const sessionLanguages = getUserLanguages(session, false);
    const siteLanguage = sessionLanguages.find(language => siteLanguages.includes(language));
    if (siteLanguage)
        return siteLanguage;
    const cookieLanguages = getCookieLanguage();
    if (cookieLanguages && siteLanguages.includes(cookieLanguages))
        return cookieLanguages;
    return siteLanguages[0];
};
export const getFocusModeInfo = (session) => {
    const { activeFocusMode, focusModes } = getCurrentUser(session);
    return {
        active: activeFocusMode ?? getCookieActiveFocusMode() ?? null,
        all: focusModes ?? getCookieAllFocusModes() ?? [],
    };
};
//# sourceMappingURL=session.js.map