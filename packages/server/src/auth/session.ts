import { DEFAULT_LANGUAGE, generatePK, type Session, type SessionUser } from "@vrooli/shared";
import { type Request } from "express";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { PasswordAuthService, type UserDataForPasswordAuth } from "./email.js";
import { REFRESH_TOKEN_EXPIRATION_MS } from "./jwt.js";
import { RequestService } from "./request.js";

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
     * Creates and records a new session in the database.
     * @param userId The ID of the user for the session.
     * @param authId The ID of the authentication record used.
     * @param req The Express request object.
     * @returns The created session data from Prisma.
     */
    static async createAndRecordSession(userId: string, authId: string, req: Request) {
        const deviceInfo = RequestService.getDeviceInfo(req);
        const ipAddress = req.ip;
        const sessionData = await DbProvider.get().session.create({
            data: {
                id: generatePK(),
                device_info: deviceInfo,
                expires_at: new Date(Date.now() + REFRESH_TOKEN_EXPIRATION_MS),
                last_refresh_at: new Date(),
                ip_address: ipAddress,
                revokedAt: null,
                user: { connect: { id: BigInt(userId) } },
                auth: { connect: { id: BigInt(authId) } },
            },
            // Use the select statement consistent with other parts of the auth logic
            select: PasswordAuthService.selectUserForPasswordAuth().sessions.select,
        });
        // Prisma might return BigInts, ensure the return type matches PrismaSessionData if necessary
        return sessionData;
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
    static async createUserSession(
        userData: Omit<UserDataForPasswordAuth, "auths" | "emails" | "sessions">,
        sessionData: UserDataForPasswordAuth["sessions"][0],
    ): Promise<SessionUser> {
        if (!userData || !sessionData) {
            throw new CustomError("0510", "NotFound", { userData });
        }

        // Check if user has any verified phone numbers
        const phoneNumberVerified = userData.phones?.some(phone => phone.verifiedAt !== null) ?? false;

        // Check if user has received the phone verification reward
        const hasReceivedPhoneVerificationReward = userData.plan?.receivedFreeTrialAt !== null;

        // Return shaped SessionUser object
        const result: SessionUser = {
            __typename: "SessionUser" as const,
            id: userData.id.toString(),
            publicId: userData.publicId,
            credits: (userData.creditAccount?.currentBalance ?? BigInt(0)) + "", // Convert to string because BigInt can't be serialized
            creditAccountId: userData.creditAccount ? userData.creditAccount.id.toString() : undefined,
            handle: userData.handle ?? undefined,
            hasPremium: new Date(userData.plan?.expiresAt ?? 0) > new Date(),
            hasReceivedPhoneVerificationReward,
            languages: userData.languages,
            name: userData.name,
            phoneNumberVerified,
            profileImage: userData.profileImage,
            session: {
                __typename: "SessionUserSession" as const,
                id: sessionData.id.toString(),
                lastRefreshAt: sessionData.last_refresh_at.toISOString(),
            },
            theme: userData.theme,
            updatedAt: userData.updatedAt.toISOString(),
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

