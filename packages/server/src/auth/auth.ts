import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { CODE, COOKIE } from '@shared/consts';
import { CustomError } from '../error';
import { Session, SessionUser } from '../schema/types';
import { RecursivePartial } from '../types';
import { genErrorCode, logger, LogLevel } from '../logger';
import { isSafeOrigin } from '../utils';
import { getUserId } from '../models';
import { uuidValidate } from '@shared/uuid';

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
        if (!req.fromSafeOrigin) throw new CustomError(CODE.Unauthorized, 'Unsafe origin with expired/missing API token', { code: genErrorCode('0247') });
        else next();
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
            if (!req.fromSafeOrigin) throw new CustomError(CODE.Unauthorized, 'Unsafe origin with expired/missing API token', { code: genErrorCode('0248') });
            else next();
            return;
        }
        // Now, set token and role variables for other middleware to use
        req.apiToken = payload.apiToken ?? false;
        req.isLoggedIn = payload.isLoggedIn === true && req.fromSafeOrigin === true && Array.isArray(req.users) && req.users.length > 0;
        req.users = req.users ?? [];
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
        users: session.users ?? [] as any[],
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
 * Middleware that restricts access to logged in users
 */
export async function requireLoggedIn(req: Request, _: any, next: any) {
    if (!req.isLoggedIn) return new CustomError(CODE.Unauthorized, 'You must be logged in to access this resource', { code: genErrorCode('0018') });
    next();
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

/**
 * Asserts that a request meets the specifiec requirements TODO need better api token validation, like uuidValidate
 * @param req The request object
 * @param conditions The conditions to check
 */
export const assertRequestFrom = (req: Request, conditions: RequestConditions) => {
    // Determine if user data is found in the request
    const userId = getUserId(req);
    const hasUserData = req.isLoggedIn === true && uuidValidate(userId);
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
}