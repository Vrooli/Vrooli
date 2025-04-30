import { DEFAULT_LANGUAGE, Session, SessionUser } from "@local/shared";
import { Request } from "express";
import { CustomError } from "../events/error.js";
import { UserDataForPasswordAuth } from "./email.js";

export class SessionService {
    private static instance: SessionService;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): SessionService {
        if (!SessionService.instance) {
            SessionService.instance = new SessionService();
        }
        return SessionService.instance;
    }

    /**
    * Finds current user in Request object. Also validates that the user data is valid
    * @param session The Request's session property
    * @returns First userId in Session object, or null if not found/invalid
    */
    static getUser<User extends { id: string }>(req: { session: { users?: User[] | null | undefined, userId?: string | null | undefined } }): User | null {
        if (typeof req !== "object" || req === null) {
            return null;
        }
        const { session } = req;
        if (typeof session !== "object" || session === null) {
            return null;
        }
        if (session.userId && typeof session.userId === "string") {
            return { id: session.userId, languages: [DEFAULT_LANGUAGE] } as unknown as User;
        }
        if (Array.isArray(session.users) && session.users.length > 0) {
            const user = session.users[0];
            if (user !== undefined && typeof user.id === "string") {
                return user;
            }

        }
        return null;
    }

    /**
     * Creates SessionUser object from user.
     * @param userData User data
     * @param session Current session data
     */
    static async createUserSession(userData: Omit<UserDataForPasswordAuth, "auths" | "emails" | "sessions">, sessionData: UserDataForPasswordAuth["sessions"][0]): Promise<SessionUser> {
        if (!userData || !sessionData) {
            throw new CustomError("0510", "NotFound", { userData });
        }
        // Return shaped SessionUser object
        const result: SessionUser = {
            __typename: "SessionUser" as const,
            id: userData.id,
            credits: (userData.premium?.credits ?? BigInt(0)) + "", // Convert to string because BigInt can't be serialized
            handle: userData.handle ?? undefined,
            hasPremium: new Date(userData.premium?.expiresAt ?? 0) > new Date(),
            languages: userData.languages.map(l => l.language),
            name: userData.name,
            profileImage: userData.profileImage,
            session: {
                __typename: "SessionUserSession" as const,
                id: sessionData.id,
                lastRefreshAt: sessionData.last_refresh_at,
            },
            theme: userData.theme,
            updatedAt: userData.updatedAt,
        };
        return result;
    }

    static createBaseSession(req: Request): Session {
        return {
            __typename: "Session" as const,
            isLoggedIn: req.session?.isLoggedIn ?? false,
            timeZone: req.session?.timeZone,
            users: req.session?.users ?? [],
        };
    }

    /**
     * Creates session object from user and existing session data
     * @param userData User data
     * @param session Current session data
     * @param req Express request object (with current session data)
     * @returns Updated session object, with user data added to the START of the users array
     */
    static async createSession(userData: UserDataForPasswordAuth, sessionData: UserDataForPasswordAuth["sessions"][0], req: Request): Promise<Session> {
        const sessionUser = await SessionService.createUserSession(userData, sessionData);
        return {
            ...this.createBaseSession(req),
            isLoggedIn: true,
            // Make sure users are unique by id
            users: [sessionUser, ...(req.session?.users ?? []).filter((u: SessionUser) => u.id !== sessionUser.id)],
        };
    }
}

