import { COOKIE, uuidValidate } from "@local/shared";
import cookie from "cookie";
import { NextFunction, Request, Response } from "express";
import jwt from "jsonwebtoken";
import { Socket } from "socket.io";
import { ExtendedError } from "socket.io/dist/namespace";
import { DefaultEventsMap } from "socket.io/dist/typed-events";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { UI_URL_REMOTE } from "../server";
import { ApiToken, BasicToken, RecursivePartial, RecursivePartialNullable, SessionData, SessionToken, SessionUserToken } from "../types";
import { isSafeOrigin } from "../utils/origin";

const SESSION_MILLI = 30 * 86400 * 1000; // 30 days

interface JwtKeys {
    privateKey: string;
    publicKey: string;
}
let jwtKeys: JwtKeys | null = null;
const getJwtKeys = (): JwtKeys => {
    // If jwtKeys is already defined, return it immediately
    if (jwtKeys) {
        return jwtKeys;
    }

    // Load the keys from process.env. They may be in a single line, so replace \n with actual newlines
    const privateKey = process.env.JWT_PRIV ?? "";
    const publicKey = process.env.JWT_PUB ?? "";

    // Check if the keys are available and log an error if not
    if (privateKey.length <= 0) {
        logger.error("JWT private key not found");
    }
    if (publicKey.length <= 0) {
        logger.error("JWT public key not found");
    }

    // Store the keys in jwtKeys for future use
    jwtKeys = { privateKey, publicKey };

    return jwtKeys;
};

/**
 * Parses a request's accept-language header
 * @param req The request
 * @returns A list of languages without any subtags
 */
const parseAcceptLanguage = (req: { headers: Record<string, any> }): string[] => {
    const acceptString = req.headers["accept-language"];
    // Default to english if not found or a wildcard
    if (!acceptString || acceptString === "*") return ["en"];
    // Strip q values
    let acceptValues = acceptString.split(",").map((lang: string) => lang.split(";")[0]);
    // Remove subtags
    acceptValues = acceptValues.map((lang: string) => lang.split("-")[0]);
    return acceptValues;
};

const verifyJwt = (token: string, publicKey: string): Promise<any> => {
    return new Promise((resolve, reject) => {
        jwt.verify(token, publicKey, { algorithms: ["RS256"] }, (error: any, payload: any) => {
            if (error) {
                reject(error);
            } else {
                resolve(payload);
            }
        });
    });
};

/**
 * Verifies if a user is authenticated, using an http cookie. 
 * Also populates helpful request properties, which can be used by endpoints
 * @param req The request object
 * @param res The response object. Has to be passed in, but it's not used.
 * @param next The next function to call.
 */
export async function authenticate(req: Request, res: Response, next: NextFunction) {
    const handleUnauthenticatedRequest = (req: Request, next: NextFunction) => {
        let error: CustomError | undefined;
        // If from unsafe origin, deny access.
        if (!req.session.fromSafeOrigin) error = new CustomError("0247", "UnsafeOriginNoApiToken", req.session.languages);
        next(error);
    };
    try {
        const { cookies } = req;
        // Initialize session
        req.session = {
            fromSafeOrigin: isSafeOrigin(req),  // Useful for handling public API requests.
            languages: parseAcceptLanguage(req), // Language fallback for error messages
        };
        req.session.languages = parseAcceptLanguage(req); // Useful for error messages
        // Check if a valid session cookie was supplied
        const token = cookies[COOKIE.Jwt];
        if (token === null || token === undefined) {
            handleUnauthenticatedRequest(req, next);
            return;
        }
        // Verify that the session token is valid and not expired
        const payload = await verifyJwt(token, getJwtKeys().publicKey);
        if (!isFinite(payload.exp) || payload.exp < Date.now()) {
            handleUnauthenticatedRequest(req, next);
            return;
        }
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
        next();

    } catch (error) {
        if (error instanceof jwt.JsonWebTokenError) {
            handleUnauthenticatedRequest(req, next);
            return;
        }
        logger.error("Error authenticating request", { trace: "0451", error });
        res.clearCookie(COOKIE.Jwt);
        next(error);
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
export const authenticateSocket = async (
    socket: Socket<DefaultEventsMap, DefaultEventsMap, DefaultEventsMap, any>,
    next: (err?: ExtendedError | undefined) => void,
) => {
    try {
        // Add IP address to socket object, so we can use it later
        socket.req = { ip: socket.handshake.address };
        // Initialize session
        socket.session = {
            fromSafeOrigin: true,  // Always from a safe origin due to cors settings
            languages: parseAcceptLanguage(socket.handshake), // Language fallback for error messages
            validToken: false,
        };
        // Get token
        const cookies = cookie.parse(socket.handshake.headers.cookie ?? "");
        const token = cookies[COOKIE.Jwt];
        if (!token) {
            return next(new Error("Unauthorized"));
        }
        // Verify that the session token is valid and not expired
        const payload = await verifyJwt(token, getJwtKeys().publicKey);
        if (!isFinite(payload.exp) || payload.exp < Date.now()) {
            throw new Error("Token expiration is invalid");
        }
        // Set token and role variables for other middleware to use
        socket.session.apiToken = payload.apiToken ?? false;
        socket.session.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
        socket.session.timeZone = payload.timeZone ?? "UTC";
        // Keep session token users, but make sure they all have unique ids
        socket.session.users = [...new Map((payload.users ?? []).map((user: SessionUserToken) => [user.id, user])).values()] as SessionUserToken[];
        // Find preferred languages for first user. Combine with languages in request header
        if (socket.session.users.length && socket.session.users[0].languages && socket.session.users[0].languages.length) {
            let languages: string[] = socket.session.users[0].languages;
            languages.push(...(socket.session.languages ?? []));
            languages = [...new Set(languages)];
            socket.session.languages = languages;
        }
        socket.session.validToken = true;
        next();
    } catch (error) {
        return next(new Error("Unauthorized"));
    }
};

/**
 * Generates the minimum data required for a session token
 */
const basicToken = (): BasicToken => ({
    iat: Date.now(),
    iss: UI_URL_REMOTE,
    exp: Date.now() + SESSION_MILLI,
});

/**
 * Generates a JSON Web Token (JWT) for user authentication (including guest access).
 * The token is added to the "res" object as a cookie.
 * @param res 
 * @param session 
 * @returns 
 */
export async function generateSessionJwt(
    res: Response,
    session: RecursivePartialNullable<Pick<SessionData, "isLoggedIn" | "languages" | "timeZone" | "users">>,
): Promise<void> {
    const tokenContents: SessionToken = {
        ...basicToken(),
        isLoggedIn: session.isLoggedIn ?? false,
        timeZone: session.timeZone ?? undefined,
        // Make sure users are unique by id
        users: [...new Map((session.users ?? []).map((user) => [user.id, {
            id: user.id,
            activeFocusMode: user.activeFocusMode ? {
                mode: {
                    id: user.activeFocusMode.mode?.id,
                    reminderList: user.activeFocusMode?.mode?.reminderList,
                },
                stopCondition: user.activeFocusMode.stopCondition,
                stopTime: user.activeFocusMode.stopTime,
            } : undefined,
            credits: user.credits + "", // Convert to string because BigInt cannot be serialized
            handle: user.handle,
            hasPremium: user.hasPremium ?? false,
            languages: user.languages ?? [],
            name: user.name ?? undefined,
            profileImage: user.profileImage ?? undefined,
            updated_at: user.updated_at,
        }])).values()],
    };
    const token = jwt.sign(tokenContents, getJwtKeys().privateKey, { algorithm: "RS256" });
    if (!res.headersSent) {
        res.cookie(COOKIE.Jwt, token, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            maxAge: SESSION_MILLI,
        });
    }
}

/**
 * Generates a JSON Web Token (JWT) for API authentication.
 * @param res 
 * @param apiToken
 * @returns 
 */
export async function generateApiJwt(res: Response, apiToken: string): Promise<void> {
    const tokenContents: ApiToken = {
        ...basicToken(),
        apiToken,
    };
    const token = jwt.sign(tokenContents, getJwtKeys().privateKey, { algorithm: "RS256" });
    if (!res.headersSent) {
        res.cookie(COOKIE.Jwt, token, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            maxAge: SESSION_MILLI,
        });
    }
}

/**
 * Update the session token with new time zone information.
 * Does not extend the max age of the token.
 */
export async function updateSessionTimeZone(req: Request, res: Response, timeZone: string): Promise<undefined> {
    if (req.session.timeZone === timeZone) return;
    const { cookies } = req;
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        logger.error("❗️ No session token found", { trace: "0006" });
        return;
    }
    try {
        const payload = await verifyJwt(token, getJwtKeys().publicKey);
        if (!isFinite(payload.exp) || payload.exp < Date.now()) {
            throw new Error("Token expiration is invalid");
        }
        const tokenContents: SessionToken = {
            ...payload,
            timeZone,
        };
        const newToken = jwt.sign(tokenContents, getJwtKeys().privateKey, { algorithm: "RS256" });

        if (!res.headersSent) {
            res.cookie(COOKIE.Jwt, newToken, {
                httpOnly: true,
                secure: process.env.NODE_ENV === "production",
                maxAge: payload.exp - Date.now(),
            });
        }
    } catch (error) {
        logger.error("❗️ Session token is invalid", { trace: "0008" });
    }
}

/**
 * Updates one or more user properties in the session token.
 * Does not extend the max age of the token.
 */
export async function updateSessionCurrentUser(req: Request, res: Response, user: RecursivePartial<SessionUserToken>): Promise<undefined> {
    const { cookies } = req;
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        logger.error("❗️ No session token found", { trace: "0445" });
        return;
    }
    try {
        const payload = await verifyJwt(token, getJwtKeys().publicKey);
        if (!isFinite(payload.exp) || payload.exp < Date.now()) {
            throw new Error("Token expiration is invalid");
        }
        const tokenContents: SessionToken = {
            ...payload,
            users: payload.users?.length > 0 ? [{
                ...payload.users[0], ...{
                    activeFocusMode: user.activeFocusMode ? {
                        mode: {
                            id: user.activeFocusMode.mode?.id,
                            reminderList: user.activeFocusMode?.mode?.reminderList,
                        },
                        stopCondition: user.activeFocusMode.stopCondition,
                        stopTime: user.activeFocusMode.stopTime,
                    } : undefined,
                    handle: user.handle,
                    hasPremium: user.hasPremium ?? false,
                    languages: user.languages ?? [],
                    name: user.name ?? undefined,
                    profileImage: user.profileImage ?? undefined,
                    updated_at: user.updated_at,
                },
            }, ...payload.users.slice(1)] : [],
        };
        const newToken = jwt.sign(tokenContents, getJwtKeys().privateKey, { algorithm: "RS256" });

        if (!res.headersSent) {
            res.cookie(COOKIE.Jwt, newToken, {
                httpOnly: true,
                secure: process.env.NODE_ENV === "production",
                maxAge: payload.exp - Date.now(),
            });
        }
    } catch (error) {
        logger.error("❗️ Session token is invalid", { trace: "0447" });
    }
}

/**
 * Middleware that restricts access to logged in users
 */
export async function requireLoggedIn(req: Request, _: unknown, next: NextFunction) {
    let error: CustomError | undefined;
    if (!req.session.isLoggedIn) error = new CustomError("0018", "NotLoggedIn", req.session.languages);
    next(error);
}

/**
 * Finds current user in Request object. Also validates that the user data is valid
 * @param req Request object
 * @returns First userId in Session object, or null if not found/invalid
 */
export const getUser = (session: { users?: SessionData["users"] }): SessionUserToken | null => {
    if (!session || !Array.isArray(session?.users) || session.users.length === 0) return null;
    const user = session.users[0];
    return user !== undefined && typeof user.id === "string" && uuidValidate(user.id) ? user : null;
};

export type RequestConditions = {
    /**
     * Checks if the request is coming from an API token directly
     */
    isApiRoot?: boolean;
    /**
     * Checks if the request is coming from a user logged in via an API token, or the official Vrooli app/website
     * This allows other services to use Vrooli as a backend, in a way that 
     * we can price it accordingly.
     */
    isUser?: boolean;
    /**
     * Checks if the request is coming from a user logged in via the official Vrooli app/website
     */
    isOfficialUser?: boolean;
}

type AssertRequestFromResult<T extends RequestConditions> = T extends { isUser: true } | { isOfficialUser: true } ? SessionUserToken : undefined;

/**
 * Asserts that a request meets the specifiec requirements TODO need better api token validation, like uuidValidate
 * @param req Object with request data
 * @param conditions The conditions to check
 * @returns user data, if isUser or isOfficialUser is true
 */
export const assertRequestFrom = <Conditions extends RequestConditions>(req: { session: SessionData }, conditions: Conditions): AssertRequestFromResult<Conditions> => {
    const { session } = req;
    // Determine if user data is found in the request
    const userData = getUser(session);
    const hasUserData = session.isLoggedIn === true && Boolean(userData);
    // Determine if api token is supplied
    const hasApiToken = session.apiToken === true;
    // Check isApiRoot condition
    if (conditions.isApiRoot !== undefined) {
        const isApiRoot = hasApiToken && !hasUserData;
        if (conditions.isApiRoot === true && !isApiRoot) throw new CustomError("0265", "MustUseApiToken", session.languages);
        if (conditions.isApiRoot === false && isApiRoot) throw new CustomError("0266", "MustNotUseApiToken", session.languages);
    }
    // Check isUser condition
    if (conditions.isUser !== undefined) {
        const isUser = hasUserData && (hasApiToken || session.fromSafeOrigin === true);
        if (conditions.isUser === true && !isUser) throw new CustomError("0267", "NotLoggedIn", session.languages);
        if (conditions.isUser === false && isUser) throw new CustomError("0268", "NotLoggedIn", session.languages);
    }
    // Check isOfficialUser condition
    if (conditions.isOfficialUser !== undefined) {
        const isOfficialUser = hasUserData && !hasApiToken && session.fromSafeOrigin === true;
        if (conditions.isOfficialUser === true && !isOfficialUser) throw new CustomError("0269", "NotLoggedInOfficial", session.languages);
        if (conditions.isOfficialUser === false && isOfficialUser) throw new CustomError("0270", "NotLoggedInOfficial", session.languages);
    }
    return conditions.isUser === true || conditions.isOfficialUser === true ? userData as any : undefined;
};
