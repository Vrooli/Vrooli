import { COOKIE, HttpStatus, mergeDeep } from "@local/shared";
import cookie from "cookie";
import { NextFunction, Request, Response } from "express";
import { type JwtPayload } from "jsonwebtoken";
import { Socket } from "socket.io";
import { ExtendedError } from "socket.io/dist/namespace";
import { DefaultEventsMap } from "socket.io/dist/typed-events";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { ApiToken, RecursivePartial, RecursivePartialNullable, SessionData, SessionToken, SessionUserToken } from "../types";
import { ACCESS_TOKEN_EXPIRATION_MS, JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "./jwt";
import { RequestService } from "./request";
import { SessionService } from "./session";

type AuthenticateTokenConfig = {
    /**
     * Additional data to be added to the token
     */
    additionalData?: object;
    /**
     * Callback to modify the payload. 
     * Has same result as additionalData, but allows for more complex modifications 
     * like respecting the token size limit.
     */
    modifyPayload?: (payload: SessionToken) => SessionToken;
}

type AuthenticateTokenResult = {
    /**
     * How long the token is still valid for, in milliseconds
     */
    maxAge: number;
    /**
     * The token payload
     */
    payload: SessionToken;
    /**
     * The signed token
     */
    token: string;
}

export class AuthTokensService {
    private static instance: AuthTokensService;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): AuthTokensService {
        if (!AuthTokensService.instance) {
            AuthTokensService.instance = new AuthTokensService();
        }
        return AuthTokensService.instance;
    }

    /**
     * Checks if a token can be refreshed
     * @param payload The access token payload
     * @returns True if the token can be refreshed
     */
    static async canRefreshToken(payload: SessionToken): Promise<boolean> {
        // Check that payload has correct fields
        if (typeof payload !== "object") {
            return false;
        }
        const user = SessionService.getUser(payload);
        if (!user) {
            return false;
        }
        const sessionId = user.session?.id;
        const lastRefreshAt = user.session?.lastRefreshAt; // An ISO string
        if (!sessionId || !lastRefreshAt) {
            return false;
        }
        // Fetch session from database
        const storedSession = await prismaInstance.session.findUnique({
            where: { id: sessionId },
            select: {
                last_refresh_at: true,
                revoked: true,
            },
        });
        // Make sure session exists
        if (!storedSession) {
            return false;
        }
        // Make sure session is not revoked
        if (storedSession.revoked) {
            return false;
        }
        // Make sure refresh times match
        const storedLastRefreshAt = storedSession?.last_refresh_at?.getTime(); // Set using new Date() (i.e. a number)
        // Compare the dates
        if (storedLastRefreshAt !== new Date(lastRefreshAt).getTime()) {
            return false;
        }
        // Make sure stored refresh time is less than REFRESH_TOKEN_EXPIRATION_MS
        const currentTime = Date.now();
        const timeSinceRefresh = currentTime - storedLastRefreshAt;
        if (timeSinceRefresh > REFRESH_TOKEN_EXPIRATION_MS) {
            return false;
        }
        return true;
    }

    /**
     * Checks if a user is authenticated. 
     * Handles verifying access tokens and refreshing them if necessary.
     * @param token The access token from the request or socket
     * @param config Configuration options for the token
     * @returns The existing or new signed token, as well as its payload
     * @throws Error if the token is invalid
     */
    static async authenticateToken(token: string, config?: AuthenticateTokenConfig): Promise<AuthenticateTokenResult> {
        function getPayloadWithAdditionalData(payload: SessionToken) {
            let result = payload;
            if (config?.additionalData) {
                result = mergeDeep(payload, config.additionalData) as SessionToken;
            }
            if (config?.modifyPayload) {
                result = config.modifyPayload(result);
            }
            return result;
        }
        const hasAdditionalDataModifer = Boolean(config?.additionalData || config?.modifyPayload);

        // Check if token is the proper type
        if (token === null || token === undefined || typeof token !== "string") {
            throw new Error("NoSessionToken");
        }
        // Check the JWT signature
        let payload: SessionToken | null = null;
        let isExpired = false;
        try {
            // Attempt to verify the token
            payload = (await JsonWebToken.get().verify(token)) as SessionToken;
        } catch (error) {
            const isTokenExpired = (error as { name: string })?.name === "TokenExpiredError";
            if (isTokenExpired) {
                isExpired = true;
                payload = JsonWebToken.get().decode(token) as SessionToken;
            }
            if (!payload) {
                throw new CustomError("0574", "ErrorUnknown");
            }
        }
        // Return early if the token is not expired
        if (!isExpired) {
            const maxAge = JsonWebToken.getMaxAge(payload);
            // If we're adding additional data, sign a new token without extending the expiration
            if (hasAdditionalDataModifer) {
                const withAdditionalData = getPayloadWithAdditionalData(payload);
                const newToken = JsonWebToken.get().sign(withAdditionalData, true);
                return {
                    maxAge,
                    payload: withAdditionalData,
                    token: newToken,
                }
            }
            // Otherwise, return the existing token
            return {
                maxAge,
                payload,
                token,
            }
        }
        // Try to refresh the token if it's expired
        const isRefreshable = await this.canRefreshToken(payload);
        if (!isRefreshable) {
            throw new CustomError("0573", "SessionExpired");
        }
        // Issue a new access token
        const withAdditionalData = getPayloadWithAdditionalData(payload);
        const newToken = JsonWebToken.get().sign(withAdditionalData);
        return {
            maxAge: ACCESS_TOKEN_EXPIRATION_MS,
            payload: withAdditionalData,
            token: newToken,
        }
    }

    /**
     * Generates a JSON Web Token (JWT) for API authentication.
     * @param res 
     * @param apiToken
     */
    static async generateApiToken(res: Response, apiToken: string): Promise<void> {
        const tokenContents: ApiToken = {
            ...JsonWebToken.get().basicToken(),
            apiToken,
        };
        const token = JsonWebToken.get().sign(tokenContents);
        if (!res.headersSent) {
            res.cookie(COOKIE.Jwt, token, JsonWebToken.getJwtCookieOptions());
        }
    }

    /**
     * Generates a JSON Web Token (JWT) for user authentication (including guest access).
     * The token is added to the "res" object as a cookie.
     * @param res 
     * @param session 
     */
    static async generateSessionToken(
        res: Response,
        session: RecursivePartialNullable<Pick<SessionData, "isLoggedIn" | "languages" | "timeZone" | "users">>,
    ): Promise<void> {
        // Start with minimal token data
        let payload: SessionToken = {
            ...JsonWebToken.get().basicToken(),
            isLoggedIn: session.isLoggedIn ?? false,
            timeZone: session.timeZone ?? undefined,
            users: [],
        };

        // Make sure users are unique
        const users = [...new Map((session.users ?? []).map((user) => [user.id, user])).values()];
        payload.users = users;

        // Add users to token payload
        payload = modifyPayloadToFitUsers(payload);

        // Create token
        const token = JsonWebToken.get().sign(payload);

        if (!res.headersSent) {
            res.cookie(COOKIE.Jwt, token, JsonWebToken.getJwtCookieOptions());
        }
    }
}

export class AuthService {
    private static instance: AuthService;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    static get(): AuthService {
        if (!AuthService.instance) {
            AuthService.instance = new AuthService();
        }
        return AuthService.instance;
    }

    /**
     * Verifies if a user is authenticated, using an http cookie. 
     * Also populates helpful request properties, which can be used by endpoints
     * @param req The request object
     * @param res The response object. Has to be passed in, but it's not used.
     * @param next The next function to call.
     */
    static async authenticateRequest(req: Request, res: Response, next: NextFunction) {
        try {
            const { cookies } = req;
            // Initialize session
            req.session = {
                fromSafeOrigin: RequestService.get().isSafeOrigin(req),
                languages: RequestService.parseAcceptLanguage(req),
            };
            // Get token
            const { payload } = await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt]);
            updateSessionWithTokenPayload(req, payload);
            next();
        } catch (error) {
            res.clearCookie(COOKIE.Jwt);
            // If from unsafe origin, deny access.
            if (!req.session.fromSafeOrigin) {
                logger.error("Error authenticating request", { trace: "0451", error });
                res.status(HttpStatus.Unauthorized).json({ error: "UnsafeOrigin" });
            }
            // Otherwise, we'll continue with the request. The endpoints themselves can 
            // determine if they'll let a signed out user access them.
            else {
                next();
            }
        }
    }

    /**
     * Middleware to authenticate a user via JWT in a WebSocket handshake.
     *
     * This function parses the JWT from the cookies in the handshake, verifies it, 
     * and if the verification is successful, attaches the decoded JWT payload 
     * to the socket object as `socket.session`.
     *
     * @param socket - The socket object
     * @param next - The next function to call
     */
    static async authenticateSocket(
        socket: Socket<DefaultEventsMap, DefaultEventsMap, DefaultEventsMap, any>,
        next: (err?: ExtendedError | undefined) => void,
    ) {
        try {
            // Add IP address to socket object, so we can use it later
            socket.req = { ip: socket.handshake.address };
            // Initialize session
            socket.session = {
                fromSafeOrigin: true,  // Always from a safe origin due to cors settings
                languages: RequestService.parseAcceptLanguage(socket.handshake),
            };
            // Get token
            const cookies = cookie.parse(socket.handshake.headers.cookie ?? "");
            const { payload } = await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt])
            updateSessionWithTokenPayload(socket, payload);
            next();
        } catch (error) {
            return next(new Error("Unauthorized"));
        }
    }
}

/**
 * Updates current user in the session token, while 
 * respecting the token size limit.
 * @param payload The token payload
 * @returns The updated token payload
 */
export function modifyPayloadToFitUsers(payload: SessionToken): SessionToken {
    // Initialize new payload with empty users array
    const newPayload = { ...payload, users: [] } as SessionToken;
    // Store current users array
    const users = payload.users ?? [];
    // Add first user to the array
    if (users.length > 0) {
        newPayload.users.push(users[0]);
    }
    // Check size of the minimal token data (just the current user)
    let currPayloadSize = JsonWebToken.calculateBase64Length(newPayload);
    // If payload is too large, throw an error. We cannot return a token without at least one user.
    if (currPayloadSize > JsonWebToken.get().getMaxPayloadSize() || users.length === 0) {
        throw new CustomError("0545", "ErrorUnknown");
    }
    // Add the rest of the users until we run out of users or reach the limit
    for (let i = 1; i < users.length; i++) {
        const currPayload = users[i];
        currPayloadSize += JsonWebToken.calculateBase64Length(currPayload);
        if (currPayloadSize > JsonWebToken.get().getMaxPayloadSize()) {
            logger.warning(`Could not add user ${currPayload.id} to session token due to size limit`, { trace: "0547" });
            break;
        }
        newPayload.users.push(currPayload);
    }
    // Return the updated payload
    return newPayload;
}

/**
 * Updates one or more user properties in the session token.
 * Does not extend the max age of the token.
 */
export async function updateSessionCurrentUser(req: Request, res: Response, user: RecursivePartial<SessionUserToken>): Promise<undefined> {
    try {
        const { cookies } = req;
        function modifyPayload(payload: SessionToken) {
            // Make sure users array is present
            if (!Array.isArray(payload.users)) {
                payload.users = [];
            }
            // Update first user in the array with new data, making sure to strip any invalid data
            const firstUser = payload.users.length > 0 ? payload.users[0] : undefined;
            const firstUserPayload = SessionService.stripSessionUserToken(mergeDeep(firstUser, user) as SessionUserToken);
            // Add first user to the array
            if (payload.users.length === 0) {
                payload.users.push(firstUserPayload);
            } else {
                payload.users[0] = firstUserPayload;
            }
            // Add users until we run out of users or reach the limit
            return modifyPayloadToFitUsers(payload);
        }
        const { token, maxAge } = await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt], { modifyPayload })
        JsonWebToken.addToCookies(res, token, maxAge);
    } catch (error) {
        logger.error("❗️ Session token is invalid", { trace: "0447" });
    }
}

/**
 * Updates the session data with the token payload
 * @param req The request object
 * @param payload The token payload
 */
export function updateSessionWithTokenPayload(req: { session: SessionData }, payload: JwtPayload) {
    // Set token and role variables for other middleware to use
    req.session.apiToken = payload.apiToken ?? false;
    req.session.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
    req.session.timeZone = payload.timeZone ?? "UTC";
    // Keep session token users, but make sure they all have unique ids
    req.session.users = [...new Map((payload.users ?? []).map((user: SessionUserToken) => [user.id, user])).values()] as SessionUserToken[];
    // Find preferred languages for first user. Combine with languages in request header
    const firstUser = req.session.users[0];
    const firstUserLanguages = firstUser?.languages ?? [];
    if (firstUser && firstUserLanguages.length) {
        const languages = [...firstUserLanguages, ...(req.session.languages ?? [])];
        req.session.languages = [...new Set(languages)];
    }
    req.session.validToken = true;
}
