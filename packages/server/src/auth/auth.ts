import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { CODE, COOKIE } from '@shared/consts';
import { CustomError } from '../events/error';
import { Session, SessionUser } from '../schema/types';
import { RecursivePartial } from '../types';
import { genErrorCode, logger, LogLevel } from '../events/logger';
import { isSafeOrigin } from '../utils';
import { uuidValidate } from '@shared/uuid';
import { getUser } from '../models';

const SESSION_MILLI = 30 * 86400 * 1000;

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
    // Check if a valid session cookie was supplied
    const token = cookies[COOKIE.Jwt];
    if (token === null || token === undefined) {
        // If from unsafe origin, deny access.
        let error: CustomError | undefined;
        if (!req.fromSafeOrigin) error = new CustomError(CODE.Unauthorized, 'Unsafe origin with expired/missing API token', { code: genErrorCode('0247') });
        next(error);
        return;
    }
    if (!process.env.JWT_SECRET) {
        logger.log(LogLevel.error, '❗️ JWT_SECRET not set! Please check .env file', { code: genErrorCode('0003') });
        return;
    }
    // Verify that the session token is valid
    jwt.verify(token, process.env.JWT_SECRET, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            // If from unsafe origin, deny access.
            let error: CustomError | undefined;
            if (!req.fromSafeOrigin) error = new CustomError(CODE.Unauthorized, 'Unsafe origin with expired/missing API token', { code: genErrorCode('0248') });
            next(error);
            return;
        }
        // Now, set token and role variables for other middleware to use
        req.apiToken = payload.apiToken ?? false;
        req.isLoggedIn = payload.isLoggedIn === true && Array.isArray(payload.users) && payload.users.length > 0;
        req.timeZone = payload.timeZone ?? 'UTC';
        // Users, but make sure they all have unique ids
        req.users = [...new Map((payload.users ?? []).map((user: SessionUser) => [user.id, user])).values()] as SessionUser[];
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
        logger.log(LogLevel.error, '❗️ JWT_SECRET not set! Please check .env file', { code: genErrorCode('0004') });
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
        logger.log(LogLevel.error, '❗️ JWT_SECRET not set! Please check .env file', { code: genErrorCode('0005') });
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
        logger.log(LogLevel.error, '❗️ No session token found', { code: genErrorCode('0006') });
        return;
    }
    if (!process.env.JWT_SECRET) {
        logger.log(LogLevel.error, '❗️ JWT_SECRET not set! Please check .env file', { code: genErrorCode('0007') });
        return;
    }
    jwt.verify(token, process.env.JWT_SECRET, async (error: any, payload: any) => {
        if (error || isNaN(payload.exp) || payload.exp < Date.now()) {
            logger.log(LogLevel.error, '❗️ Session token is invalid', { code: genErrorCode('0008') });
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
    if (!req.isLoggedIn) error = new CustomError(CODE.Unauthorized, 'You must be logged in to access this resource', { code: genErrorCode('0018') });
    next(error);
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
        if (conditions.isApiRoot === true && !isApiRoot) throw new CustomError(CODE.Unauthorized, 'Call must be made from an API token.', { code: genErrorCode('0265') });
        if (conditions.isApiRoot === false && isApiRoot) throw new CustomError(CODE.Unauthorized, 'Call cannot be made from an API token.', { code: genErrorCode('0266') });
    }
    // Check isUser condition
    if (conditions.isUser !== undefined) {
        const isUser = hasUserData && (hasApiToken || req.fromSafeOrigin === true)
        if (conditions.isUser === true && !isUser) throw new CustomError(CODE.Unauthorized, 'Must be logged in.', { code: genErrorCode('0267') });
        if (conditions.isUser === false && isUser) throw new CustomError(CODE.Unauthorized, 'Must be logged in.', { code: genErrorCode('0268') });
    }
    // Check isOfficialUser condition
    if (conditions.isOfficialUser !== undefined) {
        const isOfficialUser = hasUserData && !hasApiToken && req.fromSafeOrigin === true;
        if (conditions.isOfficialUser === true && !isOfficialUser) throw new CustomError(CODE.Unauthorized, 'Must be logged in via the official Vrooli app/website.', { code: genErrorCode('0269') });
        if (conditions.isOfficialUser === false && isOfficialUser) throw new CustomError(CODE.Unauthorized, 'Must be logged in via the official Vrooli app/website.', { code: genErrorCode('0270') });
    }
    return conditions.isUser === true || conditions.isOfficialUser === true ? userData as any : undefined;
}