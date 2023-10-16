import { COOKIE, uuidValidate } from "@local/shared";
import cookie from "cookie";
import { NextFunction, Request, Response } from "express";
import fs from "fs";
import jwt from "jsonwebtoken";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { ApiToken, BasicToken, RecursivePartial, SessionData, SessionToken, SessionUserToken } from "../types";
import { isSafeOrigin } from "../utils/origin";

const SESSION_MILLI = 30 * 86400 * 1000; // 30 days

let privateKey = "";
const privateKeyFile = `${process.env.PROJECT_DIR}/jwt_priv.pem`;
if (fs.existsSync(privateKeyFile)) {
    privateKey = fs.readFileSync(privateKeyFile, "utf8");
} else {
    logger.error(`Could not find private key at ${privateKeyFile}`);
}

let publicKey = "";
const publicKeyFile = `${process.env.PROJECT_DIR}/jwt_pub.pem`;
if (fs.existsSync(publicKeyFile)) {
    publicKey = fs.readFileSync(publicKeyFile, "utf8");
} else {
    logger.error(`Could not find public key at ${publicKeyFile}`);
}

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
    let acceptValues = acceptString.split(",").map((lang) => lang.split(";")[0]);
    // Remove subtags
    acceptValues = acceptValues.map((lang) => lang.split("-")[0]);
    return acceptValues;
};

/**
 * Verifies if a user is authenticated, using an http cookie. 
 * Also populates helpful request properties, which can be used by endpoints
 * @param req The request object
 * @param res The response object. Has to be passed in, but it's not used.
 * @param next The next function to call.
 */
export async function authenticate(req: Request, res: Response, next: NextFunction) {
    try {
        const { cookies } = req;
        req.session = {} as any;
        // Check if request is coming from a safe origin. 
        // This is useful for handling public API requests.
        req.session.fromSafeOrigin = isSafeOrigin(req);
        // Add user's preferred language to the request. Later on we'll 
        // use the preferred languages set in the user's account for localization, 
        // but this is a good fallback.
        req.session.languages = parseAcceptLanguage(req);
        // Check if a valid session cookie was supplied
        const token = cookies[COOKIE.Jwt];
        if (token === null || token === undefined) {
            // If from unsafe origin, deny access.
            let error: CustomError | undefined;
            if (!req.session.fromSafeOrigin) error = new CustomError("0247", "UnsafeOriginNoApiToken", req.session.languages);
            next(error);
            return;
        }
        // Verify that the session token is valid
        jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error: any, payload: any) => {
            try {
                if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
                    // If from unsafe origin, deny access.
                    let error: CustomError | undefined;
                    if (!req.session.fromSafeOrigin) error = new CustomError("0248", "UnsafeOriginNoApiToken", req.session.languages);
                    next(error);
                    return;
                }
                // Now, set token and role variables for other middleware to use
                req.session.apiToken = payload.apiToken ?? false;
                req.session.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
                req.session.timeZone = payload.timeZone ?? "UTC";
                // Users, but make sure they all have unique ids
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
                logger.error("Error verifying token", { trace: "0450", error });
                // Remove the cookie
                res.clearCookie(COOKIE.Jwt);
                next(error);
            }
        });
    } catch (error) {
        logger.error("Error authenticating request", { trace: "0451", error });
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
 * @param {Object} socket - The socket object
 * @param {Function} next - The next function to call
 */
export const authenticateSocket = async (socket, next) => {
    try {
        // Add IP address to socket object, so we can use it later
        socket.req = { ip: socket.handshake.address };
        const cookies = cookie.parse(socket.handshake.headers.cookie);
        const token = cookies[COOKIE.Jwt];
        socket.session = {};
        // Currently, sockets are always from a safe origin, because of cors settings
        socket.session.fromSafeOrigin = true;
        // Add user's preferred language to the request. Later on we'll 
        // use the preferred languages set in the user's account for localization, 
        // but this is a good fallback.
        socket.session.languages = parseAcceptLanguage(socket.handshake);
        if (!token) {
            // No token, unauthorized
            return next(new Error("Unauthorized"));
        }

        jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error: any, payload: any) => {
            if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
                // Invalid token, unauthorized
                return next(new Error("Unauthorized"));
            }
            // Now, set token and role variables for other middleware to use
            socket.session.apiToken = payload.apiToken ?? false;
            socket.session.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
            socket.session.timeZone = payload.timeZone ?? "UTC";
            // Users, but make sure they all have unique ids
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
        });
    } catch (error) {
        // Something went wrong, deny connection
        return next(new Error("Unauthorized"));
    }
};

/**
 * Generates the minimum data required for a session token
 */
const basicToken = (): BasicToken => ({
    iat: Date.now(),
    iss: "https://vrooli.com/",
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
    session: RecursivePartial<Pick<SessionData, "isLoggedIn" | "languages" | "timeZone" | "users">>,
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
        }])).values()],
    };
    const token = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        maxAge: SESSION_MILLI,
    });
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
    const token = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        maxAge: SESSION_MILLI,
    });
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
    jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.error("❗️ Session token is invalid", { trace: "0008" });
            return;
        }
        const tokenContents: SessionToken = {
            ...payload,
            timeZone,
        };
        const newToken = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
        res.cookie(COOKIE.Jwt, newToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            // Max age should be the same as the old token
            maxAge: payload.exp - Date.now(),
        });
    });
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
    jwt.verify(token, publicKey, { algorithms: ["RS256"] }, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.error("❗️ Session token is invalid", { trace: "0447" });
            return;
        }
        const tokenContents: SessionToken = {
            ...payload,
            users: payload.users?.length > 0 ? [{
                ...payload.users[0], ...{
                    activeFocusMode: user.activeFocusMode ? {
                        mode: {
                            id: user.activeFocusMode.mode?.id,
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
        const newToken = jwt.sign(tokenContents, privateKey, { algorithm: "RS256" });
        res.cookie(COOKIE.Jwt, newToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === "production",
            // Max age should be the same as the old token
            maxAge: payload.exp - Date.now(),
        });
    });
}

/**
 * Middleware that restricts access to logged in users
 */
export async function requireLoggedIn(req: Request, _: any, next: any) {
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
