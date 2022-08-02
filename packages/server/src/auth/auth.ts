import jwt from 'jsonwebtoken';
import { Request, Response, NextFunction } from 'express';
import { CODE, COOKIE } from '@local/shared';
import { CustomError } from '../error';
import { Session } from '../schema/types';
import { RecursivePartial } from '../types';
import { genErrorCode, logger, LogLevel } from '../logger';
import { isSafeOrigin } from '../utils';

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
        req.isLoggedIn = payload.isLoggedIn === true && req.fromSafeOrigin === true;
        req.languages = payload.languages ?? [];
        req.userId = payload.userId ?? null;
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
    languages: string[] | null;
    userId: string | null;
}
interface ApiToken extends BasicToken {
    apiToken: boolean;
    userId: string;
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
        languages: (session.languages as string[]) ?? [],
        userId: session.id ?? null,
    }
    if (!process.env.JWT_SECRET) {
        logger.log(LogLevel.error, '❗️ JWT_SECRET not set! Please check .env file', { code: genErrorCode('0004') });
        return;
    }
    console.log('signing token contents:', tokenContents);
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
 * @param userId
 * @param apiToken
 * @returns 
 */
export async function generateApiJwt(res: Response, userId: string, apiToken: boolean): Promise<undefined> {
    const tokenContents: ApiToken = {
        ...basicToken(),
        apiToken,
        userId,
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
export async function requireLoggedIn(req: any, _: any, next: any) {
    if (!req.isLoggedIn) return new CustomError(CODE.Unauthorized, 'You must be logged in to access this resource', { code: genErrorCode('0018') });
    next();
}