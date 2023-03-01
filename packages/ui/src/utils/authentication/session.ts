import { Session, SessionUser } from "@shared/consts";
import { uuidValidate } from "@shared/uuid";

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