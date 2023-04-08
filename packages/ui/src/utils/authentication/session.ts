import { ActiveFocusMode, FocusMode, Session, SessionUser } from "@shared/consts";
import { uuidValidate } from "@shared/uuid";
import { getCookieActiveFocusMode, getCookieAllFocusModes, getCookieLanguage } from "utils/cookies";
import { getUserLanguages } from "utils/display/translationTools";

/**
 * Session object that indicates no user is logged in
 */
export const guestSession: Session = {
    __typename: 'Session',
    isLoggedIn: false,
    users: [],
}

/**
 * Parses session to find current user's data. 
 * @param session Session object
 * @returns data SessionUser object, or empty values
 */
export const getCurrentUser = (session: Session | null | undefined): Partial<SessionUser> => {
    if (!session || !session.isLoggedIn || !Array.isArray(session.users) || session.users.length === 0) {
        return {}
    }
    const userData = session.users[0];
    // Make sure that user data is valid, by checking ID. 
    // Can add more checks in the future
    return uuidValidate(userData.id) ? userData : {}
}

/**
 * Checks if there is a user logged in. Falls back to flag in local storage 
 * to avoid flickering of UI elements when session is loading. This means 
 * the flag must be updated elsewhere when user logs in or out.
 * @param session Session object
 * @returns True if user is logged in
 */
export const checkIfLoggedIn = (session: Session | null | undefined): boolean => {
    console.log('action checkIfLoggedIn', session, localStorage.getItem('isLoggedIn') === 'true')
    // If there is no session, check local storage
    if (!session) return localStorage.getItem('isLoggedIn') === 'true';
    // Otherwise, check session
    return session.isLoggedIn;
}

/**
 * Languages the site has translations for (2-letter codes).
 * Should be ordered by priority. 
 * 
 * Missing a language you want to use? Consider contributing to the project!
 */
export const siteLanguages = ['en'];

/**
 * Finds which language the site should be displayed in.
 * @param session Session object
 */
export const getSiteLanguage = (session: Session | null | undefined): string => {
    // Try to find languages from session. Make sure not to return default
    const sessionLanguages = getUserLanguages(session, false);
    // Find first language that is in site languages
    const siteLanguage = sessionLanguages.find(language => siteLanguages.includes(language));
    // If found, return it
    if (siteLanguage) return siteLanguage;
    // If no languages found in session, check local storage (i.e. cookies)
    const cookieLanguages = getCookieLanguage();
    // Check if it's a site language
    if (cookieLanguages && siteLanguages.includes(cookieLanguages)) return cookieLanguages;
    // Otherwise, return default (first in array)
    return siteLanguages[0];
}

/**
 * Finds which focus mode the site should be displayed in, as well as 
 * all focus modes of the user
 * @param session Session object
 */
export const getFocusModeInfo = (session: Session | null | undefined): {
    active: ActiveFocusMode | null,
    all: FocusMode[]
} => {
    // Try to find focus modes user from session
    const { activeFocusMode, focusModes } = getCurrentUser(session);
    // If not found, use cookies
    return {
        active: activeFocusMode ?? getCookieActiveFocusMode() ?? null,
        all: focusModes ?? getCookieAllFocusModes() ?? [],
    }
}
