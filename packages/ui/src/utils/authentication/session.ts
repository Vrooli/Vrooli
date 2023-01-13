import { GqlModelType, Session, SessionUser } from "@shared/consts";
import { uuidValidate } from "@shared/uuid";

/**
 * Session object that indicates no user is logged in
 */
export const guestSession: Session = {
    type: GqlModelType.Session,
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