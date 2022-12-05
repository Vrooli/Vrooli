import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { COOKIE } from '@shared/consts';
import { CustomError } from '../events/error';
import { Session, SessionUser } from '../endpoints/types';
import { RecursivePartial } from '../types';
import { logger } from '../events/logger';
import { isSafeOrigin } from '../utils';
import { uuidValidate } from '@shared/uuid';

const SESSION_MILLI = 30 * 86400 * 1000;

/**
 * Parses a request's accept-language header
 * @param req The request
 * @returns A list of languages without any subtags
 */
const parseAcceptLanguage = (req: Request): string[] => {
    let acceptString = req.headers['accept-language'];
    // Default to english if not found or a wildcard
    if (!acceptString || acceptString === '*') return ['en'];
    // Strip q values
    let acceptValues = acceptString.split(',').map((lang) => lang.split(';')[0]);
    // Remove subtags
    acceptValues = acceptValues.map((lang) => lang.split('-')[0]);
    return acceptValues;
};

/**
 * Verifies if a user is authenticated, using an http cookie. 
 * Also populates helpful request properties, which can be used by endpoints
 * @param req The request object
 * @param _ The response object. Has to be passed in, but it's not used.
 * @param next The next function to call.
 */
export async function authenticate(req: Request, _: Response, next: NextFunction) {
    const { cookies } = req;
    // Check if request is coming from a safe origin. 
    // This is useful for handling public API requests.
    req.fromSafeOrigin = isSafeOrigin(req);
    // Add user's preferred language to the request. Later on we'll 
    // use the preferred languages set in the user's account for localization, 
    // but this is a good fallback.
    req.languages = parseAcceptLanguage(req);
    // Check if a valid session cookie was supplied
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        // If from unsafe origin, deny access.
        let error: CustomError | undefined;
        if (!req.fromSafeOrigin) error = new CustomError('0247', 'UnsafeOriginNoApiToken', req.languages);
        next(error);
        return;
    }
    if (!process.env.JWT_SECRET) {
        logger.error('❗️ JWT_SECRET not set! Please check .env file', { trace: '0003' });
        return;
    }
    // Verify that the session token is valid
    jwt.verify(token, process.env.JWT_SECRET, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            // If from unsafe origin, deny access.
            let error: CustomError | undefined;
            if (!req.fromSafeOrigin) error = new CustomError('0248', 'UnsafeOriginNoApiToken', req.languages);
            next(error);
            return;
        }
        // Now, set token and role variables for other middleware to use
        req.apiToken = payload.apiToken ?? false;
        req.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
        req.timeZone = payload.timeZone ?? 'UTC';
        // Users, but make sure they all have unique ids
        req.users = [...new Map((payload.users ?? []).map((user: SessionUser) => [user.id, user])).values()] as SessionUser[];
        // Find preferred languages for first user. Combine with languages in request header
        if (req.users.length && req.users[0].languages && req.users[0].languages.length) {
            let languages: string[] = req.users[0].languages
            languages.push(...req.languages);
            languages = [...new Set(languages)];
            req.languages = languages;
        }
        req.validToken = true;
        next();
    })
}

interface BasicToken {
    iat: number;
    iss: string;
    exp: number;
}
interface SessionToken extends BasicToken {
    isLoggedIn: boolean;
    timeZone?: string;
    // Supports logging in with multiple accounts
    users: SessionUser[];
}
interface ApiToken extends BasicToken {
    apiToken: string;
};

/**
 * Generates the minimum data required for a session token
 */
const basicToken = (): BasicToken => ({
    iat: Date.now(),
    iss: `https://app.vrooli.com/`,
    exp: Date.now() + SESSION_MILLI,
})

/**
 * Generates a JSON Web Token (JWT) for user authentication (including guest access).
 * The token is added to the "res" object as a cookie.
 * @param res 
 * @param session 
 * @returns 
 */
export async function generateSessionJwt(res: Response, session: RecursivePartial<Session>): Promise<undefined> {
    const tokenContents: SessionToken = {
        ...basicToken(),
        isLoggedIn: session.isLoggedIn ?? false,
        timeZone: session.timeZone ?? undefined,
        // Make sure users are unique by id
        users: [...new Map((session.users ?? []).map((user: SessionUser) => [user.id, user])).values()],
    }
    if (!process.env.JWT_SECRET) {
        logger.error('❗️ JWT_SECRET not set! Please check .env file', { trace: '0004' });
        return;
    }
    const token = jwt.sign(tokenContents, process.env.JWT_SECRET);
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        maxAge: SESSION_MILLI
    });
}

/**
 * Generates a JSON Web Token (JWT) for API authentication.
 * @param res 
 * @param apiToken
 * @returns 
 */
export async function generateApiJwt(res: Response, apiToken: string): Promise<undefined> {
    const tokenContents: ApiToken = {
        ...basicToken(),
        apiToken,
    }
    if (!process.env.JWT_SECRET) {
        logger.error('❗️ JWT_SECRET not set! Please check .env file', { trace: '0005' });
        return;
    }
    const token = jwt.sign(tokenContents, process.env.JWT_SECRET);
    res.cookie(COOKIE.Jwt, token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        maxAge: SESSION_MILLI
    });
}

/**
 * Update the session token with new time zone information.
 * Does not extend the max age of the token.
 */
export async function updateSessionTimeZone(req: Request, res: Response, timeZone: string): Promise<undefined> {
    if (req.timeZone === timeZone) return;
    const { cookies } = req;
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        logger.error('❗️ No session token found', { trace: '0006' });
        return;
    }
    if (!process.env.JWT_SECRET) {
        logger.error('❗️ JWT_SECRET not set! Please check .env file', { trace: '0007' });
        return;
    }
    jwt.verify(token, process.env.JWT_SECRET, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.error('❗️ Session token is invalid', { trace: '0008' });
            return;
        }
        const tokenContents: SessionToken = {
            ...payload,
            timeZone,
        }
        const newToken = jwt.sign(tokenContents, process.env.JWT_SECRET as string);
        res.cookie(COOKIE.Jwt, newToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === 'production',
            // Max age should be the same as the old token
            maxAge: payload.exp - Date.now(),
        });
    })
}

/**
 * Middleware that restricts access to logged in users
 */
export async function requireLoggedIn(req: Request, _: any, next: any) {
    let error: CustomError | undefined;
    if (!req.isLoggedIn) error = new CustomError('0018', 'NotLoggedIn', req.languages);
    next(error);
}

/**
 * Finds current user in Request object. Also validates that the user data is valid
 * @param req Request object
 * @returns First userId in Session object, or null if not found/invalid
 */
export const getUser = (req: { users?: Request['users'] }): SessionUser | null => {
    if (!req || !Array.isArray(req?.users) || req.users.length === 0) return null;
    const user = req.users[0];
    return typeof user.id === 'string' && uuidValidate(user.id) ? user : null;
}

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

type AssertRequestFromResult<T extends RequestConditions> = T extends { isUser: true } | { isOfficialUser: true } ? SessionUser : undefined;

/**
 * Asserts that a request meets the specifiec requirements TODO need better api token validation, like uuidValidate
 * @param req The request object
 * @param conditions The conditions to check
 * @returns user data, if isUser or isOfficialUser is true
 */
export const assertRequestFrom = <Conditions extends RequestConditions>(req: Request, conditions: Conditions): AssertRequestFromResult<Conditions> => {
    // Determine if user data is found in the request
    const userData = getUser(req);
    const hasUserData = req.isLoggedIn === true && Boolean(userData);
    // Determine if api token is supplied
    const hasApiToken = req.apiToken === true;
    // Check isApiRoot condition
    if (conditions.isApiRoot !== undefined) {
        const isApiRoot = hasApiToken && !hasUserData;
        if (conditions.isApiRoot === true && !isApiRoot) throw new CustomError('0265', 'MustUseApiToken', req.languages);
        if (conditions.isApiRoot === false && isApiRoot) throw new CustomError('0266', 'MustNotUseApiToken', req.languages);
    }
    // Check isUser condition
    if (conditions.isUser !== undefined) {
        const isUser = hasUserData && (hasApiToken || req.fromSafeOrigin === true)
        if (conditions.isUser === true && !isUser) throw new CustomError('0267', 'NotLoggedIn', req.languages);
        if (conditions.isUser === false && isUser) throw new CustomError('0268', 'NotLoggedIn', req.languages);
    }
    // Check isOfficialUser condition
    if (conditions.isOfficialUser !== undefined) {
        const isOfficialUser = hasUserData && !hasApiToken && req.fromSafeOrigin === true;
        if (conditions.isOfficialUser === true && !isOfficialUser) throw new CustomError('0269', 'NotLoggedInOfficial', req.languages);
        if (conditions.isOfficialUser === false && isOfficialUser) throw new CustomError('0270', 'NotLoggedInOfficial', req.languages);
    }
    return conditions.isUser === true || conditions.isOfficialUser === true ? userData as any : undefined;
}