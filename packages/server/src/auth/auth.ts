import { ApiKeyPermission, COOKIE, HttpStatus, mergeDeep, SECONDS_1_MS, SessionUser } from "@local/shared";
import cookie from "cookie";
import { NextFunction, Request, Response } from "express";
import { type JwtPayload } from "jsonwebtoken";
import { Socket } from "socket.io";
// eslint-disable-next-line import/extensions
import { ExtendedError } from "socket.io/dist/namespace";
// eslint-disable-next-line import/extensions
import { DefaultEventsMap } from "socket.io/dist/typed-events";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ApiToken, RecursivePartial, SessionData, SessionToken } from "../types.js";
import { ResponseService } from "../utils/response.js";
import { JsonWebToken, REFRESH_TOKEN_EXPIRATION_MS } from "./jwt.js";
import { RequestService } from "./request.js";
import { SessionService } from "./session.js";

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
     * Checks if a token can be refreshed.
     * @param payload The access token payload
     * @returns True if the token can be refreshed
     */
    static async canRefreshToken(payload: Pick<SessionToken, "users">): Promise<boolean> {
        // Check that payload has correct fields
        if (typeof payload !== "object") {
            return false;
        }
        const user = SessionService.getUser({ session: payload });
        if (!user) {
            return false;
        }
        const sessionId = user.session?.id;
        const lastRefreshAt = user.session?.lastRefreshAt; // An ISO string
        if (!sessionId || !lastRefreshAt) {
            return false;
        }
        // Fetch session from database
        const storedSession = await DbProvider.get().session.findUnique({
            where: { id: sessionId },
            select: {
                expires_at: true,
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
        if (storedLastRefreshAt !== new Date(lastRefreshAt).getTime()) {
            return false;
        }
        // Make sure we haven't passed the expiration time
        const hasExpired = storedSession.expires_at < new Date();
        if (hasExpired) {
            return false;
        }
        return true;
    }

    static isTokenCorrectType(token: unknown): token is string {
        return typeof token === "string";
    }

    static isAccessTokenExpired(payload: Pick<SessionData, "accessExpiresAt">): boolean {
        // If field is missing/invalid, assume it's expired
        if (typeof payload.accessExpiresAt !== "number") {
            return true;
        }

        const accessExpiresAt = payload.accessExpiresAt * SECONDS_1_MS;
        const currentTime = Date.now().valueOf();
        return currentTime > accessExpiresAt;
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
        // (typically because the user is not logged in, or the token fully expired)
        if (!AuthTokensService.isTokenCorrectType(token)) {
            throw new Error("NoSessionToken");
        }
        // Verify the token. This will throw an error if the token is invalid or expired 
        // (expired as in fully expired, not the accessExpiresAt part)
        const payload = await JsonWebToken.get().verify(token) as SessionToken;
        // Check if the token is expired
        const isExpired = AuthTokensService.isAccessTokenExpired(payload);
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
                };
            }
            // Otherwise, return the existing token
            return {
                maxAge,
                payload,
                token,
            };
        }
        // Try to refresh the token if it's expired
        const isRefreshable = await this.canRefreshToken(payload);
        if (!isRefreshable) {
            throw new CustomError("0573", "SessionExpired");
        }
        // Issue a new access token
        const withAdditionalData = {
            ...getPayloadWithAdditionalData(payload),
            ...JsonWebToken.createAccessExpiresAt(),
        };
        const newToken = JsonWebToken.get().sign(withAdditionalData);
        return {
            maxAge: REFRESH_TOKEN_EXPIRATION_MS,
            payload: withAdditionalData,
            token: newToken,
        };
    }

    /**
     * Generates a JSON Web Token (JWT) for API authentication.
     * @param res 
     * @param apiToken
     * @param permissions The permissions of the API key    
     * @param userId The ID of the user that the API key belongs to
     */
    static async generateApiToken(res: Response, apiToken: string, permissions: Record<ApiKeyPermission, boolean>, userId: string): Promise<void> {
        const tokenContents: ApiToken = {
            ...JsonWebToken.get().basicToken(),
            ...JsonWebToken.createAccessExpiresAt(),
            apiToken,
            permissions,
            userId,
        };
        const token = JsonWebToken.get().sign(tokenContents);
        JsonWebToken.addToCookies(res, token, REFRESH_TOKEN_EXPIRATION_MS);
    }

    /**
     * Generates a JSON Web Token (JWT) for user authentication (including guest access).
     * The token is added to the "res" object as a cookie.
     * @param res 
     * @param session 
     */
    static async generateSessionToken(
        res: Response,
        session: Pick<SessionData, "accessExpiresAt" | "isLoggedIn" | "languages" | "timeZone" | "users">,
    ): Promise<void> {
        // Start with minimal token data
        let payload: SessionToken = {
            ...JsonWebToken.get().basicToken(),
            accessExpiresAt: session.accessExpiresAt ?? 0,
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

        JsonWebToken.addToCookies(res, token, REFRESH_TOKEN_EXPIRATION_MS);
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
     * @param res The response object
     * @param next The next function to call.
     */
    static async authenticateRequest(req: Request, res: Response, next: NextFunction) {
        const { cookies } = req;
        // Initialize session
        req.session = {
            fromSafeOrigin: RequestService.get().isSafeOrigin(req),
            languages: RequestService.parseAcceptLanguage(req),
        };
        // If from unsafe origin, deny access.
        if (!req.session.fromSafeOrigin && req.originalUrl !== "/healthcheck") {
            const trace = "0451";
            logger.error("Error authenticating request", { trace });
            return ResponseService.sendError(res, { trace, code: "UnsafeOrigin" }, HttpStatus.Forbidden);
        }
        try {
            // Authenticate token if it exists
            const token = cookies[COOKIE.Jwt];
            if (AuthTokensService.isTokenCorrectType(token)) {
                const { maxAge, payload, token: authenticatedToken } = await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt]);
                updateSessionWithTokenPayload(req, payload);
                // If the token changed, update the cookie
                if (authenticatedToken !== token) {
                    JsonWebToken.addToCookies(res, authenticatedToken, maxAge);
                }
            }
        } catch (error) {
            res.clearCookie(COOKIE.Jwt);
        } finally {
            // Always continue with the request. We'll let the endpoint handlers determine if they'll let a signed out user access them.
            next();
        }
    }

    /**
     * Middleware to authenticate a user via JWT in a WebSocket handshake.
     * 
     * NOTE: This should only be called when first connecting to the socket. 
     * And since this happens in a request (which is already authenticated), 
     * the only thing we need to do is make sure the session is set up correctly.
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
            const token = cookies[COOKIE.Jwt];
            const { maxAge, payload } = await AuthTokensService.authenticateToken(token);

            // Update socket.session with token payload
            updateSessionWithTokenPayload(socket, payload);
            // Override access expiration time from short expiration (that we use to refresh the token) to 
            // long expiration (the full expiration time of the token). This is done because we don't refresh 
            // websocket connections. Instead, we:
            // - Manually disconnect them when the session is revoked
            // - Check the long expiration before emitting events to the socket.
            // This combination gives the same functionality as refreshing the token, but without the need to
            // actually change the http cookie (which we can't do in a socket handshake).
            const expiresAt = Date.now() + maxAge;
            socket.session.accessExpiresAt = expiresAt;
            // No need to update the cookie here, as this is done by `authenticateRequest` 
            // (and we can't set cookies in a socket handshake anyway)
            next();
        } catch (error) {
            // Continue with the request. The socket handlers themselves can determine if they'll let a signed out user access them.
            return next();
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
    // If payload is too large, throw an error
    if (currPayloadSize > JsonWebToken.get().getMaxPayloadSize()) {
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
export async function updateSessionCurrentUser(req: Request, res: Response, user: RecursivePartial<SessionUser>): Promise<undefined> {
    function modifyPayload(payload: SessionToken) {
        // Make sure users array is present
        if (!Array.isArray(payload.users)) {
            payload.users = [];
        }
        if (payload.users.length === 0) {
            return payload;
        }
        // Update first user in the array with new data, making sure to strip any invalid data
        payload.users[0] = mergeDeep(payload.users[0], user) as SessionUser;
        // Add users until we run out of users or reach the limit
        return modifyPayloadToFitUsers(payload);
    }
    try {
        const { cookies } = req;
        const { token, maxAge } = await AuthTokensService.authenticateToken(cookies[COOKIE.Jwt], { modifyPayload });
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
    req.session.timeZone = payload.timeZone ?? req.session.timeZone;
    // Set API token permissions if they exist
    if (payload.apiToken && payload.permissions) {
        req.session.permissions = payload.permissions;
        req.session.userId = payload.userId;
    }
    // Keep session token users, but make sure they all have unique ids
    req.session.users = [...new Map((payload.users ?? []).map((user: SessionUser) => [user.id, user])).values()] as SessionUser[];
    // Find preferred languages for first user. Combine with languages in request header
    const firstUser = req.session.users[0];
    const firstUserLanguages = firstUser?.languages ?? [];
    if (firstUser && firstUserLanguages.length) {
        const languages = [...firstUserLanguages, ...(req.session.languages ?? [])];
        req.session.languages = [...new Set(languages)];
    }
    req.session.validToken = true;
}
